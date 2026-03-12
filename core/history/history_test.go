package history

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestEscapeTSV(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello world"},
		{"hello\tworld", "hello world"},
		{"hello\nworld", "hello world"},
		{"hello\r\nworld", "hello world"},
		{"tabs\tand\nnewlines\there", "tabs and newlines here"},
	}
	for _, tt := range tests {
		got := escapeTSV(tt.input)
		if got != tt.want {
			t.Errorf("escapeTSV(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		name string
		want int64
	}{
		{"a1b2c3d4e5f6.1741532400123456789.md", 1741532400123456789},
		{"abcdef012345.9223372036854775807.md", 9223372036854775807},
		{"notsplit.md", 0},
		{"", 0},
		{"a1b2c3d4e5f6.notanumber.md", 0},
	}
	for _, tt := range tests {
		got := extractTimestamp(tt.name)
		if got != tt.want {
			t.Errorf("extractTimestamp(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestAppendQuery(t *testing.T) {
	dir := t.TempDir()

	if err := AppendQuery(dir, "jwt auth", "jwt auth authentication", 5); err != nil {
		t.Fatalf("AppendQuery: %v", err)
	}
	if err := AppendQuery(dir, "api endpoints", "api endpoints routes", 3); err != nil {
		t.Fatalf("AppendQuery second: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, QueryHistoryFile))
	if err != nil {
		t.Fatalf("reading query.history: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Verify first line has correct fields.
	fields := strings.SplitN(lines[0], "\t", 4)
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}
	if fields[1] != "jwt auth" {
		t.Errorf("raw query = %q, want %q", fields[1], "jwt auth")
	}
	if fields[2] != "jwt auth authentication" {
		t.Errorf("expanded query = %q, want %q", fields[2], "jwt auth authentication")
	}
	if fields[3] != "5" {
		t.Errorf("result count = %q, want %q", fields[3], "5")
	}
}

func TestAppendChunks(t *testing.T) {
	dir := t.TempDir()

	ids := []string{"obj:abc123.01", "spec:def456.02"}
	if err := AppendChunks(dir, ids, 1500, "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"); err != nil {
		t.Fatalf("AppendChunks: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ChunksHistoryFile))
	if err != nil {
		t.Fatalf("reading chunks.history: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	fields := strings.SplitN(lines[0], "\t", 4)
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}
	if fields[1] != "obj:abc123.01,spec:def456.02" {
		t.Errorf("chunk IDs = %q, want %q", fields[1], "obj:abc123.01,spec:def456.02")
	}
	if fields[2] != "1500" {
		t.Errorf("token count = %q, want %q", fields[2], "1500")
	}
}

func TestSaveDocument(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("creating docs dir: %v", err)
	}

	content := "# Test Document\n\nThis is a test."

	shortHash, contentHash, filePath, err := SaveDocument(dir, content)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	// Short hash should be 12 hex chars.
	if len(shortHash) != ShortHashLen {
		t.Errorf("short hash length = %d, want %d", len(shortHash), ShortHashLen)
	}

	// Content hash should be 64 hex chars (full SHA-256).
	if len(contentHash) != 64 {
		t.Errorf("content hash length = %d, want 64", len(contentHash))
	}

	// Short hash should be prefix of content hash.
	if !strings.HasPrefix(contentHash, shortHash) {
		t.Errorf("content hash %q does not start with short hash %q", contentHash, shortHash)
	}

	// File should exist and contain the content.
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading saved document: %v", err)
	}
	if string(got) != content {
		t.Errorf("saved content mismatch:\ngot:  %q\nwant: %q", string(got), content)
	}

	// Filename should follow the expected pattern.
	base := filepath.Base(filePath)
	if !strings.HasPrefix(base, shortHash+".") || !strings.HasSuffix(base, ".md") {
		t.Errorf("unexpected filename format: %s", base)
	}
}

func TestSaveDocument_DeterministicHash(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("creating docs dir: %v", err)
	}

	content := "same content"

	hash1, _, _, err := SaveDocument(dir, content)
	if err != nil {
		t.Fatalf("SaveDocument first: %v", err)
	}

	hash2, _, _, err := SaveDocument(dir, content)
	if err != nil {
		t.Fatalf("SaveDocument second: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("same content produced different hashes: %q vs %q", hash1, hash2)
	}
}

func TestAnnotateDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Doc\n"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	if err := AnnotateDocument(path, "chunks.history write failed: permission denied"); err != nil {
		t.Fatalf("AnnotateDocument: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading annotated file: %v", err)
	}

	got := string(data)
	if !strings.Contains(got, "<!-- codectx:warning chunks.history write failed: permission denied -->") {
		t.Errorf("annotation not found in file content:\n%s", got)
	}
	if !strings.HasPrefix(got, "# Doc\n") {
		t.Error("original content was modified")
	}
}

func TestReadQueryHistory(t *testing.T) {
	dir := t.TempDir()

	// Write 5 query entries.
	for i := range 5 {
		line := strings.Join([]string{
			"100000000000000000" + string(rune('0'+i)),
			"query " + string(rune('a'+i)),
			"expanded " + string(rune('a'+i)),
			"3",
		}, "\t") + "\n"
		if err := appendToFile(filepath.Join(dir, QueryHistoryFile), line); err != nil {
			t.Fatalf("writing entry %d: %v", i, err)
		}
	}

	entries, err := ReadQueryHistory(dir, 3)
	if err != nil {
		t.Fatalf("ReadQueryHistory: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Should be the last 3 entries.
	if entries[0].RawQuery != "query c" {
		t.Errorf("first entry query = %q, want %q", entries[0].RawQuery, "query c")
	}
	if entries[2].RawQuery != "query e" {
		t.Errorf("last entry query = %q, want %q", entries[2].RawQuery, "query e")
	}
}

func TestReadQueryHistory_Empty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ReadQueryHistory(dir, 10)
	if err != nil {
		t.Fatalf("ReadQueryHistory on empty dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadChunksHistory(t *testing.T) {
	dir := t.TempDir()

	line := "1741532400123456789\tobj:abc123.01,spec:def456.02\t1500\tabcdef0123456789\n"
	if err := appendToFile(filepath.Join(dir, ChunksHistoryFile), line); err != nil {
		t.Fatalf("writing entry: %v", err)
	}

	entries, err := ReadChunksHistory(dir, 0)
	if err != nil {
		t.Fatalf("ReadChunksHistory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Timestamp != 1741532400123456789 {
		t.Errorf("timestamp = %d, want 1741532400123456789", e.Timestamp)
	}
	if len(e.ChunkIDs) != 2 || e.ChunkIDs[0] != "obj:abc123.01" {
		t.Errorf("chunk IDs = %v, want [obj:abc123.01 spec:def456.02]", e.ChunkIDs)
	}
	if e.TokenCount != 1500 {
		t.Errorf("token count = %d, want 1500", e.TokenCount)
	}
	if e.ContentHash != "abcdef0123456789" {
		t.Errorf("content hash = %q, want %q", e.ContentHash, "abcdef0123456789")
	}
}

func TestShowDocument(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("creating docs dir: %v", err)
	}

	// Write a document with known hash prefix.
	content := "# Test"
	if err := os.WriteFile(filepath.Join(docsDir, "a1b2c3d4e5f6.1741532400123456789.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing test doc: %v", err)
	}

	got, err := ShowDocument(dir, "a1b2c3d4e5f6")
	if err != nil {
		t.Fatalf("ShowDocument: %v", err)
	}
	if got != content {
		t.Errorf("ShowDocument content = %q, want %q", got, content)
	}
}

func TestShowDocument_PartialHash(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("creating docs dir: %v", err)
	}

	content := "# Partial Match"
	if err := os.WriteFile(filepath.Join(docsDir, "a1b2c3d4e5f6.1741532400123456789.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing test doc: %v", err)
	}

	got, err := ShowDocument(dir, "a1b2c3")
	if err != nil {
		t.Fatalf("ShowDocument partial: %v", err)
	}
	if got != content {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestShowDocument_NotFound(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("creating docs dir: %v", err)
	}

	_, err := ShowDocument(dir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent hash, got nil")
	}
}

func TestShowDocument_MultipleMatches_PicksNewest(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("creating docs dir: %v", err)
	}

	// Two documents with same hash prefix but different timestamps.
	if err := os.WriteFile(filepath.Join(docsDir, "a1b2c3d4e5f6.1000000000000000000.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "a1b2c3d4e5f6.2000000000000000000.md"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ShowDocument(dir, "a1b2c3d4e5f6")
	if err != nil {
		t.Fatalf("ShowDocument: %v", err)
	}
	if got != "new" {
		t.Errorf("expected newest document, got %q", got)
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write history files and a doc.
	if err := appendToFile(filepath.Join(dir, QueryHistoryFile), "line1\n"); err != nil {
		t.Fatal(err)
	}
	if err := appendToFile(filepath.Join(dir, ChunksHistoryFile), "line1\n"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "test.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Clear(dir); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	// History files should be gone.
	if _, err := os.Stat(filepath.Join(dir, QueryHistoryFile)); !os.IsNotExist(err) {
		t.Error("query.history should not exist after clear")
	}
	if _, err := os.Stat(filepath.Join(dir, ChunksHistoryFile)); !os.IsNotExist(err) {
		t.Error("chunks.history should not exist after clear")
	}

	// Docs should be empty.
	entries, _ := os.ReadDir(docsDir)
	if len(entries) != 0 {
		t.Errorf("docs/ should be empty after clear, has %d entries", len(entries))
	}
}

func TestClear_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	// Clear on empty dir should not error.
	if err := Clear(dir); err != nil {
		t.Fatalf("Clear on empty dir: %v", err)
	}
}

func TestTruncateToLastN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	// Write 10 lines.
	var content string
	for i := range 10 {
		content += "line" + string(rune('0'+i)) + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := truncateToLastN(path, 3); err != nil {
		t.Fatalf("truncateToLastN: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line7" {
		t.Errorf("first kept line = %q, want %q", lines[0], "line7")
	}
}

func TestTruncateToLastN_FewerThanN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := os.WriteFile(path, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := truncateToLastN(path, 5); err != nil {
		t.Fatalf("truncateToLastN: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Should be unchanged.
	if string(data) != "line1\nline2\n" {
		t.Errorf("content changed unexpectedly: %q", string(data))
	}
}

func TestTruncateToLastN_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.txt")

	// Should be a no-op.
	if err := truncateToLastN(path, 5); err != nil {
		t.Fatalf("truncateToLastN on nonexistent file: %v", err)
	}
}

func TestPruneDocuments(t *testing.T) {
	dir := t.TempDir()

	// Create 8 documents with different timestamps.
	for i := range 8 {
		name := "a1b2c3d4e5f6." + strings.Repeat(string(rune('0'+i)), 19) + ".md"
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := pruneDocuments(dir, 3); err != nil {
		t.Fatalf("pruneDocuments: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 files after pruning, got %d", len(entries))
	}
}

func TestPruneDocuments_NonexistentDir(t *testing.T) {
	if err := pruneDocuments("/nonexistent/path", 5); err != nil {
		t.Fatalf("pruneDocuments on nonexistent dir: %v", err)
	}
}

func TestCheckAndPrune_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a small file — well under 100MB.
	if err := os.WriteFile(filepath.Join(docsDir, "small.md"), []byte("small"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CheckAndPrune(dir); err != nil {
		t.Fatalf("CheckAndPrune: %v", err)
	}

	// File should still exist.
	if _, err := os.Stat(filepath.Join(docsDir, "small.md")); os.IsNotExist(err) {
		t.Error("file was pruned despite being under limit")
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()

	data := []byte("hello world") // 11 bytes
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	size, err := dirSize(dir)
	if err != nil {
		t.Fatalf("dirSize: %v", err)
	}

	if size != 22 {
		t.Errorf("dirSize = %d, want 22", size)
	}
}

func TestParseQueryLine(t *testing.T) {
	line := "1741532400123456789\tjwt auth\tjwt auth authentication\t5"
	entry, err := parseQueryLine(line)
	if err != nil {
		t.Fatalf("parseQueryLine: %v", err)
	}
	if entry.Timestamp != 1741532400123456789 {
		t.Errorf("timestamp = %d", entry.Timestamp)
	}
	if entry.RawQuery != "jwt auth" {
		t.Errorf("raw query = %q", entry.RawQuery)
	}
	if entry.ExpandedQuery != "jwt auth authentication" {
		t.Errorf("expanded query = %q", entry.ExpandedQuery)
	}
	if entry.ResultCount != 5 {
		t.Errorf("result count = %d", entry.ResultCount)
	}
}

func TestParseQueryLine_BadFormat(t *testing.T) {
	_, err := parseQueryLine("only\ttwo\tfields")
	if err == nil {
		t.Error("expected error for malformed line")
	}
}

func TestParseChunksLine(t *testing.T) {
	line := "1741532400123456789\tobj:abc.01,spec:def.02\t1500\tabcdef0123456789"
	entry, err := parseChunksLine(line)
	if err != nil {
		t.Fatalf("parseChunksLine: %v", err)
	}
	if entry.Timestamp != 1741532400123456789 {
		t.Errorf("timestamp = %d", entry.Timestamp)
	}
	if len(entry.ChunkIDs) != 2 {
		t.Errorf("chunk IDs count = %d, want 2", len(entry.ChunkIDs))
	}
	if entry.TokenCount != 1500 {
		t.Errorf("token count = %d", entry.TokenCount)
	}
	if entry.ContentHash != "abcdef0123456789" {
		t.Errorf("content hash = %q", entry.ContentHash)
	}
}

func TestParseChunksLine_BadFormat(t *testing.T) {
	_, err := parseChunksLine("bad")
	if err == nil {
		t.Error("expected error for malformed line")
	}
}

func TestReadLastN_AllEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, err := readLastN(path, 0)
	if err != nil {
		t.Fatalf("readLastN: %v", err)
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestReadLastN_SkipsEmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("a\n\nb\n\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, err := readLastN(path, 0)
	if err != nil {
		t.Fatalf("readLastN: %v", err)
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (skipping empty), got %d: %v", len(lines), lines)
	}
}

// ---------------------------------------------------------------------------
// ShortHash
// ---------------------------------------------------------------------------

func TestShortHash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "abcdef123456"},
		{"abcdef123456", "abcdef123456"}, // exactly ShortHashLen
		{"short", "short"},               // shorter than ShortHashLen
		{"", ""},                         // empty
	}
	for _, tt := range tests {
		got := ShortHash(tt.input)
		if got != tt.want {
			t.Errorf("ShortHash(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// EnsureDir
// ---------------------------------------------------------------------------

func TestEnsureDir(t *testing.T) {
	projectDir := t.TempDir()
	histDir := filepath.Join(projectDir, "docs", ".codectx", "history")

	if err := EnsureDir(histDir, projectDir, "docs"); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	// history/ should exist.
	info, err := os.Stat(histDir)
	if err != nil {
		t.Fatalf("history dir missing: %v", err)
	}
	if !info.IsDir() {
		t.Error("history path is not a directory")
	}

	// history/docs/ should exist.
	docsPath := filepath.Join(histDir, DocsDir)
	info, err = os.Stat(docsPath)
	if err != nil {
		t.Fatalf("history/docs dir missing: %v", err)
	}
	if !info.IsDir() {
		t.Error("history/docs path is not a directory")
	}
}

func TestEnsureDir_Idempotent(t *testing.T) {
	projectDir := t.TempDir()
	histDir := filepath.Join(projectDir, "docs", ".codectx", "history")

	// Call twice — should not error.
	if err := EnsureDir(histDir, projectDir, "docs"); err != nil {
		t.Fatalf("EnsureDir first: %v", err)
	}
	if err := EnsureDir(histDir, projectDir, "docs"); err != nil {
		t.Fatalf("EnsureDir second: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AppendQuery edge cases
// ---------------------------------------------------------------------------

func TestAppendQuery_EmptyFields(t *testing.T) {
	dir := t.TempDir()

	if err := AppendQuery(dir, "", "", 0); err != nil {
		t.Fatalf("AppendQuery with empty fields: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, QueryHistoryFile))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	fields := strings.SplitN(lines[0], "\t", 4)
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}
	if fields[1] != "" {
		t.Errorf("raw query = %q, want empty", fields[1])
	}
	if fields[3] != "0" {
		t.Errorf("result count = %q, want \"0\"", fields[3])
	}
}

// ---------------------------------------------------------------------------
// AppendChunks edge cases
// ---------------------------------------------------------------------------

func TestAppendChunks_EmptyIDs(t *testing.T) {
	dir := t.TempDir()

	if err := AppendChunks(dir, nil, 0, ""); err != nil {
		t.Fatalf("AppendChunks with nil IDs: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ChunksHistoryFile))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	fields := strings.SplitN(lines[0], "\t", 4)
	if fields[1] != "" {
		t.Errorf("chunk IDs = %q, want empty", fields[1])
	}
}

// ---------------------------------------------------------------------------
// SaveDocument — verify hash correctness
// ---------------------------------------------------------------------------

func TestSaveDocument_HashMatchesContent(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "verify hash matches"
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	_, contentHash, _, err := SaveDocument(dir, content)
	if err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	if contentHash != expectedHash {
		t.Errorf("content hash = %q, want %q", contentHash, expectedHash)
	}
}

// ---------------------------------------------------------------------------
// CheckAndPrune — over limit (actual pruning)
// ---------------------------------------------------------------------------

func TestCheckAndPrune_OverLimit(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write 10 query history lines.
	for i := range 10 {
		line := strconv.FormatInt(int64(1000000000000000000+i), 10) +
			"\tquery" + strconv.Itoa(i) + "\texpanded\t1\n"
		if err := appendToFile(filepath.Join(dir, QueryHistoryFile), line); err != nil {
			t.Fatal(err)
		}
	}

	// Write 10 chunks history lines.
	for i := range 10 {
		line := strconv.FormatInt(int64(1000000000000000000+i), 10) +
			"\tobj:abc.0" + strconv.Itoa(i) + "\t100\thash" + strconv.Itoa(i) + "\n"
		if err := appendToFile(filepath.Join(dir, ChunksHistoryFile), line); err != nil {
			t.Fatal(err)
		}
	}

	// Write 10 documents. Make them large enough so total > MaxSize.
	// Each file ~11MB to exceed 100MB total.
	bigContent := strings.Repeat("x", 11*1024*1024)
	for i := range 10 {
		ts := strconv.FormatInt(int64(1000000000000000000+i), 10)
		name := "a1b2c3d4e5f6." + ts + ".md"
		if err := os.WriteFile(filepath.Join(docsDir, name), []byte(bigContent), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := CheckAndPrune(dir); err != nil {
		t.Fatalf("CheckAndPrune: %v", err)
	}

	// After pruning, both history files should have at most PruneKeep lines.
	queryData, err := os.ReadFile(filepath.Join(dir, QueryHistoryFile))
	if err != nil {
		t.Fatalf("reading query.history: %v", err)
	}
	queryLines := strings.Split(strings.TrimRight(string(queryData), "\n"), "\n")
	if len(queryLines) != PruneKeep {
		t.Errorf("query.history: expected %d lines, got %d", PruneKeep, len(queryLines))
	}

	chunksData, err := os.ReadFile(filepath.Join(dir, ChunksHistoryFile))
	if err != nil {
		t.Fatalf("reading chunks.history: %v", err)
	}
	chunksLines := strings.Split(strings.TrimRight(string(chunksData), "\n"), "\n")
	if len(chunksLines) != PruneKeep {
		t.Errorf("chunks.history: expected %d lines, got %d", PruneKeep, len(chunksLines))
	}

	// After pruning, docs/ should have at most PruneKeep documents.
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != PruneKeep {
		t.Errorf("docs/: expected %d files, got %d", PruneKeep, len(entries))
	}

	// The kept lines should be the last PruneKeep (highest timestamps).
	if !strings.Contains(queryLines[0], "query5") {
		t.Errorf("first kept query line should be query5, got: %s", queryLines[0])
	}
}

// ---------------------------------------------------------------------------
// LogGenerate
// ---------------------------------------------------------------------------

func TestLogGenerate(t *testing.T) {
	projectDir := t.TempDir()
	histDir := filepath.Join(projectDir, "docs", ".codectx", "history")

	docContent := "# Generated document"
	chunkIDs := []string{"obj:abc.01", "spec:def.02"}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(docContent)))

	docPath, err := LogGenerate(histDir, projectDir, "docs", docContent, chunkIDs, 500, hash)
	if err != nil {
		t.Fatalf("LogGenerate: %v", err)
	}

	// Document should be saved.
	if docPath == "" {
		t.Fatal("expected non-empty docPath")
	}
	saved, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading saved doc: %v", err)
	}
	if string(saved) != docContent {
		t.Errorf("saved content mismatch")
	}

	// chunks.history should have an entry.
	data, err := os.ReadFile(filepath.Join(histDir, ChunksHistoryFile))
	if err != nil {
		t.Fatalf("reading chunks.history: %v", err)
	}
	if !strings.Contains(string(data), "obj:abc.01,spec:def.02") {
		t.Error("chunks.history missing chunk IDs")
	}
}

// ---------------------------------------------------------------------------
// LogQuery
// ---------------------------------------------------------------------------

func TestLogQuery(t *testing.T) {
	projectDir := t.TempDir()
	histDir := filepath.Join(projectDir, "docs", ".codectx", "history")

	if err := LogQuery(histDir, projectDir, "docs", "jwt auth", "jwt authentication", 5); err != nil {
		t.Fatalf("LogQuery: %v", err)
	}

	// query.history should have an entry.
	data, err := os.ReadFile(filepath.Join(histDir, QueryHistoryFile))
	if err != nil {
		t.Fatalf("reading query.history: %v", err)
	}
	if !strings.Contains(string(data), "jwt auth") {
		t.Error("query.history missing query")
	}
}
