package llm

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
	"github.com/securacore/codectx/embed"
)

// Default concurrency limits for LLM batch processing.
const (
	// defaultAPIConcurrency is the default number of concurrent API calls.
	defaultAPIConcurrency = 4

	// defaultCLIConcurrency is the default number of concurrent CLI calls.
	defaultCLIConcurrency = 2
)

// AugmentConfig holds all parameters for the LLM augmentation stage.
type AugmentConfig struct {
	// Provider is the LLM provider ("cli", "api", or "" for auto-detect).
	Provider string

	// APIKey is the Anthropic API key (empty if using CLI provider).
	APIKey string

	// Model is the compilation model name from ai.yml.
	Model string

	// ClaudeBinary is the name or path of the claude CLI binary.
	ClaudeBinary string

	// Taxonomy is the compiled taxonomy. Terms are used to build alias requests.
	Taxonomy *taxonomy.Taxonomy

	// Chunks are all compiled chunks. Used to build bridge pairs.
	Chunks []chunk.Chunk

	// TaxonomyConfig holds taxonomy settings (max_alias_count, llm_alias_generation).
	TaxonomyConfig project.TaxonomyConfig

	// InstructionsDir is the absolute path to the system/topics/ directory
	// containing taxonomy-generation/ and bridge-summaries/ subdirectories.
	InstructionsDir string

	// Concurrency is the maximum number of concurrent LLM calls per task.
	// If <= 0, a default is chosen based on the provider (4 for API, 2 for CLI).
	Concurrency int
}

// AugmentResult holds the output of the LLM augmentation stage.
type AugmentResult struct {
	// Aliases maps normalized term keys to their generated aliases.
	// Applied to taxonomy.Terms by the caller.
	Aliases map[string][]string

	// Bridges maps chunk IDs to their generated bridge summary text.
	// Applied to manifest entries by the caller.
	Bridges map[string]string

	// AliasCount is the total number of aliases generated across all terms.
	AliasCount int

	// BridgeCount is the total number of bridges generated.
	BridgeCount int

	// Seconds is the wall-clock duration of the augmentation stage.
	Seconds float64

	// Skipped is true if LLM augmentation was skipped entirely.
	Skipped bool

	// SkipReason describes why augmentation was skipped (for display).
	SkipReason string
}

// Augment runs both LLM augmentation tasks: alias generation and bridge summaries.
//
// It returns a result containing aliases and bridges to be applied by the caller.
// The function itself does not mutate the taxonomy or manifest entries.
//
// Graceful degradation: returns a valid AugmentResult with Skipped=true if the
// LLM is unavailable or disabled. Individual batch failures produce partial results
// rather than a total failure.
func Augment(ctx context.Context, cfg AugmentConfig) *AugmentResult {
	start := time.Now()

	result := &AugmentResult{
		Aliases: make(map[string][]string),
		Bridges: make(map[string]string),
	}

	// Check master switch.
	if !cfg.TaxonomyConfig.LLMAliasGeneration {
		result.Skipped = true
		result.SkipReason = "llm_alias_generation disabled"
		return result
	}

	// Create sender.
	sender, err := NewSender(cfg.Provider, cfg.APIKey, cfg.Model, cfg.ClaudeBinary)
	if err != nil {
		result.Skipped = true
		result.SkipReason = "failed to create LLM client: " + err.Error()
		return result
	}
	if sender == nil {
		result.Skipped = true
		result.SkipReason = "no LLM provider available"
		return result
	}

	// Determine concurrency.
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		if cfg.Provider == project.ProviderCLI {
			concurrency = defaultCLIConcurrency
		} else {
			concurrency = defaultAPIConcurrency
		}
	}

	// Read instruction files (with embedded fallback).
	aliasInstructions := readInstructions(cfg.InstructionsDir, "taxonomy-generation", "defaults/taxonomy-generation.md")
	bridgeInstructions := readInstructions(cfg.InstructionsDir, "bridge-summaries", "defaults/bridge-summaries.md")

	// Run alias generation and bridge generation concurrently.
	// They are independent tasks that can safely share the sender.
	var wg sync.WaitGroup
	var aliasResult *aliasResult
	var bridgeResult *bridgeResult

	// Task 1: Alias generation (concurrent batches).
	if cfg.Taxonomy != nil && len(cfg.Taxonomy.Terms) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			terms := buildAliasRequests(cfg.Taxonomy)
			aliasResult = generateAliases(ctx, aliasGenConfig{
				sender:       sender,
				terms:        terms,
				instructions: aliasInstructions,
				maxAliases:   cfg.TaxonomyConfig.MaxAliasCount,
				concurrency:  concurrency,
			})
		}()
	}

	// Task 2: Bridge summaries (concurrent batches).
	if len(cfg.Chunks) > 0 {
		pairs := buildBridgePairs(cfg.Chunks)
		if len(pairs) > 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				bridgeResult = generateBridges(ctx, bridgeGenConfig{
					sender:       sender,
					pairs:        pairs,
					instructions: bridgeInstructions,
					concurrency:  concurrency,
				})
			}()
		}
	}

	wg.Wait()

	// Collect results.
	if aliasResult != nil {
		result.Aliases = aliasResult.Aliases
		result.AliasCount = aliasResult.TotalAliases
	}
	if bridgeResult != nil {
		result.Bridges = bridgeResult.Bridges
		result.BridgeCount = len(bridgeResult.Bridges)
	}

	result.Seconds = time.Since(start).Seconds()
	return result
}

// buildAliasRequests converts taxonomy terms into aliasRequest structs.
func buildAliasRequests(tax *taxonomy.Taxonomy) []*aliasRequest {
	requests := make([]*aliasRequest, 0, len(tax.Terms))
	for key, term := range tax.Terms {
		requests = append(requests, &aliasRequest{
			Key:       key,
			Canonical: term.Canonical,
			Source:    term.Source,
			Broader:   term.Broader,
			Narrower:  term.Narrower,
			Related:   term.Related,
		})
	}
	return requests
}

// readInstructions reads a system instruction file from the instructions directory.
// If the file doesn't exist or can't be read, falls back to the embedded default.
func readInstructions(instructionsDir, topicDir, embedPath string) string {
	if instructionsDir != "" {
		path := filepath.Join(instructionsDir, topicDir, "README.md")
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return string(data)
		}
	}

	// Fallback to embedded default.
	data, err := embed.ReadFile(embedPath)
	if err != nil {
		// This should never happen since embedded files are compiled in.
		return ""
	}
	return string(data)
}
