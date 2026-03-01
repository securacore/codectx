package activate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command metadata ---

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "activate", Command.Name)
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
	a, err := shared.ParseActivateFlag("all")
	require.NoError(t, err)
	assert.True(t, a.IsAll())
}

func TestParseActivateFlag_none(t *testing.T) {
	a, err := shared.ParseActivateFlag("none")
	require.NoError(t, err)
	assert.True(t, a.IsNone())
}

func TestParseActivateFlag_granular(t *testing.T) {
	a, err := shared.ParseActivateFlag("topics:react,foundation:core")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
	assert.Equal(t, []string{"core"}, a.Map.Foundation)
}

func TestParseActivateFlag_application(t *testing.T) {
	a, err := shared.ParseActivateFlag("application:architecture")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"architecture"}, a.Map.Application)
}

func TestParseActivateFlag_unknownSection(t *testing.T) {
	_, err := shared.ParseActivateFlag("invalid:foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown section")
}

func TestParseActivateFlag_missingColon(t *testing.T) {
	_, err := shared.ParseActivateFlag("topics")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected section:id")
}

func TestParseActivateFlag_emptyID(t *testing.T) {
	_, err := shared.ParseActivateFlag("topics:")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty id")
}

// --- findPackage ---

func TestFindPackage_found(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "react", Author: "org"},
			{Name: "go", Author: "org"},
		},
	}
	assert.Equal(t, 0, findPackage(cfg, "react@org"))
	assert.Equal(t, 1, findPackage(cfg, "go@org"))
}

func TestFindPackage_notFound(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "react", Author: "org"},
		},
	}
	assert.Equal(t, -1, findPackage(cfg, "missing@org"))
}

func TestFindPackage_invalidFormat(t *testing.T) {
	cfg := &config.Config{
		Packages: []config.PackageDep{
			{Name: "react", Author: "org"},
		},
	}
	assert.Equal(t, -1, findPackage(cfg, "no-at-sign"))
}

// --- activationLabel ---

func TestActivationLabel_all(t *testing.T) {
	assert.Equal(t, "all", activationLabel(config.Activation{Mode: "all"}))
}

func TestActivationLabel_none(t *testing.T) {
	assert.Equal(t, "none", activationLabel(config.Activation{Mode: "none"}))
}

func TestActivationLabel_granular(t *testing.T) {
	a := config.Activation{Map: &config.ActivationMap{
		Topics:      []string{"react", "go"},
		Foundation:  []string{"core"},
		Application: []string{"architecture"},
	}}
	label := activationLabel(a)
	assert.Contains(t, label, "foundation: core")
	assert.Contains(t, label, "application: architecture")
	assert.Contains(t, label, "topics: react, go")
}

// --- activationEntryCount ---

func TestActivationEntryCount_none(t *testing.T) {
	assert.Equal(t, 0, activationEntryCount(config.Activation{Mode: "none"}))
}

func TestActivationEntryCount_granular(t *testing.T) {
	a := config.Activation{Map: &config.ActivationMap{
		Foundation:  []string{"a"},
		Application: []string{"arch"},
		Topics:      []string{"b", "c"},
		Prompts:     []string{"d"},
	}}
	assert.Equal(t, 5, activationEntryCount(a))
}

// --- filterManifestForIDs ---

func TestFilterManifestForIDs_all(t *testing.T) {
	m := &config.Activation{Mode: "all"}
	mf := testManifest()
	filtered := shared.FilterManifestForIDs(mf, *m)
	assert.Equal(t, mf, filtered)
}

func TestFilterManifestForIDs_none(t *testing.T) {
	m := &config.Activation{Mode: "none"}
	mf := testManifest()
	filtered := shared.FilterManifestForIDs(mf, *m)
	assert.Empty(t, filtered.Foundation)
	assert.Empty(t, filtered.Topics)
}

func TestFilterManifestForIDs_granular(t *testing.T) {
	a := config.Activation{Map: &config.ActivationMap{
		Topics: []string{"conventions"},
	}}
	mf := testManifest()
	filtered := shared.FilterManifestForIDs(mf, a)
	assert.Empty(t, filtered.Foundation)
	require.Len(t, filtered.Topics, 1)
	assert.Equal(t, "conventions", filtered.Topics[0].ID)
}

func TestFilterManifestForIDs_application(t *testing.T) {
	a := config.Activation{Map: &config.ActivationMap{
		Application: []string{"architecture"},
	}}
	mf := testManifest()
	filtered := shared.FilterManifestForIDs(mf, a)
	require.Len(t, filtered.Application, 1)
	assert.Equal(t, "architecture", filtered.Application[0].ID)
	assert.Empty(t, filtered.Foundation)
	assert.Empty(t, filtered.Topics)
}

// --- splitKey ---

func TestSplitKey(t *testing.T) {
	s, id := shared.SplitKey("topics:react")
	assert.Equal(t, "topics", s)
	assert.Equal(t, "react", id)
}

func TestSplitKey_noColon(t *testing.T) {
	s, id := shared.SplitKey("nocolon")
	assert.Equal(t, "nocolon", s)
	assert.Empty(t, id)
}

// --- parseActivateFlag: prompts and plans ---

func TestParseActivateFlag_prompts(t *testing.T) {
	a, err := shared.ParseActivateFlag("prompts:review")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"review"}, a.Map.Prompts)
}

func TestParseActivateFlag_plans(t *testing.T) {
	a, err := shared.ParseActivateFlag("plans:migration")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"migration"}, a.Map.Plans)
}

func TestParseActivateFlag_allFiveSections(t *testing.T) {
	a, err := shared.ParseActivateFlag("foundation:a,application:b,topics:c,prompts:d,plans:e")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"a"}, a.Map.Foundation)
	assert.Equal(t, []string{"b"}, a.Map.Application)
	assert.Equal(t, []string{"c"}, a.Map.Topics)
	assert.Equal(t, []string{"d"}, a.Map.Prompts)
	assert.Equal(t, []string{"e"}, a.Map.Plans)
}

// --- detectCollisions ---

func setupCollisionProject(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))

	// Write a local manifest with some entries.
	localManifest := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Philosophy"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), localManifest))

	// Create the files on disk so Sync doesn't remove them.
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "philosophy.md"), []byte("# Philosophy\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "README.md"), []byte("# React\n"), 0o644))

	return dir, docsDir
}

func TestDetectCollisions_noCollisions(t *testing.T) {
	_, docsDir := setupCollisionProject(t)

	cfg := &config.Config{
		Name:   "test-project",
		Config: &config.BuildConfig{DocsDir: docsDir},
	}

	// Package with a unique foundation entry — no overlap with local.
	pkgManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "unique-doc", Path: "foundation/unique.md"},
		},
	}

	collisions := shared.DetectCollisions(cfg, -1, pkgManifest, config.Activation{Mode: "all"})
	assert.Empty(t, collisions)
}

func TestDetectCollisions_collisionWithLocal(t *testing.T) {
	_, docsDir := setupCollisionProject(t)

	cfg := &config.Config{
		Name:   "test-project",
		Config: &config.BuildConfig{DocsDir: docsDir},
	}

	// Package has same foundation entry as local.
	pkgManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
	}

	collisions := shared.DetectCollisions(cfg, -1, pkgManifest, config.Activation{Mode: "all"})
	require.Len(t, collisions, 1)
	assert.Equal(t, "foundation", collisions[0].Section)
	assert.Equal(t, "philosophy", collisions[0].ID)
	assert.Equal(t, "local", collisions[0].Pkg)
}

func TestDetectCollisions_skipsPackageAtSkipIdx(t *testing.T) {
	_, docsDir := setupCollisionProject(t)

	// Create an installed package that shares a topic with the new activation.
	pkgDir := filepath.Join(docsDir, "packages", "pkg-a@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "topics", "go"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "topics", "go", "README.md"), []byte("# Go\n"), 0o644))
	pkgAManifest := &manifest.Manifest{
		Name: "pkg-a", Author: "org",
		Topics: []manifest.TopicEntry{{ID: "go", Path: "topics/go/README.md"}},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgAManifest))

	cfg := &config.Config{
		Name:   "test-project",
		Config: &config.BuildConfig{DocsDir: docsDir},
		Packages: []config.PackageDep{
			{Name: "pkg-a", Author: "org", Active: config.Activation{Mode: "all"}},
		},
	}

	// Re-activating pkg-a at index 0: skipIdx=0 should skip self.
	collisions := shared.DetectCollisions(cfg, 0, pkgAManifest, config.Activation{Mode: "all"})
	// Should not collide with itself.
	for _, c := range collisions {
		assert.NotEqual(t, "pkg-a@org", c.Pkg, "should not collide with itself")
	}
}

func TestDetectCollisions_skipsInactivePackages(t *testing.T) {
	_, docsDir := setupCollisionProject(t)

	// Create an installed but inactive package.
	pkgDir := filepath.Join(docsDir, "packages", "inactive@org")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	inactiveManifest := &manifest.Manifest{
		Name: "inactive", Author: "org",
		Foundation: []manifest.FoundationEntry{
			{ID: "shared-doc", Path: "foundation/shared.md"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), inactiveManifest))

	cfg := &config.Config{
		Name:   "test-project",
		Config: &config.BuildConfig{DocsDir: docsDir},
		Packages: []config.PackageDep{
			{Name: "inactive", Author: "org", Active: config.Activation{Mode: "none"}},
		},
	}

	// New package has same entry — but inactive package shouldn't create collision.
	pkgManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "shared-doc", Path: "foundation/shared.md"},
		},
	}

	collisions := shared.DetectCollisions(cfg, -1, pkgManifest, config.Activation{Mode: "all"})
	// No collision because the existing package is inactive.
	for _, c := range collisions {
		assert.NotEqual(t, "inactive@org", c.Pkg)
	}
}

func TestDetectCollisions_withActiveInstalledPackage(t *testing.T) {
	_, docsDir := setupCollisionProject(t)

	// Create an installed, active package with a topic.
	pkgDir := filepath.Join(docsDir, "packages", "existing@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "topics", "go"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "topics", "go", "README.md"), []byte("# Go\n"), 0o644))
	existingManifest := &manifest.Manifest{
		Name: "existing", Author: "org",
		Topics: []manifest.TopicEntry{{ID: "go", Path: "topics/go/README.md"}},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), existingManifest))

	cfg := &config.Config{
		Name:   "test-project",
		Config: &config.BuildConfig{DocsDir: docsDir},
		Packages: []config.PackageDep{
			{Name: "existing", Author: "org", Active: config.Activation{Mode: "all"}},
		},
	}

	// New package also has "go" topic → collision.
	pkgManifest := &manifest.Manifest{
		Topics: []manifest.TopicEntry{{ID: "go", Path: "topics/go/README.md"}},
	}

	collisions := shared.DetectCollisions(cfg, -1, pkgManifest, config.Activation{Mode: "all"})
	require.Len(t, collisions, 1)
	assert.Equal(t, "topics", collisions[0].Section)
	assert.Equal(t, "go", collisions[0].ID)
	assert.Equal(t, "existing@org", collisions[0].Pkg)
}

func TestDetectCollisions_noneActivationNoCollisions(t *testing.T) {
	_, docsDir := setupCollisionProject(t)

	cfg := &config.Config{
		Name:   "test-project",
		Config: &config.BuildConfig{DocsDir: docsDir},
	}

	// Even with overlapping entries, "none" activation produces no collisions.
	pkgManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
	}

	collisions := shared.DetectCollisions(cfg, -1, pkgManifest, config.Activation{Mode: "none"})
	assert.Empty(t, collisions)
}

// --- runCLI integration tests ---

func setupActivateProject(t *testing.T, packages []config.PackageDep) string {
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
	require.NoError(t, config.Write(shared.ConfigFile, cfg))

	m := &manifest.Manifest{
		Name: "test-project", Author: "tester", Version: "1.0.0",
		Description: "Test project",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Pre-set auto_compile=false to avoid compile and interactive prompt.
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	prefs := &preferences.Preferences{AutoCompile: preferences.BoolPtr(false)}
	require.NoError(t, preferences.Write(outputDir, prefs))

	return dir
}

func TestRunCLI_activateAll(t *testing.T) {
	setupActivateProject(t, []config.PackageDep{
		{Name: "react", Author: "org", Active: config.Activation{Mode: "none"}},
	})

	err := runCLI([]string{"react@org"}, "all")
	require.NoError(t, err)

	// Verify config was updated.
	cfg, err := config.Load(shared.ConfigFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.True(t, cfg.Packages[0].Active.IsAll())
}

func TestRunCLI_defaultsToAll(t *testing.T) {
	setupActivateProject(t, []config.PackageDep{
		{Name: "react", Author: "org", Active: config.Activation{Mode: "none"}},
	})

	// Empty activateFlag → defaults to "all".
	err := runCLI([]string{"react@org"}, "")
	require.NoError(t, err)

	cfg, err := config.Load(shared.ConfigFile)
	require.NoError(t, err)
	assert.True(t, cfg.Packages[0].Active.IsAll())
}

func TestRunCLI_activateGranular(t *testing.T) {
	setupActivateProject(t, []config.PackageDep{
		{Name: "react", Author: "org", Active: config.Activation{Mode: "none"}},
	})

	err := runCLI([]string{"react@org"}, "topics:react")
	require.NoError(t, err)

	cfg, err := config.Load(shared.ConfigFile)
	require.NoError(t, err)
	require.True(t, cfg.Packages[0].Active.IsGranular())
	assert.Equal(t, []string{"react"}, cfg.Packages[0].Active.Map.Topics)
}

func TestRunCLI_activateNone(t *testing.T) {
	setupActivateProject(t, []config.PackageDep{
		{Name: "react", Author: "org", Active: config.Activation{Mode: "all"}},
	})

	err := runCLI([]string{"react@org"}, "none")
	require.NoError(t, err)

	cfg, err := config.Load(shared.ConfigFile)
	require.NoError(t, err)
	assert.True(t, cfg.Packages[0].Active.IsNone())
}

func TestRunCLI_multiplePackages(t *testing.T) {
	setupActivateProject(t, []config.PackageDep{
		{Name: "react", Author: "org", Active: config.Activation{Mode: "none"}},
		{Name: "go", Author: "org", Active: config.Activation{Mode: "none"}},
	})

	err := runCLI([]string{"react@org", "go@org"}, "all")
	require.NoError(t, err)

	cfg, err := config.Load(shared.ConfigFile)
	require.NoError(t, err)
	assert.True(t, cfg.Packages[0].Active.IsAll())
	assert.True(t, cfg.Packages[1].Active.IsAll())
}

func TestRunCLI_packageNotFound(t *testing.T) {
	setupActivateProject(t, []config.PackageDep{
		{Name: "react", Author: "org"},
	})

	err := runCLI([]string{"missing@org"}, "all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no packages were modified")
}

func TestRunCLI_noConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runCLI([]string{"react@org"}, "all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRunCLI_invalidActivateFlag(t *testing.T) {
	setupActivateProject(t, []config.PackageDep{
		{Name: "react", Author: "org"},
	})

	err := runCLI([]string{"react@org"}, "badformat")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse --activate")
}

func TestRunCLI_collisionDetection(t *testing.T) {
	dir := setupActivateProject(t, []config.PackageDep{
		{Name: "pkg-a", Author: "org", Active: config.Activation{Mode: "none"}},
	})

	docsDir := filepath.Join(dir, "docs")

	// Create a local foundation doc.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "shared.md"),
		[]byte("# Shared\n"), 0o644))

	// Update local manifest with the entry.
	localManifest := &manifest.Manifest{
		Name: "test-project", Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "shared", Path: "foundation/shared.md", Description: "Shared doc"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), localManifest))

	// Create installed package with same entry.
	pkgDir := filepath.Join(docsDir, "packages", "pkg-a@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "foundation", "shared.md"), []byte("# Pkg Shared\n"), 0o644))
	pkgManifest := &manifest.Manifest{
		Name: "pkg-a", Author: "org",
		Foundation: []manifest.FoundationEntry{
			{ID: "shared", Path: "foundation/shared.md"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	// Should succeed (collisions are warnings, not errors).
	err := runCLI([]string{"pkg-a@org"}, "all")
	require.NoError(t, err)

	// Package should still be activated.
	cfg, err := config.Load(shared.ConfigFile)
	require.NoError(t, err)
	assert.True(t, cfg.Packages[0].Active.IsAll())
}

// --- Helpers ---

func testManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Name: "test",
		Foundation: []manifest.FoundationEntry{
			{ID: "core", Description: "Core principles"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "architecture", Description: "System architecture"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "conventions", Description: "Conventions"},
			{ID: "patterns", Description: "Patterns"},
		},
	}
}
