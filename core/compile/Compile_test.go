package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/lock"
	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// setupTestProject creates a minimal project structure in a temp directory,
// changes to it (so lock file writes succeed), and returns the project root and config.
func setupTestProject(t *testing.T) (string, *config.Config) {
	t.Helper()
	dir := t.TempDir()

	// Compile writes codectx.lock to cwd; chdir to temp dir.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
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
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

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
	assert.Equal(t, 0, result.ObjectsStored)
	assert.Equal(t, 0, result.Packages)

	// Verify compiled manifest exists.
	_, err = os.Stat(filepath.Join(result.OutputDir, "manifest.yml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(result.OutputDir, "README.md"))
	assert.NoError(t, err)

	// Verify compiled manifest is loadable.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	assert.Equal(t, "test-project", cm.Name)

	// Verify heuristics.yml was generated.
	h, err := LoadHeuristics(filepath.Join(result.OutputDir, "heuristics.yml"))
	require.NoError(t, err)
	assert.Equal(t, 0, h.Totals.Entries)
	assert.Equal(t, 0, h.Totals.Objects)
	assert.NotEmpty(t, h.CompiledAt)
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
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	assert.Equal(t, 1, result.ObjectsStored)

	// Verify the file was stored as a content-addressed object.
	hash := ContentHash([]byte("# Philosophy\n"))
	objectPath := filepath.Join(result.OutputDir, "objects", hash+".md")
	data, err := os.ReadFile(objectPath)
	require.NoError(t, err)
	assert.Equal(t, "# Philosophy\n", string(data))

	// Verify the compiled manifest references the object.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Foundation, 1)
	assert.Equal(t, ObjectPath(hash), cm.Foundation[0].Object)
	assert.Equal(t, "local", cm.Foundation[0].Source)
	assert.Equal(t, "always", cm.Foundation[0].Load)

	// Verify heuristics.yml has correct stats.
	h, err := LoadHeuristics(filepath.Join(result.OutputDir, "heuristics.yml"))
	require.NoError(t, err)
	assert.Equal(t, 1, h.Totals.Entries)
	assert.Equal(t, 1, h.Totals.AlwaysLoad)
	assert.Greater(t, h.Totals.SizeBytes, 0)
	assert.Greater(t, h.Totals.EstimatedTokens, 0)
	require.NotNil(t, h.Sections.Foundation)
	assert.Equal(t, 1, h.Sections.Foundation.Entries)
	require.Len(t, h.Packages, 1)
	assert.Equal(t, "local", h.Packages[0].Name)
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
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

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

func TestCompile_preservesPreferencesYML(t *testing.T) {
	_, cfg := setupTestProject(t)
	outputDir := cfg.OutputDir()

	// Create preferences.yml in the output directory.
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	prefsPath := filepath.Join(outputDir, "preferences.yml")
	require.NoError(t, os.WriteFile(prefsPath, []byte("auto_compile: true\n"), 0o644))

	_, err := Compile(cfg)
	require.NoError(t, err)

	// preferences.yml should survive the compile.
	data, err := os.ReadFile(prefsPath)
	require.NoError(t, err)
	assert.Equal(t, "auto_compile: true\n", string(data))
}

func TestCompile_writesLockFile(t *testing.T) {
	dir, cfg := setupTestProject(t)

	_, err := Compile(cfg)
	require.NoError(t, err)

	// Verify lock file was created in the project dir (cwd set by setupTestProject).
	_, err = os.Stat(filepath.Join(dir, "codectx.lock"))
	assert.NoError(t, err)
}

func TestCompile_dedupsSameContent(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Write a foundation doc to the local package.
	sharedContent := []byte("# Shared Philosophy\nThis is shared across packages.\n")
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "philosophy.md"), sharedContent, 0o644))

	localManifest := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Shared philosophy", Load: "always"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), localManifest))

	// Create installed package with same foundation entry and same content.
	pkgDir := filepath.Join(docsDir, "packages", "react@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "foundation", "philosophy.md"), sharedContent, 0o644))

	pkgManifest := &manifest.Manifest{
		Name:    "react",
		Author:  "org",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Shared philosophy"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	cfg.Packages = []config.PackageDep{
		{Name: "react", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	// Should report 1 duplicate, 0 conflicts.
	assert.Len(t, result.Dedup.Duplicates, 1)
	assert.Empty(t, result.Dedup.Conflicts)
	assert.Equal(t, "philosophy", result.Dedup.Duplicates[0].ID)
	assert.Equal(t, "duplicate", result.Dedup.Duplicates[0].Reason)
}

func TestCompile_conflictDifferentContent(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Local has version A of the doc.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "conventions.md"),
		[]byte("# Local Conventions\n"), 0o644))

	localManifest := &manifest.Manifest{
		Name:    "test-project",
		Author:  "tester",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "conventions", Path: "foundation/conventions.md", Description: "Local conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), localManifest))

	// Installed package has version B of the doc (different content).
	pkgDir := filepath.Join(docsDir, "packages", "go@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "foundation", "conventions.md"),
		[]byte("# Go Conventions\n"), 0o644))

	pkgManifest := &manifest.Manifest{
		Name:    "go",
		Author:  "org",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "conventions", Path: "foundation/conventions.md", Description: "Go conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	cfg.Packages = []config.PackageDep{
		{Name: "go", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	// Should report 1 conflict, 0 duplicates.
	assert.Empty(t, result.Dedup.Duplicates)
	require.Len(t, result.Dedup.Conflicts, 1)
	assert.Equal(t, "conventions", result.Dedup.Conflicts[0].ID)
	assert.Equal(t, "conflict", result.Dedup.Conflicts[0].Reason)
	assert.Equal(t, "local", result.Dedup.Conflicts[0].WinnerPkg)
	assert.Equal(t, "go@org", result.Dedup.Conflicts[0].SkippedPkg)
}

func TestCompile_inactivePackageInLockFile(t *testing.T) {
	dir, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package with a foundation entry.
	pkgDir := filepath.Join(docsDir, "packages", "inert@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "foundation", "guide.md"),
		[]byte("# Inert Guide\n"), 0o644))

	pkgManifest := &manifest.Manifest{
		Name:    "inert",
		Author:  "org",
		Version: "2.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "guide", Path: "foundation/guide.md", Description: "Inert guide"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	// Add as inactive package (mode: "none").
	cfg.Packages = []config.PackageDep{
		{Name: "inert", Author: "org", Version: "^2.0.0", Active: config.Activation{Mode: "none"}},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	// Inactive packages don't count toward processed packages.
	assert.Equal(t, 0, result.Packages)

	// But the lock file should still contain the package.
	lockPath := filepath.Join(dir, "codectx.lock")
	data, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "inert")
}

func TestCompile_multiplePackages(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create installed package A with a topic entry.
	pkgADir := filepath.Join(docsDir, "packages", "pkgA@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgADir, "topics"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgADir, "topics", "react.md"),
		[]byte("# React Conventions\n"), 0o644))

	pkgAManifest := &manifest.Manifest{
		Name:    "pkgA",
		Author:  "org",
		Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react.md", Description: "React conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgADir, "manifest.yml"), pkgAManifest))

	// Create installed package B with a foundation entry.
	pkgBDir := filepath.Join(docsDir, "packages", "pkgB@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgBDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgBDir, "foundation", "conventions.md"),
		[]byte("# Conventions\n"), 0o644))

	pkgBManifest := &manifest.Manifest{
		Name:    "pkgB",
		Author:  "org",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "conventions", Path: "foundation/conventions.md", Description: "Conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgBDir, "manifest.yml"), pkgBManifest))

	cfg.Packages = []config.PackageDep{
		{Name: "pkgA", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
		{Name: "pkgB", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	assert.Equal(t, 2, result.Packages)
	// 2 objects: pkgA topic + pkgB foundation.
	assert.Equal(t, 2, result.ObjectsStored)

	// Verify the compiled manifest contains entries from both packages.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)

	assert.Len(t, cm.Topics, 1)
	assert.Equal(t, "react", cm.Topics[0].ID)
	assert.Equal(t, "pkgA@org", cm.Topics[0].Source)
	assert.Len(t, cm.Foundation, 1)
	assert.Equal(t, "conventions", cm.Foundation[0].ID)
	assert.Equal(t, "pkgB@org", cm.Foundation[0].Source)
}

func TestCompile_lockFileContent(t *testing.T) {
	dir, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package with explicit source.
	pkgDir := filepath.Join(docsDir, "packages", "mypkg@myauthor")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "foundation", "guide.md"),
		[]byte("# Guide\n"), 0o644))

	pkgManifest := &manifest.Manifest{
		Name:    "mypkg",
		Author:  "myauthor",
		Version: "3.2.1",
		Foundation: []manifest.FoundationEntry{
			{ID: "guide", Path: "foundation/guide.md", Description: "A guide"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	cfg.Packages = []config.PackageDep{
		{
			Name:    "mypkg",
			Author:  "myauthor",
			Version: "^3.0.0",
			Source:  "https://example.com/mypkg",
			Active:  config.Activation{Mode: "all"},
		},
	}

	_, err := Compile(cfg)
	require.NoError(t, err)

	// Read back the lock file and unmarshal.
	lockPath := filepath.Join(dir, "codectx.lock")
	data, err := os.ReadFile(lockPath)
	require.NoError(t, err)

	var lck lock.Lock
	require.NoError(t, yaml.Unmarshal(data, &lck))

	// Verify compiled_at is set (today's date).
	assert.NotEmpty(t, lck.CompiledAt)

	// Verify package fields round-trip correctly.
	require.Len(t, lck.Packages, 1)
	pkg := lck.Packages[0]
	assert.Equal(t, "mypkg", pkg.Name)
	assert.Equal(t, "myauthor", pkg.Author)
	assert.Equal(t, "3.2.1", pkg.Version)
	assert.Equal(t, "https://example.com/mypkg", pkg.Source)
	assert.True(t, pkg.Active.IsAll())
}

func TestCompile_failsWhenLocalManifestInvalid(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Overwrite the local manifest with corrupt YAML.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "manifest.yml"),
		[]byte("{{{{not valid yaml"),
		0o644,
	))

	_, err := Compile(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load local manifest")
}

func TestCompile_corruptPackageManifestFails(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package directory with a corrupt manifest.
	// This is the fingerprint gap: computeFingerprint silently skips
	// corrupt manifests (line 67), but Compile itself catches it (line 105).
	pkgDir := filepath.Join(docsDir, "packages", "corrupt@org")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "manifest.yml"),
		[]byte("{{{{not valid yaml at all"),
		0o644,
	))

	cfg.Packages = []config.PackageDep{
		{Name: "corrupt", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
	}

	// Write codectx.yml so fingerprint can read it.
	require.NoError(t, config.Write("codectx.yml", cfg))

	_, err := Compile(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "corrupt@org")
}

func TestCompile_fingerprintChangesOnContentModification(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Add a foundation document.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "doc.md"),
		[]byte("# Original\n"), 0o644))

	m := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "doc", Path: "foundation/doc.md", Description: "A document"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))
	require.NoError(t, config.Write("codectx.yml", cfg))

	// First compile.
	result1, err := Compile(cfg)
	require.NoError(t, err)
	assert.False(t, result1.UpToDate)

	// Second compile without changes: should be up-to-date.
	result2, err := Compile(cfg)
	require.NoError(t, err)
	assert.True(t, result2.UpToDate)

	// Modify the file content.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "doc.md"),
		[]byte("# Modified\n"), 0o644))

	// Third compile: should detect change via fingerprint.
	result3, err := Compile(cfg)
	require.NoError(t, err)
	assert.False(t, result3.UpToDate)
}

func TestCompile_failsWhenInstalledManifestInvalid(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package directory with a corrupt manifest.
	pkgDir := filepath.Join(docsDir, "packages", "broken@org")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "manifest.yml"),
		[]byte("{{{{not valid yaml"),
		0o644,
	))

	cfg.Packages = []config.PackageDep{
		{Name: "broken", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
	}

	_, err := Compile(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken@org")
}

func TestCompile_granularActivation(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package with multiple entries across sections.
	pkgDir := filepath.Join(docsDir, "packages", "multi@org")
	for _, sub := range []string{"foundation", "topics/react", "topics/go"} {
		require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, sub), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "foundation", "philosophy.md"), []byte("# Philosophy\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "topics", "react", "README.md"), []byte("# React\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "topics", "go", "README.md"), []byte("# Go\n"), 0o644))

	pkgManifest := &manifest.Manifest{
		Name:    "multi",
		Author:  "org",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Philosophy"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React"},
			{ID: "go", Path: "topics/go/README.md", Description: "Go"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	// Only activate foundation:philosophy and topics:react (not topics:go).
	cfg.Packages = []config.PackageDep{
		{
			Name:    "multi",
			Author:  "org",
			Version: "^1.0.0",
			Active: config.Activation{
				Map: &config.ActivationMap{
					Foundation: []string{"philosophy"},
					Topics:     []string{"react"},
				},
			},
		},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Packages)
	// 2 objects: foundation/philosophy.md + topics/react/README.md (go excluded).
	assert.Equal(t, 2, result.ObjectsStored)

	// Verify the compiled manifest only has the activated entries.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)

	require.Len(t, cm.Foundation, 1)
	assert.Equal(t, "philosophy", cm.Foundation[0].ID)
	require.Len(t, cm.Topics, 1)
	assert.Equal(t, "react", cm.Topics[0].ID)

	// Verify the excluded topic was not stored as an object.
	goHash := ContentHash([]byte("# Go\n"))
	_, err = os.Stat(filepath.Join(result.OutputDir, "objects", goHash+".md"))
	assert.True(t, os.IsNotExist(err))
}

func TestCompile_recompileReplacesOldObjects(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// First compile with a foundation entry.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "original.md"),
		[]byte("# Original\n"), 0o644))

	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "original", Path: "foundation/original.md", Description: "Original"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result1, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, result1.ObjectsStored)

	originalHash := ContentHash([]byte("# Original\n"))
	replacementHash := ContentHash([]byte("# Replacement\n"))

	// Verify original object exists.
	_, err = os.Stat(filepath.Join(result1.OutputDir, "objects", originalHash+".md"))
	require.NoError(t, err)

	// Second compile: remove the old file and add a different one.
	require.NoError(t, os.Remove(filepath.Join(docsDir, "foundation", "original.md")))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "replacement.md"),
		[]byte("# Replacement\n"), 0o644))

	m2 := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "replacement", Path: "foundation/replacement.md", Description: "Replacement"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m2))

	result2, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, result2.ObjectsStored)

	// Original object should be gone (cleaned by selective wipe).
	_, err = os.Stat(filepath.Join(result2.OutputDir, "objects", originalHash+".md"))
	assert.True(t, os.IsNotExist(err))

	// Replacement object should exist.
	_, err = os.Stat(filepath.Join(result2.OutputDir, "objects", replacementHash+".md"))
	assert.NoError(t, err)
}

func TestCompile_contentAddressedDedup(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Two entries referencing files with identical content.
	content := []byte("# Shared Content\n")
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "a.md"), content, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "b.md"), content, 0o644))

	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md", Description: "A"},
			{ID: "b", Path: "foundation/b.md", Description: "B"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	// Two entries stored, but ObjectStore deduplicates at the file level.
	assert.Equal(t, 2, result.ObjectsStored)

	// Both entries in the compiled manifest reference the same object.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Foundation, 2)
	assert.Equal(t, cm.Foundation[0].Object, cm.Foundation[1].Object)

	// Only one physical object file exists.
	entries, err := os.ReadDir(filepath.Join(result.OutputDir, "objects"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestCompile_topicWithSpecAndFiles(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a topic with spec and extra files.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "go", "spec"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "go", "README.md"),
		[]byte("# Go Conventions\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "go", "spec", "README.md"),
		[]byte("# Go Spec\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "go", "extra.md"),
		[]byte("# Extra Go Notes\n"), 0o644))

	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Topics: []manifest.TopicEntry{
			{
				ID:          "go",
				Path:        "topics/go/README.md",
				Description: "Go conventions",
				Spec:        "topics/go/spec/README.md",
				Files:       []string{"topics/go/extra.md"},
			},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	// 3 objects: README.md, spec/README.md, extra.md.
	assert.Equal(t, 3, result.ObjectsStored)

	// Verify compiled manifest has spec and files references.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Topics, 1)
	assert.Equal(t, "go", cm.Topics[0].ID)
	assert.NotEmpty(t, cm.Topics[0].Object)
	assert.NotEmpty(t, cm.Topics[0].Spec)
	assert.Contains(t, cm.Topics[0].Spec, "objects/")
	require.Len(t, cm.Topics[0].Files, 1)
	assert.Contains(t, cm.Topics[0].Files[0], "objects/")

	// Verify all three objects exist on disk.
	objectEntries, err := os.ReadDir(filepath.Join(result.OutputDir, "objects"))
	require.NoError(t, err)
	assert.Len(t, objectEntries, 3)

	// Verify heuristics counts spec and files sizes.
	h, err := LoadHeuristics(filepath.Join(result.OutputDir, "heuristics.yml"))
	require.NoError(t, err)
	require.NotNil(t, h.Sections.Topics)
	assert.Equal(t, 1, h.Sections.Topics.Entries)
	// Size should include main path + spec + extra file.
	mainSize := len("# Go Conventions\n")
	specSize := len("# Go Spec\n")
	extraSize := len("# Extra Go Notes\n")
	assert.Equal(t, mainSize+specSize+extraSize, h.Sections.Topics.SizeBytes)
}

func TestCompile_planWithStateFile(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a plan entry with a state file.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "plans", "migrate"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "plans", "migrate", "README.md"),
		[]byte("# Migration Plan\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "plans", "migrate", "state.yml"),
		[]byte("phase: planning\ntasks:\n  - name: setup\n    done: false\n"), 0o644))

	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Plans: []manifest.PlanEntry{
			{
				ID:          "migrate",
				Path:        "plans/migrate/README.md",
				State:       "plans/migrate/state.yml",
				Description: "Database migration plan",
			},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	// Only the plan README should be stored as an object (state is mutable).
	assert.Equal(t, 1, result.ObjectsStored)

	// Verify the state file was copied to state/migrate.yml.
	stateDir := filepath.Join(result.OutputDir, "state")
	statePath := filepath.Join(stateDir, "migrate.yml")
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "phase: planning")

	// Verify compiled manifest references state path.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Plans, 1)
	assert.Equal(t, "state/migrate.yml", cm.Plans[0].State)
	assert.Equal(t, "local", cm.Plans[0].Source)
}

func TestCompile_fingerprintSkipWithPackages(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package with a foundation entry.
	pkgDir := filepath.Join(docsDir, "packages", "react@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "foundation", "guide.md"),
		[]byte("# React Guide\n"), 0o644))

	pkgManifest := &manifest.Manifest{
		Name:    "react",
		Author:  "org",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "guide", Path: "foundation/guide.md", Description: "React guide"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	cfg.Packages = []config.PackageDep{
		{Name: "react", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
	}

	// Write codectx.yml so fingerprint can read it.
	require.NoError(t, config.Write("codectx.yml", cfg))

	// First compile.
	result1, err := Compile(cfg)
	require.NoError(t, err)
	assert.False(t, result1.UpToDate)
	assert.Equal(t, 1, result1.Packages)

	// Second compile without changes should be up-to-date.
	result2, err := Compile(cfg)
	require.NoError(t, err)
	assert.True(t, result2.UpToDate)

	// Modify a package file and recompile — should not be up-to-date.
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "foundation", "guide.md"),
		[]byte("# React Guide (updated)\n"), 0o644))

	result3, err := Compile(cfg)
	require.NoError(t, err)
	assert.False(t, result3.UpToDate)
	assert.Equal(t, 1, result3.Packages)
}

func TestCompile_decompositionTriggered(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Generate enough content to exceed the 50KB byte threshold.
	// Each entry: ~1KB content. Need >50 entries to exceed 50KB.
	var foundationEntries []manifest.FoundationEntry
	for i := range 60 {
		id := fmt.Sprintf("doc-%03d", i)
		path := fmt.Sprintf("foundation/%s.md", id)
		// Write ~1KB file.
		content := fmt.Sprintf("# Document %03d\n%s\n", i, string(make([]byte, 900)))
		require.NoError(t, os.WriteFile(
			filepath.Join(docsDir, path), []byte(content), 0o644))
		foundationEntries = append(foundationEntries, manifest.FoundationEntry{
			ID:          id,
			Path:        path,
			Description: fmt.Sprintf("Document %d", i),
		})
	}

	// Mark first entry as always-load.
	foundationEntries[0].Load = "always"

	m := &manifest.Manifest{
		Name:       "test-project",
		Version:    "1.0.0",
		Foundation: foundationEntries,
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 60, result.ObjectsStored)

	// Verify decomposition happened: manifests/ directory should exist.
	manifestsDir := filepath.Join(result.OutputDir, "manifests")
	_, err = os.Stat(manifestsDir)
	require.NoError(t, err)

	// Load root manifest: should have always-load entry inlined
	// and a manifests reference for foundation.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)

	// Only the always-load entry should be inlined in root.
	require.Len(t, cm.Foundation, 1)
	assert.Equal(t, "doc-000", cm.Foundation[0].ID)
	assert.Equal(t, "always", cm.Foundation[0].Load)

	// Should have a manifests reference for foundation.
	require.NotEmpty(t, cm.Manifests)
	foundRef := false
	for _, ref := range cm.Manifests {
		if ref.Section == "foundation" {
			foundRef = true
			assert.Equal(t, "manifests/foundation.yml", ref.Path)
			assert.Equal(t, 59, ref.Entries) // 60 - 1 always-load = 59
		}
	}
	assert.True(t, foundRef, "should have foundation manifest reference")

	// Verify the sub-manifest file exists and is loadable.
	subCm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifests", "foundation.yml"))
	require.NoError(t, err)
	assert.Len(t, subCm.Foundation, 59)
}

func TestCompile_planStateMissing(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a plan entry where state is declared but the file is missing.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "plans", "future"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "plans", "future", "README.md"),
		[]byte("# Future Plan\n"), 0o644))

	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Plans: []manifest.PlanEntry{
			{
				ID:          "future",
				Path:        "plans/future/README.md",
				State:       "plans/future/state.yml", // does not exist on disk
				Description: "A future plan",
			},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Should succeed: missing state files are silently skipped.
	result, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ObjectsStored)

	// State directory should not exist (no state files copied).
	_, err = os.Stat(filepath.Join(result.OutputDir, "state"))
	assert.True(t, os.IsNotExist(err))
}

// --- Discovery integration tests ---

func TestCompile_discoversLocalEntries(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create files on disk but do NOT declare them in the manifest.
	// The manifest remains empty (metadata only) — exactly the bug scenario.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy.md"),
		[]byte("# Philosophy\n\nGuiding principles.\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "react", "README.md"),
		[]byte("# React\n\nComponent conventions.\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "react", "hooks.md"),
		[]byte("# Hooks\n\nHook patterns.\n"), 0o644))

	result, err := Compile(cfg)
	require.NoError(t, err)

	// Discovery should find the foundation doc and the topic.
	// 3 objects: philosophy.md, react/README.md, react/hooks.md
	assert.Equal(t, 3, result.ObjectsStored)

	// Verify the compiled manifest has the discovered entries.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Foundation, 1)
	assert.Equal(t, "philosophy", cm.Foundation[0].ID)
	assert.Equal(t, "local", cm.Foundation[0].Source)
	require.Len(t, cm.Topics, 1)
	assert.Equal(t, "react", cm.Topics[0].ID)
	require.Len(t, cm.Topics[0].Files, 1)

	// Verify heuristics reflect the discovered entries.
	h, err := LoadHeuristics(filepath.Join(result.OutputDir, "heuristics.yml"))
	require.NoError(t, err)
	assert.Equal(t, 2, h.Totals.Entries) // 1 foundation + 1 topic
	assert.Equal(t, 3, h.Totals.Objects) // 3 unique files
}

func TestCompile_discoversInstalledPackageEntries(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package with files on disk but an empty manifest.
	// This is THE critical bug scenario: manifest.yml has only metadata.
	pkgDir := filepath.Join(docsDir, "packages", "react@securacore")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "topics", "react"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "topics", "react", "README.md"),
		[]byte("# React\n\nComponent patterns.\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "topics", "react", "hooks.md"),
		[]byte("# Hooks\n\nHook patterns.\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "topics", "react", "state.md"),
		[]byte("# State Management\n\nState patterns.\n"), 0o644))

	// Write an empty manifest (metadata only — no entry arrays).
	emptyPkgManifest := &manifest.Manifest{
		Name:        "codectx-react",
		Author:      "securacore",
		Version:     "0.0.3",
		Description: "Documentation package for codectx-react",
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), emptyPkgManifest))

	cfg.Packages = []config.PackageDep{
		{Name: "react", Author: "securacore", Version: "^0.0.3", Active: config.Activation{Mode: "all"}},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Packages)
	// 3 objects: README.md + hooks.md + state.md
	assert.Equal(t, 3, result.ObjectsStored)

	// Verify the compiled manifest has the discovered topic.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Topics, 1)
	assert.Equal(t, "react", cm.Topics[0].ID)
	assert.Equal(t, "react@securacore", cm.Topics[0].Source)
	assert.NotEmpty(t, cm.Topics[0].Object)
	require.Len(t, cm.Topics[0].Files, 2) // hooks.md + state.md

	// Verify heuristics reflect the discovered entries.
	h, err := LoadHeuristics(filepath.Join(result.OutputDir, "heuristics.yml"))
	require.NoError(t, err)
	assert.Equal(t, 1, h.Totals.Entries)
	assert.Equal(t, 3, h.Totals.Objects)
	// Only react@securacore appears (local project has no entries).
	require.Len(t, h.Packages, 1)
	assert.Equal(t, "react@securacore", h.Packages[0].Name)
}

func TestCompile_discoveryWithGranularActivation(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create an installed package with files but empty manifest.
	pkgDir := filepath.Join(docsDir, "packages", "docs@org")
	for _, topicName := range []string{"react", "go", "typescript"} {
		require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "topics", topicName), 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(pkgDir, "topics", topicName, "README.md"),
			[]byte(fmt.Sprintf("# %s\n\nConventions.\n", topicName)), 0o644))
	}

	emptyPkgManifest := &manifest.Manifest{
		Name:        "docs",
		Author:      "org",
		Version:     "1.0.0",
		Description: "Documentation package",
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), emptyPkgManifest))

	// Only activate the react and go topics (not typescript).
	cfg.Packages = []config.PackageDep{
		{
			Name:    "docs",
			Author:  "org",
			Version: "^1.0.0",
			Active: config.Activation{
				Map: &config.ActivationMap{
					Topics: []string{"react", "go"},
				},
			},
		},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Packages)
	// Only 2 objects: react/README.md + go/README.md (typescript excluded).
	assert.Equal(t, 2, result.ObjectsStored)

	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)

	require.Len(t, cm.Topics, 2)
	topicIDs := []string{cm.Topics[0].ID, cm.Topics[1].ID}
	assert.Contains(t, topicIDs, "react")
	assert.Contains(t, topicIDs, "go")
}

// --- Sync integration tests ---
// These tests verify that Compile's internal Sync() call (line 45) correctly
// discovers entries, removes stale entries, infers relationships, and writes
// the synced manifest back to disk (line 48).

func TestCompile_syncRemovesStaleLocalEntries(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create a foundation file on disk.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "alive.md"),
		[]byte("# Alive\n"), 0o644))

	// Write a manifest that declares two entries: "alive" (file exists) and "stale" (file deleted).
	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "alive", Path: "foundation/alive.md", Description: "Still here"},
			{ID: "stale", Path: "foundation/stale.md", Description: "File was deleted"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)

	// Only the alive entry should appear in compiled output.
	assert.Equal(t, 1, result.ObjectsStored)
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Foundation, 1)
	assert.Equal(t, "alive", cm.Foundation[0].ID)

	// Verify the source manifest was written back WITHOUT the stale entry.
	reloaded, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, reloaded.Foundation, 1)
	assert.Equal(t, "alive", reloaded.Foundation[0].ID)
}

func TestCompile_writesBackSyncedManifest(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create files on disk but leave the manifest completely empty (metadata only).
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "discovered.md"),
		[]byte("# Discovered\n"), 0o644))

	_, err := Compile(cfg)
	require.NoError(t, err)

	// Read back the source manifest — it should now contain the discovered entry.
	reloaded, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, reloaded.Foundation, 1)
	assert.Equal(t, "discovered", reloaded.Foundation[0].ID)
	assert.Equal(t, "foundation/discovered.md", reloaded.Foundation[0].Path)
}

func TestCompile_syncInfersRelationshipsInCompiledOutput(t *testing.T) {
	_, cfg := setupTestProject(t)
	docsDir := cfg.DocsDir()

	// Create two foundation files that link to each other.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "alpha.md"),
		[]byte("# Alpha\nSee [beta](beta.md) for details.\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "beta.md"),
		[]byte("# Beta\nBuilds on [alpha](alpha.md).\n"), 0o644))

	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "alpha", Path: "foundation/alpha.md", Description: "Alpha doc"},
			{ID: "beta", Path: "foundation/beta.md", Description: "Beta doc"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	result, err := Compile(cfg)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ObjectsStored)

	// Verify the compiled manifest carries the inferred relationships.
	cm, err := LoadCompiledManifest(filepath.Join(result.OutputDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, cm.Foundation, 2)

	// Build a lookup for easier assertion.
	byID := map[string]CompiledFoundationEntry{}
	for _, e := range cm.Foundation {
		byID[e.ID] = e
	}

	// alpha links to beta → alpha depends_on beta, beta required_by alpha.
	assert.Contains(t, byID["alpha"].DependsOn, "beta", "alpha should depend on beta")
	assert.Contains(t, byID["beta"].RequiredBy, "alpha", "beta should be required by alpha")

	// beta links to alpha → beta depends_on alpha, alpha required_by beta.
	assert.Contains(t, byID["beta"].DependsOn, "alpha", "beta should depend on alpha")
	assert.Contains(t, byID["alpha"].RequiredBy, "beta", "alpha should be required by beta")

	// Verify the source manifest was also written back with relationships.
	reloaded, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	foundAlpha := false
	for _, e := range reloaded.Foundation {
		if e.ID == "alpha" {
			foundAlpha = true
			assert.Contains(t, e.DependsOn, "beta")
			assert.Contains(t, e.RequiredBy, "beta")
		}
	}
	assert.True(t, foundAlpha, "alpha entry should exist in reloaded manifest")
}

// --- storeObjects unit tests ---
// These tests exercise storeObjects directly to cover edge cases that are
// difficult to trigger through the full Compile integration path.

func TestStoreObjects_dedupByPath(t *testing.T) {
	// When two entries reference the same file path, the file should be
	// stored only once (dedup-by-path on line 247-249 of Compile.go).
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "foundation"), 0o755))

	sharedContent := []byte("# Shared\nShared content.\n")
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "foundation", "shared.md"), sharedContent, 0o644))

	store := NewObjectStore(filepath.Join(dir, "objects"))
	unified := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/shared.md"},
			{ID: "b", Path: "foundation/shared.md"}, // same path as "a"
		},
	}
	srcDirs := map[string]string{"local": srcDir}
	provenance := map[string]string{
		"foundation:a": "local",
		"foundation:b": "local",
	}

	pathToHash, stored, err := storeObjects(store, unified, srcDirs, provenance)
	require.NoError(t, err)

	// Only 1 object should be stored (same path deduped).
	assert.Equal(t, 1, stored)
	assert.Len(t, pathToHash, 1)
	assert.Equal(t, ContentHash(sharedContent), pathToHash["foundation/shared.md"])
}

func TestStoreObjects_emptyProvenanceKeySkipsFile(t *testing.T) {
	// When provenance has no entry for a section:id key, srcDir is "" and
	// the file is silently skipped (line 252-254 of Compile.go).
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "foundation", "orphan.md"), []byte("# Orphan\n"), 0o644))

	store := NewObjectStore(filepath.Join(dir, "objects"))
	unified := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "orphan", Path: "foundation/orphan.md"},
		},
	}
	srcDirs := map[string]string{"local": srcDir}
	// Provenance is empty — no mapping for "foundation:orphan".
	provenance := map[string]string{}

	pathToHash, stored, err := storeObjects(store, unified, srcDirs, provenance)
	require.NoError(t, err)

	// File should be skipped entirely.
	assert.Equal(t, 0, stored)
	assert.Empty(t, pathToHash)
}

func TestStoreObjects_skipsMissingFiles(t *testing.T) {
	// When a file referenced by an entry doesn't exist on disk, it should
	// be silently skipped (line 258-259 of Compile.go).
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "foundation"), 0o755))
	// Do NOT write the file to disk.

	store := NewObjectStore(filepath.Join(dir, "objects"))
	unified := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "missing", Path: "foundation/missing.md"},
		},
	}
	srcDirs := map[string]string{"local": srcDir}
	provenance := map[string]string{"foundation:missing": "local"}

	pathToHash, stored, err := storeObjects(store, unified, srcDirs, provenance)
	require.NoError(t, err)

	assert.Equal(t, 0, stored)
	assert.Empty(t, pathToHash)
}

func TestStoreObjects_readErrorNonNotExist(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denial as root")
	}
	// When a file exists but is unreadable, storeObjects should return an error
	// (line 261 of Compile.go).
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "foundation"), 0o755))

	unreadable := filepath.Join(srcDir, "foundation", "secret.md")
	require.NoError(t, os.WriteFile(unreadable, []byte("# Secret\n"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o644) })

	store := NewObjectStore(filepath.Join(dir, "objects"))
	unified := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "secret", Path: "foundation/secret.md"},
		},
	}
	srcDirs := map[string]string{"local": srcDir}
	provenance := map[string]string{"foundation:secret": "local"}

	_, _, err := storeObjects(store, unified, srcDirs, provenance)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestStoreObjects_storeAsFailureDuringPass2(t *testing.T) {
	// When StoreAs fails during pass 2 (line 320-321), storeObjects should
	// return the error with the count of files stored before the failure.
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "foundation", "doc.md"), []byte("# Doc\n"), 0o644))

	// Create a blocker file where the objects directory needs to be.
	objPath := filepath.Join(dir, "objects")
	require.NoError(t, os.WriteFile(objPath, []byte("not a dir"), 0o644))

	store := NewObjectStore(objPath)
	unified := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "doc", Path: "foundation/doc.md"},
		},
	}
	srcDirs := map[string]string{"local": srcDir}
	provenance := map[string]string{"foundation:doc": "local"}

	_, stored, err := storeObjects(store, unified, srcDirs, provenance)
	require.Error(t, err)
	assert.Equal(t, 0, stored)
	assert.Contains(t, err.Error(), "create objects directory")
}

func TestStoreObjects_applicationSpecAndFiles(t *testing.T) {
	// Verify storeObjects correctly collects Spec and Files from application entries.
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "application", "arch", "spec"), 0o755))

	mainContent := []byte("# Architecture\n")
	specContent := []byte("# Spec\n")
	fileContent := []byte("# Details\n")

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "application", "arch", "README.md"), mainContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "application", "arch", "spec", "README.md"), specContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "application", "arch", "details.md"), fileContent, 0o644))

	store := NewObjectStore(filepath.Join(dir, "objects"))
	unified := &manifest.Manifest{
		Application: []manifest.ApplicationEntry{
			{
				ID:    "arch",
				Path:  "application/arch/README.md",
				Spec:  "application/arch/spec/README.md",
				Files: []string{"application/arch/details.md"},
			},
		},
	}
	srcDirs := map[string]string{"local": srcDir}
	provenance := map[string]string{"application:arch": "local"}

	pathToHash, stored, err := storeObjects(store, unified, srcDirs, provenance)
	require.NoError(t, err)

	assert.Equal(t, 3, stored)
	assert.Len(t, pathToHash, 3)
	assert.Equal(t, ContentHash(mainContent), pathToHash["application/arch/README.md"])
	assert.Equal(t, ContentHash(specContent), pathToHash["application/arch/spec/README.md"])
	assert.Equal(t, ContentHash(fileContent), pathToHash["application/arch/details.md"])
}

func TestStoreObjects_emptyRelPathSkipped(t *testing.T) {
	// Entries with empty Path should be silently skipped (line 244-246).
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	store := NewObjectStore(filepath.Join(dir, "objects"))
	unified := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "empty", Path: ""},
		},
	}
	srcDirs := map[string]string{"local": srcDir}
	provenance := map[string]string{"foundation:empty": "local"}

	pathToHash, stored, err := storeObjects(store, unified, srcDirs, provenance)
	require.NoError(t, err)
	assert.Equal(t, 0, stored)
	assert.Empty(t, pathToHash)
}
