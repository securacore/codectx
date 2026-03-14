package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/manifest"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// NewHeuristics
// ---------------------------------------------------------------------------

func TestNewHeuristics_SetsCompiledAt(t *testing.T) {
	h := manifest.NewHeuristics("0.1.0", "cl100k_base")
	if h.CompiledAt == "" {
		t.Error("expected compiled_at to be set")
	}
}

func TestNewHeuristics_SetsVersion(t *testing.T) {
	h := manifest.NewHeuristics("1.2.3", "cl100k_base")
	if h.CompilerVersion != "1.2.3" {
		t.Errorf("expected version %q, got %q", "1.2.3", h.CompilerVersion)
	}
}

func TestNewHeuristics_SetsEncoding(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	if h.Encoding != "cl100k_base" {
		t.Errorf("expected encoding %q, got %q", "cl100k_base", h.Encoding)
	}
}

func TestNewHeuristics_InitializesSections(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")

	if h.Sources == nil {
		t.Error("expected Sources section to be initialized")
	}
	if h.Chunks == nil {
		t.Error("expected Chunks section to be initialized")
	}
	if h.Taxonomy == nil {
		t.Error("expected Taxonomy section to be initialized")
	}
	if h.Session == nil {
		t.Error("expected Session section to be initialized")
	}
	if h.BM25 == nil {
		t.Error("expected BM25 section to be initialized")
	}
	if h.Incremental == nil {
		t.Error("expected Incremental section to be initialized")
	}
	if h.Timing == nil {
		t.Error("expected Timing section to be initialized")
	}
}

func TestNewHeuristics_IncrementalHasSystemInstructionsChanged(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	if h.Incremental.SystemInstructionsChanged == nil {
		t.Error("expected SystemInstructionsChanged to be initialized")
	}
}

func TestNewHeuristics_SectionsHaveZeroDefaults(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")

	if h.Sources.TotalFiles != 0 {
		t.Errorf("expected 0 total_files, got %d", h.Sources.TotalFiles)
	}
	if h.Chunks.Total != 0 {
		t.Errorf("expected 0 total chunks, got %d", h.Chunks.Total)
	}
	if h.Taxonomy.CanonicalTerms != 0 {
		t.Errorf("expected 0 canonical_terms, got %d", h.Taxonomy.CanonicalTerms)
	}
	if h.Session.TotalTokens != 0 {
		t.Errorf("expected 0 session total_tokens, got %d", h.Session.TotalTokens)
	}
	if h.Timing.TotalSeconds != 0 {
		t.Errorf("expected 0 total_seconds, got %f", h.Timing.TotalSeconds)
	}
}

// ---------------------------------------------------------------------------
// SetSources
// ---------------------------------------------------------------------------

func TestSetSources(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetSources(&manifest.SourcesSection{
		TotalFiles: 342, LocalFiles: 298, PackageFiles: 44,
		New: 3, Modified: 7, Unchanged: 332, SpecFiles: 86,
	})

	s := h.Sources
	if s.TotalFiles != 342 {
		t.Errorf("TotalFiles: expected 342, got %d", s.TotalFiles)
	}
	if s.LocalFiles != 298 {
		t.Errorf("LocalFiles: expected 298, got %d", s.LocalFiles)
	}
	if s.PackageFiles != 44 {
		t.Errorf("PackageFiles: expected 44, got %d", s.PackageFiles)
	}
	if s.New != 3 {
		t.Errorf("New: expected 3, got %d", s.New)
	}
	if s.Modified != 7 {
		t.Errorf("Modified: expected 7, got %d", s.Modified)
	}
	if s.Unchanged != 332 {
		t.Errorf("Unchanged: expected 332, got %d", s.Unchanged)
	}
	if s.SpecFiles != 86 {
		t.Errorf("SpecFiles: expected 86, got %d", s.SpecFiles)
	}
}

// ---------------------------------------------------------------------------
// SetChunkStats
// ---------------------------------------------------------------------------

func TestSetChunkStats(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetChunkStats(&manifest.ChunksSection{
		Total: 4850, Objects: 3842, Specs: 879, System: 129,
		TotalTokens: 2134500, AverageTokens: 440, MinTokens: 203, MaxTokens: 1847, Oversized: 4,
	})

	c := h.Chunks
	if c.Total != 4850 {
		t.Errorf("Total: expected 4850, got %d", c.Total)
	}
	if c.Objects != 3842 {
		t.Errorf("Objects: expected 3842, got %d", c.Objects)
	}
	if c.Specs != 879 {
		t.Errorf("Specs: expected 879, got %d", c.Specs)
	}
	if c.System != 129 {
		t.Errorf("System: expected 129, got %d", c.System)
	}
	if c.TotalTokens != 2134500 {
		t.Errorf("TotalTokens: expected 2134500, got %d", c.TotalTokens)
	}
	if c.AverageTokens != 440 {
		t.Errorf("AverageTokens: expected 440, got %d", c.AverageTokens)
	}
	if c.MinTokens != 203 {
		t.Errorf("MinTokens: expected 203, got %d", c.MinTokens)
	}
	if c.MaxTokens != 1847 {
		t.Errorf("MaxTokens: expected 1847, got %d", c.MaxTokens)
	}
	if c.Oversized != 4 {
		t.Errorf("Oversized: expected 4, got %d", c.Oversized)
	}
}

// ---------------------------------------------------------------------------
// SetBM25Stats
// ---------------------------------------------------------------------------

func TestSetBM25Stats(t *testing.T) {
	// Build a minimal index with known stats.
	idx := index.New(1.2, 0.75)

	// We can't easily populate the internal BM25 structs, so test with
	// an empty index — stats should all be zero.
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetBM25Stats(idx)

	if h.BM25.Objects == nil {
		t.Fatal("expected BM25.Objects to be set")
	}
	if h.BM25.Specs == nil {
		t.Fatal("expected BM25.Specs to be set")
	}
	if h.BM25.System == nil {
		t.Fatal("expected BM25.System to be set")
	}

	// Empty index — all zero.
	if h.BM25.Objects.IndexedTerms != 0 {
		t.Errorf("expected 0 indexed_terms for empty index, got %d", h.BM25.Objects.IndexedTerms)
	}
	if h.BM25.Objects.IndexedChunks != 0 {
		t.Errorf("expected 0 indexed_chunks for empty index, got %d", h.BM25.Objects.IndexedChunks)
	}
}

// ---------------------------------------------------------------------------
// SetTiming
// ---------------------------------------------------------------------------

func TestSetTiming(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetTiming(&manifest.TimingSection{
		TotalSeconds:       47.3,
		ParseValidate:      2.1,
		StripNormalize:     1.8,
		Chunking:           8.4,
		BM25Indexing:       3.2,
		TaxonomyExtraction: 0.0,
		ManifestGeneration: 4.5,
		ContextAssembly:    3.8,
		SyncEntryPoints:    1.4,
	})

	if h.Timing.TotalSeconds != 47.3 {
		t.Errorf("TotalSeconds: expected 47.3, got %f", h.Timing.TotalSeconds)
	}
	if h.Timing.Chunking != 8.4 {
		t.Errorf("Chunking: expected 8.4, got %f", h.Timing.Chunking)
	}
}

// ---------------------------------------------------------------------------
// SetIncremental
// ---------------------------------------------------------------------------

func TestSetIncremental(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetIncremental(&manifest.IncrementalSection{
		FullRecompile: false,
		StagesSkipped: []string{"taxonomy_extraction"},
		StagesRerun:   []string{"chunking", "bm25_indexing", "manifest_generation"},
		SystemInstructionsChanged: &manifest.SystemInstructionsChanged{
			ContextAssembly: true,
		},
	})

	if h.Incremental.FullRecompile {
		t.Error("expected FullRecompile to be false")
	}
	if len(h.Incremental.StagesSkipped) != 1 {
		t.Errorf("expected 1 skipped stage, got %d", len(h.Incremental.StagesSkipped))
	}
	if len(h.Incremental.StagesRerun) != 3 {
		t.Errorf("expected 3 rerun stages, got %d", len(h.Incremental.StagesRerun))
	}
	if !h.Incremental.SystemInstructionsChanged.ContextAssembly {
		t.Error("expected ContextAssembly to be true")
	}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func TestHeuristics_WriteTo_RoundTrip(t *testing.T) {
	h := manifest.NewHeuristics("0.1.0", "cl100k_base")
	h.SetSources(&manifest.SourcesSection{
		TotalFiles: 100, LocalFiles: 90, PackageFiles: 10,
		New: 5, Modified: 3, Unchanged: 92, SpecFiles: 20,
	})
	h.SetChunkStats(&manifest.ChunksSection{
		Total: 500, Objects: 400, Specs: 80, System: 20,
		TotalTokens: 220000, AverageTokens: 440, MinTokens: 200, MaxTokens: 1800, Oversized: 2,
	})
	h.SetTiming(&manifest.TimingSection{
		TotalSeconds:  10.5,
		ParseValidate: 1.2,
		Chunking:      3.4,
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "heuristics.yml")

	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading heuristics: %v", err)
	}

	content := string(data)

	// Should have the header comment.
	if !strings.HasPrefix(content, "# codectx compilation heuristics") {
		t.Error("expected header comment")
	}

	// Should be valid YAML.
	var loaded manifest.Heuristics
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling heuristics: %v", err)
	}

	if loaded.CompilerVersion != "0.1.0" {
		t.Errorf("round-trip version: expected %q, got %q", "0.1.0", loaded.CompilerVersion)
	}
	if loaded.Encoding != "cl100k_base" {
		t.Errorf("round-trip encoding: expected %q, got %q", "cl100k_base", loaded.Encoding)
	}
	if loaded.Sources.TotalFiles != 100 {
		t.Errorf("round-trip total_files: expected 100, got %d", loaded.Sources.TotalFiles)
	}
	if loaded.Chunks.Total != 500 {
		t.Errorf("round-trip chunks total: expected 500, got %d", loaded.Chunks.Total)
	}
	if loaded.Timing.TotalSeconds != 10.5 {
		t.Errorf("round-trip timing total_seconds: expected 10.5, got %f", loaded.Timing.TotalSeconds)
	}
}

func TestHeuristics_WriteTo_2SpaceIndent(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")

	dir := t.TempDir()
	path := filepath.Join(dir, "heuristics.yml")
	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Contains(content, "\t") {
		t.Error("heuristics should not contain tabs")
	}

	// Verify nested keys use 2-space indent.
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent > 0 && indent%2 != 0 {
			t.Errorf("line %d: indent %d is not a multiple of 2: %q", i+1, indent, line)
		}
	}
}

func TestHeuristics_WriteTo_ContainsAllSections(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetSources(&manifest.SourcesSection{
		TotalFiles: 10, LocalFiles: 8, PackageFiles: 2,
		New: 1, Modified: 1, Unchanged: 8, SpecFiles: 3,
	})
	h.SetChunkStats(&manifest.ChunksSection{
		Total: 50, Objects: 40, Specs: 8, System: 2,
		TotalTokens: 22000, AverageTokens: 440, MinTokens: 200, MaxTokens: 800, Oversized: 0,
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "heuristics.yml")
	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// All top-level sections should be present.
	sections := []string{
		"compiled_at:", "compiler_version:", "encoding:",
		"sources:", "chunks:", "taxonomy:", "session:",
		"bm25:", "incremental:", "timing:",
	}
	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("expected section %q in output", section)
		}
	}
}

func TestHeuristics_WriteTo_TaxonomyPlaceholderZeros(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")

	dir := t.TempDir()
	path := filepath.Join(dir, "heuristics.yml")
	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded manifest.Heuristics
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	if loaded.Taxonomy.CanonicalTerms != 0 {
		t.Errorf("expected 0 canonical_terms placeholder, got %d", loaded.Taxonomy.CanonicalTerms)
	}
	if loaded.Taxonomy.TotalAliases != 0 {
		t.Errorf("expected 0 total_aliases placeholder, got %d", loaded.Taxonomy.TotalAliases)
	}
}

func TestHeuristics_WriteTo_SessionPlaceholderZeros(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")

	dir := t.TempDir()
	path := filepath.Join(dir, "heuristics.yml")
	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded manifest.Heuristics
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	if loaded.Session.TotalTokens != 0 {
		t.Errorf("expected 0 session total_tokens placeholder, got %d", loaded.Session.TotalTokens)
	}
	if loaded.Session.Budget != 0 {
		t.Errorf("expected 0 session budget placeholder, got %d", loaded.Session.Budget)
	}
}

// ---------------------------------------------------------------------------
// HeuristicsPath
// ---------------------------------------------------------------------------

func TestHeuristicsPath(t *testing.T) {
	got := manifest.HeuristicsPath("/project/.codectx/compiled")
	expected := filepath.Join("/project/.codectx/compiled", "heuristics.yml")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// ---------------------------------------------------------------------------
// SetSession
// ---------------------------------------------------------------------------

func TestSetSession_PopulatesFields(t *testing.T) {
	h := manifest.NewHeuristics("0.1.0", "cl100k_base")
	h.SetSession(28450, 30000, "94.8%", []manifest.SessionEntry{
		{Path: "foundation/coding-standards", Tokens: 8200},
		{Path: "foundation/architecture", Tokens: 6100},
	})

	if h.Session.TotalTokens != 28450 {
		t.Errorf("expected 28450, got %d", h.Session.TotalTokens)
	}
	if h.Session.Budget != 30000 {
		t.Errorf("expected 30000, got %d", h.Session.Budget)
	}
	if h.Session.Utilization != "94.8%" {
		t.Errorf("expected %q, got %q", "94.8%", h.Session.Utilization)
	}
	if len(h.Session.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(h.Session.Entries))
	}
	if h.Session.Entries[0].Tokens != 8200 {
		t.Errorf("expected 8200, got %d", h.Session.Entries[0].Tokens)
	}
}

// ---------------------------------------------------------------------------
// Set* methods with nil sections
// ---------------------------------------------------------------------------

func TestSetSources_Nil(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetSources(nil)
	if h.Sources != nil {
		t.Error("expected Sources to be nil after SetSources(nil)")
	}
}

func TestSetChunkStats_Nil(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetChunkStats(nil)
	if h.Chunks != nil {
		t.Error("expected Chunks to be nil after SetChunkStats(nil)")
	}
}

func TestSetTaxonomyStats_Nil(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetTaxonomyStats(nil)
	if h.Taxonomy != nil {
		t.Error("expected Taxonomy to be nil after SetTaxonomyStats(nil)")
	}
}

func TestSetTiming_Nil(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetTiming(nil)
	if h.Timing != nil {
		t.Error("expected Timing to be nil after SetTiming(nil)")
	}
}

func TestSetIncremental_Nil(t *testing.T) {
	h := manifest.NewHeuristics("dev", "cl100k_base")
	h.SetIncremental(nil)
	if h.Incremental != nil {
		t.Error("expected Incremental to be nil after SetIncremental(nil)")
	}
}

func TestSetSession_SerializesToYAML(t *testing.T) {
	h := manifest.NewHeuristics("0.1.0", "cl100k_base")
	h.SetSession(1000, 5000, "20.0%", []manifest.SessionEntry{
		{Path: "foundation/test", Tokens: 1000},
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "heuristics.yml")
	if err := h.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "total_tokens: 1000") {
		t.Error("expected total_tokens in YAML")
	}
	if !strings.Contains(content, "budget: 5000") {
		t.Error("expected budget in YAML")
	}
	if !strings.Contains(content, "20.0%") {
		t.Error("expected utilization in YAML")
	}
}
