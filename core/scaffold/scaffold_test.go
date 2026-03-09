package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/scaffold"
)

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

func TestInit_ErrorsWhenAlreadyInitialized(t *testing.T) {
	dir := t.TempDir()

	// First init should succeed.
	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "test",
	})
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Second init should fail with ErrAlreadyInitialized.
	_, err = scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "test",
	})
	if err == nil {
		t.Fatal("expected error on second init")
	}
	if err != project.ErrAlreadyInitialized {
		t.Errorf("expected ErrAlreadyInitialized, got: %v", err)
	}
}

func TestInit_ErrorsWhenParentHasConfig(t *testing.T) {
	parent := t.TempDir()

	// Init in parent.
	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: parent,
		Name:       "parent",
	})
	if err != nil {
		t.Fatalf("parent init failed: %v", err)
	}

	// Try to init in a child directory.
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}

	_, err = scaffold.Init(scaffold.Options{
		ProjectDir: child,
		Name:       "child",
	})
	if err == nil {
		t.Fatal("expected error when parent has codectx.yml")
	}
	if err != project.ErrAlreadyInitialized {
		t.Errorf("expected ErrAlreadyInitialized, got: %v", err)
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
