package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/scaffold"
)

// ---------------------------------------------------------------------------
// Init tests
// ---------------------------------------------------------------------------

func TestInit_CreatesFullDirectoryStructure(t *testing.T) {
	dir := t.TempDir()

	result, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "test-project",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	docsRoot := filepath.Join(dir, "docs")

	// Verify result fields.
	if result.ProjectDir != dir {
		t.Errorf("expected project dir %s, got %s", dir, result.ProjectDir)
	}
	if result.DocsRoot != docsRoot {
		t.Errorf("expected docs root %s, got %s", docsRoot, result.DocsRoot)
	}
	if result.Root != "docs" {
		t.Errorf("expected root %q, got %q", "docs", result.Root)
	}
	if result.DirsCreated == 0 {
		t.Error("expected directories to be created")
	}
	if result.FilesCreated == 0 {
		t.Error("expected files to be created")
	}

	// Verify all expected directories exist.
	expectedDirs := []string{
		"docs/foundation",
		"docs/topics",
		"docs/plans",
		"docs/prompts",
		"docs/system/foundation/compiler-philosophy",
		"docs/system/topics/taxonomy-generation",
		"docs/system/topics/bridge-summaries",
		"docs/system/topics/context-assembly",
		"docs/system/plans",
		"docs/system/prompts",
		"docs/.codectx/compiled/objects",
		"docs/.codectx/compiled/specs",
		"docs/.codectx/compiled/system",
		"docs/.codectx/compiled/bm25/objects",
		"docs/.codectx/compiled/bm25/specs",
		"docs/.codectx/compiled/bm25/system",
		"docs/.codectx/packages",
	}

	for _, d := range expectedDirs {
		path := filepath.Join(dir, d)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}
}

func TestInit_CreatesConfigFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "config-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// codectx.yml at project root.
	configPath := filepath.Join(dir, project.ConfigFileName)
	cfg, err := project.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Name != "config-test" {
		t.Errorf("expected name %q, got %q", "config-test", cfg.Name)
	}
	if cfg.Root != "docs" {
		t.Errorf("expected root %q, got %q", "docs", cfg.Root)
	}
	if cfg.Version != "0.1.0" {
		t.Errorf("expected version %q, got %q", "0.1.0", cfg.Version)
	}

	// ai.yml in .codectx/.
	aiPath := filepath.Join(dir, "docs", ".codectx", "ai.yml")
	aiData, err := os.ReadFile(aiPath)
	if err != nil {
		t.Fatalf("reading ai.yml: %v", err)
	}
	aiContent := string(aiData)
	if !strings.Contains(aiContent, "cl100k_base") {
		t.Error("expected ai.yml to contain encoding")
	}
	if !strings.Contains(aiContent, "claude-sonnet-4-20250514") {
		t.Error("expected ai.yml to contain model name")
	}

	// preferences.yml in .codectx/.
	prefsPath := filepath.Join(dir, "docs", ".codectx", "preferences.yml")
	prefsData, err := os.ReadFile(prefsPath)
	if err != nil {
		t.Fatalf("reading preferences.yml: %v", err)
	}
	prefsContent := string(prefsData)
	if !strings.Contains(prefsContent, "target_tokens: 450") {
		t.Error("expected preferences.yml to contain chunking settings")
	}
	if !strings.Contains(prefsContent, "k1: 1.2") {
		t.Error("expected preferences.yml to contain BM25 settings")
	}
}

func TestInit_CreatesSystemDefaults(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "system-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	docsRoot := filepath.Join(dir, "docs")

	// Verify all default system documentation files exist and have content.
	expectedFiles := map[string]string{
		"system/foundation/compiler-philosophy/README.md":      "Compiler Philosophy",
		"system/foundation/compiler-philosophy/README.spec.md": "Compiler Philosophy Reasoning",
		"system/topics/taxonomy-generation/README.md":          "Taxonomy Alias Generation",
		"system/topics/taxonomy-generation/README.spec.md":     "Taxonomy Generation Reasoning",
		"system/topics/bridge-summaries/README.md":             "Bridge Summary Generation",
		"system/topics/bridge-summaries/README.spec.md":        "Bridge Summary Reasoning",
		"system/topics/context-assembly/README.md":             "Context Assembly Instructions",
	}

	for path, expectedContent := range expectedFiles {
		fullPath := filepath.Join(docsRoot, path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", path, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("expected file %s to have content", path)
			continue
		}
		if !strings.Contains(string(data), expectedContent) {
			t.Errorf("expected file %s to contain %q", path, expectedContent)
		}
	}
}

func TestInit_CreatesGitignore(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "gitignore-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gitignorePath := filepath.Join(dir, "docs", ".codectx", ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)
	expectedEntries := []string{
		".codectx/compiled/",
		".codectx/packages/",
		".codectx/ai.local.yml",
		"!.codectx/ai.yml",
		"!.codectx/preferences.yml",
	}

	for _, entry := range expectedEntries {
		if !strings.Contains(content, entry) {
			t.Errorf("expected .gitignore to contain %q", entry)
		}
	}
}

func TestInit_CustomRoot(t *testing.T) {
	dir := t.TempDir()

	result, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Root:       "ai-docs",
		Name:       "custom-root-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDocsRoot := filepath.Join(dir, "ai-docs")
	if result.DocsRoot != expectedDocsRoot {
		t.Errorf("expected docs root %s, got %s", expectedDocsRoot, result.DocsRoot)
	}
	if result.Root != "ai-docs" {
		t.Errorf("expected root %q, got %q", "ai-docs", result.Root)
	}

	// Verify config has the custom root.
	cfg, err := project.LoadConfig(filepath.Join(dir, project.ConfigFileName))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Root != "ai-docs" {
		t.Errorf("expected root %q, got %q", "ai-docs", cfg.Root)
	}

	// Verify the custom root directory exists with expected subdirs.
	foundationDir := filepath.Join(dir, "ai-docs", "foundation")
	if _, err := os.Stat(foundationDir); err != nil {
		t.Errorf("expected %s to exist: %v", foundationDir, err)
	}
}

func TestInit_DefaultNameFromDirectory(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		// Name intentionally omitted — should default to dir basename.
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := project.LoadConfig(filepath.Join(dir, project.ConfigFileName))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	expectedName := filepath.Base(dir)
	if cfg.Name != expectedName {
		t.Errorf("expected name %q (from dir basename), got %q", expectedName, cfg.Name)
	}
}

func TestInit_ErrorsWithEmptyProjectDir(t *testing.T) {
	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: "",
		Name:       "test",
	})
	if err == nil {
		t.Fatal("expected error for empty project dir")
	}
}

func TestInit_OverwritesOnSecondRun(t *testing.T) {
	// Init no longer checks for already-initialized — that's Check()'s job.
	// Running Init twice should succeed and overwrite existing files.
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "first-run",
	})
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Second run with a different name should succeed and overwrite.
	_, err = scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "second-run",
	})
	if err != nil {
		t.Fatalf("second init should succeed (no internal guard): %v", err)
	}

	// Verify the config has the second name.
	cfg, err := project.LoadConfig(filepath.Join(dir, project.ConfigFileName))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Name != "second-run" {
		t.Errorf("expected name %q, got %q", "second-run", cfg.Name)
	}
}

func TestInit_ModelAndEncodingPassthrough(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "model-test",
		Model:      "gpt-4o",
		Encoding:   "o200k_base",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	aiPath := filepath.Join(dir, "docs", ".codectx", "ai.yml")
	data, err := os.ReadFile(aiPath)
	if err != nil {
		t.Fatalf("reading ai.yml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "gpt-4o") {
		t.Error("expected ai.yml to contain custom model 'gpt-4o'")
	}
	if !strings.Contains(content, "o200k_base") {
		t.Error("expected ai.yml to contain custom encoding 'o200k_base'")
	}
	// Should NOT contain the default model.
	// Note: consumption.model also gets set to the custom model.
	// Both compilation and consumption should show gpt-4o.
}

func TestInit_ModelWithoutEncoding(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "partial-model-test",
		Model:      "gemini-2.0-flash",
		// Encoding intentionally omitted — should keep default.
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	aiPath := filepath.Join(dir, "docs", ".codectx", "ai.yml")
	data, err := os.ReadFile(aiPath)
	if err != nil {
		t.Fatalf("reading ai.yml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "gemini-2.0-flash") {
		t.Error("expected ai.yml to contain custom model")
	}
	if !strings.Contains(content, "cl100k_base") {
		t.Error("expected ai.yml to retain default encoding when not overridden")
	}
}

func TestInit_GitInit(t *testing.T) {
	dir := t.TempDir()

	result, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "git-test",
		GitInit:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.GitInitialized {
		t.Error("expected GitInitialized to be true")
	}

	// Verify .git/ directory was created.
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		t.Fatalf("expected .git/ to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected .git to be a directory")
	}
}

func TestInit_GitInitFalse(t *testing.T) {
	dir := t.TempDir()

	result, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "no-git-test",
		GitInit:    false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.GitInitialized {
		t.Error("expected GitInitialized to be false")
	}

	// Verify .git/ was NOT created.
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		t.Error("expected .git/ to NOT exist when GitInit is false")
	}
}

// ---------------------------------------------------------------------------
// Check tests
// ---------------------------------------------------------------------------

func TestCheck_CleanDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := scaffold.Check(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AlreadyInitialized {
		t.Error("expected AlreadyInitialized to be false in empty dir")
	}
	if result.NestedProject {
		t.Error("expected NestedProject to be false")
	}
	if result.HasGit {
		t.Error("expected HasGit to be false")
	}
	if result.RootConflict {
		t.Error("expected RootConflict to be false")
	}
	if !result.Writable {
		t.Error("expected Writable to be true for temp dir")
	}
}

func TestCheck_AlreadyInitialized(t *testing.T) {
	dir := t.TempDir()

	// Scaffold first.
	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "test",
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	result, err := scaffold.Check(dir, "")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if !result.AlreadyInitialized {
		t.Error("expected AlreadyInitialized to be true after init")
	}
}

func TestCheck_NestedProject(t *testing.T) {
	parent := t.TempDir()

	// Init in parent.
	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: parent,
		Name:       "parent",
	})
	if err != nil {
		t.Fatalf("parent init failed: %v", err)
	}

	// Create child directory.
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Check(child, "")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if !result.NestedProject {
		t.Error("expected NestedProject to be true")
	}
	if result.NestedProjectPath != parent {
		t.Errorf("expected NestedProjectPath %q, got %q", parent, result.NestedProjectPath)
	}
}

func TestCheck_HasGit(t *testing.T) {
	dir := t.TempDir()

	// Create a .git directory.
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Check(dir, "")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if !result.HasGit {
		t.Error("expected HasGit to be true")
	}
}

func TestCheck_RootConflict(t *testing.T) {
	dir := t.TempDir()

	// Create a docs/ directory with content.
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "existing.md"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Check(dir, "")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if !result.RootConflict {
		t.Error("expected RootConflict to be true when docs/ has files")
	}
}

func TestCheck_RootConflictCustomRoot(t *testing.T) {
	dir := t.TempDir()

	// Create a custom root with content.
	customRoot := filepath.Join(dir, "ai-docs")
	if err := os.MkdirAll(customRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(customRoot, "README.md"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Check(dir, "ai-docs")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if !result.RootConflict {
		t.Error("expected RootConflict to be true for custom root with files")
	}
}

func TestCheck_NoRootConflictEmptyDir(t *testing.T) {
	dir := t.TempDir()

	// Create an empty docs/ directory — no conflict.
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Check(dir, "")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if result.RootConflict {
		t.Error("expected RootConflict to be false for empty docs/")
	}
}

func TestCheck_NoRootConflictNonexistent(t *testing.T) {
	dir := t.TempDir()

	// docs/ doesn't exist at all — no conflict.
	result, err := scaffold.Check(dir, "")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if result.RootConflict {
		t.Error("expected RootConflict to be false when docs/ doesn't exist")
	}
}

func TestCheck_NestedSkippedWhenAlreadyInitialized(t *testing.T) {
	// When AlreadyInitialized is true, the nested check should be skipped.
	// This tests the optimization in Check() that avoids unnecessary Discover calls.
	parent := t.TempDir()

	// Init in parent.
	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: parent,
		Name:       "parent",
	})
	if err != nil {
		t.Fatalf("parent init failed: %v", err)
	}

	// Check the parent itself — AlreadyInitialized should be true,
	// NestedProject should be false (we don't walk up when already init).
	result, err := scaffold.Check(parent, "")
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if !result.AlreadyInitialized {
		t.Error("expected AlreadyInitialized to be true")
	}
	if result.NestedProject {
		t.Error("expected NestedProject to be false when AlreadyInitialized is true")
	}
}
