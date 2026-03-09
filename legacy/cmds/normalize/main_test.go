package normalize

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command structure ---

func TestCommand_structure(t *testing.T) {
	assert.Equal(t, "normalize", Command.Name)
	assert.Equal(t, "AI-driven terminology normalization across documentation", Command.Usage)
}

func TestCommand_flags(t *testing.T) {
	flags := Command.Flags
	require.Len(t, flags, 1)

	names := make(map[string]bool)
	for _, f := range flags {
		for _, n := range f.Names() {
			names[n] = true
		}
	}
	assert.True(t, names["dry-run"], "expected --dry-run flag")
}

func TestCommand_actionIsWired(t *testing.T) {
	assert.NotNil(t, Command.Action)
}

func TestCommand_hasDescription(t *testing.T) {
	assert.Contains(t, Command.Description, "terminology")
	assert.Contains(t, Command.Description, "AI integration")
}

// --- countDocsFiles ---

func TestCountDocsFiles_countsMarkdownOnly(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "go"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Docs\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "philosophy", "README.md"), []byte("# Philosophy\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "go", "README.md"), []byte("# Go\n"), 0o644))

	// Non-md files should be skipped.
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "metadata.yml"), []byte("name: test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "schema.json"), []byte("{}"), 0o644))

	assert.Equal(t, 3, countDocsFiles(docsDir))
}

func TestCountDocsFiles_skipsPackagesDir(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "packages", "react@org"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))

	// Files in packages/ should not be counted.
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "packages", "react@org", "README.md"), []byte("# React\n"), 0o644))
	// Files outside packages/ should be counted.
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "philosophy", "README.md"), []byte("# Philosophy\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Docs\n"), 0o644))

	assert.Equal(t, 2, countDocsFiles(docsDir))
}

func TestCountDocsFiles_emptyDir(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	assert.Equal(t, 0, countDocsFiles(docsDir))
}

func TestCountDocsFiles_nestedPackages(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	// Deep nesting inside packages/ should still be skipped.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "packages", "a@b", "topics", "deep"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "packages", "a@b", "topics", "deep", "README.md"), []byte("# Deep\n"), 0o644))

	assert.Equal(t, 0, countDocsFiles(docsDir))
}

// --- dryRun ---

func TestDryRun_noError(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Docs\n"), 0o644))

	err := dryRun(docsDir, &mockLauncher{id: "claude"}, "### Foundation\n\n- **philosophy**: Guiding principles\n")
	assert.NoError(t, err)
}

func TestDryRun_emptyDocsDir(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	err := dryRun(docsDir, &mockLauncher{id: "opencode"}, "")
	assert.NoError(t, err)
}

func TestDryRun_noExistingSummary(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Docs\n"), 0o644))

	err := dryRun(docsDir, &mockLauncher{id: "claude"}, "No existing documentation.")
	assert.NoError(t, err)
}

// --- mockLauncher ---

type mockLauncher struct {
	id string
}

func (m *mockLauncher) ID() string     { return m.id }
func (m *mockLauncher) Binary() string { return "/usr/bin/echo" }
func (m *mockLauncher) NewSessionArgs(_, _ string) []string {
	return []string{"mock"}
}
func (m *mockLauncher) ResumeArgs(_, _ string) []string {
	return []string{"mock"}
}
func (m *mockLauncher) SupportsSessionID() bool                    { return false }
func (m *mockLauncher) FindLatestSession(_ string) (string, error) { return "", nil }
