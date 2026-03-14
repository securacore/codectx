package compile

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/securacore/codectx/core/bridge"
	"github.com/securacore/codectx/core/chunk"
	codectx "github.com/securacore/codectx/core/context"
	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/link"
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

	// BM25 holds flat BM25 index parameters from preferences.yml.
	BM25 project.BM25Config

	// BM25F holds field-weighted BM25F index parameters from preferences.yml.
	BM25F project.BM25FConfig

	// Validation holds validation settings from preferences.yml.
	Validation project.ValidationConfig

	// Taxonomy holds taxonomy extraction settings from preferences.yml.
	Taxonomy project.TaxonomyConfig

	// ActiveDeps maps package names to active status.
	// Only packages with a true value are included in compilation.
	ActiveDeps map[string]bool

	// Session holds session context configuration from codectx.yml.
	// If nil or AlwaysLoaded is empty, context assembly is skipped.
	Session *project.SessionConfig

	// Incremental enables incremental compilation mode. When true and
	// previous hashes exist, only changed files are reprocessed through
	// expensive pipeline stages. Default: true.
	Incremental bool
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

	// Cross-references.
	CrossRefLinks int // total forward cross-reference links across all documents
	CrossRefDocs  int // number of documents with at least one cross-reference

	// Bridge generation.
	DetBridgeCount int // deterministic bridges (heading + RAKE + last sentence)

	// Incremental compilation.
	IncrementalMode bool // true if incremental mode was active (not first compile)
	NewFiles        int  // files not in previous hashes
	ModifiedFiles   int  // files whose hash changed
	UnchangedFiles  int  // files with identical hashes
	DeletedFiles    int  // files in previous hashes but no longer discovered

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

// parsedFile holds a parsed source file with both original and stripped ASTs.
type parsedFile struct {
	source   SourceFile
	doc      *markdown.Document
	stripped *markdown.Document
}

// pipelineState carries intermediate results between pipeline stages.
// Each stage reads from and writes to this shared state, keeping Run()
// as a thin orchestrator.
type pipelineState struct {
	cfg      Config
	result   *Result
	progress ProgressFunc

	// Source discovery.
	sources    []SourceFile
	fileHashes map[string]string

	// Incremental detection.
	incremental      bool
	changeSet        *ChangeSet
	prevManifest     *manifest.Manifest
	prevTaxonomy     *taxonomy.Taxonomy
	prevSystemHashes map[string]string

	// Parse/chunk output.
	parsed            []parsedFile
	blocksBySource    map[string]*markdown.Document
	newChunks         []chunk.Chunk
	unchangedChunks   []chunk.Chunk
	unchangedChunkIDs map[string]bool
	allChunks         []chunk.Chunk

	// Index.
	idx      *index.Index
	fieldIdx *index.FieldIndex

	// Taxonomy.
	taxResult    *taxonomy.Result
	systemHashes map[string]string

	// Heuristics (accumulated across stages).
	heur *manifest.Heuristics
}

// Run executes the full compilation pipeline. It discovers source files,
// parses and strips markdown, counts tokens, chunks documents, builds
// BM25 and BM25F search indexes, extracts taxonomy terms, generates
// deterministic bridge summaries, and produces all manifest files.
//
// When cfg.Incremental is true and previous hashes exist, only changed files
// are re-parsed and re-chunked. Unchanged chunks are loaded from disk.
// All chunks (new + unchanged) are used for BM25, taxonomy, and manifests.
//
// The progress callback is invoked at each stage transition. It may be nil.
func Run(cfg Config, progress ProgressFunc) (*Result, error) {
	if progress == nil {
		progress = func(string, string) {}
	}

	totalStart := time.Now()

	ps := &pipelineState{
		cfg:      cfg,
		result:   &Result{MinTokens: math.MaxInt},
		progress: progress,
	}

	// --- Stage: Discover source files ---
	if err := ps.stageDiscover(); err != nil {
		return nil, err
	}

	if ps.result.TotalFiles == 0 {
		ps.progress(StageManifest, "No files to compile")
		if err := writeEmptyManifests(cfg); err != nil {
			return nil, err
		}
		ps.result.TotalSeconds = time.Since(totalStart).Seconds()
		ps.result.MinTokens = 0
		return ps.result, nil
	}

	// --- Hash source files ---
	if err := ps.stageHash(); err != nil {
		return nil, err
	}

	// --- Incremental detection ---
	ps.stageIncrementalDetection()

	// --- Prepare output directories ---
	if err := ps.stagePrepareOutputDirs(); err != nil {
		return nil, err
	}

	// --- Parse, validate, strip, count tokens ---
	if err := ps.stageParse(); err != nil {
		return nil, err
	}

	// --- Chunk + load unchanged ---
	if err := ps.stageChunk(); err != nil {
		return nil, err
	}

	// --- Write chunk files ---
	if err := ps.stageWriteChunks(); err != nil {
		return nil, err
	}

	// --- Build BM25 index ---
	if err := ps.stageIndex(); err != nil {
		return nil, err
	}

	// --- Extract taxonomy ---
	if err := ps.stageTaxonomy(); err != nil {
		return nil, err
	}

	// --- Build BM25F field-weighted index (needs taxonomy terms) ---
	if err := ps.stageFieldIndex(); err != nil {
		return nil, err
	}

	// --- Generate manifests ---
	if err := ps.stageManifests(); err != nil {
		return nil, err
	}

	// --- Assemble session context ---
	if err := ps.stageContext(); err != nil {
		return nil, err
	}

	// --- Update linked entry points ---
	ps.stageLink()

	// --- Finalize heuristics and write ---
	if err := ps.stageFinalize(totalStart); err != nil {
		return nil, err
	}

	ps.result.TotalSeconds = time.Since(totalStart).Seconds()

	return ps.result, nil
}

// stageDiscover scans for markdown source files.
func (ps *pipelineState) stageDiscover() error {
	ps.progress(StageDiscover, "Scanning for markdown files")

	sources, err := DiscoverSources(ps.cfg.RootDir, ps.cfg.ActiveDeps)
	if err != nil {
		return fmt.Errorf("discovering sources: %w", err)
	}

	ps.sources = sources
	ps.result.TotalFiles = len(sources)
	for _, s := range sources {
		if s.IsSpec {
			ps.result.SpecFiles++
		}
	}

	return nil
}

// stageHash computes SHA-256 content hashes for all source files.
func (ps *pipelineState) stageHash() error {
	ps.fileHashes = make(map[string]string, len(ps.sources))
	for _, src := range ps.sources {
		h, err := manifest.HashFile(src.AbsPath)
		if err != nil {
			return fmt.Errorf("hashing %s: %w", src.Path, err)
		}
		ps.fileHashes[src.Path] = h
	}
	return nil
}

// stageIncrementalDetection compares current hashes against previous
// compilation to classify files as new, modified, or unchanged.
func (ps *pipelineState) stageIncrementalDetection() {
	if !ps.cfg.Incremental {
		ps.result.NewFiles = ps.result.TotalFiles
		return
	}

	prevHashes, prevErr := manifest.LoadHashes(manifest.HashesPath(ps.cfg.CompiledDir))
	if prevErr != nil || len(prevHashes.Files) == 0 {
		// First compile or corrupted hashes — treat as full recompile.
		ps.result.NewFiles = ps.result.TotalFiles
		return
	}

	ps.changeSet = ClassifyFiles(ps.fileHashes, prevHashes)
	ps.prevSystemHashes = prevHashes.System

	// Only use incremental mode if there are actually unchanged files.
	if ps.changeSet.UnchangedCount > 0 {
		ps.incremental = true

		// Load previous manifest for reconstructing unchanged chunks.
		// Errors are intentionally ignored — incremental mode gracefully
		// degrades to a full recompile if previous artifacts are missing.
		ps.prevManifest, _ = manifest.LoadManifest(manifest.EntryPath(ps.cfg.CompiledDir))

		// Load previous taxonomy for preserving aliases on unchanged terms.
		// Same graceful degradation as above.
		ps.prevTaxonomy, _ = taxonomy.Load(taxonomy.TaxonomyPath(ps.cfg.CompiledDir))
	}

	ps.result.IncrementalMode = ps.incremental

	if ps.incremental {
		ps.result.NewFiles = ps.changeSet.NewCount
		ps.result.ModifiedFiles = ps.changeSet.ModifiedCount
		ps.result.UnchangedFiles = ps.changeSet.UnchangedCount
		ps.result.DeletedFiles = len(ps.changeSet.Deleted)
	} else {
		ps.result.NewFiles = ps.result.TotalFiles
	}
}

// stagePrepareOutputDirs sets up compiled output directories. In incremental
// mode, preserves existing files and removes only stale/modified chunks.
// In full mode, wipes and recreates all directories.
func (ps *pipelineState) stagePrepareOutputDirs() error {
	if ps.incremental {
		ps.progress(StagePrepare, "Preserving unchanged chunk files")
		if err := EnsureOutputDirs(ps.cfg.CompiledDir); err != nil {
			return fmt.Errorf("ensuring output directories: %w", err)
		}

		if ps.prevManifest != nil {
			if err := ps.removeStaleChunkFiles(); err != nil {
				return err
			}
		}
	} else {
		ps.progress(StagePrepare, "Cleaning output directories")
		if err := PrepareOutputDirs(ps.cfg.CompiledDir); err != nil {
			return fmt.Errorf("preparing output directories: %w", err)
		}
	}
	return nil
}

// removeStaleChunkFiles removes chunk files for deleted and modified sources
// during incremental compilation.
func (ps *pipelineState) removeStaleChunkFiles() error {
	// Remove chunks for deleted sources.
	if len(ps.changeSet.Deleted) > 0 {
		deletedChunkIDs := chunksForSources(ps.prevManifest, ps.changeSet.Deleted)
		if err := RemoveChunkFiles(ps.cfg.CompiledDir, deletedChunkIDs); err != nil {
			return fmt.Errorf("removing deleted chunks: %w", err)
		}
	}

	// Remove chunks for modified sources (they will be re-chunked).
	var modifiedSources []string
	for path, status := range ps.changeSet.Status {
		if status == FileModified {
			modifiedSources = append(modifiedSources, path)
		}
	}
	if len(modifiedSources) > 0 {
		modifiedChunkIDs := chunksForSources(ps.prevManifest, modifiedSources)
		if err := RemoveChunkFiles(ps.cfg.CompiledDir, modifiedChunkIDs); err != nil {
			return fmt.Errorf("removing modified chunks: %w", err)
		}
	}

	return nil
}

// sourcesToProcess returns the source files that need parsing/chunking.
// In incremental mode, only new and modified files are returned.
func (ps *pipelineState) sourcesToProcess() []SourceFile {
	if !ps.incremental {
		return ps.sources
	}

	var toProcess []SourceFile
	for _, src := range ps.sources {
		status := ps.changeSet.Status[src.Path]
		if status == FileNew || status == FileModified {
			toProcess = append(toProcess, src)
		}
	}
	return toProcess
}

// stageParse reads, parses, validates, strips, and counts tokens for source files.
func (ps *pipelineState) stageParse() error {
	toProcess := ps.sourcesToProcess()
	ps.progress(StageParse, fmt.Sprintf("Processing %d files", len(toProcess)))

	parseStart := time.Now()

	counter, err := tokens.New(ps.cfg.Encoding)
	if err != nil {
		return fmt.Errorf("creating token counter: %w", err)
	}

	ps.parsed = make([]parsedFile, 0, len(toProcess))

	for _, src := range toProcess {
		data, err := os.ReadFile(src.AbsPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", src.Path, err)
		}

		doc := markdown.Parse(data)

		vResult := markdown.ValidateFile(doc, ps.cfg.Validation.RequireHeadings)
		for _, w := range vResult.Warnings {
			ps.result.Warnings = append(ps.result.Warnings, fmt.Sprintf("%s: %s", src.Path, w))
		}

		stripped := markdown.Strip(doc)

		if err := tokens.CountBlocks(stripped, counter); err != nil {
			return fmt.Errorf("counting tokens for %s: %w", src.Path, err)
		}

		ps.parsed = append(ps.parsed, parsedFile{
			source:   src,
			doc:      doc,
			stripped: stripped,
		})
	}

	ps.result.ParseSeconds = time.Since(parseStart).Seconds()
	return nil
}

// stageChunk splits parsed files into chunks and loads unchanged chunks
// from disk in incremental mode.
func (ps *pipelineState) stageChunk() error {
	toProcess := ps.sourcesToProcess()
	ps.progress(StageChunk, fmt.Sprintf("Chunking %d files", len(toProcess)))

	chunkStart := time.Now()

	opts := chunk.OptionsFromConfig(ps.cfg.Chunking)
	ps.blocksBySource = make(map[string]*markdown.Document, len(ps.parsed))

	for _, pf := range ps.parsed {
		ct := chunk.ClassifySource(pf.source.Path, ps.cfg.SystemDir)
		chunks, err := chunk.ChunkDocument(pf.stripped, pf.source.Path, ct, opts)
		if err != nil {
			return fmt.Errorf("chunking %s: %w", pf.source.Path, err)
		}
		ps.newChunks = append(ps.newChunks, chunks...)
		ps.blocksBySource[pf.source.Path] = pf.doc
	}

	// Load unchanged chunks from disk in incremental mode.
	ps.unchangedChunkIDs = make(map[string]bool)
	if ps.incremental && ps.prevManifest != nil {
		ps.loadUnchangedChunks()
	}

	// Combine new + unchanged chunks for downstream stages.
	ps.allChunks = make([]chunk.Chunk, 0, len(ps.newChunks)+len(ps.unchangedChunks))
	ps.allChunks = append(ps.allChunks, ps.newChunks...)
	ps.allChunks = append(ps.allChunks, ps.unchangedChunks...)

	if err := chunk.CheckCollisions(ps.allChunks); err != nil {
		return fmt.Errorf("chunk collision detected: %w", err)
	}

	ps.computeChunkStats()
	ps.result.ChunkSeconds = time.Since(chunkStart).Seconds()

	return nil
}

// loadUnchangedChunks loads chunks from disk for sources that haven't changed.
func (ps *pipelineState) loadUnchangedChunks() {
	var unchangedSources []string
	for path, status := range ps.changeSet.Status {
		if status == FileUnchanged {
			unchangedSources = append(unchangedSources, path)
		}
	}

	loadedChunks := chunksFromManifest(ps.prevManifest, unchangedSources)
	for _, lc := range loadedChunks {
		loaded, loadErr := chunk.LoadChunkFromDisk(ps.cfg.CompiledDir, lc)
		if loadErr != nil {
			// If a chunk file is missing, skip it — will be treated as if modified.
			ps.result.Warnings = append(ps.result.Warnings,
				fmt.Sprintf("incremental: failed to load chunk %s: %v", lc.ID, loadErr))
			continue
		}
		ps.unchangedChunks = append(ps.unchangedChunks, loaded)
		ps.unchangedChunkIDs[loaded.ID] = true
	}
}

// computeChunkStats aggregates chunk statistics into the result.
func (ps *pipelineState) computeChunkStats() {
	for i := range ps.allChunks {
		c := &ps.allChunks[i]
		switch c.Type {
		case chunk.ChunkObject:
			ps.result.ObjectChunks++
		case chunk.ChunkSpec:
			ps.result.SpecChunks++
		case chunk.ChunkSystem:
			ps.result.SystemChunks++
		}
		ps.result.TotalTokens += c.Tokens
		if c.Tokens < ps.result.MinTokens {
			ps.result.MinTokens = c.Tokens
		}
		if c.Tokens > ps.result.MaxTokens {
			ps.result.MaxTokens = c.Tokens
		}
		if c.Oversized {
			ps.result.Oversized++
		}
	}
	ps.result.TotalChunks = len(ps.allChunks)
	if ps.result.TotalChunks > 0 {
		ps.result.AvgTokens = ps.result.TotalTokens / ps.result.TotalChunks
	}
}

// stageWriteChunks writes new/modified chunk files to disk.
func (ps *pipelineState) stageWriteChunks() error {
	ps.progress(StageWrite, fmt.Sprintf("Writing %d chunk files", len(ps.newChunks)))

	if _, err := WriteChunkFiles(ps.cfg.CompiledDir, ps.newChunks); err != nil {
		return fmt.Errorf("writing chunk files: %w", err)
	}
	return nil
}

// stageIndex builds the flat BM25 search index from all chunks.
// The field-weighted BM25F index is built separately in stageFieldIndex
// after taxonomy extraction provides per-chunk terms.
func (ps *pipelineState) stageIndex() error {
	ps.progress(StageIndex, "Building search index (bm25)")

	indexStart := time.Now()

	ps.idx = index.NewFromConfig(ps.cfg.BM25)
	ps.idx.BuildFromChunks(ps.allChunks)

	if err := ps.idx.Save(ps.cfg.CompiledDir); err != nil {
		return fmt.Errorf("saving BM25 index: %w", err)
	}

	ps.result.IndexSeconds = time.Since(indexStart).Seconds()
	return nil
}

// stageFieldIndex builds the field-weighted BM25F index from all chunks.
// This runs after taxonomy extraction so that per-chunk terms are available
// for the terms field. Both BM25 and BM25F indexes are always built so
// users can switch between them without recompiling.
func (ps *pipelineState) stageFieldIndex() error {
	ps.progress(StageIndex, "Building field-weighted index (bm25f)")

	fieldStart := time.Now()

	ps.fieldIdx = index.NewFieldIndex(ps.cfg.BM25F)

	// Build a map of chunk ID → term keys for the terms field.
	var termsByChunk map[string][]string
	if ps.taxResult != nil && ps.taxResult.ChunkTerms != nil {
		termsByChunk = ps.taxResult.ChunkTerms
	}

	ps.fieldIdx.BuildFieldIndexFromChunks(ps.allChunks, termsByChunk)

	if err := ps.fieldIdx.SaveFieldIndex(ps.cfg.CompiledDir); err != nil {
		return fmt.Errorf("saving BM25F field index: %w", err)
	}

	ps.result.IndexSeconds += time.Since(fieldStart).Seconds()
	return nil
}

// stageTaxonomy extracts taxonomy terms from all chunks and preserves
// aliases from unchanged terms in incremental mode.
func (ps *pipelineState) stageTaxonomy() error {
	ps.progress(StageTaxonomy, fmt.Sprintf("Extracting taxonomy from %d chunks", ps.result.TotalChunks))

	// Compute per-subdirectory system instruction hashes.
	topicsDir := filepath.Join(ps.cfg.RootDir, ps.cfg.SystemDir, "topics")
	var err error
	ps.systemHashes, err = manifest.HashSystemDirs(topicsDir)
	if err != nil {
		return fmt.Errorf("hashing system instruction directories: %w", err)
	}

	taxonomyInstructionsHash := ps.systemHashes["taxonomy-generation"]

	ps.taxResult = taxonomy.Extract(ps.allChunks, ps.cfg.Taxonomy, ps.cfg.Encoding, taxonomyInstructionsHash)
	ps.result.TaxonomyTerms = ps.taxResult.Stats.CanonicalTerms
	ps.result.TaxonomySeconds = ps.taxResult.Seconds

	// Preserve existing aliases for terms whose source chunks haven't changed.
	if ps.incremental && ps.prevTaxonomy != nil {
		preserveUnchangedAliases(ps.taxResult.Taxonomy, ps.prevTaxonomy, ps.taxResult.ChunkTerms, ps.unchangedChunkIDs)
	}

	// Write taxonomy.yml.
	taxPath := taxonomy.TaxonomyPath(ps.cfg.CompiledDir)
	if err := ps.taxResult.Taxonomy.WriteTo(taxPath); err != nil {
		return fmt.Errorf("writing taxonomy: %w", err)
	}

	return nil
}

// stageManifests generates manifest, metadata, and hashes files.
func (ps *pipelineState) stageManifests() error {
	ps.progress(StageManifest, "Generating manifest files")

	manifestStart := time.Now()

	// Derive bridge hash from the per-subdirectory system hashes.
	var bridgeHash *string
	if h, ok := ps.systemHashes["bridge-summaries"]; ok {
		bridgeHash = &h
	}

	mfst := manifest.BuildManifest(ps.allChunks, ps.cfg.Encoding, bridgeHash, ps.taxResult.ChunkTerms)

	// Generate deterministic bridge summaries for all adjacent chunk pairs.
	// These use heading transitions, RAKE key phrases, and last-sentence
	// extraction — no LLM calls required.
	detBridges := bridge.GenerateAll(ps.allChunks, mfst)
	for id, b := range detBridges {
		if entry := mfst.LookupEntry(id); entry != nil {
			text := b
			entry.BridgeToNext = &text
		}
	}
	ps.result.DetBridgeCount = len(detBridges)

	// Preserve bridges from previous manifest for unchanged chunk pairs.
	if ps.incremental && ps.prevManifest != nil {
		preserveUnchangedBridges(mfst, ps.prevManifest, ps.unchangedChunkIDs)
	}

	meta := manifest.BuildMetadata(ps.allChunks, ps.blocksBySource)
	hashes := manifest.BuildHashes(ps.fileHashes, ps.systemHashes)

	// Count cross-reference statistics.
	for _, entry := range meta.Documents {
		if len(entry.ReferencesTo) > 0 {
			ps.result.CrossRefLinks += len(entry.ReferencesTo)
			ps.result.CrossRefDocs++
		}
	}

	ps.heur = manifest.NewHeuristics(ps.cfg.Version, ps.cfg.Encoding)
	ps.heur.SetSources(&manifest.SourcesSection{
		TotalFiles:   ps.result.TotalFiles,
		LocalFiles:   ps.result.TotalFiles - ps.result.SpecFiles,
		PackageFiles: 0,
		New:          ps.result.NewFiles,
		Modified:     ps.result.ModifiedFiles,
		Unchanged:    ps.result.UnchangedFiles,
		SpecFiles:    ps.result.SpecFiles,
	})
	ps.heur.SetChunkStats(&manifest.ChunksSection{
		Total:         ps.result.TotalChunks,
		Objects:       ps.result.ObjectChunks,
		Specs:         ps.result.SpecChunks,
		System:        ps.result.SystemChunks,
		TotalTokens:   ps.result.TotalTokens,
		AverageTokens: ps.result.AvgTokens,
		MinTokens:     ps.result.MinTokens,
		MaxTokens:     ps.result.MaxTokens,
		Oversized:     ps.result.Oversized,
	})
	ps.heur.SetBM25Stats(ps.idx)

	ps.result.ManifestSeconds = time.Since(manifestStart).Seconds()

	return writeManifests(ps.cfg.CompiledDir, mfst, meta, hashes, ps.heur)
}

// stageContext assembles always-loaded session context.
func (ps *pipelineState) stageContext() error {
	if ps.cfg.Session == nil || len(ps.cfg.Session.AlwaysLoaded) == 0 {
		return nil
	}

	ps.progress(StageContext, "Assembling session context")

	contextStart := time.Now()

	packagesDir := project.PackagesPath(ps.cfg.RootDir)
	resolved, resolveErr := codectx.Resolve(ps.cfg.RootDir, packagesDir, ps.cfg.Session.AlwaysLoaded)
	if resolveErr != nil {
		return fmt.Errorf("resolving session context: %w", resolveErr)
	}

	budget := ps.cfg.Session.EffectiveBudget()

	assembly, assembleErr := codectx.Assemble(resolved, ps.cfg.Encoding, budget)
	if assembleErr != nil {
		return fmt.Errorf("assembling session context: %w", assembleErr)
	}

	if err := codectx.WriteContextMD(ps.cfg.CompiledDir, assembly); err != nil {
		return fmt.Errorf("writing context.md: %w", err)
	}

	ps.result.SessionTokens = assembly.TotalTokens
	ps.result.SessionBudget = assembly.Budget
	ps.result.Warnings = append(ps.result.Warnings, assembly.Warnings...)

	for _, entry := range assembly.Entries {
		ps.result.SessionEntries = append(ps.result.SessionEntries, SessionEntryResult{
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
	ps.heur.SetSession(assembly.TotalTokens, assembly.Budget, utilization, heurEntries)

	ps.result.ContextSeconds = time.Since(contextStart).Seconds()
	return nil
}

// stageLink updates existing AI tool entry point files if their context
// path is stale. Non-fatal: failures are recorded as warnings.
func (ps *pipelineState) stageLink() {
	if ps.cfg.ProjectDir == "" {
		return
	}

	contextRelPath := contextRelativePath(ps.cfg)
	needsUpdate := link.NeedsUpdate(ps.cfg.ProjectDir, contextRelPath)

	if len(needsUpdate) == 0 {
		return
	}

	ps.progress(StageLink, fmt.Sprintf("Updating %d entry point(s)", len(needsUpdate)))

	linkStart := time.Now()

	if _, linkErr := link.Write(ps.cfg.ProjectDir, contextRelPath, needsUpdate); linkErr != nil {
		// Non-fatal: warn but don't fail compilation.
		ps.result.Warnings = append(ps.result.Warnings,
			fmt.Sprintf("failed to update entry points: %v", linkErr))
	}

	ps.result.LinkSeconds = time.Since(linkStart).Seconds()
}

// stageFinalize populates final heuristics (taxonomy, timing, incremental)
// and writes all manifest files. This must be called after all other stages.
func (ps *pipelineState) stageFinalize(totalStart time.Time) error {
	// Populate taxonomy heuristics.
	// Count total aliases across all terms (corpus + dictionary).
	totalAliases := 0
	for _, term := range ps.taxResult.Taxonomy.Terms {
		totalAliases += len(term.Aliases)
	}
	var avgAliases float64
	if ps.taxResult.Stats.CanonicalTerms > 0 {
		avgAliases = float64(totalAliases) / float64(ps.taxResult.Stats.CanonicalTerms)
	}

	ps.heur.SetTaxonomyStats(&manifest.TaxonomySection{
		CanonicalTerms:               ps.taxResult.Stats.CanonicalTerms,
		TotalAliases:                 totalAliases,
		AverageAliasesPerTerm:        avgAliases,
		TermsFromHeadings:            ps.taxResult.Stats.TermsFromHeadings,
		TermsFromCodeIdents:          ps.taxResult.Stats.TermsFromCodeIdents,
		TermsFromBoldTerms:           ps.taxResult.Stats.TermsFromBoldTerms,
		TermsFromStructuredPositions: ps.taxResult.Stats.TermsFromStructured,
		TermsFromPOSExtraction:       ps.taxResult.Stats.TermsFromPOS,
		CorpusAbbreviationPairs:      ps.taxResult.Stats.CorpusAbbreviationPairs,
		TermsWithCorpusAliases:       ps.taxResult.Stats.TermsWithCorpusAliases,
	})

	timing := &manifest.TimingSection{
		TotalSeconds:       time.Since(totalStart).Seconds(),
		ParseValidate:      ps.result.ParseSeconds,
		StripNormalize:     0, // combined with parse in this implementation
		Chunking:           ps.result.ChunkSeconds,
		BM25Indexing:       ps.result.IndexSeconds,
		TaxonomyExtraction: ps.result.TaxonomySeconds,
		ManifestGeneration: ps.result.ManifestSeconds,
		ContextAssembly:    ps.result.ContextSeconds,
		SyncEntryPoints:    ps.result.LinkSeconds,
	}
	ps.heur.SetTiming(timing)

	// Populate incremental heuristics.
	if ps.incremental {
		instrChanges := DetectInstructionChanges(ps.systemHashes, ps.prevSystemHashes)

		ps.heur.SetIncremental(&manifest.IncrementalSection{
			FullRecompile: false,
			StagesSkipped: []string{},
			StagesRerun: []string{
				"parse_validate", "chunking", "bm25_indexing",
				"taxonomy_extraction", "manifest_generation",
			},
			SystemInstructionsChanged: &manifest.SystemInstructionsChanged{
				ContextAssembly: instrChanges.ContextAssembly,
			},
		})
	} else {
		ps.heur.SetIncremental(&manifest.IncrementalSection{
			FullRecompile: true,
		})
	}

	return ps.heur.WriteTo(manifest.HeuristicsPath(ps.cfg.CompiledDir))
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

// writeManifests writes manifest, metadata, and hashes files to the compiled directory.
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
