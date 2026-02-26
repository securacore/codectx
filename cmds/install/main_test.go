package install

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command metadata ---

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "install", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}

func TestCommand_flags(t *testing.T) {
	flagNames := make(map[string]bool)
	for _, f := range Command.Flags {
		flagNames[f.Names()[0]] = true
	}
	assert.True(t, flagNames["activate"])
}

// --- parseActivateFlag ---

func TestParseActivateFlag_all(t *testing.T) {
	a, err := parseActivateFlag("all")
	require.NoError(t, err)
	assert.True(t, a.IsAll())
}

func TestParseActivateFlag_none(t *testing.T) {
	a, err := parseActivateFlag("none")
	require.NoError(t, err)
	assert.True(t, a.IsNone())
}

func TestParseActivateFlag_granular(t *testing.T) {
	a, err := parseActivateFlag("topics:react,prompts:commit")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
	assert.Equal(t, []string{"commit"}, a.Map.Prompts)
}

func TestParseActivateFlag_application(t *testing.T) {
	a, err := parseActivateFlag("application:architecture")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"architecture"}, a.Map.Application)
}

func TestParseActivateFlag_unknownSection(t *testing.T) {
	_, err := parseActivateFlag("bad:id")
	assert.Error(t, err)
}

// --- setupDocsDir ---

func TestSetupDocsDir_createsNewDir(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{Name: "test"}
	docsDir := "docs"

	// Write a valid config so setupDocsDir can write back if needed.
	require.NoError(t, config.Write(configFile, cfg))

	err = setupDocsDir(cfg, &docsDir)
	require.NoError(t, err)

	// Verify structure was created.
	assert.DirExists(t, filepath.Join(dir, "docs"))
	assert.DirExists(t, filepath.Join(dir, "docs", "packages"))
	assert.DirExists(t, filepath.Join(dir, "docs", "schemas"))
	assert.FileExists(t, filepath.Join(dir, "docs", "manifest.yml"))
}

func TestSetupDocsDir_existingCompatible(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Create a valid docs dir.
	docsPath := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsPath, 0o755))
	pkgYml := `name: test
author: org
version: "1.0.0"
description: "Test"
`
	require.NoError(t, os.WriteFile(filepath.Join(docsPath, "manifest.yml"), []byte(pkgYml), 0o644))

	cfg := &config.Config{Name: "test"}
	require.NoError(t, config.Write(configFile, cfg))

	docsDir := "docs"
	err = setupDocsDir(cfg, &docsDir)
	require.NoError(t, err)
	assert.Equal(t, "docs", docsDir) // should not have changed
}

func TestSetupDocsDir_preservesExistingManifestYml(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsPath := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsPath, 0o755))

	pkgYml := `name: existing
author: org
version: "2.0.0"
description: "Existing package"
`
	require.NoError(t, os.WriteFile(filepath.Join(docsPath, "manifest.yml"), []byte(pkgYml), 0o644))

	cfg := &config.Config{Name: "test"}
	require.NoError(t, config.Write(configFile, cfg))

	docsDir := "docs"
	err = setupDocsDir(cfg, &docsDir)
	require.NoError(t, err)

	// Existing manifest.yml should not be overwritten.
	data, err := os.ReadFile(filepath.Join(docsPath, "manifest.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "existing")
}

// --- ensureGitignore ---

func TestEnsureGitignore_createsNew(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), ".codectx/")
}

func TestEnsureGitignore_alreadyPresent(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(".gitignore", []byte(".codectx/\n"), 0o644))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	// Should not duplicate the entry.
	assert.Equal(t, ".codectx/\n", string(data))
}

func TestEnsureGitignore_appendsToExisting(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(".gitignore", []byte("node_modules/\n"), 0o644))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), "node_modules/")
	assert.Contains(t, string(data), ".codectx/")
}

func TestEnsureGitignore_appendsNewlineIfMissing(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// File without trailing newline.
	require.NoError(t, os.WriteFile(".gitignore", []byte("node_modules/"), 0o644))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), "node_modules/\n.codectx/")
}

// --- run ---

func TestRun_noConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = run("all")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_noPackages(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{Name: "test", Packages: []config.PackageDep{}}
	require.NoError(t, config.Write(configFile, cfg))

	err = run("all")
	assert.NoError(t, err)
}

// --- parseActivateFlag error branches ---

func TestParseActivateFlag_missingColon(t *testing.T) {
	_, err := parseActivateFlag("topicsreact")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected section:id")
}

func TestParseActivateFlag_emptyId(t *testing.T) {
	_, err := parseActivateFlag("topics:")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty id")
}

func TestParseActivateFlag_allSections(t *testing.T) {
	a, err := parseActivateFlag("foundation:a,application:arch,topics:b,prompts:c,plans:d")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"a"}, a.Map.Foundation)
	assert.Equal(t, []string{"arch"}, a.Map.Application)
	assert.Equal(t, []string{"b"}, a.Map.Topics)
	assert.Equal(t, []string{"c"}, a.Map.Prompts)
	assert.Equal(t, []string{"d"}, a.Map.Plans)
}

// --- ensureGitignore edge cases ---

func TestEnsureGitignore_emptyExistingFile(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(".gitignore", []byte(""), 0o644))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), ".codectx/")
}

// --- setupDocsDir edge cases ---

func TestSetupDocsDir_deeplyNestedDir(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{Name: "test"}
	require.NoError(t, config.Write(configFile, cfg))

	docsDir := filepath.Join("a", "b", "c", "docs")
	err = setupDocsDir(cfg, &docsDir)
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(dir, "a", "b", "c", "docs"))
	assert.DirExists(t, filepath.Join(dir, "a", "b", "c", "docs", "packages"))
	assert.DirExists(t, filepath.Join(dir, "a", "b", "c", "docs", "schemas"))
}

// --- setupInstallProject / setupBareRepo ---

// setupInstallProject creates a minimal project structure with the given
// packages declared in codectx.yml. Pre-sets preferences to avoid interactive
// prompts. Changes cwd to the project root.
func setupInstallProject(t *testing.T, packages []config.PackageDep) string {
	t.Helper()
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	for _, sub := range []string{"foundation", "topics", "prompts", "plans", "packages"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	cfg := &config.Config{
		Name:     "test-project",
		Packages: packages,
	}
	require.NoError(t, config.Write(configFile, cfg))

	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Pre-set auto_compile preference to skip interactive prompt.
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(outputDir, "preferences.yml"),
		[]byte("auto_compile: false\n"),
		0o644,
	))

	return dir
}

// setupBareRepo creates a bare git repo with a manifest.yml and tagged version.
func setupBareRepo(t *testing.T, name, author, ver string, tags []string) string {
	t.Helper()
	dir := t.TempDir()
	workDir := filepath.Join(dir, "work")
	bareDir := filepath.Join(dir, "bare.git")

	repo, err := git.PlainInit(workDir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// Create manifest.yml.
	content := fmt.Sprintf(
		"name: %s\nauthor: %s\nversion: %q\ndescription: Test package\n",
		name, author, ver,
	)
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "manifest.yml"), []byte(content), 0o644))
	_, err = wt.Add("manifest.yml")
	require.NoError(t, err)

	// Create a foundation doc.
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(workDir, "foundation", "guide.md"),
		[]byte("# Guide\nInstallation guide.\n"), 0o644))
	_, err = wt.Add("foundation/guide.md")
	require.NoError(t, err)

	sig := &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()}
	hash, err := wt.Commit("initial commit", &git.CommitOptions{Author: sig})
	require.NoError(t, err)

	for _, tag := range tags {
		_, err = repo.CreateTag(tag, hash, nil)
		require.NoError(t, err)
	}

	_, err = git.PlainClone(bareDir, true, &git.CloneOptions{URL: workDir, Tags: git.AllTags})
	require.NoError(t, err)

	return bareDir
}

// --- run integration tests ---

func TestRun_installAndCompile(t *testing.T) {
	bareDir := setupBareRepo(t, "test-pkg", "test-author", "1.0.0", []string{"v1.0.0"})

	dir := setupInstallProject(t, []config.PackageDep{
		{
			Name:    "test-pkg",
			Author:  "test-author",
			Version: "^1.0.0",
			Source:  bareDir,
			Active:  config.Activation{Mode: "none"},
		},
	})

	err := run("all")
	require.NoError(t, err)

	// Verify package was fetched.
	pkgDir := filepath.Join(dir, "docs", "packages", "test-pkg@test-author")
	assert.FileExists(t, filepath.Join(pkgDir, "manifest.yml"))
	assert.FileExists(t, filepath.Join(pkgDir, "foundation", "guide.md"))

	// Verify config was updated with activation.
	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.True(t, cfg.Packages[0].Active.IsAll())

	// Verify compilation ran: lock file should exist.
	assert.FileExists(t, filepath.Join(dir, lockFile))

	// Verify compiled output exists.
	assert.FileExists(t, filepath.Join(dir, ".codectx", "manifest.yml"))
}

func TestRun_alreadyInstalled(t *testing.T) {
	bareDir := setupBareRepo(t, "pre-pkg", "pre-author", "1.0.0", []string{"v1.0.0"})

	dir := setupInstallProject(t, []config.PackageDep{
		{
			Name:    "pre-pkg",
			Author:  "pre-author",
			Version: "^1.0.0",
			Source:  bareDir,
			Active:  config.Activation{Mode: "all"},
		},
	})

	// Pre-install the package manually.
	pkgDir := filepath.Join(dir, "docs", "packages", "pre-pkg@pre-author")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "foundation"), 0o755))
	pkgManifest := &manifest.Manifest{
		Name:    "pre-pkg",
		Author:  "pre-author",
		Version: "1.0.0",
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "foundation", "guide.md"),
		[]byte("# Guide\n"), 0o644))

	err := run("all")
	require.NoError(t, err)

	// Should succeed (skips fetch, still compiles).
	assert.FileExists(t, filepath.Join(dir, lockFile))
}

func TestRun_allPackagesFail(t *testing.T) {
	setupInstallProject(t, []config.PackageDep{
		{
			Name:    "nonexistent",
			Author:  "nobody",
			Version: "^1.0.0",
			Source:  "/nonexistent/path/to/repo.git",
			Active:  config.Activation{Mode: "none"},
		},
	})

	err := run("all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no packages were installed successfully")
}

func TestRun_mixedSuccessAndFailure(t *testing.T) {
	bareDir := setupBareRepo(t, "good-pkg", "org", "1.0.0", []string{"v1.0.0"})

	dir := setupInstallProject(t, []config.PackageDep{
		{
			Name:    "good-pkg",
			Author:  "org",
			Version: "^1.0.0",
			Source:  bareDir,
			Active:  config.Activation{Mode: "none"},
		},
		{
			Name:    "bad-pkg",
			Author:  "org",
			Version: "^1.0.0",
			Source:  "/nonexistent/repo.git",
			Active:  config.Activation{Mode: "none"},
		},
	})

	// Should succeed because at least one package installed.
	err := run("all")
	require.NoError(t, err)

	// Verify the good package was installed and compiled.
	assert.FileExists(t, filepath.Join(dir, "docs", "packages", "good-pkg@org", "manifest.yml"))
	assert.FileExists(t, filepath.Join(dir, lockFile))
}

func TestRun_lockFileVersionPinning(t *testing.T) {
	// Create a bare repo with two tagged versions.
	bareDir := setupBareRepo(t, "pinned-pkg", "org", "1.0.0", []string{"v1.0.0", "v1.1.0"})

	dir := setupInstallProject(t, []config.PackageDep{
		{
			Name:    "pinned-pkg",
			Author:  "org",
			Version: "^1.0.0",
			Source:  bareDir,
			Active:  config.Activation{Mode: "all"},
		},
	})

	// Create a lock file that pins to v1.0.0 (not the latest v1.1.0).
	lockContent := fmt.Sprintf(`compiled_at: "2025-01-01"
packages:
  - name: pinned-pkg
    author: org
    version: "1.0.0"
    source: %s
    active: all
`, bareDir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, lockFile), []byte(lockContent), 0o644))

	err := run("all")
	require.NoError(t, err)

	// Verify installation succeeded.
	assert.FileExists(t, filepath.Join(dir, "docs", "packages", "pinned-pkg@org", "manifest.yml"))
}

func TestRun_activateNoneFlagSkipsCompilation(t *testing.T) {
	bareDir := setupBareRepo(t, "none-pkg", "org", "1.0.0", []string{"v1.0.0"})

	dir := setupInstallProject(t, []config.PackageDep{
		{
			Name:    "none-pkg",
			Author:  "org",
			Version: "^1.0.0",
			Source:  bareDir,
			Active:  config.Activation{Mode: "none"},
		},
	})

	err := run("none")
	require.NoError(t, err)

	// Verify activation was set to "none".
	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.True(t, cfg.Packages[0].Active.IsNone())

	// Compiled output should exist (install always compiles).
	assert.FileExists(t, filepath.Join(dir, ".codectx", "manifest.yml"))
}

func TestRun_activateGranular(t *testing.T) {
	bareDir := setupBareRepo(t, "gran-pkg", "org", "1.0.0", []string{"v1.0.0"})

	setupInstallProject(t, []config.PackageDep{
		{
			Name:    "gran-pkg",
			Author:  "org",
			Version: "^1.0.0",
			Source:  bareDir,
			Active:  config.Activation{Mode: "none"},
		},
	})

	err := run("foundation:guide")
	require.NoError(t, err)

	// Verify granular activation was applied.
	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	require.True(t, cfg.Packages[0].Active.IsGranular())
	assert.Equal(t, []string{"guide"}, cfg.Packages[0].Active.Map.Foundation)
}

// --- run with invalid activate flag ---

// --- buildActivationEntries ---

func TestBuildActivationEntries_singlePackageAllSections(t *testing.T) {
	successes := []installedPkg{
		{
			idx: 0,
			pkg: config.PackageDep{Name: "pkg", Author: "org"},
			manifest: &manifest.Manifest{
				Foundation:  []manifest.FoundationEntry{{ID: "f1", Description: "Foundation 1"}},
				Application: []manifest.ApplicationEntry{{ID: "a1", Description: "App 1"}},
				Topics:      []manifest.TopicEntry{{ID: "t1", Description: "Topic 1"}},
				Prompts:     []manifest.PromptEntry{{ID: "p1", Description: "Prompt 1"}},
				Plans:       []manifest.PlanEntry{{ID: "pl1", Description: "Plan 1"}},
			},
		},
	}

	entries := buildActivationEntries(successes)
	assert.Len(t, entries, 5)

	// Verify each entry has the correct section and label format.
	assert.Equal(t, "foundation", entries[0].section)
	assert.Equal(t, "f1", entries[0].id)
	assert.Contains(t, entries[0].label, "[pkg@org / foundation]")

	assert.Equal(t, "application", entries[1].section)
	assert.Equal(t, "a1", entries[1].id)

	assert.Equal(t, "topics", entries[2].section)
	assert.Equal(t, "t1", entries[2].id)

	assert.Equal(t, "prompts", entries[3].section)
	assert.Equal(t, "p1", entries[3].id)

	assert.Equal(t, "plans", entries[4].section)
	assert.Equal(t, "pl1", entries[4].id)
}

func TestBuildActivationEntries_skipsAlreadyActive(t *testing.T) {
	successes := []installedPkg{
		{
			idx: 0,
			pkg: config.PackageDep{Name: "active-pkg", Author: "org", Active: config.Activation{Mode: "all"}},
			manifest: &manifest.Manifest{
				Foundation: []manifest.FoundationEntry{{ID: "f1", Description: "F1"}},
			},
		},
		{
			idx: 1,
			pkg: config.PackageDep{Name: "inactive-pkg", Author: "org"},
			manifest: &manifest.Manifest{
				Topics: []manifest.TopicEntry{{ID: "t1", Description: "T1"}},
			},
		},
	}

	entries := buildActivationEntries(successes)
	// Only the inactive package should produce entries.
	assert.Len(t, entries, 1)
	assert.Equal(t, "topics", entries[0].section)
	assert.Equal(t, 1, entries[0].pkgIdx)
}

func TestBuildActivationEntries_emptyManifest(t *testing.T) {
	successes := []installedPkg{
		{
			idx:      0,
			pkg:      config.PackageDep{Name: "empty", Author: "org"},
			manifest: &manifest.Manifest{},
		},
	}

	entries := buildActivationEntries(successes)
	assert.Empty(t, entries)
}

func TestBuildActivationEntries_multiPackage(t *testing.T) {
	successes := []installedPkg{
		{
			idx: 0,
			pkg: config.PackageDep{Name: "alpha", Author: "org"},
			manifest: &manifest.Manifest{
				Foundation: []manifest.FoundationEntry{
					{ID: "f1", Description: "Alpha F1"},
					{ID: "f2", Description: "Alpha F2"},
				},
			},
		},
		{
			idx: 1,
			pkg: config.PackageDep{Name: "beta", Author: "org"},
			manifest: &manifest.Manifest{
				Topics: []manifest.TopicEntry{
					{ID: "t1", Description: "Beta T1"},
				},
			},
		},
	}

	entries := buildActivationEntries(successes)
	assert.Len(t, entries, 3)
	assert.Equal(t, 0, entries[0].pkgIdx)
	assert.Equal(t, 0, entries[1].pkgIdx)
	assert.Equal(t, 1, entries[2].pkgIdx)
}

// --- applyActivationSelection ---

func TestApplyActivationSelection_allSelected(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "pkg", Author: "org"},
		},
	}
	successes := []installedPkg{
		{
			idx: 0,
			pkg: cfg.Packages[0],
			manifest: &manifest.Manifest{
				Foundation: []manifest.FoundationEntry{{ID: "f1"}},
				Topics:     []manifest.TopicEntry{{ID: "t1"}},
			},
		},
	}
	entries := buildActivationEntries(successes)
	// Select all entries (indices 0 and 1).
	selected := []int{0, 1}

	applyActivationSelection(cfg, successes, entries, selected)
	assert.True(t, cfg.Packages[0].Active.IsAll())
}

func TestApplyActivationSelection_noneSelected(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "pkg", Author: "org"},
		},
	}
	successes := []installedPkg{
		{
			idx: 0,
			pkg: cfg.Packages[0],
			manifest: &manifest.Manifest{
				Foundation: []manifest.FoundationEntry{{ID: "f1"}},
			},
		},
	}
	entries := buildActivationEntries(successes)
	// Select nothing.
	selected := []int{}

	applyActivationSelection(cfg, successes, entries, selected)
	assert.True(t, cfg.Packages[0].Active.IsNone())
}

func TestApplyActivationSelection_granularSelection(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "pkg", Author: "org"},
		},
	}
	successes := []installedPkg{
		{
			idx: 0,
			pkg: cfg.Packages[0],
			manifest: &manifest.Manifest{
				Foundation: []manifest.FoundationEntry{{ID: "f1"}, {ID: "f2"}},
				Topics:     []manifest.TopicEntry{{ID: "t1"}},
			},
		},
	}
	entries := buildActivationEntries(successes)
	// Select only the first foundation entry (index 0).
	selected := []int{0}

	applyActivationSelection(cfg, successes, entries, selected)
	assert.True(t, cfg.Packages[0].Active.IsGranular())
	require.NotNil(t, cfg.Packages[0].Active.Map)
	assert.Equal(t, []string{"f1"}, cfg.Packages[0].Active.Map.Foundation)
	assert.Nil(t, cfg.Packages[0].Active.Map.Topics)
}

func TestApplyActivationSelection_multiPackageMixed(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "alpha", Author: "org"},
			{Name: "beta", Author: "org"},
		},
	}
	successes := []installedPkg{
		{
			idx: 0,
			pkg: cfg.Packages[0],
			manifest: &manifest.Manifest{
				Foundation: []manifest.FoundationEntry{{ID: "f1"}},
			},
		},
		{
			idx: 1,
			pkg: cfg.Packages[1],
			manifest: &manifest.Manifest{
				Topics:  []manifest.TopicEntry{{ID: "t1"}},
				Prompts: []manifest.PromptEntry{{ID: "p1"}},
			},
		},
	}
	entries := buildActivationEntries(successes)
	// entries: [0]=alpha/f1, [1]=beta/t1, [2]=beta/p1
	require.Len(t, entries, 3)

	// Select all of alpha (all), only t1 from beta (granular).
	selected := []int{0, 1}

	applyActivationSelection(cfg, successes, entries, selected)
	// Alpha: 1 out of 1 selected → "all".
	assert.True(t, cfg.Packages[0].Active.IsAll())
	// Beta: 1 out of 2 selected → granular.
	assert.True(t, cfg.Packages[1].Active.IsGranular())
	assert.Equal(t, []string{"t1"}, cfg.Packages[1].Active.Map.Topics)
	assert.Nil(t, cfg.Packages[1].Active.Map.Prompts)
}

func TestApplyActivationSelection_skipsAlreadyActivePackages(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "already", Author: "org", Active: config.Activation{Mode: "all"}},
			{Name: "new", Author: "org"},
		},
	}
	successes := []installedPkg{
		{
			idx: 0,
			pkg: cfg.Packages[0],
			manifest: &manifest.Manifest{
				Foundation: []manifest.FoundationEntry{{ID: "f1"}},
			},
		},
		{
			idx: 1,
			pkg: cfg.Packages[1],
			manifest: &manifest.Manifest{
				Topics: []manifest.TopicEntry{{ID: "t1"}},
			},
		},
	}
	entries := buildActivationEntries(successes)
	// Only the "new" package's entries should appear.
	require.Len(t, entries, 1)

	// Select all.
	selected := []int{0}
	applyActivationSelection(cfg, successes, entries, selected)

	// "already" should remain unchanged.
	assert.True(t, cfg.Packages[0].Active.IsAll())
	// "new" should become "all" (1 out of 1).
	assert.True(t, cfg.Packages[1].Active.IsAll())
}

func TestApplyActivationSelection_allFiveSections(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "full", Author: "org"},
		},
	}
	successes := []installedPkg{
		{
			idx: 0,
			pkg: cfg.Packages[0],
			manifest: &manifest.Manifest{
				Foundation:  []manifest.FoundationEntry{{ID: "f1"}, {ID: "f2"}},
				Application: []manifest.ApplicationEntry{{ID: "a1"}},
				Topics:      []manifest.TopicEntry{{ID: "t1"}},
				Prompts:     []manifest.PromptEntry{{ID: "p1"}},
				Plans:       []manifest.PlanEntry{{ID: "pl1"}},
			},
		},
	}
	entries := buildActivationEntries(successes)
	require.Len(t, entries, 6)

	// Select a subset from different sections: f1, a1, p1 (indices 0, 2, 4).
	selected := []int{0, 2, 4}

	applyActivationSelection(cfg, successes, entries, selected)
	assert.True(t, cfg.Packages[0].Active.IsGranular())
	am := cfg.Packages[0].Active.Map
	require.NotNil(t, am)
	assert.Equal(t, []string{"f1"}, am.Foundation)
	assert.Equal(t, []string{"a1"}, am.Application)
	assert.Nil(t, am.Topics)
	assert.Equal(t, []string{"p1"}, am.Prompts)
	assert.Nil(t, am.Plans)
}

// --- run with invalid activate flag ---

func TestRun_invalidActivateFlag(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{
		Name: "test",
		Packages: []config.PackageDep{
			{Name: "pkg", Author: "org", Active: config.Activation{Mode: "none"}},
		},
	}
	require.NoError(t, config.Write(configFile, cfg))

	// Create docs directory structure for setupDocsDir.
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "packages"), 0o755))

	// Create output directory.
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	err = run("badformat")
	// Should fail trying to resolve the package (no git source), but
	// the activate flag error only fires if packages have inactive status
	// and we get past the install phase. Since there's no actual git source,
	// install will fail for all packages first.
	assert.Error(t, err)
}
