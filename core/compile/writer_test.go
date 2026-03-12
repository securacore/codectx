package compile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/project"
)

func TestPrepareOutputDirs_CreatesDirectories(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}

	if err := compile.PrepareOutputDirs(compiledDir); err != nil {
		t.Fatalf("PrepareOutputDirs: %v", err)
	}

	// Verify all chunk output directories exist.
	for _, ct := range []chunk.ChunkType{chunk.ChunkObject, chunk.ChunkSpec, chunk.ChunkSystem} {
		dir := filepath.Join(compiledDir, chunk.OutputDir(ct))
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("expected directory %s to exist", dir)
		}
	}

	// Verify BM25 directories exist.
	for _, ct := range []chunk.ChunkType{chunk.ChunkObject, chunk.ChunkSpec, chunk.ChunkSystem} {
		dir := filepath.Join(compiledDir, project.BM25Dir, chunk.OutputDir(ct))
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("expected BM25 directory %s to exist", dir)
		}
	}
}

func TestPrepareOutputDirs_CleansExisting(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	objDir := filepath.Join(compiledDir, chunk.OutputDir(chunk.ChunkObject))

	// Create directory and add a file.
	if err := os.MkdirAll(objDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}
	staleFile := filepath.Join(objDir, "old-chunk.md")
	if err := os.WriteFile(staleFile, []byte("stale"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	if err := compile.PrepareOutputDirs(compiledDir); err != nil {
		t.Fatalf("PrepareOutputDirs: %v", err)
	}

	// Stale file should be gone.
	if _, err := os.Stat(staleFile); err == nil {
		t.Error("expected stale file to be removed")
	}

	// Directory should still exist (recreated).
	if info, err := os.Stat(objDir); err != nil || !info.IsDir() {
		t.Error("expected objects directory to be recreated")
	}
}

func TestPrepareOutputDirs_Idempotent(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}

	// Run twice — second should succeed cleanly.
	if err := compile.PrepareOutputDirs(compiledDir); err != nil {
		t.Fatalf("first PrepareOutputDirs: %v", err)
	}
	if err := compile.PrepareOutputDirs(compiledDir); err != nil {
		t.Fatalf("second PrepareOutputDirs: %v", err)
	}
}

func makeTestChunk(id string, ct chunk.ChunkType, content string, seq int) chunk.Chunk {
	return chunk.Chunk{
		ID:       id,
		Type:     ct,
		Source:   "test/source.md",
		Heading:  "Test",
		Sequence: seq,
		Tokens:   100,
		Blocks: []markdown.Block{
			{Type: markdown.BlockParagraph, Content: content},
		},
		Content:     content,
		TotalInFile: 1,
	}
}

func TestWriteChunkFile_WritesRenderedContent(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := compile.PrepareOutputDirs(compiledDir); err != nil {
		t.Fatal(err)
	}

	c := makeTestChunk("obj:abcdef0123456789.1", chunk.ChunkObject, "Test content here.", 1)

	if err := compile.WriteChunkFile(compiledDir, &c); err != nil {
		t.Fatalf("WriteChunkFile: %v", err)
	}

	outPath := filepath.Join(compiledDir, chunk.OutputDir(chunk.ChunkObject), chunk.OutputFilename(&c))
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "codectx:meta") {
		t.Error("expected chunk file to contain codectx:meta header")
	}
	if !strings.Contains(content, "Test content here.") {
		t.Error("expected chunk file to contain content")
	}
	if !strings.Contains(content, "obj:abcdef0123456789.1") {
		t.Error("expected chunk file to contain chunk ID")
	}
}

func TestWriteChunkFile_NilChunkReturnsError(t *testing.T) {
	if err := compile.WriteChunkFile(t.TempDir(), nil); err == nil {
		t.Error("expected error for nil chunk")
	}
}

func TestWriteChunkFiles_WritesAll(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := compile.PrepareOutputDirs(compiledDir); err != nil {
		t.Fatal(err)
	}

	chunks := []chunk.Chunk{
		makeTestChunk("obj:aaaa000000000000.1", chunk.ChunkObject, "Object content", 1),
		makeTestChunk("spec:bbbb000000000000.1", chunk.ChunkSpec, "Spec content", 1),
		makeTestChunk("sys:cccc000000000000.1", chunk.ChunkSystem, "System content", 1),
	}

	written, err := compile.WriteChunkFiles(compiledDir, chunks)
	if err != nil {
		t.Fatalf("WriteChunkFiles: %v", err)
	}

	if written != 3 {
		t.Errorf("expected 3 files written, got %d", written)
	}

	// Verify each file exists.
	for i := range chunks {
		c := &chunks[i]
		outPath := filepath.Join(compiledDir, chunk.OutputDir(c.Type), chunk.OutputFilename(c))
		if _, err := os.Stat(outPath); err != nil {
			t.Errorf("expected chunk file %s to exist", outPath)
		}
	}
}

func TestWriteChunkFiles_EmptySlice(t *testing.T) {
	written, err := compile.WriteChunkFiles(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("WriteChunkFiles: %v", err)
	}
	if written != 0 {
		t.Errorf("expected 0 files written, got %d", written)
	}
}

// ---------------------------------------------------------------------------
// EnsureOutputDirs
// ---------------------------------------------------------------------------

func TestEnsureOutputDirs_CreatesDirectories(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}

	if err := compile.EnsureOutputDirs(compiledDir); err != nil {
		t.Fatalf("EnsureOutputDirs: %v", err)
	}

	for _, ct := range []chunk.ChunkType{chunk.ChunkObject, chunk.ChunkSpec, chunk.ChunkSystem} {
		dir := filepath.Join(compiledDir, chunk.OutputDir(ct))
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("expected directory %s to exist", dir)
		}
	}
}

func TestEnsureOutputDirs_PreservesExistingFiles(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	objDir := filepath.Join(compiledDir, chunk.OutputDir(chunk.ChunkObject))

	// Create directory and add a file.
	if err := os.MkdirAll(objDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}
	existingFile := filepath.Join(objDir, "existing-chunk.md")
	if err := os.WriteFile(existingFile, []byte("keep me"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	if err := compile.EnsureOutputDirs(compiledDir); err != nil {
		t.Fatalf("EnsureOutputDirs: %v", err)
	}

	// File should still be there.
	data, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("existing file removed: %v", err)
	}
	if string(data) != "keep me" {
		t.Error("existing file content changed")
	}
}

// ---------------------------------------------------------------------------
// RemoveChunkFiles
// ---------------------------------------------------------------------------

func TestRemoveChunkFiles_RemovesExisting(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := compile.PrepareOutputDirs(compiledDir); err != nil {
		t.Fatal(err)
	}

	// Write a chunk.
	c := makeTestChunk("obj:abcdef0123456789.1", chunk.ChunkObject, "Remove me", 1)
	if err := compile.WriteChunkFile(compiledDir, &c); err != nil {
		t.Fatal(err)
	}

	// Verify it exists.
	outPath := filepath.Join(compiledDir, chunk.OutputDir(chunk.ChunkObject), chunk.OutputFilename(&c))
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("chunk file should exist: %v", err)
	}

	// Remove it.
	if err := compile.RemoveChunkFiles(compiledDir, []string{"obj:abcdef0123456789.1"}); err != nil {
		t.Fatalf("RemoveChunkFiles: %v", err)
	}

	// Should be gone.
	if _, err := os.Stat(outPath); err == nil {
		t.Error("chunk file should have been removed")
	}
}

func TestRemoveChunkFiles_IgnoresMissing(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := compile.PrepareOutputDirs(compiledDir); err != nil {
		t.Fatal(err)
	}

	// Removing nonexistent chunk should not error.
	if err := compile.RemoveChunkFiles(compiledDir, []string{"obj:nonexistent0000.1"}); err != nil {
		t.Fatalf("RemoveChunkFiles: %v", err)
	}
}

func TestRemoveChunkFiles_EmptyList(t *testing.T) {
	if err := compile.RemoveChunkFiles(t.TempDir(), nil); err != nil {
		t.Fatalf("RemoveChunkFiles with empty list: %v", err)
	}
}

func TestRemoveChunkFiles_UnparseableID(t *testing.T) {
	// An ID that cannot be parsed by ChunkFilePath should be silently skipped.
	compiledDir := filepath.Join(t.TempDir(), project.CompiledDir)
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}

	err := compile.RemoveChunkFiles(compiledDir, []string{"not-a-valid-chunk-id", "also:bad"})
	if err != nil {
		t.Fatalf("RemoveChunkFiles with unparseable IDs should not error: %v", err)
	}
}

func TestWriteChunkFile_NonWritableDir(t *testing.T) {
	// compiledDir doesn't have the output subdirectory — should error on write.
	compiledDir := t.TempDir()

	c := makeTestChunk("obj:abcdef0123456789.1", chunk.ChunkObject, "Content", 1)
	err := compile.WriteChunkFile(compiledDir, &c)
	if err == nil {
		t.Error("expected error when output directory doesn't exist")
	}
}

func TestWriteChunkFiles_StopsOnError(t *testing.T) {
	// Write to a directory without proper subdirs — should fail on first chunk.
	compiledDir := t.TempDir()

	chunks := []chunk.Chunk{
		makeTestChunk("obj:aaaa000000000000.1", chunk.ChunkObject, "Content 1", 1),
		makeTestChunk("obj:bbbb000000000000.1", chunk.ChunkObject, "Content 2", 1),
	}

	written, err := compile.WriteChunkFiles(compiledDir, chunks)
	if err == nil {
		t.Error("expected error when output directory doesn't exist")
	}
	if written != 0 {
		t.Errorf("expected 0 files written before error, got %d", written)
	}
}
