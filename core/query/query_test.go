package query_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/query"
)

// testFixture creates a temporary compiled directory with BM25 indexes,
// manifest, and chunk files for testing. Returns the compiledDir path.
func testFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Define test chunks.
	chunks := []chunk.Chunk{
		{
			ID: "obj:abc123.1", Type: chunk.ChunkObject,
			Source: "topics/auth.md", Heading: "Auth > Login",
			Sequence: 1, TotalInFile: 3, Tokens: 400,
			Content: "JWT authentication login flow with refresh tokens and session management",
		},
		{
			ID: "obj:abc123.2", Type: chunk.ChunkObject,
			Source: "topics/auth.md", Heading: "Auth > Tokens",
			Sequence: 2, TotalInFile: 3, Tokens: 350,
			Content: "Token validation refresh expiry rotation strategies for secure access",
		},
		{
			ID: "obj:abc123.3", Type: chunk.ChunkObject,
			Source: "topics/auth.md", Heading: "Auth > Sessions",
			Sequence: 3, TotalInFile: 3, Tokens: 300,
			Content: "Session management cookies server side storage and invalidation",
		},
		{
			ID: "spec:def456.1", Type: chunk.ChunkSpec,
			Source: "topics/auth.spec.md", Heading: "Auth > Login",
			Sequence: 1, TotalInFile: 1, Tokens: 250,
			Content: "Reasoning behind JWT choice versus session cookies tradeoffs",
		},
		{
			ID: "sys:ghi789.1", Type: chunk.ChunkSystem,
			Source: "system/topics/compile.md", Heading: "Compiler",
			Sequence: 1, TotalInFile: 1, Tokens: 200,
			Content: "System documentation for the compilation pipeline stages",
		},
	}

	// Build and save BM25 indexes.
	idx := index.New(1.2, 0.75)
	idx.BuildFromChunks(chunks)
	if err := idx.Save(dir); err != nil {
		t.Fatalf("saving indexes: %v", err)
	}

	// Build and save manifest.
	mfst := manifest.BuildManifest(chunks, "cl100k_base", nil)
	manifestPath := filepath.Join(dir, "manifest.yml")
	if err := mfst.WriteTo(manifestPath); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}

	// Write chunk files on disk.
	writeChunkFile(t, dir, "objects", "abc123.1.md",
		"<!-- codectx:meta\nid: obj:abc123.1\n-->\n\n## Login\n\nJWT authentication login flow.")
	writeChunkFile(t, dir, "objects", "abc123.2.md",
		"<!-- codectx:meta\nid: obj:abc123.2\n-->\n\n## Tokens\n\nToken validation and rotation.")
	writeChunkFile(t, dir, "objects", "abc123.3.md",
		"<!-- codectx:meta\nid: obj:abc123.3\n-->\n\n## Sessions\n\nSession management overview.")
	writeChunkFile(t, dir, "specs", "def456.1.md",
		"<!-- codectx:meta\nid: spec:def456.1\n-->\n\n## Login Reasoning\n\nWhy JWT was chosen.")
	writeChunkFile(t, dir, "system", "ghi789.1.md",
		"<!-- codectx:meta\nid: sys:ghi789.1\n-->\n\n## Compiler\n\nCompilation pipeline docs.")

	return dir
}

// writeChunkFile writes a chunk file to the compiled directory.
func writeChunkFile(t *testing.T, compiledDir, subdir, filename, content string) {
	t.Helper()
	dir := filepath.Join(compiledDir, subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating chunk dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("writing chunk file: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CompiledDir
// ---------------------------------------------------------------------------

func TestCompiledDir(t *testing.T) {
	cfg := &project.Config{Root: "docs"}
	got := query.CompiledDir("/home/user/myproject", cfg)
	want := filepath.Join("/home/user/myproject", "docs", ".codectx", "compiled")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompiledDir_EmptyRoot(t *testing.T) {
	cfg := &project.Config{Root: ""}
	got := query.CompiledDir("/project", cfg)
	// Empty root defaults to "docs" via RootDir.
	want := filepath.Join("/project", "docs", ".codectx", "compiled")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// RunQuery
// ---------------------------------------------------------------------------

func TestRunQuery_ReturnsResults(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunQuery(compiledDir, "jwt authentication login", 10)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}

	if result.RawQuery != "jwt authentication login" {
		t.Errorf("RawQuery = %q, want %q", result.RawQuery, "jwt authentication login")
	}

	// Should have at least one instruction result (obj chunks contain jwt/auth terms).
	if len(result.Instructions) == 0 {
		t.Error("expected at least one instruction result")
	}

	// Verify first result has enriched metadata.
	first := result.Instructions[0]
	if first.ChunkID == "" {
		t.Error("ChunkID should not be empty")
	}
	if first.Score <= 0 {
		t.Error("Score should be positive")
	}
	if first.Heading == "" {
		t.Error("Heading should not be empty")
	}
	if first.Source == "" {
		t.Error("Source should not be empty")
	}
	if first.Tokens <= 0 {
		t.Error("Tokens should be positive")
	}
}

func TestRunQuery_SystemResults(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunQuery(compiledDir, "compilation pipeline", 10)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}

	if len(result.System) == 0 {
		t.Error("expected at least one system result for 'compilation pipeline'")
	}
}

func TestRunQuery_SpecResults(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunQuery(compiledDir, "jwt tradeoffs reasoning", 10)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}

	if len(result.Reasoning) == 0 {
		t.Error("expected at least one reasoning result for 'jwt tradeoffs'")
	}
}

func TestRunQuery_TopNLimitsResults(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunQuery(compiledDir, "auth token session", 1)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}

	if len(result.Instructions) > 1 {
		t.Errorf("expected at most 1 instruction result, got %d", len(result.Instructions))
	}
}

func TestRunQuery_CollectsRelated(t *testing.T) {
	compiledDir := testFixture(t)

	// Query for "login" — should match obj:abc123.1 which has adjacency
	// links to abc123.2 (next). abc123.2 may or may not be in results.
	result, err := query.RunQuery(compiledDir, "login flow", 1)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}

	// With topN=1, the top instruction result should be abc123.1.
	// Related should include adjacent chunks not in the top results.
	// We can't guarantee exact IDs due to BM25 scoring, but Related should
	// be populated if adjacency data exists.
	if len(result.Instructions) == 0 {
		t.Skip("no instruction results to check related chunks")
	}

	// At least verify Related doesn't contain the same IDs as Instructions.
	resultIDs := make(map[string]bool)
	for _, e := range result.Instructions {
		resultIDs[e.ChunkID] = true
	}
	for _, rel := range result.Related {
		if resultIDs[rel.ChunkID] {
			t.Errorf("related chunk %s also in instructions", rel.ChunkID)
		}
	}
}

func TestRunQuery_NoMatch(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunQuery(compiledDir, "xyznonexistentterm", 10)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}

	total := len(result.Instructions) + len(result.Reasoning) + len(result.System)
	if total != 0 {
		t.Errorf("expected 0 results, got %d", total)
	}
}

func TestRunQuery_MissingIndex(t *testing.T) {
	// Empty dir with no indexes.
	dir := t.TempDir()

	_, err := query.RunQuery(dir, "test", 10)
	if err == nil {
		t.Error("expected error for missing indexes")
	}
}

// ---------------------------------------------------------------------------
// CollectRelated
// ---------------------------------------------------------------------------

func TestCollectRelated_FindsAdjacent(t *testing.T) {
	compiledDir := testFixture(t)

	// Load manifest for the test.
	mfst, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	// obj:abc123.1 has adj next -> obj:abc123.2
	seen := map[string]bool{"obj:abc123.1": true}
	related := query.CollectRelated([]string{"obj:abc123.1"}, mfst, seen)

	found := false
	for _, rel := range related {
		if rel.ChunkID == "obj:abc123.2" {
			found = true
			if rel.Heading == "" {
				t.Error("related entry heading should not be empty")
			}
			if rel.Tokens <= 0 {
				t.Error("related entry tokens should be positive")
			}
		}
	}
	if !found {
		t.Error("expected obj:abc123.2 in related (adjacent next)")
	}
}

func TestCollectRelated_ExcludesSeen(t *testing.T) {
	compiledDir := testFixture(t)

	mfst, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	// Mark abc123.2 as already seen — it should not appear in related.
	seen := map[string]bool{"obj:abc123.1": true, "obj:abc123.2": true}
	related := query.CollectRelated([]string{"obj:abc123.1"}, mfst, seen)

	for _, rel := range related {
		if rel.ChunkID == "obj:abc123.2" {
			t.Error("obj:abc123.2 should be excluded (already seen)")
		}
	}
}

func TestCollectRelated_MaxFive(t *testing.T) {
	compiledDir := testFixture(t)

	mfst, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	// Even if there are many adjacent chunks, at most 5 should be returned.
	seen := make(map[string]bool)
	related := query.CollectRelated([]string{"obj:abc123.1", "obj:abc123.2", "obj:abc123.3"}, mfst, seen)

	if len(related) > 5 {
		t.Errorf("expected at most 5 related entries, got %d", len(related))
	}
}

func TestCollectRelated_NoAdjacency(t *testing.T) {
	compiledDir := testFixture(t)

	mfst, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	// Spec chunks don't have adjacency data.
	seen := map[string]bool{"spec:def456.1": true}
	related := query.CollectRelated([]string{"spec:def456.1"}, mfst, seen)

	if len(related) != 0 {
		t.Errorf("expected 0 related for spec chunk (no adjacency), got %d", len(related))
	}
}

func TestCollectRelated_UnknownChunkID(t *testing.T) {
	compiledDir := testFixture(t)

	mfst, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	seen := make(map[string]bool)
	related := query.CollectRelated([]string{"obj:nonexistent.1"}, mfst, seen)

	if len(related) != 0 {
		t.Errorf("expected 0 related for nonexistent chunk, got %d", len(related))
	}
}
