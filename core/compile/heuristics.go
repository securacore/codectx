package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/securacore/codectx/core/manifest"
)

const heuristicsFile = "heuristics.yml"

// tokensPerByte is the rough conversion factor from bytes to tokens.
// ~4 characters per token for English/Markdown content.
const tokensPerByte = 0.25

// Heuristics is the sidecar metadata generated during compilation.
// It provides size and token estimates for the compiled documentation set.
// Not part of the AI loading protocol; used by tooling and the generated
// README for richer context.
type Heuristics struct {
	CompiledAt string             `yaml:"compiled_at"`
	Totals     HeuristicsTotals   `yaml:"totals"`
	Sections   HeuristicsSections `yaml:"sections"`
	Packages   []PackageStats     `yaml:"packages"`
}

// HeuristicsTotals holds aggregate stats for the entire documentation set.
type HeuristicsTotals struct {
	Entries         int `yaml:"entries"`
	Objects         int `yaml:"objects"`
	SizeBytes       int `yaml:"size_bytes"`
	EstimatedTokens int `yaml:"estimated_tokens"`
	AlwaysLoad      int `yaml:"always_load"`
}

// HeuristicsSections holds per-section stats.
type HeuristicsSections struct {
	Foundation *SectionStats `yaml:"foundation,omitempty"`
	Topics     *SectionStats `yaml:"topics,omitempty"`
	Prompts    *SectionStats `yaml:"prompts,omitempty"`
	Plans      *SectionStats `yaml:"plans,omitempty"`
}

// SectionStats holds stats for a single section.
type SectionStats struct {
	Entries         int `yaml:"entries"`
	SizeBytes       int `yaml:"size_bytes"`
	EstimatedTokens int `yaml:"estimated_tokens"`
	AlwaysLoad      int `yaml:"always_load,omitempty"`
}

// PackageStats holds stats for a single package source.
type PackageStats struct {
	Name            string `yaml:"name"`
	Entries         int    `yaml:"entries"`
	SizeBytes       int    `yaml:"size_bytes"`
	EstimatedTokens int    `yaml:"estimated_tokens"`
}

// estimateTokens converts byte count to estimated token count.
func estimateTokens(bytes int) int {
	return int(float64(bytes) * tokensPerByte)
}

// generateHeuristics builds heuristics metadata from the unified manifest
// and stored objects. It reads object sizes from the object store to
// compute byte counts and token estimates.
func generateHeuristics(
	unified *manifest.Manifest,
	pathToHash map[string]string,
	provenance map[string]string,
	objectsDir string,
) *Heuristics {
	h := &Heuristics{
		CompiledAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Track unique objects and their sizes.
	objectSizes := make(map[string]int) // hash -> byte count
	readObjectSize := func(hash string) int {
		if hash == "" {
			return 0
		}
		if size, ok := objectSizes[hash]; ok {
			return size
		}
		path := filepath.Join(objectsDir, hash+".md")
		info, err := os.Stat(path)
		if err != nil {
			return 0
		}
		size := int(info.Size())
		objectSizes[hash] = size
		return size
	}

	// Per-package accumulators.
	pkgEntries := make(map[string]int)
	pkgBytes := make(map[string]int)

	// Per-section accumulators.
	var foundationStats, topicStats, promptStats, planStats SectionStats

	// Track total entries and always-load count.
	totalEntries := 0
	totalAlwaysLoad := 0

	// Helper to accumulate entry stats.
	addEntry := func(section, id, relPath string, stats *SectionStats, alwaysLoad bool) int {
		hash := pathToHash[relPath]
		size := readObjectSize(hash)
		stats.Entries++
		stats.SizeBytes += size
		if alwaysLoad {
			stats.AlwaysLoad++
			totalAlwaysLoad++
		}
		totalEntries++

		pkg := provenance[section+":"+id]
		pkgEntries[pkg]++
		pkgBytes[pkg] += size

		return size
	}

	// Foundation entries.
	for _, e := range unified.Foundation {
		addEntry("foundation", e.ID, e.Path, &foundationStats, e.Load == "always")
	}

	// Topic entries (include spec and files in size).
	for _, e := range unified.Topics {
		size := addEntry("topics", e.ID, e.Path, &topicStats, false)
		_ = size

		// Add spec size.
		if e.Spec != "" {
			if hash, ok := pathToHash[e.Spec]; ok {
				specSize := readObjectSize(hash)
				topicStats.SizeBytes += specSize
				pkg := provenance["topics:"+e.ID]
				pkgBytes[pkg] += specSize
			}
		}
		// Add extra files size.
		for _, f := range e.Files {
			if hash, ok := pathToHash[f]; ok {
				fileSize := readObjectSize(hash)
				topicStats.SizeBytes += fileSize
				pkg := provenance["topics:"+e.ID]
				pkgBytes[pkg] += fileSize
			}
		}
	}

	// Prompt entries.
	for _, e := range unified.Prompts {
		addEntry("prompts", e.ID, e.Path, &promptStats, false)
	}

	// Plan entries.
	for _, e := range unified.Plans {
		addEntry("plans", e.ID, e.Path, &planStats, false)
	}

	// Compute token estimates for sections.
	foundationStats.EstimatedTokens = estimateTokens(foundationStats.SizeBytes)
	topicStats.EstimatedTokens = estimateTokens(topicStats.SizeBytes)
	promptStats.EstimatedTokens = estimateTokens(promptStats.SizeBytes)
	planStats.EstimatedTokens = estimateTokens(planStats.SizeBytes)

	// Populate sections (only non-empty).
	if foundationStats.Entries > 0 {
		h.Sections.Foundation = &foundationStats
	}
	if topicStats.Entries > 0 {
		h.Sections.Topics = &topicStats
	}
	if promptStats.Entries > 0 {
		h.Sections.Prompts = &promptStats
	}
	if planStats.Entries > 0 {
		h.Sections.Plans = &planStats
	}

	// Compute totals.
	totalBytes := foundationStats.SizeBytes + topicStats.SizeBytes +
		promptStats.SizeBytes + planStats.SizeBytes
	h.Totals = HeuristicsTotals{
		Entries:         totalEntries,
		Objects:         len(objectSizes),
		SizeBytes:       totalBytes,
		EstimatedTokens: estimateTokens(totalBytes),
		AlwaysLoad:      totalAlwaysLoad,
	}

	// Build per-package stats (sorted: local first, then alphabetical).
	pkgNames := make([]string, 0, len(pkgEntries))
	for name := range pkgEntries {
		if name == "local" {
			continue
		}
		pkgNames = append(pkgNames, name)
	}
	// Sort package names.
	sortStrings(pkgNames)

	// Local first.
	if pkgEntries["local"] > 0 {
		h.Packages = append(h.Packages, PackageStats{
			Name:            "local",
			Entries:         pkgEntries["local"],
			SizeBytes:       pkgBytes["local"],
			EstimatedTokens: estimateTokens(pkgBytes["local"]),
		})
	}
	for _, name := range pkgNames {
		h.Packages = append(h.Packages, PackageStats{
			Name:            name,
			Entries:         pkgEntries[name],
			SizeBytes:       pkgBytes[name],
			EstimatedTokens: estimateTokens(pkgBytes[name]),
		})
	}

	return h
}

// sortStrings sorts a slice of strings in place (simple insertion sort;
// package lists are small).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// WriteHeuristics writes the heuristics sidecar to the output directory.
func WriteHeuristics(path string, h *Heuristics) error {
	data, err := yaml.Marshal(h)
	if err != nil {
		return fmt.Errorf("marshal heuristics: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write heuristics: %w", err)
	}
	return nil
}

// LoadHeuristics reads heuristics from a file.
func LoadHeuristics(path string) (*Heuristics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read heuristics: %w", err)
	}
	var h Heuristics
	if err := yaml.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parse heuristics: %w", err)
	}
	return &h, nil
}
