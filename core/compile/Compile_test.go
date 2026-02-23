package compile

import (
	"os"
	"path/filepath"
	"testing"

	"securacore/codectx/core/config"
	"securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestProject creates a minimal project structure in a temp directory,
// changes to it (so lock file writes succeed), and returns the project root and config.
func setupTestProject(t *testing.T) (string, *config.Config) {
	t.Helper()
	dir := t.TempDir()

	// Compile writes codectx.lock to cwd; chdir to temp dir.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	outputDir := filepath.Join(dir, ".codectx")

	// Create docs directory structure.
	for _, sub := range []string{"foundation", "topics", "prompts", "plans"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	// Write a local package manifest.
	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project for compile",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), m))

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			DocsDir:   docsDir,
			OutputDir: outputDir,
		},
		Packages: []config.PackageDep{},
	}

	return dir, cfg
}

func TestCompile_emptyProject(t *testing.T) {
	dir, cfg := setupTestProject(t)

	result, err := Compile(cfg)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dir, ".codectx"), result.OutputDir)
	assert.Equal(t, 0, result.FilesCopied)
	assert.Equal(t, 0, result.Packages)

	// Verify outputs exist.
	_, err = os.Stat(filepath.Join(result.OutputDir, "package.yml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(result.OutputDir, "README.md"))
	assert.NoError(t, err)
}

func TestCompile_withLocalFiles(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Add a foundation document to the manifest.
	foundationPath := filepath.Join(docsDir, "foundation", "philosophy.md")
	require.NoError(t, os.WriteFile(foundationPath, []byte("# Philosophy\n"), 0o644))

	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Core philosophy", Load: "always"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	assert.Equal(t, 1, result.FilesCopied)

	// Verify the file was copied.
	copiedPath := filepath.Join(result.OutputDir, "foundation", "philosophy.md")
	data, err := os.ReadFile(copiedPath)
	require.NoError(t, err)
	assert.Equal(t, "# Philosophy\n", string(data))
}

func TestCompile_readmeContent(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Add entries to test README generation.
	foundationPath := filepath.Join(docsDir, "foundation", "doc.md")
	require.NoError(t, os.WriteFile(foundationPath, []byte("doc"), 0o644))

	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project",
		Foundation: []manifest.FoundationEntry{
			{ID: "doc", Path: "foundation/doc.md", Description: "A document"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	readmePath := filepath.Join(result.OutputDir, "README.md")
	data, err := os.ReadFile(readmePath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# test-project")
	assert.Contains(t, content, "## Loading Protocol")
	assert.Contains(t, content, "## Sections")
	assert.Contains(t, content, "**Foundation**: 1 document")
}

func TestCompile_cleansOutputDirectory(t *testing.T) {
	_, cfg := setupTestProject(t)
	outputDir := cfg.OutputDir()

	// Create stale file in output directory.
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	stalePath := filepath.Join(outputDir, "stale.txt")
	require.NoError(t, os.WriteFile(stalePath, []byte("stale"), 0o644))

	_, err := Compile(cfg)
	require.NoError(t, err)

	// Stale file should be gone.
	_, err = os.Stat(stalePath)
	assert.True(t, os.IsNotExist(err))
}

func TestCompile_writesLockFile(t *testing.T) {
	dir, cfg := setupTestProject(t)

	_, err := Compile(cfg)
	require.NoError(t, err)

	// Verify lock file was created in the project dir (cwd set by setupTestProject).
	_, err = os.Stat(filepath.Join(dir, "codectx.lock"))
	assert.NoError(t, err)
}
