package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
)

// ---------------------------------------------------------------------------
// LoadManifest
// ---------------------------------------------------------------------------

func TestLoadManifest_RoundTrip(t *testing.T) {
	// Build a manifest, write it, then load it back.
	chunks := []chunk.Chunk{
		{ID: "obj:abc123.1", Type: chunk.ChunkObject, Source: "topics/auth.md", Heading: "Auth", Sequence: 1, TotalInFile: 2, Tokens: 400},
		{ID: "obj:abc123.2", Type: chunk.ChunkObject, Source: "topics/auth.md", Heading: "Auth > Tokens", Sequence: 2, TotalInFile: 2, Tokens: 350},
		{ID: "spec:def456.1", Type: chunk.ChunkSpec, Source: "topics/auth.spec.md", Heading: "Auth", Sequence: 1, TotalInFile: 1, Tokens: 300},
		{ID: "sys:ghi789.1", Type: chunk.ChunkSystem, Source: "system/topics/compile.md", Heading: "Compile", Sequence: 1, TotalInFile: 1, Tokens: 200},
	}

	original := manifest.BuildManifest(chunks, "cl100k_base", nil, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")
	if err := original.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	loaded, err := manifest.LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// Verify structure.
	if loaded.TotalChunks != 4 {
		t.Errorf("TotalChunks = %d, want 4", loaded.TotalChunks)
	}
	if loaded.TotalObjectChunks != 2 {
		t.Errorf("TotalObjectChunks = %d, want 2", loaded.TotalObjectChunks)
	}
	if loaded.TotalSpecChunks != 1 {
		t.Errorf("TotalSpecChunks = %d, want 1", loaded.TotalSpecChunks)
	}
	if loaded.TotalSystemChunks != 1 {
		t.Errorf("TotalSystemChunks = %d, want 1", loaded.TotalSystemChunks)
	}
	if loaded.TotalTokens != 1250 {
		t.Errorf("TotalTokens = %d, want 1250", loaded.TotalTokens)
	}

	// Verify entries.
	if e := loaded.LookupEntry("obj:abc123.1"); e == nil {
		t.Error("obj:abc123.1 not found")
	} else if e.Tokens != 400 {
		t.Errorf("obj:abc123.1 tokens = %d, want 400", e.Tokens)
	}

	if e := loaded.LookupEntry("spec:def456.1"); e == nil {
		t.Error("spec:def456.1 not found")
	}

	if e := loaded.LookupEntry("sys:ghi789.1"); e == nil {
		t.Error("sys:ghi789.1 not found")
	}
}

func TestLoadManifest_FileNotFound(t *testing.T) {
	_, err := manifest.LoadManifest("/nonexistent/manifest.yml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadManifest_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")
	if err := os.WriteFile(path, []byte(":\n\t\tinvalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := manifest.LoadManifest(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadManifest_EmptyMaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")
	if err := os.WriteFile(path, []byte("total_chunks: 0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := manifest.LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// Maps should be non-nil even when empty.
	if loaded.Objects == nil {
		t.Error("Objects map should be non-nil")
	}
	if loaded.Specs == nil {
		t.Error("Specs map should be non-nil")
	}
	if loaded.System == nil {
		t.Error("System map should be non-nil")
	}
}

// ---------------------------------------------------------------------------
// LoadHashes
// ---------------------------------------------------------------------------

func TestLoadHashes_RoundTrip(t *testing.T) {
	files := map[string]string{
		"docs/topics/auth/jwt.md": "sha256:abc123def456",
		"docs/topics/api/rest.md": "sha256:789012345678",
	}
	system := map[string]string{
		"taxonomy-generation": "sha256:aaa111bbb222",
		"bridge-summaries":    "sha256:ccc333ddd444",
	}

	original := manifest.BuildHashes(files, system)

	dir := t.TempDir()
	path := filepath.Join(dir, "hashes.yml")
	if err := original.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	loaded, err := manifest.LoadHashes(path)
	if err != nil {
		t.Fatalf("LoadHashes: %v", err)
	}

	if len(loaded.Files) != 2 {
		t.Errorf("Files count = %d, want 2", len(loaded.Files))
	}
	if len(loaded.System) != 2 {
		t.Errorf("System count = %d, want 2", len(loaded.System))
	}
	if loaded.Files["docs/topics/auth/jwt.md"] != "sha256:abc123def456" {
		t.Error("file hash mismatch for jwt.md")
	}
	if loaded.System["taxonomy-generation"] != "sha256:aaa111bbb222" {
		t.Error("system hash mismatch for taxonomy-generation")
	}
	if loaded.CompiledAt == "" {
		t.Error("expected compiled_at to be set")
	}
}

func TestLoadHashes_FileNotFound(t *testing.T) {
	_, err := manifest.LoadHashes("/nonexistent/hashes.yml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadHashes_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hashes.yml")
	if err := os.WriteFile(path, []byte(":\n\t\tinvalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := manifest.LoadHashes(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadHashes_EmptyMaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hashes.yml")
	if err := os.WriteFile(path, []byte("compiled_at: '2025-01-01'\n"), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := manifest.LoadHashes(path)
	if err != nil {
		t.Fatalf("LoadHashes: %v", err)
	}

	if loaded.Files == nil {
		t.Error("Files map should be non-nil")
	}
	if loaded.System == nil {
		t.Error("System map should be non-nil")
	}
}

// ---------------------------------------------------------------------------
// LoadMetadata
// ---------------------------------------------------------------------------

func TestLoadMetadata_RoundTrip(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:abc123.1", Type: chunk.ChunkObject, Source: "topics/auth.md", Heading: "Auth", Sequence: 1, TotalInFile: 1, Tokens: 400},
	}

	original := manifest.BuildMetadata(chunks, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.yml")
	if err := original.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	loaded, err := manifest.LoadMetadata(path)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	if len(loaded.Documents) != 1 {
		t.Fatalf("Documents count = %d, want 1", len(loaded.Documents))
	}

	doc, ok := loaded.Documents["topics/auth.md"]
	if !ok {
		t.Fatal("topics/auth.md not found in documents")
	}
	if doc.TotalTokens != 400 {
		t.Errorf("TotalTokens = %d, want 400", doc.TotalTokens)
	}
	if len(doc.Chunks) != 1 || doc.Chunks[0] != "obj:abc123.1" {
		t.Errorf("Chunks = %v, want [obj:abc123.1]", doc.Chunks)
	}
}

func TestLoadMetadata_FileNotFound(t *testing.T) {
	_, err := manifest.LoadMetadata("/nonexistent/metadata.yml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadManifest_CorruptedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")
	// Write a truncated/corrupted YAML file.
	if err := os.WriteFile(path, []byte("total_chunks: 10\nobjects:\n  obj:abc.1:\n    heading: "), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := manifest.LoadManifest(path)
	// A truncated YAML may parse without error (partial data) or with error.
	// Either way, the function should not panic.
	if err != nil {
		// Error path — acceptable for corrupted data.
		return
	}
	// If it parsed, verify it doesn't have the full expected data.
	if loaded.TotalChunks != 10 {
		t.Logf("partial parse: total_chunks = %d", loaded.TotalChunks)
	}
}

func TestLoadMetadata_EmptyDocuments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.yml")
	if err := os.WriteFile(path, []byte("compiled_at: '2025-01-01'\n"), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := manifest.LoadMetadata(path)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}

	if loaded.Documents == nil {
		t.Error("Documents map should be non-nil")
	}
}
