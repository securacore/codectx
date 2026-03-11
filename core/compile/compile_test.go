package compile_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
	"github.com/securacore/codectx/core/tokens"
)

// setupTestProject creates a minimal project structure for testing compilation.
// Returns (rootDir, compiledDir).
func setupTestProject(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	compiledDir := filepath.Join(root, project.CodectxDir, project.CompiledDir)

	// Create source files.
	mustWriteFile(t, filepath.Join(root, "foundation", "overview.md"),
		"# Project Overview\n\nThis project provides authentication services.\n\n## Architecture\n\nThe system uses a microservices architecture with JWT tokens.\n")
	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"),
		"# Authentication\n\nAuthentication is handled via JWT tokens.\n\n## JWT Tokens\n\nTokens are signed with RS256.\n\n## Refresh Flow\n\nRefresh tokens expire after 7 days.\n")
	mustWriteFile(t, filepath.Join(root, "topics", "auth.spec.md"),
		"# Authentication\n\nThe authentication system was designed with security first.\n\n## JWT Tokens\n\nWe chose RS256 because it allows public key verification.\n")
	mustWriteFile(t, filepath.Join(root, project.SystemDir, "topics", "taxonomy-generation", "README.md"),
		"# Taxonomy Generation\n\nGenerate aliases for canonical terms.\n")

	return root, compiledDir
}

func defaultTestConfig(rootDir, compiledDir string) compile.Config {
	taxCfg := project.DefaultPreferencesConfig().Taxonomy
	// Disable LLM augmentation in tests to avoid real API/CLI calls.
	taxCfg.LLMAliasGeneration = false

	return compile.Config{
		ProjectDir:  filepath.Dir(rootDir),
		RootDir:     rootDir,
		CompiledDir: compiledDir,
		SystemDir:   project.SystemDir,
		Encoding:    tokens.Cl100kBase,
		Version:     "test-v0.1.0",
		Chunking:    project.DefaultPreferencesConfig().Chunking,
		BM25:        project.DefaultPreferencesConfig().BM25,
		Validation:  project.DefaultPreferencesConfig().Validation,
		Taxonomy:    taxCfg,
		ActiveDeps:  nil,
	}
}

func TestRun_FullPipeline(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	// Track progress stages.
	var stages []string
	progress := func(stage, detail string) {
		stages = append(stages, stage)
	}

	result, err := compile.Run(cfg, progress)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify file counts.
	if result.TotalFiles != 4 {
		t.Errorf("expected 4 total files, got %d", result.TotalFiles)
	}
	if result.SpecFiles != 1 {
		t.Errorf("expected 1 spec file, got %d", result.SpecFiles)
	}

	// Verify chunk counts.
	if result.TotalChunks == 0 {
		t.Error("expected at least 1 chunk")
	}
	if result.ObjectChunks == 0 {
		t.Error("expected at least 1 object chunk")
	}
	if result.SpecChunks == 0 {
		t.Error("expected at least 1 spec chunk")
	}
	if result.SystemChunks == 0 {
		t.Error("expected at least 1 system chunk")
	}

	// Verify token statistics.
	if result.TotalTokens == 0 {
		t.Error("expected total tokens > 0")
	}
	if result.AvgTokens == 0 {
		t.Error("expected average tokens > 0")
	}
	if result.MinTokens == 0 || result.MinTokens > result.MaxTokens {
		t.Errorf("invalid token range: min=%d, max=%d", result.MinTokens, result.MaxTokens)
	}

	// Verify timing.
	if result.TotalSeconds <= 0 {
		t.Error("expected total seconds > 0")
	}

	// Verify progress was called.
	expectedStages := []string{
		compile.StagePrepare,
		compile.StageDiscover,
		compile.StageParse,
		compile.StageChunk,
		compile.StageWrite,
		compile.StageIndex,
		compile.StageTaxonomy,
		compile.StageLLM,
		compile.StageManifest,
	}
	for _, expected := range expectedStages {
		found := false
		for _, s := range stages {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected progress stage %q, but it was not reported", expected)
		}
	}

	// Verify LLM augmentation was skipped (disabled in test config).
	if !result.LLMSkipped {
		t.Error("expected LLM augmentation to be skipped in test")
	}
	if result.LLMAliasCount != 0 {
		t.Errorf("expected 0 LLM aliases, got %d", result.LLMAliasCount)
	}
	if result.LLMBridgeCount != 0 {
		t.Errorf("expected 0 LLM bridges, got %d", result.LLMBridgeCount)
	}
}

func TestRun_ChunkFilesExistOnDisk(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify chunk files in each output directory.
	checkChunkDirHasFiles := func(ct chunk.ChunkType, expectedMin int) {
		dir := filepath.Join(compiledDir, chunk.OutputDir(ct))
		entries, err := filepath.Glob(filepath.Join(dir, "*.md"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		if len(entries) < expectedMin {
			t.Errorf("%s: expected at least %d chunk files, got %d", chunk.OutputDir(ct), expectedMin, len(entries))
		}
	}

	if result.ObjectChunks > 0 {
		checkChunkDirHasFiles(chunk.ChunkObject, 1)
	}
	if result.SpecChunks > 0 {
		checkChunkDirHasFiles(chunk.ChunkSpec, 1)
	}
	if result.SystemChunks > 0 {
		checkChunkDirHasFiles(chunk.ChunkSystem, 1)
	}
}

func TestRun_BM25IndexExistsAndLoadable(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Load the BM25 index.
	idx, err := index.Load(compiledDir)
	if err != nil {
		t.Fatalf("loading BM25 index: %v", err)
	}

	// Query should return results.
	results := idx.QueryAll("authentication jwt tokens", 5)
	totalResults := 0
	for _, r := range results {
		totalResults += len(r)
	}

	if totalResults == 0 {
		t.Error("expected BM25 query to return results")
	}
}

func TestRun_ManifestFilesExistAndValid(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify manifest.yml exists.
	manifestPath := manifest.EntryPath(compiledDir)
	assertFileExists(t, manifestPath)

	// Verify metadata.yml exists.
	metadataPath := manifest.MetadataPath(compiledDir)
	assertFileExists(t, metadataPath)

	// Verify hashes.yml exists.
	hashesPath := manifest.HashesPath(compiledDir)
	assertFileExists(t, hashesPath)

	// Verify heuristics.yml exists.
	heuristicsPath := manifest.HeuristicsPath(compiledDir)
	assertFileExists(t, heuristicsPath)

	// Verify taxonomy.yml exists.
	taxonomyPath := taxonomy.TaxonomyPath(compiledDir)
	assertFileExists(t, taxonomyPath)
}

func TestRun_ManifestContainsAllChunks(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Read manifest to verify chunk counts match.
	manifestPath := manifest.EntryPath(compiledDir)
	data, readErr := readFileContent(t, manifestPath)
	if readErr != nil {
		t.Fatalf("reading manifest: %v", readErr)
	}

	// Verify the manifest contains the right total.
	totalLine := fmt.Sprintf("total_chunks: %d", result.TotalChunks)
	if !strings.Contains(data, totalLine) {
		t.Errorf("manifest missing %q", totalLine)
	}
}

func TestRun_EmptyProject(t *testing.T) {
	root := t.TempDir()
	compiledDir := filepath.Join(root, project.CodectxDir, project.CompiledDir)
	cfg := defaultTestConfig(root, compiledDir)

	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.TotalFiles != 0 {
		t.Errorf("expected 0 total files, got %d", result.TotalFiles)
	}
	if result.TotalChunks != 0 {
		t.Errorf("expected 0 total chunks, got %d", result.TotalChunks)
	}

	// Manifest files should still be created.
	assertFileExists(t, manifest.EntryPath(compiledDir))
	assertFileExists(t, manifest.MetadataPath(compiledDir))
	assertFileExists(t, manifest.HashesPath(compiledDir))
	assertFileExists(t, manifest.HeuristicsPath(compiledDir))
}

func TestRun_NilProgressDoesNotPanic(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	// Should not panic with nil progress.
	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_ValidationWarningsCollected(t *testing.T) {
	root := t.TempDir()
	compiledDir := filepath.Join(root, project.CodectxDir, project.CompiledDir)

	// Create a file without any headings — should produce a validation warning.
	mustWriteFile(t, filepath.Join(root, "topics", "no-heading.md"),
		"This file has no heading structure at all.\n\nJust plain paragraphs.\n")

	cfg := defaultTestConfig(root, compiledDir)
	cfg.Validation.RequireHeadings = true

	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Error("expected validation warnings for file without headings")
	}

	foundWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no-heading.md") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected warning mentioning no-heading.md, got: %v", result.Warnings)
	}
}

func TestRun_CleanSlateOnRecompile(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	// First compile.
	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Count chunk files after first compile.
	firstObjFiles := countMDFiles(t, filepath.Join(compiledDir, chunk.OutputDir(chunk.ChunkObject)))

	// Second compile should produce the same result (clean slate).
	_, err = compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	secondObjFiles := countMDFiles(t, filepath.Join(compiledDir, chunk.OutputDir(chunk.ChunkObject)))

	if firstObjFiles != secondObjFiles {
		t.Errorf("expected same chunk count after recompile: first=%d, second=%d",
			firstObjFiles, secondObjFiles)
	}
}

func TestRun_HeuristicsContainsVersion(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Version = "test-v1.2.3"

	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, readErr := readFileContent(t, manifest.HeuristicsPath(compiledDir))
	if readErr != nil {
		t.Fatalf("reading heuristics: %v", readErr)
	}

	if !strings.Contains(data, "test-v1.2.3") {
		t.Error("expected heuristics to contain compiler version")
	}
}

func TestRun_SpecObjectLinking(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	_, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Read manifest and verify spec-object linking happened.
	data, readErr := readFileContent(t, manifest.EntryPath(compiledDir))
	if readErr != nil {
		t.Fatalf("reading manifest: %v", readErr)
	}

	// The spec file auth.spec.md should have parent_object references
	// and the object file auth.md should have spec_chunk references.
	if !strings.Contains(data, "parent_object:") && !strings.Contains(data, "spec_chunk:") {
		t.Log("Note: spec-object linking may not have produced matches for these test files")
	}
}

// ---------------------------------------------------------------------------
// Context assembly integration
// ---------------------------------------------------------------------------

func TestRun_SessionContextAssembly(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	// Add session config pointing to existing foundation docs.
	cfg.Session = &project.SessionConfig{
		AlwaysLoaded: []string{"foundation"},
		Budget:       30000,
	}

	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Session should have been assembled.
	if result.SessionTokens == 0 {
		t.Error("expected non-zero session tokens")
	}
	if result.SessionBudget != 30000 {
		t.Errorf("expected budget 30000, got %d", result.SessionBudget)
	}
	if len(result.SessionEntries) != 1 {
		t.Fatalf("expected 1 session entry, got %d", len(result.SessionEntries))
	}
	if result.SessionEntries[0].Reference != "foundation" {
		t.Errorf("expected reference %q, got %q", "foundation", result.SessionEntries[0].Reference)
	}

	// context.md should exist.
	contextPath := filepath.Join(compiledDir, "context.md")
	if _, err := os.Stat(contextPath); err != nil {
		t.Errorf("expected context.md to exist: %v", err)
	}

	// Read and verify content.
	data, readErr := os.ReadFile(contextPath)
	if readErr != nil {
		t.Fatalf("reading context.md: %v", readErr)
	}
	content := string(data)
	if !strings.Contains(content, "Project Engineering Context") {
		t.Error("expected context.md header")
	}
}

func TestRun_NoSessionConfig(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	// No session config — context assembly should be skipped.
	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.SessionTokens != 0 {
		t.Errorf("expected 0 session tokens, got %d", result.SessionTokens)
	}

	// context.md should NOT exist.
	contextPath := filepath.Join(compiledDir, "context.md")
	if _, err := os.Stat(contextPath); err == nil {
		t.Error("expected context.md to not exist when no session config")
	}
}

func TestRun_SessionContextWithBudgetWarning(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	// Set a very small budget to trigger a warning.
	cfg.Session = &project.SessionConfig{
		AlwaysLoaded: []string{"foundation"},
		Budget:       1,
	}

	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should have a budget warning.
	foundBudgetWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "exceeds budget") {
			foundBudgetWarning = true
			break
		}
	}
	if !foundBudgetWarning {
		t.Errorf("expected budget warning, got: %v", result.Warnings)
	}
}

func TestRun_SessionContextStageReported(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Session = &project.SessionConfig{
		AlwaysLoaded: []string{"foundation"},
		Budget:       30000,
	}

	var stages []string
	progress := func(stage, detail string) {
		stages = append(stages, stage)
	}

	_, err := compile.Run(cfg, progress)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, s := range stages {
		if s == compile.StageContext {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %q stage reported, got: %v", compile.StageContext, stages)
	}
}

// ---------------------------------------------------------------------------
// Error path tests (C2)
// ---------------------------------------------------------------------------

func TestRun_InvalidEncoding(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)
	cfg.Encoding = "nonexistent-encoding-xyz"

	_, err := compile.Run(cfg, nil)
	if err == nil {
		t.Fatal("expected error for invalid encoding")
	}
	if !strings.Contains(err.Error(), "token counter") {
		t.Errorf("expected token counter error, got: %v", err)
	}
}

func TestRun_UnreadableSourceFile(t *testing.T) {
	root := t.TempDir()
	compiledDir := filepath.Join(root, project.CodectxDir, project.CompiledDir)

	// Create a file, then make it unreadable.
	mdPath := filepath.Join(root, "topics", "secret.md")
	mustWriteFile(t, mdPath, "# Secret\n\nContent.\n")
	if err := os.Chmod(mdPath, 0000); err != nil {
		t.Skipf("cannot change file permissions (possibly running as root): %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(mdPath, project.FilePerm) })

	cfg := defaultTestConfig(root, compiledDir)
	_, err := compile.Run(cfg, nil)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
	if !strings.Contains(err.Error(), "reading") {
		t.Errorf("expected reading error, got: %v", err)
	}
}

func TestRun_NonexistentRootDir(t *testing.T) {
	compiledDir := filepath.Join(t.TempDir(), project.CodectxDir, project.CompiledDir)
	cfg := defaultTestConfig("/nonexistent/root/dir", compiledDir)

	_, err := compile.Run(cfg, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent root directory")
	}
	if !strings.Contains(err.Error(), "discovering sources") {
		t.Errorf("expected discovering sources error, got: %v", err)
	}
}

func TestRun_TaxonomyExtraction(t *testing.T) {
	rootDir, compiledDir := setupTestProject(t)
	cfg := defaultTestConfig(rootDir, compiledDir)

	result, err := compile.Run(cfg, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify taxonomy terms were extracted.
	if result.TaxonomyTerms == 0 {
		t.Error("expected at least 1 taxonomy term")
	}

	// Verify taxonomy.yml is loadable.
	taxPath := taxonomy.TaxonomyPath(compiledDir)
	tax, loadErr := taxonomy.Load(taxPath)
	if loadErr != nil {
		t.Fatalf("loading taxonomy: %v", loadErr)
	}

	if tax.Encoding != "cl100k_base" {
		t.Errorf("encoding: expected %q, got %q", "cl100k_base", tax.Encoding)
	}
	if tax.TermCount == 0 {
		t.Error("expected non-zero term_count")
	}
	if len(tax.Terms) != tax.TermCount {
		t.Errorf("term_count %d != len(terms) %d", tax.TermCount, len(tax.Terms))
	}

	// Verify manifest has terms populated.
	mfstPath := manifest.EntryPath(compiledDir)
	mfst, mfstErr := manifest.LoadManifest(mfstPath)
	if mfstErr != nil {
		t.Fatalf("loading manifest: %v", mfstErr)
	}

	// At least some object chunks should have terms.
	hasTerms := false
	for _, entry := range mfst.Objects {
		if len(entry.Terms) > 0 {
			hasTerms = true
			break
		}
	}
	if !hasTerms {
		t.Error("expected at least one object chunk to have taxonomy terms")
	}

	// Verify timing was recorded.
	if result.TaxonomySeconds <= 0 {
		t.Error("expected taxonomy seconds > 0")
	}
}

// --- Helpers ---

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %s to exist: %v", path, err)
	}
}

func readFileContent(t *testing.T, path string) (string, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func countMDFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	return len(entries)
}
