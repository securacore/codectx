package compile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
)

// ---------------------------------------------------------------------------
// Incremental compilation integration tests
// ---------------------------------------------------------------------------

func TestRun_IncrementalFirstCompileIsFullRecompile(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// First compile with Incremental=true but no previous hashes should
	// behave as a full recompile.
	if result.IncrementalMode {
		t.Error("first compile should not report incremental mode")
	}
	if result.NewFiles != result.TotalFiles {
		t.Errorf("expected all files as new: NewFiles=%d, TotalFiles=%d",
			result.NewFiles, result.TotalFiles)
	}

	// Heuristics should say full recompile.
	heurData, err := readFileContent(t, manifest.HeuristicsPath(compiledDir))
	if err != nil {
		t.Fatalf("reading heuristics: %v", err)
	}
	if !strings.Contains(heurData, "full_recompile: true") {
		t.Error("expected full_recompile: true in heuristics")
	}
}

func TestRun_IncrementalSecondCompileNoChanges(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	// First compile.
	firstResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Second compile with no changes.
	secondResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if !secondResult.IncrementalMode {
		t.Error("second compile should use incremental mode")
	}
	if secondResult.NewFiles != 0 {
		t.Errorf("expected 0 new files, got %d", secondResult.NewFiles)
	}
	if secondResult.ModifiedFiles != 0 {
		t.Errorf("expected 0 modified files, got %d", secondResult.ModifiedFiles)
	}
	if secondResult.UnchangedFiles != firstResult.TotalFiles {
		t.Errorf("expected %d unchanged files, got %d",
			firstResult.TotalFiles, secondResult.UnchangedFiles)
	}

	// Should produce same total chunks.
	if secondResult.TotalChunks != firstResult.TotalChunks {
		t.Errorf("chunk count changed: first=%d, second=%d",
			firstResult.TotalChunks, secondResult.TotalChunks)
	}

	// Should produce same total tokens.
	if secondResult.TotalTokens != firstResult.TotalTokens {
		t.Errorf("token count changed: first=%d, second=%d",
			firstResult.TotalTokens, secondResult.TotalTokens)
	}

	// Heuristics should say incremental.
	heurData, err := readFileContent(t, manifest.HeuristicsPath(compiledDir))
	if err != nil {
		t.Fatalf("reading heuristics: %v", err)
	}
	if !strings.Contains(heurData, "full_recompile: false") {
		t.Error("expected full_recompile: false in heuristics")
	}
}

func TestRun_IncrementalWithModifiedFile(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	// First compile.
	firstResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Modify one file.
	authPath := filepath.Join(rootDir, "topics", "auth.md")
	mustWriteFile(t, authPath,
		"# Authentication v2\n\nAuthentication is now handled via OAuth2.\n\n## OAuth2\n\nOAuth2 replaces JWT tokens.\n")

	// Second compile should detect the modification.
	secondResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if !secondResult.IncrementalMode {
		t.Error("expected incremental mode")
	}
	if secondResult.ModifiedFiles != 1 {
		t.Errorf("expected 1 modified file, got %d", secondResult.ModifiedFiles)
	}
	if secondResult.NewFiles != 0 {
		t.Errorf("expected 0 new files, got %d", secondResult.NewFiles)
	}
	if secondResult.UnchangedFiles != firstResult.TotalFiles-1 {
		t.Errorf("expected %d unchanged files, got %d",
			firstResult.TotalFiles-1, secondResult.UnchangedFiles)
	}
}

func TestRun_IncrementalWithNewFile(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	// First compile.
	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Add a new file.
	mustWriteFile(t, filepath.Join(rootDir, "topics", "new-topic.md"),
		"# New Topic\n\nThis is a brand new topic.\n")

	// Second compile should detect the new file.
	secondResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if !secondResult.IncrementalMode {
		t.Error("expected incremental mode")
	}
	if secondResult.NewFiles != 1 {
		t.Errorf("expected 1 new file, got %d", secondResult.NewFiles)
	}
}

func TestRun_IncrementalWithDeletedFile(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	// First compile.
	firstResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Delete a file.
	specPath := filepath.Join(rootDir, "topics", "auth.spec.md")
	if err := os.Remove(specPath); err != nil {
		t.Fatalf("removing spec file: %v", err)
	}

	// Second compile should detect the deletion.
	secondResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if !secondResult.IncrementalMode {
		t.Error("expected incremental mode")
	}
	if secondResult.DeletedFiles != 1 {
		t.Errorf("expected 1 deleted file, got %d", secondResult.DeletedFiles)
	}

	// Total files should be one less.
	if secondResult.TotalFiles != firstResult.TotalFiles-1 {
		t.Errorf("expected %d total files, got %d",
			firstResult.TotalFiles-1, secondResult.TotalFiles)
	}
}

func TestRun_IncrementalPreservesChunkFiles(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	// First compile.
	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Count chunk files after first compile.
	firstCount := countAllChunkFiles(t, compiledDir)

	// Second compile with no changes — should preserve chunk files.
	_, err = compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	secondCount := countAllChunkFiles(t, compiledDir)

	if secondCount != firstCount {
		t.Errorf("chunk file count changed: first=%d, second=%d",
			firstCount, secondCount)
	}
}

func TestRun_IncrementalManifestConsistency(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	// First compile.
	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	firstManifest, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		t.Fatalf("loading first manifest: %v", err)
	}

	// Second compile (no changes).
	_, err = compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	secondManifest, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		t.Fatalf("loading second manifest: %v", err)
	}

	// Same chunk counts.
	if secondManifest.TotalChunks != firstManifest.TotalChunks {
		t.Errorf("manifest total_chunks changed: first=%d, second=%d",
			firstManifest.TotalChunks, secondManifest.TotalChunks)
	}

	// All chunk IDs should be preserved.
	for id := range firstManifest.Objects {
		if secondManifest.LookupEntry(id) == nil {
			t.Errorf("chunk %s missing from second manifest", id)
		}
	}
	for id := range firstManifest.Specs {
		if secondManifest.LookupEntry(id) == nil {
			t.Errorf("chunk %s missing from second manifest", id)
		}
	}
	for id := range firstManifest.System {
		if secondManifest.LookupEntry(id) == nil {
			t.Errorf("chunk %s missing from second manifest", id)
		}
	}
}

func TestRun_IncrementalTaxonomyPreserved(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	// First compile.
	firstResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Second compile (no changes).
	secondResult, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	// Taxonomy term count should be the same.
	if secondResult.TaxonomyTerms != firstResult.TaxonomyTerms {
		t.Errorf("taxonomy terms changed: first=%d, second=%d",
			firstResult.TaxonomyTerms, secondResult.TaxonomyTerms)
	}

	// Taxonomy file should be valid.
	tax, taxErr := taxonomy.Load(taxonomy.TaxonomyPath(compiledDir))
	if taxErr != nil {
		t.Fatalf("loading taxonomy after incremental: %v", taxErr)
	}
	if tax.TermCount != firstResult.TaxonomyTerms {
		t.Errorf("taxonomy term_count = %d, want %d", tax.TermCount, firstResult.TaxonomyTerms)
	}
}

func TestRun_IncrementalHashesPerSubdir(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = true

	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	hashes, err := manifest.LoadHashes(manifest.HashesPath(compiledDir))
	if err != nil {
		t.Fatalf("LoadHashes: %v", err)
	}

	// Should have per-subdirectory system hashes for existing system dirs.
	// The test project creates a taxonomy-generation system dir, which
	// will be hashed if present. Context-assembly may or may not exist
	// depending on the test fixture. At minimum, verify the format.
	for name, hash := range hashes.System {
		if !strings.HasPrefix(hash, "sha256:") {
			t.Errorf("system hash for %q missing sha256: prefix: %q", name, hash)
		}
	}

	// The old blob key should NOT be present.
	if _, ok := hashes.System[project.SystemDir]; ok {
		t.Error("should not have blob system hash — expected per-subdirectory")
	}
}

func TestRun_IncrementalDisabledAlwaysFullRecompile(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Incremental = false

	// First compile.
	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Second compile with incremental=false.
	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if result.IncrementalMode {
		t.Error("expected non-incremental mode when flag is false")
	}
	if result.NewFiles != result.TotalFiles {
		t.Errorf("expected all files as new in non-incremental mode")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func countAllChunkFiles(t *testing.T, compiledDir string) int {
	t.Helper()
	total := 0
	for _, ct := range []chunk.ChunkType{chunk.ChunkObject, chunk.ChunkSpec, chunk.ChunkSystem} {
		total += countMDFiles(t, filepath.Join(compiledDir, chunk.OutputDir(ct)))
	}
	return total
}
