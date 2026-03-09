package shared

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAutoCompileProject creates a minimal project structure with
// auto_compile set as specified. Returns the project root.
func setupAutoCompileProject(t *testing.T, autoCompile *bool) string {
	t.Helper()
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Write codectx.yml.
	cfg := &config.Config{Name: "test-project"}
	require.NoError(t, config.Write("codectx.yml", cfg))

	// Create docs directory with a minimal manifest.
	docsDir := filepath.Join(dir, "docs")
	for _, sub := range []string{"foundation", "topics", "prompts", "plans", "packages"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}
	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "test",
		Version:     "1.0.0",
		Description: "Test project",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Create output directory and set preferences.
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	if autoCompile != nil {
		prefs := &preferences.Preferences{AutoCompile: autoCompile}
		require.NoError(t, preferences.Write(outputDir, prefs))
	}

	return dir
}

func TestMaybeAutoCompile_autoCompileFalse(t *testing.T) {
	val := false
	setupAutoCompileProject(t, &val)

	cfg, err := config.Load("codectx.yml")
	require.NoError(t, err)

	// Should return nil immediately (no compilation).
	err = MaybeAutoCompile(cfg)
	assert.NoError(t, err)

	// Verify no compilation output was created.
	outputDir := cfg.OutputDir()
	entries, _ := os.ReadDir(filepath.Join(outputDir, "objects"))
	assert.Empty(t, entries)
}

func TestMaybeAutoCompile_autoCompileTrue(t *testing.T) {
	val := true
	setupAutoCompileProject(t, &val)

	cfg, err := config.Load("codectx.yml")
	require.NoError(t, err)

	err = MaybeAutoCompile(cfg)
	assert.NoError(t, err)
}

func TestMaybeAutoCompile_autoCompileTrue_compilesSuccessfully(t *testing.T) {
	val := true
	setupAutoCompileProject(t, &val)

	cfg, err := config.Load("codectx.yml")
	require.NoError(t, err)

	err = MaybeAutoCompile(cfg)
	require.NoError(t, err)

	// Verify compilation happened — output dir should have compiled files.
	outputDir := cfg.OutputDir()
	_, err = os.Stat(outputDir)
	assert.NoError(t, err)
}

func TestMaybeAutoCompile_compileError(t *testing.T) {
	val := true
	setupAutoCompileProject(t, &val)

	cfg, err := config.Load("codectx.yml")
	require.NoError(t, err)

	// Corrupt the local manifest so compile.Compile fails.
	docsDir := cfg.DocsDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "manifest.yml"),
		[]byte("{{{{invalid yaml"),
		0o644,
	))

	err = MaybeAutoCompile(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile:")
}

func TestMaybeAutoCompile_preferencesLoadError(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Config with an output dir that doesn't exist and has an unreadable
	// preferences file.
	cfg := &config.Config{
		Name: "test",
		Config: &config.BuildConfig{
			OutputDir: filepath.Join(dir, "bad-output"),
		},
	}

	outputDir := filepath.Join(dir, "bad-output")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	// Write an unreadable preferences file.
	prefsPath := filepath.Join(outputDir, "preferences.yml")
	require.NoError(t, os.WriteFile(prefsPath, []byte("{{invalid yaml"), 0o644))

	err = MaybeAutoCompile(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load preferences")
}
