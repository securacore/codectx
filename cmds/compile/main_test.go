package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupProject creates a minimal project in a temp directory and
// changes cwd into it. Returns the project root. Caller should not
// need to restore cwd; t.Cleanup handles it.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	for _, sub := range []string{"foundation", "topics", "prompts", "plans"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	// Write codectx.yml (must pass schema validation).
	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	// Write docs/package.yml.
	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project for compile command",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), m))

	return dir
}

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "compile", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}

func TestRun_emptyProject(t *testing.T) {
	dir := setupProject(t)

	err := run()
	require.NoError(t, err)

	// Verify compiled output exists.
	outputDir := filepath.Join(dir, ".codectx")
	_, err = os.Stat(filepath.Join(outputDir, "manifest.yml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(outputDir, "README.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(outputDir, "heuristics.yml"))
	assert.NoError(t, err)
}

func TestRun_withFoundationEntry(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Add a foundation document.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy.md"),
		[]byte("# Philosophy\n"), 0o644))

	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Core philosophy"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), m))

	err := run()
	require.NoError(t, err)

	// Verify objects directory was populated.
	outputDir := filepath.Join(dir, ".codectx")
	entries, err := os.ReadDir(filepath.Join(outputDir, "objects"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestRun_upToDate(t *testing.T) {
	setupProject(t)

	// First compile succeeds.
	err := run()
	require.NoError(t, err)

	// Second compile should succeed (up-to-date path).
	err = run()
	require.NoError(t, err)
}

func TestRun_missingConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// No codectx.yml exists.
	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_invalidConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Write invalid YAML.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, configFile),
		[]byte("{{{{not valid"), 0o644))

	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_missingPackageManifest(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Write a valid codectx.yml but no docs/package.yml.
	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile")
}

func TestRun_writesLockFile(t *testing.T) {
	dir := setupProject(t)

	err := run()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "codectx.lock"))
	assert.NoError(t, err)
}

func TestRun_customOutputDir(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "custom-docs")
	outputDir := filepath.Join(dir, "custom-output")
	for _, sub := range []string{"foundation", "topics", "prompts", "plans"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			DocsDir:   docsDir,
			OutputDir: outputDir,
		},
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), m))

	err = run()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(outputDir, "manifest.yml"))
	assert.NoError(t, err)
}
