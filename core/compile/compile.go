package compile

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/securacore/codectx/core/chunk"
	codectx "github.com/securacore/codectx/core/context"
	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/link"
	"github.com/securacore/codectx/core/llm"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
	"github.com/securacore/codectx/core/tokens"
)

// Stage names for progress reporting.
const (
	StagePrepare  = "prepare"
	StageDiscover = "discover"
	StageParse    = "parse"
	StageChunk    = "chunk"
	StageWrite    = "write"
	StageIndex    = "index"
	StageTaxonomy = "taxonomy"
	StageLLM      = "llm"
	StageManifest = "manifest"
	StageContext  = "context"
	StageLink     = "link"
)

// Config holds all parameters needed to run the compilation pipeline.
// The CLI layer assembles this from the project's config files.
type Config struct {
	// ProjectDir is the absolute path to the project root (where codectx.yml lives).
	ProjectDir string

	// RootDir is the absolute path to the documentation root (e.g., /path/to/project/docs).
	RootDir string

	// CompiledDir is the absolute path to .codectx/compiled/.
	CompiledDir string

	// SystemDir is the relative directory name for system docs (typically "system").
	SystemDir string

	// Encoding is the tokenizer encoding from ai.yml.
	Encoding string

	// Version is the compiler version string.
	Version string

	// Chunking holds chunking configuration from preferences.yml.
	Chunking project.ChunkingConfig

	// BM25 holds BM25 index parameters from preferences.yml.
	BM25 project.BM25Config

	// Validation holds validation settings from preferences.yml.
	Validation project.ValidationConfig

	// Taxonomy holds taxonomy extraction settings from preferences.yml.
	Taxonomy project.TaxonomyConfig

	// Model is the compilation model name from ai.yml (e.g. "claude-sonnet-4-20250514").
	Model string

	// Provider is the LLM provider from ai.yml ("cli", "api", or "" for auto-detect).
	Provider string

	// APIKey is the Anthropic API key for the API provider. Empty if using CLI.
	APIKey string

	// ActiveDeps maps package names to active status.
	// Only packages with a true value are included in compilation.
	ActiveDeps map[string]bool

	// Session holds session context configuration from codectx.yml.
	// If nil or AlwaysLoaded is empty, context assembly is skipped.
	Session *project.SessionConfig
}

// Result holds compilation statistics for display by the CLI layer.
type Result struct {
	// Source counts.
	TotalFiles int
	SpecFiles  int

	// Chunk counts by type.
	TotalChunks  int
	ObjectChunks int
	SpecChunks   int
	SystemChunks int

	// Token statistics.
	TotalTokens int
	AvgTokens   int
	MinTokens   int
	MaxTokens   int
	Oversized   int

	// Validation output.
	Warnings []string

	// Session context assembly.
	SessionTokens  int
	SessionBudget  int
	SessionEntries []SessionEntryResult

	// Taxonomy extraction.
	TaxonomyTerms int

	// LLM augmentation.
	LLMAliasCount  int
	LLMBridgeCount int
	LLMSeconds     float64
	LLMSkipped     bool
	LLMSkipReason  string

	// Timing.
	TotalSeconds    float64
	ParseSeconds    float64
	ChunkSeconds    float64
	IndexSeconds    float64
	TaxonomySeconds float64
	ManifestSeconds float64
	ContextSeconds  float64
	LinkSeconds     float64
}

// SessionEntryResult holds the assembly result for a single always_loaded entry.
type SessionEntryResult struct {
	Reference string
	Title     string
	Tokens    int
}

// ProgressFunc is called by Run at the start of each pipeline stage.
// The stage parameter is one of the Stage* constants. The detail string
// provides additional context (e.g., file counts).
type ProgressFunc func(stage, detail string)

// Run executes the full compilation pipeline. It discovers source files,
// parses and strips markdown, counts tokens, chunks documents, builds the
// BM25 search index, extracts taxonomy terms, runs LLM augmentation for
// alias generation and bridge summaries, and generates all manifest files.
//
// The progress callback is invoked at each stage transition. It may be nil.
func Run(cfg Config, progress ProgressFunc) (*Result, error) {
	if progress == nil {
		progress = func(string, string) {}
	}

	totalStart := time.Now()
	result := &Result{
		MinTokens: math.MaxInt,
	}

	// --- Stage: Prepare output directories ---
	progress(StagePrepare, "Cleaning output directories")

	if err := PrepareOutputDirs(cfg.CompiledDir); err != nil {
		return nil, fmt.Errorf("preparing output directories: %w", err)
	}

	// --- Stage: Discover source files ---
	progress(StageDiscover, "Scanning for markdown files")

	sources, err := DiscoverSources(cfg.RootDir, cfg.ActiveDeps)
	if err != nil {
		return nil, fmt.Errorf("discovering sources: %w", err)
	}

	result.TotalFiles = len(sources)
	for _, s := range sources {
		if s.IsSpec {
			result.SpecFiles++
		}
	}

	if result.TotalFiles == 0 {
		// No files to compile. Write empty manifests and return.
		progress(StageManifest, "No files to compile")
		if err := writeEmptyManifests(cfg); err != nil {
			return nil, err
		}
		result.TotalSeconds = time.Since(totalStart).Seconds()
		result.MinTokens = 0
		return result, nil
	}

	// --- Stage: Parse, validate, strip, count tokens ---
	progress(StageParse, fmt.Sprintf("Processing %d files", result.TotalFiles))

	parseStart := time.Now()

	counter, err := tokens.New(cfg.Encoding)
	if err != nil {
		return nil, fmt.Errorf("creating token counter: %w", err)
	}

	type parsedFile struct {
		source   SourceFile
		doc      *markdown.Document
		stripped *markdown.Document
	}

	parsed := make([]parsedFile, 0, len(sources))

	for _, src := range sources {
		data, err := os.ReadFile(src.AbsPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", src.Path, err)
		}

		// Parse markdown to AST.
		doc := markdown.Parse(data)

		// Validate the document.
		vResult := markdown.ValidateFile(doc, cfg.Validation.RequireHeadings)
		for _, w := range vResult.Warnings {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", src.Path, w))
		}

		// Strip formatting overhead.
		stripped := markdown.Strip(doc)

		// Count tokens on stripped blocks.
		if err := tokens.CountBlocks(stripped, counter); err != nil {
			return nil, fmt.Errorf("counting tokens for %s: %w", src.Path, err)
		}

		parsed = append(parsed, parsedFile{
			source:   src,
			doc:      doc,
			stripped: stripped,
		})
	}

	result.ParseSeconds = time.Since(parseStart).Seconds()

	// --- Stage: Chunk ---
	progress(StageChunk, fmt.Sprintf("Chunking %d files", result.TotalFiles))

	chunkStart := time.Now()

	opts := chunk.OptionsFromConfig(cfg.Chunking)

	var allChunks []chunk.Chunk
	// Map from source path to original parsed document (for metadata title extraction).
	blocksBySource := make(map[string]*markdown.Document, len(parsed))

	for _, pf := range parsed {
		ct := chunk.ClassifySource(pf.source.Path, cfg.SystemDir)
		chunks, err := chunk.ChunkDocument(pf.stripped, pf.source.Path, ct, opts)
		if err != nil {
			return nil, fmt.Errorf("chunking %s: %w", pf.source.Path, err)
		}
		allChunks = append(allChunks, chunks...)
		// Use the original (non-stripped) doc for title derivation.
		blocksBySource[pf.source.Path] = pf.doc
	}

	// Check for hash collisions across all chunks.
	if err := chunk.CheckCollisions(allChunks); err != nil {
		return nil, fmt.Errorf("chunk collision detected: %w", err)
	}

	// Compute chunk statistics.
	for i := range allChunks {
		c := &allChunks[i]
		switch c.Type {
		case chunk.ChunkObject:
			result.ObjectChunks++
		case chunk.ChunkSpec:
			result.SpecChunks++
		case chunk.ChunkSystem:
			result.SystemChunks++
		}
		result.TotalTokens += c.Tokens
		if c.Tokens < result.MinTokens {
			result.MinTokens = c.Tokens
		}
		if c.Tokens > result.MaxTokens {
			result.MaxTokens = c.Tokens
		}
		if c.Oversized {
			result.Oversized++
		}
	}
	result.TotalChunks = len(allChunks)
	if result.TotalChunks > 0 {
		result.AvgTokens = result.TotalTokens / result.TotalChunks
	}

	result.ChunkSeconds = time.Since(chunkStart).Seconds()

	// --- Stage: Write chunk files ---
	progress(StageWrite, fmt.Sprintf("Writing %d chunk files", result.TotalChunks))

	if _, err := WriteChunkFiles(cfg.CompiledDir, allChunks); err != nil {
		return nil, fmt.Errorf("writing chunk files: %w", err)
	}

	// --- Stage: Build BM25 index ---
	progress(StageIndex, "Building search index")

	indexStart := time.Now()

	idx := index.NewFromConfig(cfg.BM25)
	idx.BuildFromChunks(allChunks)

	if err := idx.Save(cfg.CompiledDir); err != nil {
		return nil, fmt.Errorf("saving BM25 index: %w", err)
	}

	result.IndexSeconds = time.Since(indexStart).Seconds()

	// --- Stage: Extract taxonomy ---
	progress(StageTaxonomy, fmt.Sprintf("Extracting taxonomy from %d chunks", result.TotalChunks))

	// Hash the taxonomy-generation system directory for cache invalidation.
	taxonomyInstructionsHash := ""
	taxonomyDir := filepath.Join(cfg.RootDir, cfg.SystemDir, "topics", "taxonomy-generation")
	if info, err := os.Stat(taxonomyDir); err == nil && info.IsDir() {
		h, err := manifest.HashDir(taxonomyDir)
		if err != nil {
			return nil, fmt.Errorf("hashing taxonomy instructions: %w", err)
		}
		taxonomyInstructionsHash = h
	}

	taxResult := taxonomy.Extract(allChunks, cfg.Taxonomy, cfg.Encoding, taxonomyInstructionsHash)
	result.TaxonomyTerms = taxResult.Stats.CanonicalTerms
	result.TaxonomySeconds = taxResult.Seconds

	// Write taxonomy.yml.
	taxPath := taxonomy.TaxonomyPath(cfg.CompiledDir)
	if err := taxResult.Taxonomy.WriteTo(taxPath); err != nil {
		return nil, fmt.Errorf("writing taxonomy: %w", err)
	}

	// --- Stage: LLM Augmentation ---
	progress(StageLLM, "Augmenting with LLM")

	llmStart := time.Now()
	instructionsDir := filepath.Join(cfg.RootDir, cfg.SystemDir, "topics")

	augResult := llm.Augment(context.Background(), llm.AugmentConfig{
		Provider:        cfg.Provider,
		APIKey:          cfg.APIKey,
		Model:           cfg.Model,
		ClaudeBinary:    "claude",
		Taxonomy:        taxResult.Taxonomy,
		Chunks:          allChunks,
		TaxonomyConfig:  cfg.Taxonomy,
		InstructionsDir: instructionsDir,
	})

	// Apply aliases to taxonomy and rewrite.
	if !augResult.Skipped && len(augResult.Aliases) > 0 {
		for key, aliases := range augResult.Aliases {
			if term, ok := taxResult.Taxonomy.Terms[key]; ok {
				term.Aliases = aliases
			}
		}
		taxResult.Taxonomy.CompiledWith = cfg.Model
		if err := taxResult.Taxonomy.WriteTo(taxPath); err != nil {
			return nil, fmt.Errorf("rewriting taxonomy with aliases: %w", err)
		}
	}

	result.LLMAliasCount = augResult.AliasCount
	result.LLMBridgeCount = augResult.BridgeCount
	result.LLMSeconds = time.Since(llmStart).Seconds()
	result.LLMSkipped = augResult.Skipped
	result.LLMSkipReason = augResult.SkipReason

	// --- Stage: Generate manifests ---
	progress(StageManifest, "Generating manifest files")

	manifestStart := time.Now()

	// Hash source files.
	fileHashes := make(map[string]string, len(sources))
	for _, src := range sources {
		h, err := manifest.HashFile(src.AbsPath)
		if err != nil {
			return nil, fmt.Errorf("hashing %s: %w", src.Path, err)
		}
		fileHashes[src.Path] = h
	}

	// Hash system directory if it exists.
	systemHashes := make(map[string]string)
	systemAbsDir := filepath.Join(cfg.RootDir, cfg.SystemDir)
	if info, err := os.Stat(systemAbsDir); err == nil && info.IsDir() {
		h, err := manifest.HashDir(systemAbsDir)
		if err != nil {
			return nil, fmt.Errorf("hashing system directory: %w", err)
		}
		systemHashes[cfg.SystemDir] = h
	}

	// Hash bridge-summaries directory for manifest cache invalidation.
	var bridgeHash *string
	bridgeDir := filepath.Join(cfg.RootDir, cfg.SystemDir, "topics", "bridge-summaries")
	if info, err := os.Stat(bridgeDir); err == nil && info.IsDir() {
		h, hashErr := manifest.HashDir(bridgeDir)
		if hashErr != nil {
			return nil, fmt.Errorf("hashing bridge instructions: %w", hashErr)
		}
		bridgeHash = &h
	}

	// Build manifest artifacts with taxonomy terms and bridge hash.
	mfst := manifest.BuildManifest(allChunks, cfg.Encoding, bridgeHash, taxResult.ChunkTerms)

	// Apply bridge summaries to manifest entries.
	if !augResult.Skipped && len(augResult.Bridges) > 0 {
		for id, bridge := range augResult.Bridges {
			if entry := mfst.LookupEntry(id); entry != nil {
				b := bridge
				entry.BridgeToNext = &b
			}
		}
	}
	meta := manifest.BuildMetadata(allChunks, blocksBySource)
	hashes := manifest.BuildHashes(fileHashes, systemHashes)

	heur := manifest.NewHeuristics(cfg.Version, cfg.Encoding)
	heur.SetSources(&manifest.SourcesSection{
		TotalFiles:   result.TotalFiles,
		LocalFiles:   result.TotalFiles - result.SpecFiles,
		PackageFiles: 0,
		New:          result.TotalFiles,
		Modified:     0,
		Unchanged:    0,
		SpecFiles:    result.SpecFiles,
	})
	heur.SetChunkStats(&manifest.ChunksSection{
		Total:         result.TotalChunks,
		Objects:       result.ObjectChunks,
		Specs:         result.SpecChunks,
		System:        result.SystemChunks,
		TotalTokens:   result.TotalTokens,
		AverageTokens: result.AvgTokens,
		MinTokens:     result.MinTokens,
		MaxTokens:     result.MaxTokens,
		Oversized:     result.Oversized,
	})
	heur.SetBM25Stats(idx)

	result.ManifestSeconds = time.Since(manifestStart).Seconds()

	// --- Stage: Assemble session context ---
	if cfg.Session != nil && len(cfg.Session.AlwaysLoaded) > 0 {
		progress(StageContext, "Assembling session context")

		contextStart := time.Now()

		packagesDir := project.PackagesPath(cfg.RootDir)
		resolved, resolveErr := codectx.Resolve(cfg.RootDir, packagesDir, cfg.Session.AlwaysLoaded)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolving session context: %w", resolveErr)
		}

		budget := cfg.Session.EffectiveBudget()

		assembly, assembleErr := codectx.Assemble(resolved, cfg.Encoding, budget)
		if assembleErr != nil {
			return nil, fmt.Errorf("assembling session context: %w", assembleErr)
		}

		if err := codectx.WriteContextMD(cfg.CompiledDir, assembly); err != nil {
			return nil, fmt.Errorf("writing context.md: %w", err)
		}

		result.SessionTokens = assembly.TotalTokens
		result.SessionBudget = assembly.Budget
		result.Warnings = append(result.Warnings, assembly.Warnings...)

		for _, entry := range assembly.Entries {
			result.SessionEntries = append(result.SessionEntries, SessionEntryResult{
				Reference: entry.Reference,
				Title:     entry.Title,
				Tokens:    entry.Tokens,
			})
		}

		// Populate heuristics session section.
		heurEntries := make([]manifest.SessionEntry, len(assembly.Entries))
		for i, e := range assembly.Entries {
			heurEntries[i] = manifest.SessionEntry{
				Path:   e.Reference,
				Tokens: e.Tokens,
			}
		}
		utilization := fmt.Sprintf("%.1f%%", assembly.Utilization)
		heur.SetSession(assembly.TotalTokens, assembly.Budget, utilization, heurEntries)

		result.ContextSeconds = time.Since(contextStart).Seconds()
	}

	// --- Stage: Update linked entry points ---
	if cfg.ProjectDir != "" {
		contextRelPath := contextRelativePath(cfg)
		needsUpdate := link.NeedsUpdate(cfg.ProjectDir, contextRelPath)

		if len(needsUpdate) > 0 {
			progress(StageLink, fmt.Sprintf("Updating %d entry point(s)", len(needsUpdate)))

			linkStart := time.Now()

			if _, linkErr := link.Write(cfg.ProjectDir, contextRelPath, needsUpdate); linkErr != nil {
				// Non-fatal: warn but don't fail compilation.
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("failed to update entry points: %v", linkErr))
			}

			result.LinkSeconds = time.Since(linkStart).Seconds()
		}
	}

	// Populate taxonomy heuristics.
	var avgAliases float64
	if taxResult.Stats.CanonicalTerms > 0 {
		avgAliases = float64(result.LLMAliasCount) / float64(taxResult.Stats.CanonicalTerms)
	}
	heur.SetTaxonomyStats(&manifest.TaxonomySection{
		CanonicalTerms:               taxResult.Stats.CanonicalTerms,
		TotalAliases:                 result.LLMAliasCount,
		AverageAliasesPerTerm:        avgAliases,
		TermsFromHeadings:            taxResult.Stats.TermsFromHeadings,
		TermsFromCodeIdents:          taxResult.Stats.TermsFromCodeIdents,
		TermsFromBoldTerms:           taxResult.Stats.TermsFromBoldTerms,
		TermsFromStructuredPositions: taxResult.Stats.TermsFromStructured,
		TermsFromPOSExtraction:       taxResult.Stats.TermsFromPOS,
		AliasesFromLLM:               result.LLMAliasCount,
	})

	timing := &manifest.TimingSection{
		TotalSeconds:       time.Since(totalStart).Seconds(),
		ParseValidate:      result.ParseSeconds,
		StripNormalize:     0, // combined with parse in this implementation
		Chunking:           result.ChunkSeconds,
		BM25Indexing:       result.IndexSeconds,
		TaxonomyExtraction: result.TaxonomySeconds,
		LLMAugmentation:    result.LLMSeconds,
		ManifestGeneration: result.ManifestSeconds,
		ContextAssembly:    result.ContextSeconds,
		SyncEntryPoints:    result.LinkSeconds,
	}
	heur.SetTiming(timing)
	heur.SetIncremental(&manifest.IncrementalSection{
		FullRecompile: true,
	})

	if err := writeManifests(cfg.CompiledDir, mfst, meta, hashes, heur); err != nil {
		return nil, err
	}

	result.TotalSeconds = time.Since(totalStart).Seconds()

	return result, nil
}

// contextRelativePath computes the path to context.md relative to the project root.
func contextRelativePath(cfg Config) string {
	contextAbsPath := codectx.ContextPath(cfg.CompiledDir)
	relPath, err := filepath.Rel(cfg.ProjectDir, contextAbsPath)
	if err != nil {
		return contextAbsPath
	}
	return filepath.ToSlash(relPath)
}

// writeEmptyManifests writes empty manifest files for projects with no source files.
func writeEmptyManifests(cfg Config) error {
	mfst := manifest.BuildManifest(nil, cfg.Encoding, nil, nil)
	meta := manifest.BuildMetadata(nil, nil)
	hashes := manifest.BuildHashes(nil, nil)

	heur := manifest.NewHeuristics(cfg.Version, cfg.Encoding)
	heur.SetSources(&manifest.SourcesSection{})
	heur.SetChunkStats(&manifest.ChunksSection{})
	heur.SetIncremental(&manifest.IncrementalSection{FullRecompile: true})

	return writeManifests(cfg.CompiledDir, mfst, meta, hashes, heur)
}

// writeManifests writes all four compiled manifest files to the compiled directory.
func writeManifests(compiledDir string, mfst *manifest.Manifest, meta *manifest.Metadata, hashes *manifest.Hashes, heur *manifest.Heuristics) error {
	if err := mfst.WriteTo(manifest.EntryPath(compiledDir)); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	if err := meta.WriteTo(manifest.MetadataPath(compiledDir)); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}
	if err := hashes.WriteTo(manifest.HashesPath(compiledDir)); err != nil {
		return fmt.Errorf("writing hashes: %w", err)
	}
	if err := heur.WriteTo(manifest.HeuristicsPath(compiledDir)); err != nil {
		return fmt.Errorf("writing heuristics: %w", err)
	}
	return nil
}
