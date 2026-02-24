package activate

import (
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"

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
	a, err := parseActivateFlag("topics:react,foundation:core")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
	assert.Equal(t, []string{"core"}, a.Map.Foundation)
}

func TestParseActivateFlag_unknownSection(t *testing.T) {
	_, err := parseActivateFlag("invalid:foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown section")
}

func TestParseActivateFlag_missingColon(t *testing.T) {
	_, err := parseActivateFlag("topics")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected section:id")
}

func TestParseActivateFlag_emptyID(t *testing.T) {
	_, err := parseActivateFlag("topics:")
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
		Topics:     []string{"react", "go"},
		Foundation: []string{"core"},
	}}
	label := activationLabel(a)
	assert.Contains(t, label, "foundation: core")
	assert.Contains(t, label, "topics: react, go")
}

// --- activationEntryCount ---

func TestActivationEntryCount_none(t *testing.T) {
	assert.Equal(t, 0, activationEntryCount(config.Activation{Mode: "none"}))
}

func TestActivationEntryCount_granular(t *testing.T) {
	a := config.Activation{Map: &config.ActivationMap{
		Foundation: []string{"a"},
		Topics:     []string{"b", "c"},
		Prompts:    []string{"d"},
	}}
	assert.Equal(t, 4, activationEntryCount(a))
}

// --- filterManifestForIDs ---

func TestFilterManifestForIDs_all(t *testing.T) {
	m := &config.Activation{Mode: "all"}
	mf := testManifest()
	filtered := filterManifestForIDs(mf, *m)
	assert.Equal(t, mf, filtered)
}

func TestFilterManifestForIDs_none(t *testing.T) {
	m := &config.Activation{Mode: "none"}
	mf := testManifest()
	filtered := filterManifestForIDs(mf, *m)
	assert.Empty(t, filtered.Foundation)
	assert.Empty(t, filtered.Topics)
}

func TestFilterManifestForIDs_granular(t *testing.T) {
	a := config.Activation{Map: &config.ActivationMap{
		Topics: []string{"conventions"},
	}}
	mf := testManifest()
	filtered := filterManifestForIDs(mf, a)
	assert.Empty(t, filtered.Foundation)
	require.Len(t, filtered.Topics, 1)
	assert.Equal(t, "conventions", filtered.Topics[0].ID)
}

// --- splitKey ---

func TestSplitKey(t *testing.T) {
	s, id := splitKey("topics:react")
	assert.Equal(t, "topics", s)
	assert.Equal(t, "react", id)
}

func TestSplitKey_noColon(t *testing.T) {
	s, id := splitKey("nocolon")
	assert.Equal(t, "nocolon", s)
	assert.Empty(t, id)
}

// --- Helpers ---

func testManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Name: "test",
		Foundation: []manifest.FoundationEntry{
			{ID: "core", Description: "Core principles"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "conventions", Description: "Conventions"},
			{ID: "patterns", Description: "Patterns"},
		},
	}
}
