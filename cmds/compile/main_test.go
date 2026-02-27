package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/compile"
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

	// Write docs/manifest.yml.
	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project for compile command",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

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
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

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

	// Write a valid codectx.yml but no docs/manifest.yml.
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

func TestRun_withDedupAndConflict(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Write a local foundation entry.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "shared.md"),
		[]byte("# Shared\nLocal version.\n"), 0o644))

	// Update local manifest to include the foundation entry.
	localManifest := &manifest.Manifest{
		Name:    "test-project",
		Author:  "tester",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "shared", Path: "foundation/shared.md", Description: "Shared doc"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), localManifest))

	// Create an installed package with overlapping entries.
	pkgDir := filepath.Join(docsDir, "packages", "testpkg@testorg")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "foundation"), 0o755))

	// Same content as local (will be deduplicated).
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "foundation", "shared.md"),
		[]byte("# Shared\nLocal version.\n"), 0o644))

	// Different content than local with a different ID that would collide
	// if the package had the same ID — but let's make a true conflict:
	// same ID, different content.
	// We'll add a second foundation entry to the package with the same ID "shared".
	// Actually, since mergeManifestDedup checks the "shared" key which local
	// already occupies, the package entry with same content will be a duplicate.
	// Let's add a second pair: same ID, different content = conflict.
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "foundation", "conflicting.md"),
		[]byte("# Conflicting\nPackage version with different content.\n"), 0o644))

	// Write the package manifest with one duplicate (shared) and one conflict.
	// For the conflict, we need the local to also have the same ID.
	// Update: add "conflicting" to local first with different content.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "conflicting.md"),
		[]byte("# Conflicting\nOriginal local content.\n"), 0o644))

	localManifest.Foundation = append(localManifest.Foundation,
		manifest.FoundationEntry{ID: "conflicting", Path: "foundation/conflicting.md", Description: "Conflicting doc"})
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), localManifest))

	pkgManifest := &manifest.Manifest{
		Name:    "testpkg",
		Author:  "testorg",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "shared", Path: "foundation/shared.md", Description: "Shared doc"},
			{ID: "conflicting", Path: "foundation/conflicting.md", Description: "Conflicting doc"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	// Update codectx.yml to reference the package with active: all.
	cfg := &config.Config{
		Name: "test-project",
		Packages: []config.PackageDep{
			{
				Name:    "testpkg",
				Author:  "testorg",
				Version: "1.0.0",
				Source:  "https://github.com/testorg/testpkg",
				Active:  config.Activation{Mode: "all"},
			},
		},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	// Run compile — exercises dedup + conflict branches in run().
	err := run()
	require.NoError(t, err)

	// Verify compiled output exists (the function should not error even
	// with duplicates and conflicts).
	outputDir := filepath.Join(dir, ".codectx")
	_, err = os.Stat(filepath.Join(outputDir, "manifest.yml"))
	assert.NoError(t, err)
}

// --- formatSize ---

func TestFormatSize_bytes(t *testing.T) {
	assert.Equal(t, "0 B", formatSize(0))
	assert.Equal(t, "1 B", formatSize(1))
	assert.Equal(t, "500 B", formatSize(500))
	assert.Equal(t, "1023 B", formatSize(1023))
}

func TestFormatSize_kilobytes(t *testing.T) {
	assert.Equal(t, "1.0 kB", formatSize(1024))
	assert.Equal(t, "2.5 kB", formatSize(2560))
	assert.Equal(t, "10.0 kB", formatSize(10240))
	assert.Equal(t, "1023.0 kB", formatSize(1023*1024))
}

func TestFormatSize_megabytes(t *testing.T) {
	assert.Equal(t, "1.0 MB", formatSize(1048576))
	assert.Equal(t, "2.5 MB", formatSize(int(2.5*1024*1024)))
	assert.Equal(t, "10.0 MB", formatSize(10*1024*1024))
}

// --- formatTokens ---

func TestFormatTokens_below1000(t *testing.T) {
	assert.Equal(t, "~0", formatTokens(0))
	assert.Equal(t, "~1", formatTokens(1))
	assert.Equal(t, "~500", formatTokens(500))
	assert.Equal(t, "~999", formatTokens(999))
}

func TestFormatTokens_exact_thousands(t *testing.T) {
	assert.Equal(t, "~1k", formatTokens(1000))
	assert.Equal(t, "~2k", formatTokens(2000))
	assert.Equal(t, "~10k", formatTokens(10000))
}

func TestFormatTokens_fractional_thousands(t *testing.T) {
	assert.Equal(t, "~1.5k", formatTokens(1500))
	assert.Equal(t, "~2.3k", formatTokens(2300))
	assert.Equal(t, "~12.5k", formatTokens(12500))
}

// --- compileModel bubbletea ---

func TestCompileModel_init(t *testing.T) {
	m := newCompileModel()
	assert.Equal(t, "Compiling...", m.message)
	assert.False(t, m.done)
	assert.Nil(t, m.result)
	assert.Nil(t, m.err)
	cmd := m.Init()
	assert.NotNil(t, cmd) // spinner.Tick
}

func TestCompileModel_compileMsg(t *testing.T) {
	m := newCompileModel()
	result := &compile.Result{OutputDir: "/out", ObjectsStored: 5}

	updated, cmd := m.Update(compileMsg{result: result, err: nil})
	cm := updated.(compileModel)

	assert.True(t, cm.done)
	assert.Equal(t, result, cm.result)
	assert.Nil(t, cm.err)
	assert.NotNil(t, cmd) // tea.Quit
}

func TestCompileModel_compileMsgError(t *testing.T) {
	m := newCompileModel()

	updated, _ := m.Update(compileMsg{err: assert.AnError})
	cm := updated.(compileModel)

	assert.True(t, cm.done)
	assert.Error(t, cm.err)
}

func TestCompileModel_progressMsg(t *testing.T) {
	m := newCompileModel()

	updated, cmd := m.Update(progressMsg{Message: "Processing foundation..."})
	cm := updated.(compileModel)

	assert.Equal(t, "Processing foundation...", cm.message)
	assert.False(t, cm.done)
	assert.Nil(t, cmd)
}

func TestCompileModel_viewNotDone(t *testing.T) {
	m := newCompileModel()
	view := m.View()
	assert.Contains(t, view, "Compiling...")
}

func TestCompileModel_viewDone(t *testing.T) {
	m := newCompileModel()
	m.done = true
	view := m.View()
	assert.Empty(t, view)
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
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	err = run()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(outputDir, "manifest.yml"))
	assert.NoError(t, err)
}
