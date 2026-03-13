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
		"docs/system/foundation/documentation-protocol",
		"docs/system/foundation/history",
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
		"docs/.codectx/history/docs",
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
		"system/foundation/documentation-protocol/README.md":   "Documentation Protocol",
		"system/foundation/history/README.md":                  "History",
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

	// .gitignore should be at the project root (not inside .codectx/).
	gitignorePath := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)
	expectedEntries := []string{
		"docs/.codectx/compiled/",
		"docs/.codectx/packages/",
		"docs/.codectx/history/",
		"docs/.codectx/ai.local.yml",
		"!docs/.codectx/ai.yml",
		"!docs/.codectx/preferences.yml",
	}

	for _, entry := range expectedEntries {
		if !strings.Contains(content, entry) {
			t.Errorf("expected .gitignore to contain %q", entry)
		}
	}

	// Verify the old location does NOT exist.
	oldPath := filepath.Join(dir, "docs", ".codectx", ".gitignore")
	if _, err := os.Stat(oldPath); err == nil {
		t.Error("expected .gitignore to NOT exist at old location docs/.codectx/.gitignore")
	}
}

func TestInit_GitignoreIdempotent(t *testing.T) {
	dir := t.TempDir()

	// Run init twice.
	for i := range 2 {
		_, err := scaffold.Init(scaffold.Options{
			ProjectDir: dir,
			Name:       "idempotent-test",
		})
		if err != nil {
			t.Fatalf("init run %d: unexpected error: %v", i+1, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// Count occurrences of a managed pattern — should appear exactly once.
	count := strings.Count(content, "docs/.codectx/compiled/")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of compiled/ pattern, got %d", count)
	}
}

func TestInit_GitignorePreservesExisting(t *testing.T) {
	dir := t.TempDir()

	// Create a pre-existing .gitignore with custom content.
	existing := "# My project ignores\nnode_modules/\n*.log\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "preserve-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// Existing content should be preserved.
	if !strings.Contains(content, "node_modules/") {
		t.Error("expected existing node_modules/ entry to be preserved")
	}
	if !strings.Contains(content, "*.log") {
		t.Error("expected existing *.log entry to be preserved")
	}

	// codectx entries should also be present.
	if !strings.Contains(content, "docs/.codectx/compiled/") {
		t.Error("expected codectx entries to be present")
	}
}

func TestInit_GitignoreCustomRoot(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Root:       "custom",
		Name:       "custom-root-gitignore",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// Should use custom root path.
	if !strings.Contains(content, "custom/.codectx/compiled/") {
		t.Error("expected .gitignore to use custom root 'custom'")
	}
	// Should NOT contain default "docs" root.
	if strings.Contains(content, "docs/.codectx/") {
		t.Error("expected .gitignore to NOT contain default 'docs' root with custom root")
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

func TestInit_ProviderPassthrough(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "provider-test",
		Provider:   "cli",
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
	if !strings.Contains(content, "provider: cli") {
		t.Error("expected ai.yml to contain provider: cli")
	}
}

func TestInit_NonWritableDir(t *testing.T) {
	// Use a non-existent subdirectory of /dev/null (on macOS/Linux) or
	// a read-only directory to trigger directory creation failure.
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "readonly-parent", "child")

	// Create parent and make it read-only.
	parent := filepath.Join(dir, "readonly-parent")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: targetDir,
		Name:       "fail-test",
	})
	if err == nil {
		t.Error("expected error when directory creation fails")
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

// ---------------------------------------------------------------------------
// Init .gitkeep tests
// ---------------------------------------------------------------------------

func TestInit_CreatesGitkeepInEmptyContentDirs(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "gitkeep-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	contentDirs := []string{"foundation", "topics", "plans", "prompts"}
	for _, d := range contentDirs {
		gitkeepPath := filepath.Join(dir, "docs", d, ".gitkeep")
		if _, err := os.Stat(gitkeepPath); err != nil {
			t.Errorf("expected .gitkeep in docs/%s: %v", d, err)
		}
	}
}

func TestInit_NoGitkeepInCodectxDir(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "no-gitkeep-codectx",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// .codectx directories should NOT have .gitkeep.
	codectxDirs := []string{
		"docs/.codectx/compiled",
		"docs/.codectx/packages",
		"docs/.codectx/history",
	}
	for _, d := range codectxDirs {
		gitkeepPath := filepath.Join(dir, d, ".gitkeep")
		if _, err := os.Stat(gitkeepPath); err == nil {
			t.Errorf("unexpected .gitkeep in %s", d)
		}
	}
}

// ---------------------------------------------------------------------------
// Maintain tests
// ---------------------------------------------------------------------------

func TestMaintain_NoActionsOnFreshProject(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "maintain-test",
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cfg, err := project.LoadConfig(filepath.Join(dir, project.ConfigFileName))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	result, err := scaffold.Maintain(dir, cfg)
	if err != nil {
		t.Fatalf("Maintain: %v", err)
	}

	// Fresh project should have no actions (everything already exists).
	if result.HasActions() {
		t.Errorf("expected no actions on fresh project, got: dirs=%d files=%d +gitkeep=%d -gitkeep=%d",
			result.DirsCreated, result.FilesRestored, result.GitkeepsAdded, result.GitkeepsRemoved)
	}
}

func TestMaintain_RestoresDeletedDirectory(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "restore-dir-test",
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cfg, err := project.LoadConfig(filepath.Join(dir, project.ConfigFileName))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	// Delete the topics directory entirely.
	topicsDir := filepath.Join(dir, "docs", "topics")
	if err := os.RemoveAll(topicsDir); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Maintain(dir, cfg)
	if err != nil {
		t.Fatalf("Maintain: %v", err)
	}

	if result.DirsCreated == 0 {
		t.Error("expected at least one directory to be recreated")
	}

	// Verify topics/ was restored with .gitkeep.
	if _, err := os.Stat(topicsDir); err != nil {
		t.Errorf("expected topics/ to be restored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(topicsDir, ".gitkeep")); err != nil {
		t.Errorf("expected .gitkeep in restored topics/: %v", err)
	}
}

func TestMaintain_RestoresDeletedSystemFile(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "restore-file-test",
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cfg, err := project.LoadConfig(filepath.Join(dir, project.ConfigFileName))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	// Delete a system default file.
	sysFile := filepath.Join(dir, "docs", "system", "foundation", "documentation-protocol", "README.md")
	if err := os.Remove(sysFile); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Maintain(dir, cfg)
	if err != nil {
		t.Fatalf("Maintain: %v", err)
	}

	if result.FilesRestored == 0 {
		t.Error("expected at least one file to be restored")
	}

	// Verify file was restored.
	data, err := os.ReadFile(sysFile)
	if err != nil {
		t.Fatalf("expected file to be restored: %v", err)
	}
	if !strings.Contains(string(data), "Documentation Protocol") {
		t.Error("restored file doesn't have expected content")
	}
}

func TestMaintain_RemovesGitkeepWhenContentAdded(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "gitkeep-remove-test",
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cfg, err := project.LoadConfig(filepath.Join(dir, project.ConfigFileName))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	// Add content to the topics directory.
	topicsDir := filepath.Join(dir, "docs", "topics")
	if err := os.WriteFile(filepath.Join(topicsDir, "auth.md"), []byte("# Auth"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Maintain(dir, cfg)
	if err != nil {
		t.Fatalf("Maintain: %v", err)
	}

	if result.GitkeepsRemoved == 0 {
		t.Error("expected .gitkeep to be removed from topics/")
	}

	// Verify .gitkeep is gone.
	gitkeepPath := filepath.Join(topicsDir, ".gitkeep")
	if _, err := os.Stat(gitkeepPath); err == nil {
		t.Error(".gitkeep should be removed when directory has content")
	}
}

func TestMaintain_AddsGitkeepWhenContentRemoved(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffold.Init(scaffold.Options{
		ProjectDir: dir,
		Name:       "gitkeep-add-test",
	})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cfg, err := project.LoadConfig(filepath.Join(dir, project.ConfigFileName))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	// Add content then run maintain to remove .gitkeep.
	topicsDir := filepath.Join(dir, "docs", "topics")
	contentFile := filepath.Join(topicsDir, "auth.md")
	if err := os.WriteFile(contentFile, []byte("# Auth"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = scaffold.Maintain(dir, cfg)

	// Now remove the content file.
	if err := os.Remove(contentFile); err != nil {
		t.Fatal(err)
	}

	result, err := scaffold.Maintain(dir, cfg)
	if err != nil {
		t.Fatalf("Maintain: %v", err)
	}

	if result.GitkeepsAdded == 0 {
		t.Error("expected .gitkeep to be added to empty topics/")
	}

	// Verify .gitkeep is back.
	gitkeepPath := filepath.Join(topicsDir, ".gitkeep")
	if _, err := os.Stat(gitkeepPath); err != nil {
		t.Errorf("expected .gitkeep to be restored: %v", err)
	}
}

func TestMaintain_HasActionsReportsBool(t *testing.T) {
	r := &scaffold.MaintainResult{}
	if r.HasActions() {
		t.Error("expected HasActions to be false for zero result")
	}

	r.DirsCreated = 1
	if !r.HasActions() {
		t.Error("expected HasActions to be true when dirs created")
	}
}

func TestInitPackage_CreatesPackageStructure(t *testing.T) {
	dir := t.TempDir()

	result, err := scaffold.InitPackage(scaffold.PackageOptions{
		ProjectDir:  dir,
		Root:        "docs",
		Name:        "react-patterns",
		Org:         "community",
		Description: "React component patterns",
	})
	if err != nil {
		t.Fatalf("InitPackage: %v", err)
	}

	if result.DirsCreated == 0 {
		t.Error("expected directories to be created")
	}
	if result.FilesCreated == 0 {
		t.Error("expected files to be created")
	}

	// Verify package/ content directories.
	expectedDirs := []string{
		"package/foundation",
		"package/topics",
		"package/plans",
		"package/prompts",
	}
	for _, d := range expectedDirs {
		info, err := os.Stat(filepath.Join(dir, d))
		if err != nil {
			t.Errorf("expected %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}

	// Verify package/codectx.yml.
	manifestPath := filepath.Join(dir, "package", "codectx.yml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("package manifest not created: %v", err)
	}
	if !strings.Contains(string(data), "react-patterns") {
		t.Error("manifest should contain package name")
	}
	if !strings.Contains(string(data), "community") {
		t.Error("manifest should contain org")
	}

	// Verify README.md.
	readmePath := filepath.Join(dir, "README.md")
	readmeData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("README.md not created: %v", err)
	}
	if !strings.Contains(string(readmeData), "react-patterns@community") {
		t.Error("README should contain package reference")
	}
	if !strings.Contains(string(readmeData), "codectx add") {
		t.Error("README should contain install instructions")
	}

	// Verify GHA workflow.
	workflowPath := filepath.Join(dir, ".github", "workflows", "release.yml")
	if _, err := os.Stat(workflowPath); err != nil {
		t.Errorf("GHA workflow not created: %v", err)
	}

	// Verify docs/ authoring project was initialized.
	docsConfig := filepath.Join(dir, "codectx.yml")
	cfgData, err := os.ReadFile(docsConfig)
	if err != nil {
		t.Fatalf("project config not created: %v", err)
	}
	if !strings.Contains(string(cfgData), "type: package") {
		t.Error("project config should have type: package")
	}

	// Verify docs/ directories exist.
	docsDir := filepath.Join(dir, "docs")
	if _, err := os.Stat(docsDir); err != nil {
		t.Errorf("docs/ directory not created: %v", err)
	}

	// Verify InitResult is populated.
	if result.InitResult == nil {
		t.Error("InitResult should be populated")
	}
}

func TestInitPackage_EmptyProjectDir(t *testing.T) {
	_, err := scaffold.InitPackage(scaffold.PackageOptions{
		ProjectDir: "",
		Name:       "test",
	})
	if err == nil {
		t.Error("expected error for empty project directory")
	}
}

func TestInitPackage_GitInit(t *testing.T) {
	dir := t.TempDir()

	result, err := scaffold.InitPackage(scaffold.PackageOptions{
		ProjectDir: dir,
		Root:       "docs",
		Name:       "test-pkg",
		Org:        "org",
		GitInit:    true,
	})
	if err != nil {
		t.Fatalf("InitPackage: %v", err)
	}

	if !result.GitInitialized {
		t.Error("expected GitInitialized to be true")
	}

	// Verify .git directory exists.
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		t.Errorf(".git directory not created: %v", err)
	}
}
