package compile

import (
	"testing"

	"securacore/codectx/core/config"
	"securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Name:        "test-pkg",
		Author:      "test-author",
		Version:     "1.0.0",
		Description: "Test package",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Core philosophy"},
			{ID: "conventions", Path: "foundation/conventions.md", Description: "Conventions"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React conventions"},
			{ID: "go", Path: "topics/go/README.md", Description: "Go conventions"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "review", Path: "prompts/review/README.md", Description: "Code review"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "migration", Path: "plans/migration/README.md", State: "plans/migration/state.yml", Description: "Migration plan"},
		},
	}
}

// --- filterManifest ---

func TestFilterManifest_none(t *testing.T) {
	m := testManifest()
	activation := config.Activation{Mode: "none"}

	filtered := filterManifest(m, activation)

	assert.Equal(t, m.Name, filtered.Name)
	assert.Equal(t, m.Author, filtered.Author)
	assert.Empty(t, filtered.Foundation)
	assert.Empty(t, filtered.Topics)
	assert.Empty(t, filtered.Prompts)
	assert.Empty(t, filtered.Plans)
}

func TestFilterManifest_all(t *testing.T) {
	m := testManifest()
	activation := config.Activation{Mode: "all"}

	filtered := filterManifest(m, activation)

	// "all" returns the original manifest pointer.
	assert.Equal(t, m, filtered)
}

func TestFilterManifest_granularFoundation(t *testing.T) {
	m := testManifest()
	activation := config.Activation{
		Map: &config.ActivationMap{
			Foundation: []string{"philosophy"},
		},
	}

	filtered := filterManifest(m, activation)

	require.Len(t, filtered.Foundation, 1)
	assert.Equal(t, "philosophy", filtered.Foundation[0].ID)
	assert.Empty(t, filtered.Topics)
	assert.Empty(t, filtered.Prompts)
	assert.Empty(t, filtered.Plans)
}

func TestFilterManifest_granularMixed(t *testing.T) {
	m := testManifest()
	activation := config.Activation{
		Map: &config.ActivationMap{
			Foundation: []string{"conventions"},
			Topics:     []string{"react"},
			Plans:      []string{"migration"},
		},
	}

	filtered := filterManifest(m, activation)

	require.Len(t, filtered.Foundation, 1)
	assert.Equal(t, "conventions", filtered.Foundation[0].ID)
	require.Len(t, filtered.Topics, 1)
	assert.Equal(t, "react", filtered.Topics[0].ID)
	assert.Empty(t, filtered.Prompts)
	require.Len(t, filtered.Plans, 1)
	assert.Equal(t, "migration", filtered.Plans[0].ID)
}

func TestFilterManifest_granularNoMatches(t *testing.T) {
	m := testManifest()
	activation := config.Activation{
		Map: &config.ActivationMap{
			Foundation: []string{"nonexistent"},
		},
	}

	filtered := filterManifest(m, activation)

	assert.Empty(t, filtered.Foundation)
}

func TestFilterManifest_emptyActivation(t *testing.T) {
	m := testManifest()
	activation := config.Activation{} // IsNone() == true

	filtered := filterManifest(m, activation)

	assert.Empty(t, filtered.Foundation)
	assert.Empty(t, filtered.Topics)
}

// --- mergeManifest ---

func TestMergeManifest(t *testing.T) {
	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "local", Path: "foundation/local.md", Description: "Local doc"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "remote", Path: "foundation/remote.md", Description: "Remote doc"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React"},
		},
	}

	mergeManifest(dst, src)

	require.Len(t, dst.Foundation, 2)
	assert.Equal(t, "local", dst.Foundation[0].ID)
	assert.Equal(t, "remote", dst.Foundation[1].ID)
	require.Len(t, dst.Topics, 1)
	assert.Equal(t, "react", dst.Topics[0].ID)
}

func TestMergeManifest_emptySource(t *testing.T) {
	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "local", Path: "foundation/local.md", Description: "Local"},
		},
	}
	src := &manifest.Manifest{}

	mergeManifest(dst, src)

	require.Len(t, dst.Foundation, 1)
	assert.Equal(t, "local", dst.Foundation[0].ID)
}

// --- toSet ---

func TestToSet(t *testing.T) {
	s := toSet([]string{"a", "b", "c"})
	assert.True(t, s["a"])
	assert.True(t, s["b"])
	assert.True(t, s["c"])
	assert.False(t, s["d"])
}

func TestToSet_empty(t *testing.T) {
	s := toSet([]string{})
	assert.Len(t, s, 0)
}
