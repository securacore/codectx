package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- ShortHash ---

func TestShortHash(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"full hex hash", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", "a1b2c3d4e5f6"},
		{"with prefix", "sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", "a1b2c3d4e5f6"},
		{"short input", "abcdef", "abcdef"},
		{"exactly 12", "a1b2c3d4e5f6", "a1b2c3d4e5f6"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortHash(tt.in)
			if got != tt.want {
				t.Errorf("ShortHash(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- Hash computation ---

func TestQueryHash(t *testing.T) {
	h := QueryHash("jwt authentication")
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("QueryHash should have sha256: prefix, got %q", h)
	}
	if len(h) != 7+64 { // "sha256:" + 64 hex chars
		t.Errorf("QueryHash length = %d, want %d", len(h), 71)
	}

	// Same input should produce same hash.
	h2 := QueryHash("jwt authentication")
	if h != h2 {
		t.Error("QueryHash not deterministic")
	}

	// Different input should produce different hash.
	h3 := QueryHash("different query")
	if h == h3 {
		t.Error("different inputs produced same hash")
	}
}

func TestChunkSetHash(t *testing.T) {
	// Order should not matter.
	h1 := ChunkSetHash([]string{"obj:abc.01", "spec:def.02"})
	h2 := ChunkSetHash([]string{"spec:def.02", "obj:abc.01"})
	if h1 != h2 {
		t.Error("ChunkSetHash should be order-independent")
	}

	if !strings.HasPrefix(h1, "sha256:") {
		t.Errorf("ChunkSetHash should have sha256: prefix, got %q", h1)
	}
}

func TestContentHash(t *testing.T) {
	h := ContentHash([]byte("hello world"))
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("ContentHash should have sha256: prefix, got %q", h)
	}

	// Deterministic.
	h2 := ContentHash([]byte("hello world"))
	if h != h2 {
		t.Error("ContentHash not deterministic")
	}
}

// --- EnsureDir ---

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	histDir := filepath.Join(tmpDir, "history")

	// Create a minimal .gitignore target.
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureDir(histDir, tmpDir, "docs"); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	for _, sub := range []string{QueriesDir, ChunksDir, DocsDir} {
		path := filepath.Join(histDir, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("subdirectory %q not created: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", sub)
		}
	}
}

// --- WriteQueryEntry / ReadQueryHistory ---

func TestWriteAndReadQueryEntries(t *testing.T) {
	histDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(histDir, QueriesDir), 0755); err != nil {
		t.Fatal(err)
	}

	// Write 3 entries with distinct timestamps.
	for i, raw := range []string{"first query", "second query", "third query"} {
		entry := QueryEntry{
			Ts:          int64(1000000000000000000 + i*1000000000),
			QueryHash:   QueryHash(raw),
			Raw:         raw,
			Expanded:    raw + " expanded",
			ResultCount: (i + 1) * 10,
			CompileHash: "sha256:compilehash000000000000000000000000000000000000000000000000",
			Caller:      "test",
			SessionID:   "sess_test",
			Model:       "test-model",
		}
		if err := WriteQueryEntry(histDir, entry); err != nil {
			t.Fatalf("WriteQueryEntry %d: %v", i, err)
		}
	}

	// Read all.
	entries, err := ReadQueryHistory(histDir, 0)
	if err != nil {
		t.Fatalf("ReadQueryHistory: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Should be sorted newest first.
	if entries[0].Raw != "third query" {
		t.Errorf("first entry should be newest, got %q", entries[0].Raw)
	}
	if entries[2].Raw != "first query" {
		t.Errorf("last entry should be oldest, got %q", entries[2].Raw)
	}

	// Read with limit.
	limited, err := ReadQueryHistory(histDir, 2)
	if err != nil {
		t.Fatalf("ReadQueryHistory with limit: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(limited))
	}
}

// --- WriteChunksEntry / ReadChunksHistory ---

func TestWriteAndReadChunksEntries(t *testing.T) {
	histDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(histDir, ChunksDir), 0755); err != nil {
		t.Fatal(err)
	}

	chunks := []string{"obj:abc.01", "spec:def.02"}
	entry := ChunksEntry{
		Ts:           1700000000000000000,
		ChunkSetHash: ChunkSetHash(chunks),
		Chunks:       chunks,
		TokenCount:   1500,
		ContentHash:  "sha256:contenthash00000000000000000000000000000000000000000000000000",
		CompileHash:  "sha256:compilehash000000000000000000000000000000000000000000000000",
		DocFile:      "1700000000000000000.contenthash0.md",
		CacheHit:     false,
		Caller:       "claude",
		SessionID:    "sess_abc",
		Model:        "claude-sonnet",
	}

	if err := WriteChunksEntry(histDir, entry); err != nil {
		t.Fatalf("WriteChunksEntry: %v", err)
	}

	entries, err := ReadChunksHistory(histDir, 0)
	if err != nil {
		t.Fatalf("ReadChunksHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.TokenCount != 1500 {
		t.Errorf("TokenCount = %d, want 1500", got.TokenCount)
	}
	if got.Caller != "claude" {
		t.Errorf("Caller = %q, want %q", got.Caller, "claude")
	}
	if len(got.Chunks) != 2 {
		t.Errorf("Chunks length = %d, want 2", len(got.Chunks))
	}
}

// --- SaveDocument ---

func TestSaveDocument(t *testing.T) {
	histDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(histDir, DocsDir), 0755); err != nil {
		t.Fatal(err)
	}

	content := []byte("# Test Document\n\nSome content.")
	contentHash := ContentHash(content)
	ts := int64(1700000000000000000)

	filename, err := SaveDocument(histDir, content, contentHash, ts)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	// Verify filename format.
	if !strings.HasPrefix(filename, "1700000000000000000.") {
		t.Errorf("filename should start with timestamp, got %q", filename)
	}
	if !strings.HasSuffix(filename, ".md") {
		t.Errorf("filename should end with .md, got %q", filename)
	}

	// Verify content was written correctly.
	path := filepath.Join(histDir, DocsDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved document: %v", err)
	}
	if string(data) != string(content) {
		t.Error("saved content does not match")
	}
}

// --- ShowDocument ---

func TestShowDocument(t *testing.T) {
	histDir := t.TempDir()
	docsDir := filepath.Join(histDir, DocsDir)
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a doc with known hash in filename.
	content := "test document content"
	hash := "a1b2c3d4e5f6"
	filename := "1700000000000000000." + hash + ".md"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ShowDocument(histDir, "a1b2c3")
	if err != nil {
		t.Fatalf("ShowDocument: %v", err)
	}
	if got != content {
		t.Errorf("content mismatch: got %q", got)
	}
}

func TestShowDocument_NotFound(t *testing.T) {
	histDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(histDir, DocsDir), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := ShowDocument(histDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing document")
	}
}

func TestShowDocument_MultipleMatches(t *testing.T) {
	histDir := t.TempDir()
	docsDir := filepath.Join(histDir, DocsDir)
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	hash := "a1b2c3d4e5f6"
	// Older document.
	if err := os.WriteFile(filepath.Join(docsDir, "1700000000000000000."+hash+".md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	// Newer document.
	if err := os.WriteFile(filepath.Join(docsDir, "1700000001000000000."+hash+".md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ShowDocument(histDir, "a1b2c3")
	if err != nil {
		t.Fatalf("ShowDocument: %v", err)
	}
	if got != "new" {
		t.Errorf("should return newest document, got %q", got)
	}
}

// --- AnnotateDocument ---

func TestAnnotateDocument(t *testing.T) {
	histDir := t.TempDir()
	docsDir := filepath.Join(histDir, DocsDir)
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	docFile := "1700000000000000000.a1b2c3d4e5f6.md"
	if err := os.WriteFile(filepath.Join(docsDir, docFile), []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := AnnotateDocument(histDir, docFile, "test warning"); err != nil {
		t.Fatalf("AnnotateDocument: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(docsDir, docFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "codectx:warning test warning") {
		t.Error("annotation not found in document")
	}
}

// --- Clear ---

func TestClear(t *testing.T) {
	histDir := t.TempDir()
	for _, sub := range []string{QueriesDir, ChunksDir, DocsDir} {
		dir := filepath.Join(histDir, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "test.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := Clear(histDir); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	for _, sub := range []string{QueriesDir, ChunksDir, DocsDir} {
		entries, _ := os.ReadDir(filepath.Join(histDir, sub))
		if len(entries) != 0 {
			t.Errorf("%s should be empty after clear, has %d files", sub, len(entries))
		}
	}
}

// --- PruneDirectory ---

func TestPruneDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create 8 files with ascending timestamps.
	for i := 0; i < 8; i++ {
		ts := 1700000000000000000 + i*1000000000
		fname := filepath.Join(dir, fmt.Sprintf("%019d.a1b2c3d4e5f6.json", ts))
		if err := os.WriteFile(fname, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	err := PruneDirectory(dir, 3)
	if err != nil {
		t.Fatalf("PruneDirectory: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 3 {
		t.Errorf("expected 3 files after pruning, got %d", len(entries))
	}

	// Verify the 3 newest files survived.
	for _, e := range entries {
		name := e.Name()
		// The 3 newest should have timestamps >= 1700000005000000000.
		if name < "1700000005" {
			t.Errorf("old file %q should have been pruned", name)
		}
	}
}

func TestPruneDirectory_BelowThreshold(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%019d.hash.json", i)), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	err := PruneDirectory(dir, 5)
	if err != nil {
		t.Fatalf("PruneDirectory: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 3 {
		t.Errorf("should not prune when below threshold, got %d files", len(entries))
	}
}

// --- GenerateCacheLookup ---

func TestGenerateCacheLookup_Hit(t *testing.T) {
	histDir := t.TempDir()
	compiledDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(histDir, ChunksDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(histDir, DocsDir), 0755); err != nil {
		t.Fatal(err)
	}

	// Write a hashes.yml for compile hash.
	hashesContent := "test: data"
	if err := os.WriteFile(filepath.Join(compiledDir, "hashes.yml"), []byte(hashesContent), 0644); err != nil {
		t.Fatal(err)
	}

	chunkIDs := []string{"obj:abc.01", "spec:def.02"}
	chunkSetHash := ChunkSetHash(chunkIDs)
	compileHash := ContentHash([]byte(hashesContent))

	// Create a doc file.
	docFile := "1700000000000000000.a1b2c3d4e5f6.md"
	if err := os.WriteFile(filepath.Join(histDir, DocsDir, docFile), []byte("cached content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a matching chunks entry.
	entry := ChunksEntry{
		Ts:           1700000000000000000,
		ChunkSetHash: chunkSetHash,
		Chunks:       chunkIDs,
		TokenCount:   500,
		ContentHash:  "sha256:a1b2c3d4e5f6000000000000000000000000000000000000000000000000",
		CompileHash:  compileHash,
		DocFile:      docFile,
		CacheHit:     false,
		Caller:       "test",
		SessionID:    "unknown",
		Model:        "unknown",
	}
	if err := WriteChunksEntry(histDir, entry); err != nil {
		t.Fatalf("WriteChunksEntry: %v", err)
	}

	// Lookup should hit.
	path, hit := GenerateCacheLookup(histDir, chunkIDs, compiledDir)
	if !hit {
		t.Fatal("expected cache hit")
	}
	if path == "" {
		t.Fatal("expected non-empty path on cache hit")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "cached content" {
		t.Errorf("cached content mismatch: got %q", string(content))
	}
}

func TestGenerateCacheLookup_Miss_CompileChanged(t *testing.T) {
	histDir := t.TempDir()
	compiledDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(histDir, ChunksDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(histDir, DocsDir), 0755); err != nil {
		t.Fatal(err)
	}

	// Current hashes.yml.
	if err := os.WriteFile(filepath.Join(compiledDir, "hashes.yml"), []byte("new data"), 0644); err != nil {
		t.Fatal(err)
	}

	chunkIDs := []string{"obj:abc.01"}
	chunkSetHash := ChunkSetHash(chunkIDs)

	// Create entry with OLD compile hash.
	docFile := "1700000000000000000.a1b2c3d4e5f6.md"
	if err := os.WriteFile(filepath.Join(histDir, DocsDir, docFile), []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	entry := ChunksEntry{
		Ts:           1700000000000000000,
		ChunkSetHash: chunkSetHash,
		Chunks:       chunkIDs,
		TokenCount:   500,
		ContentHash:  "sha256:a1b2c3d4e5f6000000000000000000000000000000000000000000000000",
		CompileHash:  "sha256:oldcompilehash00000000000000000000000000000000000000000000",
		DocFile:      docFile,
		Caller:       "test",
		SessionID:    "unknown",
		Model:        "unknown",
	}
	if err := WriteChunksEntry(histDir, entry); err != nil {
		t.Fatalf("WriteChunksEntry: %v", err)
	}

	_, hit := GenerateCacheLookup(histDir, chunkIDs, compiledDir)
	if hit {
		t.Fatal("expected cache miss when compile hash changed")
	}
}

func TestGenerateCacheLookup_Miss_DocPruned(t *testing.T) {
	histDir := t.TempDir()
	compiledDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(histDir, ChunksDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(histDir, DocsDir), 0755); err != nil {
		t.Fatal(err)
	}

	hashesContent := "test data"
	if err := os.WriteFile(filepath.Join(compiledDir, "hashes.yml"), []byte(hashesContent), 0644); err != nil {
		t.Fatal(err)
	}

	chunkIDs := []string{"obj:abc.01"}
	chunkSetHash := ChunkSetHash(chunkIDs)
	compileHash := ContentHash([]byte(hashesContent))

	// Create chunks entry but NO doc file (simulates pruning).
	entry := ChunksEntry{
		Ts:           1700000000000000000,
		ChunkSetHash: chunkSetHash,
		Chunks:       chunkIDs,
		TokenCount:   500,
		ContentHash:  "sha256:a1b2c3d4e5f6000000000000000000000000000000000000000000000000",
		CompileHash:  compileHash,
		DocFile:      "1700000000000000000.a1b2c3d4e5f6.md",
		Caller:       "test",
		SessionID:    "unknown",
		Model:        "unknown",
	}
	if err := WriteChunksEntry(histDir, entry); err != nil {
		t.Fatalf("WriteChunksEntry: %v", err)
	}

	_, hit := GenerateCacheLookup(histDir, chunkIDs, compiledDir)
	if hit {
		t.Fatal("expected cache miss when doc file is pruned")
	}
}

func TestGenerateCacheLookup_Miss_NoEntries(t *testing.T) {
	histDir := t.TempDir()
	compiledDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(histDir, ChunksDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(compiledDir, "hashes.yml"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	_, hit := GenerateCacheLookup(histDir, []string{"obj:abc.01"}, compiledDir)
	if hit {
		t.Fatal("expected cache miss with no entries")
	}
}

// --- LogGenerate ---

func TestLogGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	histDir := filepath.Join(tmpDir, "history")
	projectDir := tmpDir

	// Create .gitignore.
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	content := []byte("# Generated Document")
	contentHash := ContentHash(content)
	compileHash := "sha256:compilehash000000000000000000000000000000000000000000000000"
	caller := CallerContext{Caller: "test", SessionID: "sess_123", Model: "test-model"}

	docFile, err := LogGenerate(histDir, projectDir, "docs", content,
		[]string{"obj:abc.01"}, 500, contentHash, compileHash, false, caller)
	if err != nil {
		t.Fatalf("LogGenerate: %v", err)
	}

	if docFile == "" {
		t.Fatal("expected non-empty docFile")
	}

	// Verify doc was written.
	docPath := filepath.Join(histDir, DocsDir, docFile)
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("doc not written: %v", err)
	}
	if string(data) != string(content) {
		t.Error("doc content mismatch")
	}

	// Verify chunks entry was written.
	entries, err := ReadChunksHistory(histDir, 0)
	if err != nil {
		t.Fatalf("ReadChunksHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 chunks entry, got %d", len(entries))
	}
	if entries[0].Caller != "test" {
		t.Errorf("entry.Caller = %q, want %q", entries[0].Caller, "test")
	}
}

// --- LogQuery ---

func TestLogQuery(t *testing.T) {
	tmpDir := t.TempDir()
	histDir := filepath.Join(tmpDir, "history")
	projectDir := tmpDir

	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	compileHash := "sha256:compilehash000000000000000000000000000000000000000000000000"
	caller := CallerContext{Caller: "claude", SessionID: "sess_abc", Model: "claude-sonnet"}

	if err := LogQuery(histDir, projectDir, "docs", "jwt auth", "jwt auth token", 15, compileHash, caller); err != nil {
		t.Fatalf("LogQuery: %v", err)
	}

	entries, err := ReadQueryHistory(histDir, 0)
	if err != nil {
		t.Fatalf("ReadQueryHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 query entry, got %d", len(entries))
	}
	if entries[0].Raw != "jwt auth" {
		t.Errorf("entry.Raw = %q, want %q", entries[0].Raw, "jwt auth")
	}
	if entries[0].Caller != "claude" {
		t.Errorf("entry.Caller = %q, want %q", entries[0].Caller, "claude")
	}
}

// --- JSON format verification ---

func TestEntryJSON_QueryRoundTrip(t *testing.T) {
	entry := QueryEntry{
		Ts:          1700000000000000000,
		QueryHash:   "sha256:abc123",
		Raw:         "test query",
		Expanded:    "test query expanded",
		ResultCount: 5,
		CompileHash: "sha256:xyz789",
		Caller:      "test",
		SessionID:   "sess_1",
		Model:       "model-1",
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var decoded QueryEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Ts != entry.Ts || decoded.Raw != entry.Raw || decoded.Caller != entry.Caller {
		t.Error("JSON round-trip failed")
	}
}

func TestEntryJSON_ChunksRoundTrip(t *testing.T) {
	entry := ChunksEntry{
		Ts:           1700000000000000000,
		ChunkSetHash: "sha256:abc123",
		Chunks:       []string{"obj:abc.01", "spec:def.02"},
		TokenCount:   1500,
		ContentHash:  "sha256:content123",
		CompileHash:  "sha256:compile789",
		DocFile:      "1700000000000000000.abc123456789.md",
		CacheHit:     true,
		Caller:       "claude",
		SessionID:    "sess_abc",
		Model:        "claude-sonnet",
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var decoded ChunksEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.CacheHit != true {
		t.Error("CacheHit not preserved through JSON round-trip")
	}
	if len(decoded.Chunks) != 2 {
		t.Errorf("Chunks length = %d, want 2", len(decoded.Chunks))
	}
}

// --- dirSize ---

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world!"), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := dirSize(dir)
	if err != nil {
		t.Fatal(err)
	}
	if size != 11 { // 5 + 6
		t.Errorf("dirSize = %d, want 11", size)
	}
}

// --- ReadQueryHistory / ReadChunksHistory empty dir ---

func TestReadQueryHistory_EmptyDir(t *testing.T) {
	histDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(histDir, QueriesDir), 0755); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadQueryHistory(histDir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for empty dir, got %d", len(entries))
	}
}

func TestReadChunksHistory_EmptyDir(t *testing.T) {
	histDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(histDir, ChunksDir), 0755); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadChunksHistory(histDir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for empty dir, got %d", len(entries))
	}
}

// --- Filename format verification ---

func TestFilenameFormat_TimestampFirst(t *testing.T) {
	histDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(histDir, QueriesDir), 0755); err != nil {
		t.Fatal(err)
	}

	entry := QueryEntry{
		Ts:          1700000000000000000,
		QueryHash:   "sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6abcd",
		Raw:         "test",
		ResultCount: 1,
		CompileHash: "sha256:xyz",
		Caller:      "test",
		SessionID:   "unknown",
		Model:       "unknown",
	}

	if err := WriteQueryEntry(histDir, entry); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(filepath.Join(histDir, QueriesDir))
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	name := entries[0].Name()
	// Should start with timestamp.
	if !strings.HasPrefix(name, "1700000000000000000.") {
		t.Errorf("filename should start with timestamp, got %q", name)
	}
	// Should end with .json.
	if !strings.HasSuffix(name, ".json") {
		t.Errorf("filename should end with .json, got %q", name)
	}
	// Should contain hash.
	if !strings.Contains(name, "a1b2c3d4e5f6") {
		t.Errorf("filename should contain short hash, got %q", name)
	}
}

// --- CompileHash ---

func TestCompileHash(t *testing.T) {
	dir := t.TempDir()
	content := "compiled_at: 2025-01-01\nfiles:\n  docs/test.md: sha256:abc\n"
	if err := os.WriteFile(filepath.Join(dir, "hashes.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := CompileHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("CompileHash should have prefix, got %q", hash)
	}

	// Same content = same hash.
	hash2, _ := CompileHash(dir)
	if hash != hash2 {
		t.Error("CompileHash not deterministic")
	}
}

func TestCompileHash_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := CompileHash(dir)
	if err == nil {
		t.Fatal("expected error for missing hashes.yml")
	}
}
