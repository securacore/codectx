package chunk_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
)

// ---------------------------------------------------------------------------
// ChunkFilePath
// ---------------------------------------------------------------------------

func TestChunkFilePath_ObjectChunk(t *testing.T) {
	got, err := chunk.ChunkFilePath("/compiled", "obj:a1b2c3d4.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("/compiled", "objects", "a1b2c3d4.3.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestChunkFilePath_SpecChunk(t *testing.T) {
	got, err := chunk.ChunkFilePath("/compiled", "spec:def456.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("/compiled", "specs", "def456.1.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestChunkFilePath_SystemChunk(t *testing.T) {
	got, err := chunk.ChunkFilePath("/compiled", "sys:ghi789.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("/compiled", "system", "ghi789.1.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestChunkFilePath_InvalidFormat(t *testing.T) {
	_, err := chunk.ChunkFilePath("/compiled", "nocolon")
	if err == nil {
		t.Error("expected error for invalid chunk ID")
	}
}

func TestChunkFilePath_UnknownPrefix(t *testing.T) {
	_, err := chunk.ChunkFilePath("/compiled", "bad:abc.1")
	if err == nil {
		t.Error("expected error for unknown prefix")
	}
}

// ---------------------------------------------------------------------------
// ReadChunkContent
// ---------------------------------------------------------------------------

func TestReadChunkContent_StripsMeta(t *testing.T) {
	dir := t.TempDir()

	// Create the directory structure.
	objDir := filepath.Join(dir, "objects")
	if err := os.MkdirAll(objDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := "<!-- codectx:meta\nid: obj:abc123.1\ntype: object\n-->\n\nThis is the actual content.\nWith multiple lines.\n"
	if err := os.WriteFile(filepath.Join(objDir, "abc123.1.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := chunk.ReadChunkContent(dir, "obj:abc123.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "This is the actual content.") {
		t.Errorf("expected content body, got %q", got)
	}
	if strings.Contains(got, "codectx:meta") {
		t.Error("meta header should be stripped")
	}
}

func TestReadChunkContent_NoMetaHeader(t *testing.T) {
	dir := t.TempDir()
	objDir := filepath.Join(dir, "objects")
	if err := os.MkdirAll(objDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := "Just plain content without a meta header.\n"
	if err := os.WriteFile(filepath.Join(objDir, "abc123.1.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := chunk.ReadChunkContent(dir, "obj:abc123.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != content {
		t.Errorf("expected raw content, got %q", got)
	}
}

func TestReadChunkContent_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := chunk.ReadChunkContent(dir, "obj:nonexistent.1")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// ---------------------------------------------------------------------------
// CompiledOutputDirs
// ---------------------------------------------------------------------------

func TestCompiledOutputDirs_ReturnsAllDirs(t *testing.T) {
	dirs := chunk.CompiledOutputDirs()

	// Should have 6 dirs: 3 chunk dirs + 3 BM25 dirs.
	if len(dirs) != 6 {
		t.Fatalf("expected 6 dirs, got %d: %v", len(dirs), dirs)
	}

	// Check chunk dirs are present.
	expected := map[string]bool{
		"objects": false,
		"specs":   false,
		"system":  false,
	}
	for _, d := range dirs {
		if _, ok := expected[d]; ok {
			expected[d] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected %q in output dirs", name)
		}
	}

	// Check BM25 dirs contain "bm25/".
	bm25Count := 0
	for _, d := range dirs {
		if strings.Contains(d, "bm25") {
			bm25Count++
		}
	}
	if bm25Count != 3 {
		t.Errorf("expected 3 BM25 dirs, got %d", bm25Count)
	}
}
