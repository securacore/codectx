package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/securacore/codectx/core/cmdx"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/tokenizer"
)

const heuristicsFile = "heuristics.yml"

// Heuristics is the sidecar metadata generated during compilation.
// It provides size and token estimates for the compiled documentation set.
// Not part of the AI loading protocol; used by tooling and the generated
// README for richer context.
type Heuristics struct {
	CompiledAt            string                   `yaml:"compiled_at"`
	Totals                HeuristicsTotals         `yaml:"totals"`
	Sections              HeuristicsSections       `yaml:"sections"`
	Packages              []PackageStats           `yaml:"packages"`
	GlobalDictOpportunity *cmdx.GlobalDictAnalysis `yaml:"global_dict_opportunity,omitempty"`
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
	Foundation  *SectionStats `yaml:"foundation,omitempty"`
	Application *SectionStats `yaml:"application,omitempty"`
	Topics      *SectionStats `yaml:"topics,omitempty"`
	Prompts     *SectionStats `yaml:"prompts,omitempty"`
	Plans       *SectionStats `yaml:"plans,omitempty"`
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

// countTokens returns the o200k_base token count for the given content.
// This replaces the previous rough heuristic of bytes * 0.25.
func countTokens(content []byte) int {
	return tokenizer.CountTokensBytes(content)
}

// generateHeuristics builds heuristics metadata from the unified manifest
// and stored objects. It reads object sizes from the object store to
// compute byte counts and token estimates. The ext parameter is the file
// extension used by the object store (e.g., ".md" or ".cmdx").
func generateHeuristics(
	unified *manifest.Manifest,
	pathToHash map[string]string,
	provenance map[string]string,
	objectsDir string,
	ext ...string,
) *Heuristics {
	objExt := ".md"
	if len(ext) > 0 && ext[0] != "" {
		objExt = ext[0]
	}
	h := &Heuristics{
		CompiledAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Track unique objects: byte size and real token count.
	type objectMeasure struct {
		size   int
		tokens int
	}
	objectCache := make(map[string]objectMeasure) // hash -> measure
	readObject := func(hash string) objectMeasure {
		if hash == "" {
			return objectMeasure{}
		}
		if m, ok := objectCache[hash]; ok {
			return m
		}
		path := filepath.Join(objectsDir, hash+objExt)
		data, err := os.ReadFile(path)
		if err != nil {
			return objectMeasure{}
		}
		m := objectMeasure{
			size:   len(data),
			tokens: countTokens(data),
		}
		objectCache[hash] = m
		return m
	}

	// Per-package accumulators.
	pkgEntries := make(map[string]int)
	pkgBytes := make(map[string]int)
	pkgTokens := make(map[string]int)

	// Per-section accumulators.
	var foundationStats, applicationStats, topicStats, promptStats, planStats SectionStats

	// Track total entries and always-load count.
	totalEntries := 0
	totalAlwaysLoad := 0

	// Helper to accumulate entry stats.
	addEntry := func(section, id, relPath string, stats *SectionStats, alwaysLoad bool) {
		hash := pathToHash[relPath]
		m := readObject(hash)
		stats.Entries++
		stats.SizeBytes += m.size
		stats.EstimatedTokens += m.tokens
		if alwaysLoad {
			stats.AlwaysLoad++
			totalAlwaysLoad++
		}
		totalEntries++

		pkg := provenance[section+":"+id]
		pkgEntries[pkg]++
		pkgBytes[pkg] += m.size
		pkgTokens[pkg] += m.tokens
	}

	// Foundation entries.
	for _, e := range unified.Foundation {
		addEntry("foundation", e.ID, e.Path, &foundationStats, e.Load == "always")
	}

	// Application entries (include spec and files in size).
	for _, e := range unified.Application {
		addEntry("application", e.ID, e.Path, &applicationStats, e.Load == "always")

		// Add spec size and tokens.
		if e.Spec != "" {
			if hash, ok := pathToHash[e.Spec]; ok {
				m := readObject(hash)
				applicationStats.SizeBytes += m.size
				applicationStats.EstimatedTokens += m.tokens
				pkg := provenance["application:"+e.ID]
				pkgBytes[pkg] += m.size
				pkgTokens[pkg] += m.tokens
			}
		}
		// Add extra files size and tokens.
		for _, f := range e.Files {
			if hash, ok := pathToHash[f]; ok {
				m := readObject(hash)
				applicationStats.SizeBytes += m.size
				applicationStats.EstimatedTokens += m.tokens
				pkg := provenance["application:"+e.ID]
				pkgBytes[pkg] += m.size
				pkgTokens[pkg] += m.tokens
			}
		}
	}

	// Topic entries (include spec and files in size).
	for _, e := range unified.Topics {
		addEntry("topics", e.ID, e.Path, &topicStats, false)

		// Add spec size and tokens.
		if e.Spec != "" {
			if hash, ok := pathToHash[e.Spec]; ok {
				m := readObject(hash)
				topicStats.SizeBytes += m.size
				topicStats.EstimatedTokens += m.tokens
				pkg := provenance["topics:"+e.ID]
				pkgBytes[pkg] += m.size
				pkgTokens[pkg] += m.tokens
			}
		}
		// Add extra files size and tokens.
		for _, f := range e.Files {
			if hash, ok := pathToHash[f]; ok {
				m := readObject(hash)
				topicStats.SizeBytes += m.size
				topicStats.EstimatedTokens += m.tokens
				pkg := provenance["topics:"+e.ID]
				pkgBytes[pkg] += m.size
				pkgTokens[pkg] += m.tokens
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

	// Token counts are now accumulated inline via addEntry and the
	// spec/files loops — no separate estimation pass needed.

	// Populate sections (only non-empty).
	if foundationStats.Entries > 0 {
		h.Sections.Foundation = &foundationStats
	}
	if applicationStats.Entries > 0 {
		h.Sections.Application = &applicationStats
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
	totalBytes := foundationStats.SizeBytes + applicationStats.SizeBytes +
		topicStats.SizeBytes + promptStats.SizeBytes + planStats.SizeBytes
	totalTokens := foundationStats.EstimatedTokens + applicationStats.EstimatedTokens +
		topicStats.EstimatedTokens + promptStats.EstimatedTokens + planStats.EstimatedTokens
	h.Totals = HeuristicsTotals{
		Entries:         totalEntries,
		Objects:         len(objectCache),
		SizeBytes:       totalBytes,
		EstimatedTokens: totalTokens,
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
			EstimatedTokens: pkgTokens["local"],
		})
	}
	for _, name := range pkgNames {
		h.Packages = append(h.Packages, PackageStats{
			Name:            name,
			Entries:         pkgEntries[name],
			SizeBytes:       pkgBytes[name],
			EstimatedTokens: pkgTokens[name],
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

// loadHeuristics reads heuristics from a file.
func loadHeuristics(path string) (*Heuristics, error) {
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
