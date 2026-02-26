package compile

import (
	"crypto/sha256"
	"fmt"
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
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
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
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Description: "Core philosophy", Load: "always"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

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
		filepath.Join(dir, "docs", "foundation", "philosophy", "README.md"),
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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "docs", "foundation", "conventions"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "docs", "foundation", "conventions", "README.md"),
		[]byte("# Conventions\n"),
		0o644,
	))

	m := &manifest.Manifest{
		Name:        "fp-test",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Fingerprint test",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Description: "Core philosophy", Load: "always"},
			{ID: "conventions", Path: "foundation/conventions/README.md", Description: "Conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(dir, "docs", "manifest.yml"), m))

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
		filepath.Join(dir, "docs", "foundation", "philosophy", "README.md"),
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

func TestHashFile_includesPathInHash(t *testing.T) {
	dir := t.TempDir()
	content := []byte("same content")
	pathA := filepath.Join(dir, "a.md")
	pathB := filepath.Join(dir, "b.md")
	require.NoError(t, os.WriteFile(pathA, content, 0o644))
	require.NoError(t, os.WriteFile(pathB, content, 0o644))

	h1 := sha256.New()
	require.NoError(t, hashFile(h1, pathA))
	h2 := sha256.New()
	require.NoError(t, hashFile(h2, pathB))

	assert.NotEqual(t, fmt.Sprintf("%x", h1.Sum(nil)), fmt.Sprintf("%x", h2.Sum(nil)),
		"same content at different paths should produce different hashes")
}

func TestHashFile_missingFile(t *testing.T) {
	h := sha256.New()
	err := hashFile(h, "/nonexistent/file.md")
	assert.Error(t, err)
}

func TestHashManifestFiles_sortedDeterminism(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.md"), []byte("bravo"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("alpha"), 0o644))

	m1 := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "f1", Path: "a.md"},
			{ID: "f2", Path: "b.md"},
		},
	}
	m2 := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "f2", Path: "b.md"},
			{ID: "f1", Path: "a.md"},
		},
	}

	h1 := sha256.New()
	require.NoError(t, hashManifestFiles(h1, m1, dir))
	h2 := sha256.New()
	require.NoError(t, hashManifestFiles(h2, m2, dir))

	assert.Equal(t, fmt.Sprintf("%x", h1.Sum(nil)), fmt.Sprintf("%x", h2.Sum(nil)),
		"different entry order should produce same hash due to sorting")
}

func TestHashManifestFiles_skipsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "f1", Path: "missing.md"},
		},
	}

	h := sha256.New()
	err := hashManifestFiles(h, m, dir)
	assert.NoError(t, err, "missing files should be silently skipped")
}

func TestHashManifestFiles_includesSpecAndFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "topic.md"), []byte("topic"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spec.md"), []byte("spec"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.md"), []byte("extra"), 0o644))

	withSpec := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "t1", Path: "topic.md", Spec: "spec.md", Files: []string{"extra.md"}},
		},
	}
	withoutSpec := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "t1", Path: "topic.md"},
		},
	}

	h1 := sha256.New()
	require.NoError(t, hashManifestFiles(h1, withSpec, dir))
	h2 := sha256.New()
	require.NoError(t, hashManifestFiles(h2, withoutSpec, dir))

	assert.NotEqual(t, fmt.Sprintf("%x", h1.Sum(nil)), fmt.Sprintf("%x", h2.Sum(nil)),
		"spec and files should contribute to the hash")
}

func TestHashManifestFiles_applicationSection(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.md"), []byte("app content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app-spec.md"), []byte("spec content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app-file.md"), []byte("file content"), 0o644))

	m := &manifest.Manifest{
		Application: []manifest.ApplicationEntry{
			{ID: "a1", Path: "app.md", Spec: "app-spec.md", Files: []string{"app-file.md"}},
		},
	}
	withoutApp := &manifest.Manifest{}

	h1 := sha256.New()
	require.NoError(t, hashManifestFiles(h1, m, dir))
	h2 := sha256.New()
	require.NoError(t, hashManifestFiles(h2, withoutApp, dir))

	assert.NotEqual(t, fmt.Sprintf("%x", h1.Sum(nil)), fmt.Sprintf("%x", h2.Sum(nil)),
		"application entries with spec and files should change the hash")
}

func TestHashManifestFiles_promptsSection(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "prompt.md"), []byte("prompt content"), 0o644))

	m := &manifest.Manifest{
		Prompts: []manifest.PromptEntry{
			{ID: "p1", Path: "prompt.md"},
		},
	}
	empty := &manifest.Manifest{}

	h1 := sha256.New()
	require.NoError(t, hashManifestFiles(h1, m, dir))
	h2 := sha256.New()
	require.NoError(t, hashManifestFiles(h2, empty, dir))

	assert.NotEqual(t, fmt.Sprintf("%x", h1.Sum(nil)), fmt.Sprintf("%x", h2.Sum(nil)),
		"prompts entries should change the hash")
}

func TestHashManifestFiles_plansWithState(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plan.md"), []byte("plan content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plan.yml"), []byte("state: active"), 0o644))

	withState := &manifest.Manifest{
		Plans: []manifest.PlanEntry{
			{ID: "pl1", Path: "plan.md", PlanState: "plan.yml"},
		},
	}
	withoutState := &manifest.Manifest{
		Plans: []manifest.PlanEntry{
			{ID: "pl1", Path: "plan.md"},
		},
	}

	h1 := sha256.New()
	require.NoError(t, hashManifestFiles(h1, withState, dir))
	h2 := sha256.New()
	require.NoError(t, hashManifestFiles(h2, withoutState, dir))

	assert.NotEqual(t, fmt.Sprintf("%x", h1.Sum(nil)), fmt.Sprintf("%x", h2.Sum(nil)),
		"plan state files should change the hash")
}

func TestHashManifestFiles_applicationSpecChangeDetected(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.md"), []byte("app"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spec.md"), []byte("spec v1"), 0o644))

	m := &manifest.Manifest{
		Application: []manifest.ApplicationEntry{
			{ID: "a1", Path: "app.md", Spec: "spec.md"},
		},
	}

	h1 := sha256.New()
	require.NoError(t, hashManifestFiles(h1, m, dir))
	hash1 := fmt.Sprintf("%x", h1.Sum(nil))

	// Modify spec content.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spec.md"), []byte("spec v2"), 0o644))

	h2 := sha256.New()
	require.NoError(t, hashManifestFiles(h2, m, dir))
	hash2 := fmt.Sprintf("%x", h2.Sum(nil))

	assert.NotEqual(t, hash1, hash2,
		"changing an application spec file should produce a different hash")
}
