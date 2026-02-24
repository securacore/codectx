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

func setupFingerprintProject(t *testing.T) (string, *config.Config) {
	t.Helper()
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	outputDir := filepath.Join(dir, ".codectx")

	for _, sub := range []string{"foundation", "topics"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	// Write a foundation file.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy.md"),
		[]byte("# Philosophy\n\nCore principles.\n"),
		0o644,
	))

	// Write local manifest.
	m := &manifest.Manifest{
		Name:        "fp-test",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Fingerprint test",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Core philosophy", Load: "always"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), m))

	cfg := &config.Config{
		Name: "fp-test",
		Config: &config.BuildConfig{
			DocsDir:   docsDir,
			OutputDir: outputDir,
		},
		Packages: []config.PackageDep{},
	}

	// Write codectx.yml (fingerprint reads it from cwd).
	require.NoError(t, config.Write("codectx.yml", cfg))

	return dir, cfg
}

func TestComputeFingerprint_deterministic(t *testing.T) {
	_, cfg := setupFingerprintProject(t)

	fp1, err := computeFingerprint(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, fp1)

	fp2, err := computeFingerprint(cfg)
	require.NoError(t, err)

	assert.Equal(t, fp1, fp2, "same inputs should produce same fingerprint")
}

func TestComputeFingerprint_changesWhenFileChanges(t *testing.T) {
	dir, cfg := setupFingerprintProject(t)

	fp1, err := computeFingerprint(cfg)
	require.NoError(t, err)

	// Modify the foundation file.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "docs", "foundation", "philosophy.md"),
		[]byte("# Philosophy\n\nUpdated principles.\n"),
		0o644,
	))

	fp2, err := computeFingerprint(cfg)
	require.NoError(t, err)

	assert.NotEqual(t, fp1, fp2, "different content should produce different fingerprint")
}

func TestComputeFingerprint_changesWhenConfigChanges(t *testing.T) {
	_, cfg := setupFingerprintProject(t)

	fp1, err := computeFingerprint(cfg)
	require.NoError(t, err)

	// Modify the config file.
	cfg.Name = "changed-name"
	require.NoError(t, config.Write("codectx.yml", cfg))

	fp2, err := computeFingerprint(cfg)
	require.NoError(t, err)

	assert.NotEqual(t, fp1, fp2, "config change should produce different fingerprint")
}

func TestComputeFingerprint_changesWhenManifestChanges(t *testing.T) {
	dir, cfg := setupFingerprintProject(t)

	fp1, err := computeFingerprint(cfg)
	require.NoError(t, err)

	// Add another entry to the manifest.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "docs", "foundation", "conventions.md"),
		[]byte("# Conventions\n"),
		0o644,
	))

	m := &manifest.Manifest{
		Name:        "fp-test",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Fingerprint test",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Core philosophy", Load: "always"},
			{ID: "conventions", Path: "foundation/conventions.md", Description: "Conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(dir, "docs", "package.yml"), m))

	fp2, err := computeFingerprint(cfg)
	require.NoError(t, err)

	assert.NotEqual(t, fp1, fp2, "manifest change should produce different fingerprint")
}

func TestSaveAndLoadFingerprint(t *testing.T) {
	dir := t.TempDir()

	err := saveFingerprint(dir, "abc123")
	require.NoError(t, err)

	loaded := loadFingerprint(dir)
	assert.Equal(t, "abc123", loaded)
}

func TestLoadFingerprint_missingFile(t *testing.T) {
	dir := t.TempDir()
	loaded := loadFingerprint(dir)
	assert.Empty(t, loaded)
}

func TestCompile_incrementalSkip(t *testing.T) {
	_, cfg := setupFingerprintProject(t)

	// First compile should run.
	result1, err := Compile(cfg)
	require.NoError(t, err)
	assert.False(t, result1.UpToDate)
	assert.Equal(t, 1, result1.ObjectsStored)

	// Second compile with no changes should be up-to-date.
	result2, err := Compile(cfg)
	require.NoError(t, err)
	assert.True(t, result2.UpToDate)
	assert.Equal(t, 0, result2.ObjectsStored)
}

func TestCompile_incrementalRebuildsAfterChange(t *testing.T) {
	dir, cfg := setupFingerprintProject(t)

	// First compile.
	result1, err := Compile(cfg)
	require.NoError(t, err)
	assert.False(t, result1.UpToDate)

	// Modify a source file.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "docs", "foundation", "philosophy.md"),
		[]byte("# Philosophy\n\nNew content.\n"),
		0o644,
	))

	// Second compile should detect change.
	result2, err := Compile(cfg)
	require.NoError(t, err)
	assert.False(t, result2.UpToDate)
	assert.Equal(t, 1, result2.ObjectsStored)
}

func TestCompile_fingerprintPreservedDuringClean(t *testing.T) {
	_, cfg := setupFingerprintProject(t)

	// Compile to create fingerprint.
	_, err := Compile(cfg)
	require.NoError(t, err)

	// Verify fingerprint file exists.
	fpPath := filepath.Join(cfg.OutputDir(), ".fingerprint")
	_, err = os.Stat(fpPath)
	assert.NoError(t, err)

	// The fingerprint should match (so second compile is up-to-date).
	fp := loadFingerprint(cfg.OutputDir())
	assert.NotEmpty(t, fp)
}
