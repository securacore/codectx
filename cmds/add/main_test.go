package add

import (
	"os"
	"path/filepath"
	"testing"

	"securacore/codectx/core/config"
	"securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestParseActivateFlag_allSections(t *testing.T) {
	a, err := parseActivateFlag("foundation:a,topics:b,prompts:c,plans:d")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"a"}, a.Map.Foundation)
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

// --- detectCollisions ---

func setupCollisionTest(t *testing.T) (string, *config.Config) {
	t.Helper()
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")

	// Create local manifest with a foundation entry.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation"), 0o755))

	localManifest := &manifest.Manifest{
		Name:    "test-project",
		Author:  "tester",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Philosophy"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "package.yml"), localManifest))

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
			{ID: "conventions", Path: "foundation/conventions.md"},
		},
	}

	collisions := detectCollisions(cfg, newManifest, config.Activation{Mode: "all"})
	assert.Empty(t, collisions)
}

func TestDetectCollisions_foundationCollision(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
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

func TestDetectCollisions_granularActivationNoCollision(t *testing.T) {
	_, cfg := setupCollisionTest(t)

	// New manifest has both colliding and non-colliding entries,
	// but we only activate the non-colliding one.
	newManifest := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
			{ID: "unique", Path: "foundation/unique.md"},
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
	require.NoError(t, manifest.Write(filepath.Join(pkgDir, "package.yml"), pkgManifest))

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
