package shared

import (
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ParseActivateFlag ---

func TestParseActivateFlag_all(t *testing.T) {
	a, err := ParseActivateFlag("all")
	require.NoError(t, err)
	assert.True(t, a.IsAll())
}

func TestParseActivateFlag_none(t *testing.T) {
	a, err := ParseActivateFlag("none")
	require.NoError(t, err)
	assert.True(t, a.IsNone())
}

func TestParseActivateFlag_granular(t *testing.T) {
	a, err := ParseActivateFlag("topics:react,foundation:philosophy")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
	assert.Equal(t, []string{"philosophy"}, a.Map.Foundation)
}

func TestParseActivateFlag_allSections(t *testing.T) {
	a, err := ParseActivateFlag("foundation:a,application:b,topics:c,prompts:d,plans:e")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"a"}, a.Map.Foundation)
	assert.Equal(t, []string{"b"}, a.Map.Application)
	assert.Equal(t, []string{"c"}, a.Map.Topics)
	assert.Equal(t, []string{"d"}, a.Map.Prompts)
	assert.Equal(t, []string{"e"}, a.Map.Plans)
}

func TestParseActivateFlag_unknownSection(t *testing.T) {
	_, err := ParseActivateFlag("invalid:foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown section")
}

func TestParseActivateFlag_missingColon(t *testing.T) {
	_, err := ParseActivateFlag("topics")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected section:id")
}

func TestParseActivateFlag_emptyID(t *testing.T) {
	_, err := ParseActivateFlag("topics:")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty id")
}

func TestParseActivateFlag_whitespace(t *testing.T) {
	a, err := ParseActivateFlag("topics:react , foundation:philosophy")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
	assert.Equal(t, []string{"philosophy"}, a.Map.Foundation)
}

// --- FilterManifestForIDs ---

func TestFilterManifestForIDs_all(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "a"}},
		Topics:     []manifest.TopicEntry{{ID: "b"}},
	}
	filtered := FilterManifestForIDs(m, config.Activation{Mode: "all"})
	assert.Equal(t, m, filtered)
}

func TestFilterManifestForIDs_none(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "a"}},
	}
	filtered := FilterManifestForIDs(m, config.Activation{Mode: "none"})
	assert.Empty(t, filtered.Foundation)
}

func TestFilterManifestForIDs_granular(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "keep"}, {ID: "drop"}},
		Topics:     []manifest.TopicEntry{{ID: "also-keep"}, {ID: "also-drop"}},
	}
	activation := config.Activation{
		Map: &config.ActivationMap{
			Foundation: []string{"keep"},
			Topics:     []string{"also-keep"},
		},
	}
	filtered := FilterManifestForIDs(m, activation)
	require.Len(t, filtered.Foundation, 1)
	assert.Equal(t, "keep", filtered.Foundation[0].ID)
	require.Len(t, filtered.Topics, 1)
	assert.Equal(t, "also-keep", filtered.Topics[0].ID)
}

// --- ToSet ---

func TestToSet(t *testing.T) {
	s := ToSet([]string{"a", "b", "c"})
	assert.True(t, s["a"])
	assert.True(t, s["b"])
	assert.True(t, s["c"])
	assert.False(t, s["d"])
}

func TestToSet_empty(t *testing.T) {
	s := ToSet(nil)
	assert.Len(t, s, 0)
}

// --- SplitKey ---

func TestSplitKey_normal(t *testing.T) {
	section, id := SplitKey("topics:react")
	assert.Equal(t, "topics", section)
	assert.Equal(t, "react", id)
}

func TestSplitKey_noColon(t *testing.T) {
	section, id := SplitKey("nocolon")
	assert.Equal(t, "nocolon", section)
	assert.Equal(t, "", id)
}

func TestSplitKey_multipleColons(t *testing.T) {
	section, id := SplitKey("a:b:c")
	assert.Equal(t, "a", section)
	assert.Equal(t, "b:c", id)
}

func TestSplitKey_empty(t *testing.T) {
	section, id := SplitKey("")
	assert.Equal(t, "", section)
	assert.Equal(t, "", id)
}

// --- ConfigFile constant ---

func TestConfigFile(t *testing.T) {
	assert.Equal(t, "codectx.yml", ConfigFile)
}
