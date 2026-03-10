package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/project"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testChunks() []chunk.Chunk {
	return []chunk.Chunk{
		{
			ID:      "obj:aaa.1",
			Type:    chunk.ChunkObject,
			Source:  "docs/topics/auth/jwt.md",
			Heading: "Authentication > JWT",
			Content: "JWT authentication token validation refresh flow",
			Tokens:  50,
		},
		{
			ID:      "obj:bbb.1",
			Type:    chunk.ChunkObject,
			Source:  "docs/topics/auth/jwt.md",
			Heading: "Authentication > JWT > Refresh",
			Content: "JWT refresh token rotation automatic renewal",
			Tokens:  45,
		},
		{
			ID:      "spec:ccc.1",
			Type:    chunk.ChunkSpec,
			Source:  "docs/topics/auth/jwt.spec.md",
			Heading: "Authentication > JWT",
			Content: "We chose JWT because stateless authentication reduces database load",
			Tokens:  40,
		},
		{
			ID:      "sys:ddd.1",
			Type:    chunk.ChunkSystem,
			Source:  "system/topics/taxonomy/README.md",
			Heading: "Taxonomy > Rules",
			Content: "Taxonomy generation rules for alias extraction and term normalization",
			Tokens:  55,
		},
		{
			ID:      "obj:eee.1",
			Type:    chunk.ChunkObject,
			Source:  "docs/topics/database/connection.md",
			Heading: "Database > Connection Pooling",
			Content: "database connection pooling configuration maximum idle connections",
			Tokens:  48,
		},
	}
}

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNew_CreatesThreeIndexes(t *testing.T) {
	idx := New(1.2, 0.75)
	if len(idx.Indexes) != 3 {
		t.Fatalf("expected 3 indexes, got %d", len(idx.Indexes))
	}
	for _, it := range AllIndexTypes() {
		if _, ok := idx.Indexes[it]; !ok {
			t.Errorf("missing index for type %q", it)
		}
	}
}

func TestNewFromConfig(t *testing.T) {
	cfg := project.BM25Config{K1: 2.0, B: 0.5}
	idx := NewFromConfig(cfg)
	bm25 := idx.Indexes[IndexObjects]
	if bm25.K1 != 2.0 {
		t.Errorf("expected k1=2.0, got %f", bm25.K1)
	}
	if bm25.B != 0.5 {
		t.Errorf("expected b=0.5, got %f", bm25.B)
	}
}

// ---------------------------------------------------------------------------
// IndexTypeForChunk
// ---------------------------------------------------------------------------

func TestIndexTypeForChunk(t *testing.T) {
	tests := []struct {
		ct       chunk.ChunkType
		expected IndexType
	}{
		{chunk.ChunkObject, IndexObjects},
		{chunk.ChunkSpec, IndexSpecs},
		{chunk.ChunkSystem, IndexSystem},
		{chunk.ChunkType("unknown"), IndexObjects}, // default
	}

	for _, tt := range tests {
		got := IndexTypeForChunk(tt.ct)
		if got != tt.expected {
			t.Errorf("IndexTypeForChunk(%q) = %q, want %q", tt.ct, got, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// AllIndexTypes
// ---------------------------------------------------------------------------

func TestAllIndexTypes(t *testing.T) {
	types := AllIndexTypes()
	if len(types) != 3 {
		t.Fatalf("expected 3 index types, got %d", len(types))
	}
}

// ---------------------------------------------------------------------------
// BuildFromChunks
// ---------------------------------------------------------------------------

func TestBuildFromChunks_RoutesCorrectly(t *testing.T) {
	idx := New(1.2, 0.75)
	chunks := testChunks()
	idx.BuildFromChunks(chunks)

	// 3 object chunks, 1 spec chunk, 1 system chunk.
	if idx.Indexes[IndexObjects].DocCount != 3 {
		t.Errorf("objects: expected 3 docs, got %d", idx.Indexes[IndexObjects].DocCount)
	}
	if idx.Indexes[IndexSpecs].DocCount != 1 {
		t.Errorf("specs: expected 1 doc, got %d", idx.Indexes[IndexSpecs].DocCount)
	}
	if idx.Indexes[IndexSystem].DocCount != 1 {
		t.Errorf("system: expected 1 doc, got %d", idx.Indexes[IndexSystem].DocCount)
	}
}

func TestBuildFromChunks_EmptyChunks(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(nil)

	for _, it := range AllIndexTypes() {
		if idx.Indexes[it].DocCount != 0 {
			t.Errorf("%s: expected 0 docs, got %d", it, idx.Indexes[it].DocCount)
		}
	}
}

func TestBuildFromChunks_PreservesChunkIDs(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	objIDs := idx.Indexes[IndexObjects].DocIDs
	expected := []string{"obj:aaa.1", "obj:bbb.1", "obj:eee.1"}
	if len(objIDs) != len(expected) {
		t.Fatalf("expected %d object IDs, got %d", len(expected), len(objIDs))
	}
	for i, want := range expected {
		if objIDs[i] != want {
			t.Errorf("object ID %d: expected %q, got %q", i, want, objIDs[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Query
// ---------------------------------------------------------------------------

func TestQuery_FindsRelevantChunks(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	results := idx.Query(IndexObjects, "JWT authentication", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'JWT authentication' in objects")
	}

	// The JWT-related object chunks should be returned.
	found := map[string]bool{}
	for _, r := range results {
		found[r.ChunkID] = true
	}
	if !found["obj:aaa.1"] {
		t.Error("expected obj:aaa.1 in results")
	}
}

func TestQuery_SpecIndexIsolated(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	// Query the spec index for "JWT".
	results := idx.Query(IndexSpecs, "JWT", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'JWT' in specs")
	}

	// Should only return spec chunks.
	for _, r := range results {
		if r.ChunkID != "spec:ccc.1" {
			t.Errorf("unexpected chunk in spec results: %q", r.ChunkID)
		}
	}
}

func TestQuery_SystemIndexIsolated(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	results := idx.Query(IndexSystem, "taxonomy", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'taxonomy' in system")
	}
	if results[0].ChunkID != "sys:ddd.1" {
		t.Errorf("expected sys:ddd.1, got %q", results[0].ChunkID)
	}
}

func TestQuery_NoResultsForUnrelatedQuery(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	results := idx.Query(IndexObjects, "completely unrelated xyzzy", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestQuery_EmptyQuery(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	results := idx.Query(IndexObjects, "", 10)
	if results != nil {
		t.Errorf("expected nil for empty query, got %v", results)
	}
}

func TestQuery_StopwordOnlyQuery(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	results := idx.Query(IndexObjects, "the is a", 10)
	if results != nil {
		t.Errorf("expected nil for stopword-only query, got %v", results)
	}
}

func TestQuery_UnknownIndexType(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	results := idx.Query(IndexType("bogus"), "jwt", 10)
	if results != nil {
		t.Errorf("expected nil for unknown index type, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// QueryAll
// ---------------------------------------------------------------------------

func TestQueryAll_ReturnsFromMultipleIndexes(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	results := idx.QueryAll("JWT", 10)

	// JWT appears in objects and specs.
	if _, ok := results[IndexObjects]; !ok {
		t.Error("expected objects results")
	}
	if _, ok := results[IndexSpecs]; !ok {
		t.Error("expected specs results")
	}
}

func TestQueryAll_EmptyQuery(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	results := idx.QueryAll("", 10)
	if len(results) != 0 {
		t.Errorf("expected empty results for empty query, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

func TestStats_ReturnsCorrectCounts(t *testing.T) {
	idx := New(1.2, 0.75)
	idx.BuildFromChunks(testChunks())

	stats := idx.Stats(IndexObjects)
	if stats.IndexedChunks != 3 {
		t.Errorf("expected 3 indexed chunks, got %d", stats.IndexedChunks)
	}
	if stats.IndexedTerms == 0 {
		t.Error("expected non-zero indexed terms")
	}
	if stats.AvgChunkLength == 0 {
		t.Error("expected non-zero average chunk length")
	}
}

func TestStats_UnknownType(t *testing.T) {
	idx := New(1.2, 0.75)
	stats := idx.Stats(IndexType("bogus"))
	if stats.IndexedChunks != 0 {
		t.Errorf("expected 0 for unknown type, got %d", stats.IndexedChunks)
	}
}

// ---------------------------------------------------------------------------
// Serialization round-trip
// ---------------------------------------------------------------------------

func TestSaveLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	compiledDir := filepath.Join(tmpDir, "compiled")

	// Build index.
	original := New(1.2, 0.75)
	original.BuildFromChunks(testChunks())

	// Save.
	if err := original.Save(compiledDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify files exist.
	for _, it := range AllIndexTypes() {
		path := filepath.Join(compiledDir, "bm25", string(it), indexFileName)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected index file at %s: %v", path, err)
		}
	}

	// Load.
	loaded, err := Load(compiledDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded indexes match original.
	for _, it := range AllIndexTypes() {
		origBM25 := original.Indexes[it]
		loadBM25 := loaded.Indexes[it]

		if origBM25.DocCount != loadBM25.DocCount {
			t.Errorf("%s: DocCount mismatch: %d vs %d", it, origBM25.DocCount, loadBM25.DocCount)
		}
		if origBM25.K1 != loadBM25.K1 {
			t.Errorf("%s: K1 mismatch", it)
		}
		if origBM25.B != loadBM25.B {
			t.Errorf("%s: B mismatch", it)
		}
		if origBM25.AvgDocLen != loadBM25.AvgDocLen {
			t.Errorf("%s: AvgDocLen mismatch", it)
		}
	}
}

func TestSaveLoad_QueryAfterLoad(t *testing.T) {
	tmpDir := t.TempDir()
	compiledDir := filepath.Join(tmpDir, "compiled")

	// Build, save, load.
	original := New(1.2, 0.75)
	original.BuildFromChunks(testChunks())
	if err := original.Save(compiledDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded, err := Load(compiledDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Query the loaded index — should produce identical results.
	origResults := original.Query(IndexObjects, "JWT authentication", 10)
	loadResults := loaded.Query(IndexObjects, "JWT authentication", 10)

	if len(origResults) != len(loadResults) {
		t.Fatalf("result count mismatch: %d vs %d", len(origResults), len(loadResults))
	}

	for i := range origResults {
		if origResults[i].ChunkID != loadResults[i].ChunkID {
			t.Errorf("result %d: ChunkID mismatch: %q vs %q",
				i, origResults[i].ChunkID, loadResults[i].ChunkID)
		}
		if origResults[i].Score != loadResults[i].Score {
			t.Errorf("result %d: Score mismatch: %f vs %f",
				i, origResults[i].Score, loadResults[i].Score)
		}
	}
}

func TestLoad_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := Load(tmpDir)
	if err == nil {
		t.Error("expected error when loading from directory without index files")
	}
}

func TestSaveLoad_EmptyIndex(t *testing.T) {
	tmpDir := t.TempDir()
	compiledDir := filepath.Join(tmpDir, "compiled")

	empty := New(1.2, 0.75)
	// Build with no chunks — all indexes are empty.
	empty.BuildFromChunks(nil)

	if err := empty.Save(compiledDir); err != nil {
		t.Fatalf("Save empty index failed: %v", err)
	}

	loaded, err := Load(compiledDir)
	if err != nil {
		t.Fatalf("Load empty index failed: %v", err)
	}

	for _, it := range AllIndexTypes() {
		if loaded.Indexes[it].DocCount != 0 {
			t.Errorf("%s: expected 0 docs, got %d", it, loaded.Indexes[it].DocCount)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration: chunk.Chunk content is indexed, not meta header
// ---------------------------------------------------------------------------

func TestBuildFromChunks_IndexesContentNotMetaHeader(t *testing.T) {
	// The chunk Content field should be what's indexed, not the full
	// rendered output (which includes the meta header).
	chunks := []chunk.Chunk{
		{
			ID:      "obj:test.1",
			Type:    chunk.ChunkObject,
			Source:  "test.md",
			Heading: "Test Heading",
			Content: "unique-search-term-abc123",
			Tokens:  5,
		},
	}

	idx := New(1.2, 0.75)
	idx.BuildFromChunks(chunks)

	// Should find the content term.
	results := idx.Query(IndexObjects, "unique-search-term-abc123", 10)
	if len(results) == 0 {
		t.Error("expected to find chunk by its content")
	}

	// Should NOT find meta header fields like "codectx:meta" or "source:".
	metaResults := idx.Query(IndexObjects, "codectx:meta", 10)
	if len(metaResults) > 0 {
		t.Error("should not find meta header content in index")
	}
}

// ---------------------------------------------------------------------------
// Integration: Block content tokenization
// ---------------------------------------------------------------------------

func TestBuildFromChunks_TokenizesBlockContent(t *testing.T) {
	_ = markdown.Block{} // ensure import is used

	chunks := []chunk.Chunk{
		{
			ID:      "obj:code.1",
			Type:    chunk.ChunkObject,
			Content: "Use error-handling patterns with jwt.Verify for authentication",
		},
	}

	idx := New(1.2, 0.75)
	idx.BuildFromChunks(chunks)

	// Should find compound terms.
	results := idx.Query(IndexObjects, "error-handling", 10)
	if len(results) == 0 {
		t.Error("expected to find 'error-handling' as compound term")
	}

	// Should find dotted paths.
	results = idx.Query(IndexObjects, "jwt.Verify", 10)
	if len(results) == 0 {
		t.Error("expected to find 'jwt.verify' as dotted path")
	}
}

// ---------------------------------------------------------------------------
// NewFromConfig
// ---------------------------------------------------------------------------

func TestNewFromConfig_UsesConfigParams(t *testing.T) {
	cfg := project.BM25Config{K1: 1.5, B: 0.80}
	idx := NewFromConfig(cfg)

	if idx == nil {
		t.Fatal("expected non-nil index")
	}
	if len(idx.Indexes) != 3 {
		t.Errorf("expected 3 indexes, got %d", len(idx.Indexes))
	}

	// Verify the parameters propagated to each BM25 index.
	for it, bm25 := range idx.Indexes {
		if bm25.K1 != cfg.K1 {
			t.Errorf("index %s: K1 = %f, want %f", it, bm25.K1, cfg.K1)
		}
		if bm25.B != cfg.B {
			t.Errorf("index %s: B = %f, want %f", it, bm25.B, cfg.B)
		}
	}
}

func TestNewFromConfig_DefaultParams(t *testing.T) {
	defaults := project.DefaultPreferencesConfig().BM25
	idx := NewFromConfig(defaults)

	if idx == nil {
		t.Fatal("expected non-nil index")
	}

	for it, bm25 := range idx.Indexes {
		if bm25.K1 != defaults.K1 {
			t.Errorf("index %s: K1 = %f, want %f", it, bm25.K1, defaults.K1)
		}
		if bm25.B != defaults.B {
			t.Errorf("index %s: B = %f, want %f", it, bm25.B, defaults.B)
		}
	}
}
