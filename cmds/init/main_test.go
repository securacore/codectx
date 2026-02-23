package init

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRun_withName_createsDirectoryAndProject(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	err = run("test-project")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "test-project")

	// Verify the project directory was created.
	info, err := os.Stat(projectDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify codectx.yml was created inside the project dir.
	_, err = os.Stat(filepath.Join(projectDir, "codectx.yml"))
	assert.NoError(t, err)

	// Verify docs/package.yml was created.
	_, err = os.Stat(filepath.Join(projectDir, "docs", "package.yml"))
	assert.NoError(t, err)

	// Verify git was initialized.
	_, err = os.Stat(filepath.Join(projectDir, ".git"))
	assert.NoError(t, err)

	// Verify .gitignore was created with .codectx/ entry.
	data, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".codectx/")

	// Verify directory structure.
	dirs := []string{
		"docs",
		"docs/foundation",
		"docs/topics",
		"docs/prompts",
		"docs/plans",
		"docs/schemas",
		"docs/packages",
	}
	for _, d := range dirs {
		info, err := os.Stat(filepath.Join(projectDir, d))
		require.NoError(t, err, "directory %s should exist", d)
		assert.True(t, info.IsDir(), "%s should be a directory", d)
	}

	// Verify schemas were written.
	schemaFiles := []string{
		"docs/schemas/codectx.schema.json",
		"docs/schemas/package.schema.json",
		"docs/schemas/state.schema.json",
	}
	for _, f := range schemaFiles {
		_, err := os.Stat(filepath.Join(projectDir, f))
		assert.NoError(t, err, "schema %s should exist", f)
	}
}

func TestRun_failsIfAlreadyInitialized(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// First init succeeds.
	err = run("test-project")
	require.NoError(t, err)

	// Chdir into the project dir so the second init detects it.
	require.NoError(t, os.Chdir(filepath.Join(dir, "test-project")))

	// Second init fails (codectx.yml exists in cwd).
	err = run("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestEnsureGitignore_createsNewFile(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	err = ensureGitignore()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, ".codectx/\n", string(data))
}

func TestEnsureGitignore_appendsToExisting(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// Create existing .gitignore without .codectx/.
	err = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0o644)
	require.NoError(t, err)

	err = ensureGitignore()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "node_modules/")
	assert.Contains(t, string(data), ".codectx/")
}

func TestEnsureGitignore_skipsIfAlreadyPresent(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// Create existing .gitignore that already has .codectx/.
	err = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".codectx/\n"), 0o644)
	require.NoError(t, err)

	err = ensureGitignore()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	// Should not be duplicated.
	assert.Equal(t, ".codectx/\n", string(data))
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"unix newlines", "a\nb\nc", []string{"a", "b", "c"}},
		{"windows newlines", "a\r\nb\r\nc", []string{"a", "b", "c"}},
		{"trailing newline", "a\nb\n", []string{"a", "b"}},
		{"empty", "", nil},
		{"single line", "hello", []string{"hello"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, splitLines(tt.input))
		})
	}
}

func TestEnsureGit_initializesRepo(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	err = ensureGit()
	require.NoError(t, err)

	// Verify .git directory was created.
	_, err = os.Stat(filepath.Join(dir, ".git"))
	assert.NoError(t, err)

	// Verify .gitignore was created.
	_, err = os.Stat(filepath.Join(dir, ".gitignore"))
	assert.NoError(t, err)
}

func TestEnsureGit_skipsIfGitExists(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// Pre-create .git directory.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	err = ensureGit()
	require.NoError(t, err)

	// Should still create .gitignore.
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".codectx/")
}

func TestEnsureGitignore_noTrailingNewline(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// Write .gitignore without trailing newline.
	err = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/"), 0o644)
	require.NoError(t, err)

	err = ensureGitignore()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	content := string(data)
	assert.Equal(t, "node_modules/\n.codectx/\n", content)
}

func TestEnsureGitignore_substringNoFalsePositive(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// Write .gitignore with a substring that contains ".codectx/" but isn't an exact line match.
	err = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".codectx/subfolder\n"), 0o644)
	require.NoError(t, err)

	err = ensureGitignore()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	content := string(data)
	// The exact ".codectx/" line should have been appended.
	assert.Contains(t, content, ".codectx/subfolder\n")
	assert.Contains(t, content, "\n.codectx/\n")
}

func TestSplitLines_mixedLineEndings(t *testing.T) {
	result := splitLines("a\nb\r\nc")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestSplitLines_consecutiveNewlines(t *testing.T) {
	result := splitLines("a\n\nb")
	assert.Equal(t, []string{"a", "", "b"}, result)
}

func TestSplitLines_onlyNewlines(t *testing.T) {
	result := splitLines("\n\n")
	assert.Equal(t, []string{"", ""}, result)
}

func TestRun_withName_configContent(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	err = run("my-project")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "my-project")

	// Verify codectx.yml content.
	codectxData, err := os.ReadFile(filepath.Join(projectDir, "codectx.yml"))
	require.NoError(t, err)

	var codectxYAML struct {
		Name     string `yaml:"name"`
		Packages []any  `yaml:"packages"`
	}
	err = yaml.Unmarshal(codectxData, &codectxYAML)
	require.NoError(t, err)
	assert.Equal(t, "my-project", codectxYAML.Name)
	assert.Empty(t, codectxYAML.Packages)

	// Verify docs/package.yml content.
	pkgData, err := os.ReadFile(filepath.Join(projectDir, "docs", "package.yml"))
	require.NoError(t, err)

	var pkgYAML struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
	}
	err = yaml.Unmarshal(pkgData, &pkgYAML)
	require.NoError(t, err)
	assert.Equal(t, "my-project", pkgYAML.Name)
	assert.Equal(t, "0.1.0", pkgYAML.Version)
	assert.Equal(t, "Documentation package for my-project", pkgYAML.Description)
}
