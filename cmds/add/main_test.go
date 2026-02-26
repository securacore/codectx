package add

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/resolve"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command metadata ---

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "add", Command.Name)
	assert.NotEmpty(t, Command.Usage)
	assert.Equal(t, "<package> [package...]", Command.ArgsUsage)
}

func TestCommand_flags(t *testing.T) {
	require.Len(t, Command.Flags, 2)

	flagNames := make(map[string]bool)
	for _, f := range Command.Flags {
		flagNames[f.Names()[0]] = true
	}
	assert.True(t, flagNames["source"])
	assert.True(t, flagNames["activate"])
}

// --- parseActivateFlag ---

func TestParseActivateFlag_all(t *testing.T) {
	a, err := parseActivateFlag("all")
	require.NoError(t, err)
	assert.Equal(t, "all", a.Mode)
	assert.True(t, a.IsAll())
	assert.Nil(t, a.Map)
}

func TestParseActivateFlag_none(t *testing.T) {
	a, err := parseActivateFlag("none")
	require.NoError(t, err)
	assert.Equal(t, "none", a.Mode)
	assert.True(t, a.IsNone())
}

func TestParseActivateFlag_singleGranular(t *testing.T) {
	a, err := parseActivateFlag("topics:react")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
}

func TestParseActivateFlag_multipleGranular(t *testing.T) {
	a, err := parseActivateFlag("foundation:philosophy,topics:react,topics:go,plans:migration")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"philosophy"}, a.Map.Foundation)
	assert.Equal(t, []string{"react", "go"}, a.Map.Topics)
	assert.Nil(t, a.Map.Prompts)
	assert.Equal(t, []string{"migration"}, a.Map.Plans)
}

func TestParseActivateFlag_application(t *testing.T) {
	a, err := parseActivateFlag("application:architecture")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"architecture"}, a.Map.Application)
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

func TestParseActivateFlag_errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		msg   string
	}{
		{"no colon", "topicsreact", "expected section:id"},
		{"empty id", "topics:", "empty id"},
		{"unknown section", "widgets:foo", "unknown section"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseActivateFlag(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}

// --- buildCombinedEntries ---

func TestBuildCombinedEntries_singleTarget(t *testing.T) {
	targets := []*addTarget{
		{
			resolved: &resolve.ResolvedPackage{Name: "pkg", Author: "org"},
			manifest: &manifest.Manifest{
				Foundation:  []manifest.FoundationEntry{{ID: "f1", Description: "F1"}},
				Application: []manifest.ApplicationEntry{{ID: "a1", Description: "A1"}},
				Topics:      []manifest.TopicEntry{{ID: "t1", Description: "T1"}},
				Prompts:     []manifest.PromptEntry{{ID: "p1", Description: "P1"}},
				Plans:       []manifest.PlanEntry{{ID: "pl1", Description: "PL1"}},
			},
		},
	}

	entries := buildCombinedEntries(targets)
	assert.Len(t, entries, 5)

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

func TestBuildCombinedEntries_multipleTargets(t *testing.T) {
	targets := []*addTarget{
		{
			resolved: &resolve.ResolvedPackage{Name: "alpha", Author: "org"},
			manifest: &manifest.Manifest{
				Foundation: []manifest.FoundationEntry{{ID: "f1", Description: "Alpha F1"}},
			},
		},
		{
			resolved: &resolve.ResolvedPackage{Name: "beta", Author: "org"},
			manifest: &manifest.Manifest{
				Topics: []manifest.TopicEntry{{ID: "t1", Description: "Beta T1"}},
			},
		},
	}

	entries := buildCombinedEntries(targets)
	assert.Len(t, entries, 2)
	assert.Contains(t, entries[0].label, "alpha@org")
	assert.Contains(t, entries[1].label, "beta@org")
}

func TestBuildCombinedEntries_emptyManifest(t *testing.T) {
	targets := []*addTarget{
		{
			resolved: &resolve.ResolvedPackage{Name: "empty", Author: "org"},
			manifest: &manifest.Manifest{},
		},
	}

	entries := buildCombinedEntries(targets)
	assert.Empty(t, entries)
}

func TestBuildCombinedEntries_noTargets(t *testing.T) {
	entries := buildCombinedEntries(nil)
	assert.Empty(t, entries)
}

// --- resolveActivation ---

func TestResolveActivation_allSelected(t *testing.T) {
	entries := []activationEntry{
		{section: "foundation", id: "f1"},
		{section: "topics", id: "t1"},
	}
	selected := []int{0, 1}

	a := resolveActivation(entries, selected)
	assert.True(t, a.IsAll())
}

func TestResolveActivation_noneSelected(t *testing.T) {
	entries := []activationEntry{
		{section: "foundation", id: "f1"},
	}
	selected := []int{}

	a := resolveActivation(entries, selected)
	assert.True(t, a.IsNone())
}

func TestResolveActivation_granular(t *testing.T) {
	entries := []activationEntry{
		{section: "foundation", id: "f1"},
		{section: "foundation", id: "f2"},
		{section: "topics", id: "t1"},
	}
	// Select only f1 and t1.
	selected := []int{0, 2}

	a := resolveActivation(entries, selected)
	assert.True(t, a.IsGranular())
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"f1"}, a.Map.Foundation)
	assert.Equal(t, []string{"t1"}, a.Map.Topics)
}

func TestResolveActivation_allFiveSections(t *testing.T) {
	entries := []activationEntry{
		{section: "foundation", id: "f1"},
		{section: "application", id: "a1"},
		{section: "topics", id: "t1"},
		{section: "prompts", id: "p1"},
		{section: "plans", id: "pl1"},
		{section: "topics", id: "t2"},
	}
	// Select a subset across all 5 sections.
	selected := []int{0, 1, 2, 3, 4}

	a := resolveActivation(entries, selected)
	assert.True(t, a.IsGranular()) // 5 out of 6 → granular
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"f1"}, a.Map.Foundation)
	assert.Equal(t, []string{"a1"}, a.Map.Application)
	assert.Equal(t, []string{"t1"}, a.Map.Topics)
	assert.Equal(t, []string{"p1"}, a.Map.Prompts)
	assert.Equal(t, []string{"pl1"}, a.Map.Plans)
}

// --- detectCollisions ---

func setupCollisionTest(t *testing.T) (string, *config.Config) {
	t.Helper()
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")

	// Create directories and actual files so Sync stale removal doesn't drop them.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "application", "architecture"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "topics", "react"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "foundation", "philosophy", "README.md"), []byte("# Philosophy\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "application", "architecture", "README.md"), []byte("# Architecture\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "topics", "react", "README.md"), []byte("# React\n"), 0o644))

	localManifest := &manifest.Manifest{
		Name:    "test-project",
		Author:  "tester",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Description: "Philosophy"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "architecture", Path: "application/architecture/README.md", Description: "System architecture"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), localManifest))

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			DocsDir: docsDir,
		},
		Packages: []config.PackageDep{},
	}

	return dir, cfg
}

func TestDetectCollisions_noCollisions(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "conventions", Path: "foundation/conventions/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	assert.Empty(t, collisions)
}

func TestDetectCollisions_foundationCollision(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	require.Len(t, collisions, 1)
	assert.Equal(t, "foundation", collisions[0].section)
	assert.Equal(t, "philosophy", collisions[0].id)
	assert.Equal(t, "local", collisions[0].pkg)
}

func TestDetectCollisions_topicCollision(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	newManifest := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	require.Len(t, collisions, 1)
	assert.Equal(t, "topics", collisions[0].section)
	assert.Equal(t, "react", collisions[0].id)
}

func TestDetectCollisions_applicationCollision(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	newManifest := &manifest.Manifest{
		Application: []manifest.ApplicationEntry{
			{ID: "architecture", Path: "application/architecture/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	require.Len(t, collisions, 1)
	assert.Equal(t, "application", collisions[0].section)
	assert.Equal(t, "architecture", collisions[0].id)
	assert.Equal(t, "local", collisions[0].pkg)
}

func TestDetectCollisions_granularActivationNoCollision(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	// New manifest has both colliding and non-colliding entries,
	// but we only activate the non-colliding one.
	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md"},
			{ID: "unique", Path: "foundation/unique/README.md"},
		},
	}

	activation := config.Activation{
		Map: &config.ActivationMap{
			Foundation: []string{"unique"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, activation)
	assert.Empty(t, collisions)
}

func TestDetectCollisions_withExistingPackage(t *testing.T) {
	dir, cfg := setupCollisionTest(t)
	docsDir := cfg.DocsDir()

	// Set up an existing installed package with an active entry.
	pkgDir := filepath.Join(docsDir, "packages", "go@org")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	pkgManifest := &manifest.Manifest{
		Name:   "go",
		Author: "org",
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md", Description: "Go conventions"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	cfg.Packages = append(cfg.Packages, config.PackageDep{
		Name:   "go",
		Author: "org",
		Active: config.Activation{Mode: "all"},
	})

	// New manifest collides with the installed package's topic.
	newManifest := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	require.Len(t, collisions, 1)
	assert.Equal(t, "topics", collisions[0].section)
	assert.Equal(t, "go", collisions[0].id)
	assert.Equal(t, "go@org", collisions[0].pkg)

	_ = dir // keep for clarity
}

func TestDetectCollisions_granularActivationOnInstalledPackage(t *testing.T) {
	_, cfg := setupCollisionTest(t)
	docsDir := cfg.DocsDir()

	// Install a package with multiple topics but only activate one.
	pkgDir := filepath.Join(docsDir, "packages", "multi@org")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "topics", "go"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "topics", "python"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "topics", "go", "README.md"), []byte("# Go\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "topics", "python", "README.md"), []byte("# Python\n"), 0o644))

	pkgManifest := &manifest.Manifest{
		Name: "multi", Author: "org",
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md", Description: "Go"},
			{ID: "python", Path: "topics/python/README.md", Description: "Python"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	// Only activate "go", not "python".
	cfg.Packages = append(cfg.Packages, config.PackageDep{
		Name:   "multi",
		Author: "org",
		Active: config.Activation{Map: &config.ActivationMap{
			Topics: []string{"go"},
		}},
	})

	// New package has "go" (collides) and "python" (should NOT collide because
	// the installed package only has "go" activated).
	newManifest := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md"},
			{ID: "python", Path: "topics/python/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})

	// Only "go" should collide (from multi@org). "python" should not because it's
	// not activated in the installed package.
	goCollision := false
	pythonCollision := false
	for _, c := range collisions {
		if c.id == "go" && c.pkg == "multi@org" {
			goCollision = true
		}
		if c.id == "python" && c.pkg == "multi@org" {
			pythonCollision = true
		}
	}
	assert.True(t, goCollision, "should detect collision for activated 'go' topic")
	assert.False(t, pythonCollision, "should not detect collision for non-activated 'python' topic")
}

// --- filterManifestForIDs ---

func TestFilterManifestForIDs_all(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "a"}},
	}
	filtered := filterManifestForIDs(m, config.Activation{Mode: "all"})
	assert.Equal(t, m, filtered)
}

func TestFilterManifestForIDs_none(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "a"}},
	}
	filtered := filterManifestForIDs(m, config.Activation{Mode: "none"})
	assert.Empty(t, filtered.Foundation)
}

func TestFilterManifestForIDs_granular(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a"}, {ID: "b"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "c"}, {ID: "d"},
		},
	}
	activation := config.Activation{
		Map: &config.ActivationMap{
			Foundation: []string{"a"},
			Topics:     []string{"d"},
		},
	}
	filtered := filterManifestForIDs(m, activation)
	require.Len(t, filtered.Foundation, 1)
	assert.Equal(t, "a", filtered.Foundation[0].ID)
	require.Len(t, filtered.Topics, 1)
	assert.Equal(t, "d", filtered.Topics[0].ID)
}

func TestFilterManifestForIDs_application(t *testing.T) {
	m := &manifest.Manifest{
		Application: []manifest.ApplicationEntry{
			{ID: "architecture"}, {ID: "design"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react"},
		},
	}
	activation := config.Activation{
		Map: &config.ActivationMap{
			Application: []string{"architecture"},
		},
	}
	filtered := filterManifestForIDs(m, activation)
	require.Len(t, filtered.Application, 1)
	assert.Equal(t, "architecture", filtered.Application[0].ID)
	assert.Empty(t, filtered.Topics)
}

// --- splitKey ---

func TestSplitKey(t *testing.T) {
	section, id := splitKey("foundation:philosophy")
	assert.Equal(t, "foundation", section)
	assert.Equal(t, "philosophy", id)
}

func TestSplitKey_noColon(t *testing.T) {
	section, id := splitKey("noprefix")
	assert.Equal(t, "noprefix", section)
	assert.Equal(t, "", id)
}

// --- parseActivateFlag edge cases ---

func TestParseActivateFlag_emptyString(t *testing.T) {
	_, err := parseActivateFlag("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected section:id")
}

func TestParseActivateFlag_whitespace(t *testing.T) {
	a, err := parseActivateFlag("topics:react , foundation:philosophy")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
	assert.Equal(t, []string{"philosophy"}, a.Map.Foundation)
}

func TestParseActivateFlag_duplicateEntries(t *testing.T) {
	a, err := parseActivateFlag("topics:react,topics:react")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react", "react"}, a.Map.Topics)
}

func TestParseActivateFlag_colonInID(t *testing.T) {
	a, err := parseActivateFlag("topics:my:id")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"my:id"}, a.Map.Topics)
}

// --- detectCollisions edge cases ---

func TestDetectCollisions_noLocalManifest(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))
	// No manifest.yml in docsDir.

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			DocsDir: docsDir,
		},
		Packages: []config.PackageDep{},
	}

	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "unique-entry", Path: "foundation/unique/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	assert.Empty(t, collisions)
}

func TestDetectCollisions_inactivePackageSkipped(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	// Create an installed package with manifest.
	pkgDir := filepath.Join(docsDir, "packages", "mypkg@myauthor")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	pkgManifest := &manifest.Manifest{
		Name:   "mypkg",
		Author: "myauthor",
		Topics: []manifest.TopicEntry{
			{ID: "shared-topic", Path: "topics/shared/README.md"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "manifest.yml"), pkgManifest))

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			DocsDir: docsDir,
		},
		Packages: []config.PackageDep{
			{
				Name:   "mypkg",
				Author: "myauthor",
				Active: config.Activation{Mode: "none"},
			},
		},
	}

	// New manifest has the same topic ID as the inactive package.
	newManifest := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "shared-topic", Path: "topics/shared/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	assert.Empty(t, collisions)
}

func TestDetectCollisions_noneActivation(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	// New manifest has entries that would collide, but activation is "none".
	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "none"})
	assert.Empty(t, collisions)
}

// --- filterManifestForIDs edge cases ---

func TestFilterManifestForIDs_promptsAndPlans(t *testing.T) {
	m := &manifest.Manifest{
		Prompts: []manifest.PromptEntry{
			{ID: "lint", Path: "prompts/lint/README.md"},
			{ID: "review", Path: "prompts/review/README.md"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "migration", Path: "plans/migration/README.md"},
			{ID: "refactor", Path: "plans/refactor/README.md"},
		},
	}

	activation := config.Activation{
		Map: &config.ActivationMap{
			Prompts: []string{"review"},
			Plans:   []string{"migration"},
		},
	}

	filtered := filterManifestForIDs(m, activation)
	require.Len(t, filtered.Prompts, 1)
	assert.Equal(t, "review", filtered.Prompts[0].ID)
	require.Len(t, filtered.Plans, 1)
	assert.Equal(t, "migration", filtered.Plans[0].ID)
	assert.Empty(t, filtered.Foundation)
	assert.Empty(t, filtered.Topics)
}

func TestFilterManifestForIDs_emptyActivationSlice(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a"}, {ID: "b"},
		},
	}

	activation := config.Activation{
		Map: &config.ActivationMap{
			Foundation: []string{},
		},
	}

	filtered := filterManifestForIDs(m, activation)
	assert.Empty(t, filtered.Foundation)
}

func TestFilterManifestForIDs_nonexistentIDs(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a"}, {ID: "b"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "c"}, {ID: "d"},
		},
	}

	activation := config.Activation{
		Map: &config.ActivationMap{
			Foundation: []string{"nonexistent"},
			Topics:     []string{"also-missing"},
		},
	}

	filtered := filterManifestForIDs(m, activation)
	assert.Empty(t, filtered.Foundation)
	assert.Empty(t, filtered.Topics)
}

// --- splitKey edge cases ---

func TestSplitKey_emptyString(t *testing.T) {
	section, id := splitKey("")
	assert.Equal(t, "", section)
	assert.Equal(t, "", id)
}

func TestSplitKey_multipleColons(t *testing.T) {
	section, id := splitKey("a:b:c")
	assert.Equal(t, "a", section)
	assert.Equal(t, "b:c", id)
}

func TestSplitKey_colonAtStart(t *testing.T) {
	section, id := splitKey(":foo")
	assert.Equal(t, "", section)
	assert.Equal(t, "foo", id)
}

func TestSplitKey_colonAtEnd(t *testing.T) {
	section, id := splitKey("foo:")
	assert.Equal(t, "foo", section)
	assert.Equal(t, "", id)
}

// --- printActivation ---

// captureStdout runs fn and returns whatever it writes to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestPrintActivation_all(t *testing.T) {
	out := captureStdout(t, func() {
		printActivation(config.Activation{Mode: "all"})
	})
	assert.Contains(t, out, "Activation")
	assert.Contains(t, out, "all entries")
}

func TestPrintActivation_none(t *testing.T) {
	out := captureStdout(t, func() {
		printActivation(config.Activation{Mode: "none"})
	})
	assert.Contains(t, out, "Activation")
	assert.Contains(t, out, "none")
	assert.Contains(t, out, "not active")
}

func TestPrintActivation_granular(t *testing.T) {
	out := captureStdout(t, func() {
		printActivation(config.Activation{
			Map: &config.ActivationMap{
				Foundation:  []string{"philosophy"},
				Application: []string{"architecture"},
				Topics:      []string{"react", "go"},
				Prompts:     []string{"review"},
				Plans:       []string{"migration"},
			},
		})
	})
	assert.Contains(t, out, "foundation")
	assert.Contains(t, out, "philosophy")
	assert.Contains(t, out, "application")
	assert.Contains(t, out, "architecture")
	assert.Contains(t, out, "topics")
	assert.Contains(t, out, "react, go")
	assert.Contains(t, out, "prompts")
	assert.Contains(t, out, "review")
	assert.Contains(t, out, "plans")
	assert.Contains(t, out, "migration")
}

func TestPrintActivation_granularPartial(t *testing.T) {
	out := captureStdout(t, func() {
		printActivation(config.Activation{
			Map: &config.ActivationMap{
				Topics: []string{"react"},
			},
		})
	})
	assert.Contains(t, out, "topics")
	assert.Contains(t, out, "react")
	assert.NotContains(t, out, "foundation")
	assert.NotContains(t, out, "prompts")
	assert.NotContains(t, out, "plans")
}

// --- toSetLocal ---

func TestDetectCollisions_corruptInstalledPackageManifest(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	// Create an installed package directory with a corrupt manifest.
	pkgDir := filepath.Join(docsDir, "packages", "broken@org")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "manifest.yml"),
		[]byte("{{{{not valid yaml"),
		0o644,
	))

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			DocsDir: docsDir,
		},
		Packages: []config.PackageDep{
			{
				Name:   "broken",
				Author: "org",
				Active: config.Activation{Mode: "all"},
			},
		},
	}

	newManifest := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	// Corrupt installed manifest should be silently skipped (no collision, no error).
	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	assert.Empty(t, collisions)
}

func TestPrintActivation_granularEmpty(t *testing.T) {
	out := captureStdout(t, func() {
		printActivation(config.Activation{
			Map: &config.ActivationMap{
				Foundation: []string{},
				Topics:     []string{},
				Prompts:    []string{},
				Plans:      []string{},
			},
		})
	})
	// With all empty slices, the header prints but no section KVs appear.
	assert.Contains(t, out, "Activation")
	assert.NotContains(t, out, "foundation")
	assert.NotContains(t, out, "topics")
	assert.NotContains(t, out, "prompts")
	assert.NotContains(t, out, "plans")
}

// --- toSetLocal ---

func TestToSetLocal_normal(t *testing.T) {
	s := toSetLocal([]string{"a", "b", "c"})
	assert.Len(t, s, 3)
	assert.True(t, s["a"])
	assert.True(t, s["b"])
	assert.True(t, s["c"])
	assert.False(t, s["d"])
}

func TestToSetLocal_empty(t *testing.T) {
	s := toSetLocal([]string{})
	assert.Len(t, s, 0)
}

func TestToSetLocal_nil(t *testing.T) {
	s := toSetLocal(nil)
	assert.Len(t, s, 0)
}

// --- run() integration tests ---

// setupAddProject creates a minimal project structure and changes cwd.
// Returns the project root path. Cleanup restores the original cwd.
func setupAddProject(t *testing.T) string {
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

	// Write codectx.yml.
	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	// Write local manifest.yml.
	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Set auto_compile=false to avoid interactive prompt.
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(outputDir, "preferences.yml"),
		[]byte("auto_compile: false\n"),
		0o644,
	))

	return dir
}

// setupBareRepo creates a bare git repo with a tag and manifest.yml.
// Returns the path to the bare repo.
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

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Test\n"), 0o644))
	_, err = wt.Add("README.md")
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

func TestRun_addPackageSuccess(t *testing.T) {
	setupAddProject(t)
	bareDir := setupBareRepo(t, "test-pkg", "test-author", "1.0.0", []string{"v1.0.0"})

	err := Run([]string{"test-pkg@test-author"}, bareDir, "all")
	require.NoError(t, err)

	// Verify config was updated with the package.
	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.Equal(t, "test-pkg", cfg.Packages[0].Name)
	assert.Equal(t, "test-author", cfg.Packages[0].Author)
	assert.Equal(t, "^1.0.0", cfg.Packages[0].Version)
	assert.Equal(t, bareDir, cfg.Packages[0].Source)

	// Verify package was fetched to the expected directory.
	_, err = os.Stat(filepath.Join("docs", "packages", "test-pkg@test-author", "manifest.yml"))
	assert.NoError(t, err)
}

func TestRun_addPackageActivateNone(t *testing.T) {
	setupAddProject(t)
	bareDir := setupBareRepo(t, "test-pkg", "test-author", "1.0.0", []string{"v1.0.0"})

	err := Run([]string{"test-pkg@test-author"}, bareDir, "none")
	require.NoError(t, err)

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.True(t, cfg.Packages[0].Active.IsNone())
}

func TestRun_addPackageVersionPinning(t *testing.T) {
	setupAddProject(t)
	bareDir := setupBareRepo(t, "test-pkg", "test-author", "1.0.0", []string{"v1.0.0", "v1.1.0"})

	// No explicit version in input — should resolve to latest and pin with caret.
	err := Run([]string{"test-pkg@test-author"}, bareDir, "all")
	require.NoError(t, err)

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.Equal(t, "^1.1.0", cfg.Packages[0].Version)
}

func TestRun_addPackageWithExplicitVersion(t *testing.T) {
	setupAddProject(t)
	bareDir := setupBareRepo(t, "test-pkg", "test-author", "1.0.0", []string{"v1.0.0", "v1.1.0"})

	// Explicit version constraint should be preserved.
	err := Run([]string{"test-pkg@test-author:^1.0.0"}, bareDir, "all")
	require.NoError(t, err)

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.Equal(t, "^1.0.0", cfg.Packages[0].Version)
}

func TestRun_addPackageDuplicate(t *testing.T) {
	setupAddProject(t)
	bareDir := setupBareRepo(t, "test-pkg", "test-author", "1.0.0", []string{"v1.0.0"})

	// First add succeeds.
	err := Run([]string{"test-pkg@test-author"}, bareDir, "all")
	require.NoError(t, err)

	// Second add of the same package is reported as a failure.
	err = Run([]string{"test-pkg@test-author"}, bareDir, "all")
	assert.Error(t, err)
}

func TestRun_addPackageMissingAuthor(t *testing.T) {
	setupAddProject(t)

	err := Run([]string{"test-pkg"}, "", "all")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no packages were added")
}

func TestRun_addPackageInvalidSource(t *testing.T) {
	setupAddProject(t)

	err := Run([]string{"test-pkg@test-author"}, "/nonexistent/repo.git", "all")
	assert.Error(t, err)
}

func TestRun_addPackageInvalidActivateFlag(t *testing.T) {
	setupAddProject(t)
	bareDir := setupBareRepo(t, "test-pkg", "test-author", "1.0.0", []string{"v1.0.0"})

	err := Run([]string{"test-pkg@test-author"}, bareDir, "invalid:flag")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown section")
}

func TestRun_addPackageNoConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = Run([]string{"test-pkg@test-author"}, "/some/source", "all")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

// --- Multi-package integration tests ---

func TestRun_multiplePackagesSuccess(t *testing.T) {
	setupAddProject(t)
	bareDir1 := setupBareRepo(t, "pkg-a", "org-a", "1.0.0", []string{"v1.0.0"})
	bareDir2 := setupBareRepo(t, "pkg-b", "org-b", "2.0.0", []string{"v2.0.0"})

	// First add pkg-a with explicit source.
	err := Run([]string{"pkg-a@org-a"}, bareDir1, "all")
	require.NoError(t, err)

	// Then add pkg-b with a different source.
	err = Run([]string{"pkg-b@org-b"}, bareDir2, "all")
	require.NoError(t, err)

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 2)
	assert.Equal(t, "pkg-a", cfg.Packages[0].Name)
	assert.Equal(t, "pkg-b", cfg.Packages[1].Name)
}

func TestRun_multiplePackagesPartialFailure(t *testing.T) {
	setupAddProject(t)
	bareDir := setupBareRepo(t, "good-pkg", "author", "1.0.0", []string{"v1.0.0"})

	// Pass two packages: one with a valid source and one that will fail.
	// Both use the same source flag, which is only allowed for single packages
	// from the CLI, but Run() itself doesn't enforce that. However, the
	// second package has a different name@author that won't match the repo.
	// Instead, pass them separately: first succeeds, then call with bad.
	err := Run([]string{"good-pkg@author"}, bareDir, "all")
	require.NoError(t, err)

	// Now try adding one that fails (bad source).
	err = Run([]string{"bad-pkg@bad-author"}, "/nonexistent/path", "all")
	assert.Error(t, err)

	// Verify only the good package is in config.
	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.Equal(t, "good-pkg", cfg.Packages[0].Name)
}

// --- parseAndResolve tests ---

func TestParseAndResolve_duplicatePackage(t *testing.T) {
	setupAddProject(t)

	cfg, err := config.Load(configFile)
	require.NoError(t, err)

	// Add a package to config manually.
	cfg.Packages = append(cfg.Packages, config.PackageDep{
		Name:   "existing",
		Author: "org",
	})
	require.NoError(t, config.Write(configFile, cfg))
	cfg, _ = config.Load(configFile)

	_, err = parseAndResolve("existing@org", "", cfg, cfg.DocsDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestParseAndResolve_missingAuthorNoSource(t *testing.T) {
	setupAddProject(t)

	cfg, err := config.Load(configFile)
	require.NoError(t, err)

	_, err = parseAndResolve("pkgname", "", cfg, cfg.DocsDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "author required")
}

// --- printActivation edge cases ---

func TestPrintActivation_onlyPrompts(t *testing.T) {
	out := captureStdout(t, func() {
		printActivation(config.Activation{
			Map: &config.ActivationMap{
				Prompts: []string{"lint"},
			},
		})
	})
	assert.Contains(t, out, "prompts")
	assert.Contains(t, out, "lint")
	assert.NotContains(t, out, "foundation")
	assert.NotContains(t, out, "topics")
	assert.NotContains(t, out, "plans")
}

func TestPrintActivation_onlyPlans(t *testing.T) {
	out := captureStdout(t, func() {
		printActivation(config.Activation{
			Map: &config.ActivationMap{
				Plans: []string{"migration"},
			},
		})
	})
	assert.Contains(t, out, "plans")
	assert.Contains(t, out, "migration")
	assert.NotContains(t, out, "foundation")
	assert.NotContains(t, out, "topics")
	assert.NotContains(t, out, "prompts")
}

// --- detectCollisions: multiple collisions ---

func TestDetectCollisions_multipleCollisions(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	// Local manifest has foundation:philosophy and topics:react.
	// New manifest collides on both.
	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	assert.Len(t, collisions, 2)
}

// --- Sync integration tests ---

func TestRun_addPackageSyncsLocalManifest(t *testing.T) {
	dir := setupAddProject(t)
	bareDir := setupBareRepo(t, "sync-pkg", "sync-author", "1.0.0", []string{"v1.0.0"})
	docsDir := filepath.Join(dir, "docs")

	// Create a foundation file on disk that is NOT in the manifest.
	// After add, the post-add Sync should discover it.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "discovered"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "discovered", "README.md"),
		[]byte("# Discovered\n"), 0o644))

	err := Run([]string{"sync-pkg@sync-author"}, bareDir, "all")
	require.NoError(t, err)

	// Read back the local manifest — it should contain the discovered entry.
	reloaded, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, reloaded.Foundation, 1, "post-add Sync should discover the foundation file")
	assert.Equal(t, "discovered", reloaded.Foundation[0].ID)
}

func TestDetectCollisions_staleLocalEntryNotCollision(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation"), 0o755))

	// Write a manifest that declares a foundation entry, but DO NOT create the file.
	// Sync stale removal should drop it, so it won't collide.
	localManifest := &manifest.Manifest{
		Name:    "test-project",
		Author:  "tester",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "stale", Path: "foundation/stale/README.md", Description: "File was deleted"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), localManifest))

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			DocsDir: docsDir,
		},
		Packages: []config.PackageDep{},
	}

	// New package has the same ID — should NOT collide because the local
	// entry is stale (file doesn't exist) and Sync removes it.
	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "stale", Path: "foundation/stale/README.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	assert.Empty(t, collisions, "stale local entry (missing file) should not cause collision")
}
