package compile

import (
	"os"
	"path/filepath"
	"testing"

	"securacore/codectx/core/config"
	"securacore/codectx/core/lock"
	"securacore/codectx/core/manifest"

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
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), localManifest))

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
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "package.yml"), pkgManifest))

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
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), localManifest))

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
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "package.yml"), pkgManifest))

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
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "package.yml"), pkgManifest))

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
	require.NoError(t, manifest.Write(filepath.Join(pkgADir, "package.yml"), pkgAManifest))

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
	require.NoError(t, manifest.Write(filepath.Join(pkgBDir, "package.yml"), pkgBManifest))

	cfg.Packages = []config.PackageDep{
		{Name: "pkgA", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
		{Name: "pkgB", Author: "org", Version: "^1.0.0", Active: config.Activation{Mode: "all"}},
	}

	result, err := Compile(cfg)
	require.NoError(t, err)

	assert.Equal(t, 2, result.Packages)
	// Local has 0 files, pkgA has 1 (topic), pkgB has 1 (foundation).
	assert.Equal(t, 2, result.FilesCopied)

	// Verify the unified manifest contains entries from both packages.
	unifiedPath := filepath.Join(result.OutputDir, "package.yml")
	unified, err := manifest.Load(unifiedPath)
	require.NoError(t, err)

	assert.Len(t, unified.Topics, 1)
	assert.Equal(t, "react", unified.Topics[0].ID)
	assert.Len(t, unified.Foundation, 1)
	assert.Equal(t, "conventions", unified.Foundation[0].ID)
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
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "package.yml"), pkgManifest))

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
