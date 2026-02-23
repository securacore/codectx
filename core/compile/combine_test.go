package compile

import (
	"testing"

	"securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
)

func TestCollectFilePaths_allSections(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "b", Path: "topics/b/README.md", Spec: "topics/b/spec/README.md", Files: []string{"topics/b/extra.md"}},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "c", Path: "prompts/c/README.md"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "d", Path: "plans/d/README.md", State: "plans/d/state.yml"},
		},
	}

	paths := collectFilePaths(m)

	expected := []string{
		"foundation/a.md",
		"topics/b/README.md",
		"topics/b/spec/README.md",
		"topics/b/extra.md",
		"prompts/c/README.md",
		"plans/d/README.md",
		"plans/d/state.yml",
	}
	assert.Equal(t, expected, paths)
}

func TestCollectFilePaths_topicWithoutSpec(t *testing.T) {
	m := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "b", Path: "topics/b/README.md"},
		},
	}

	paths := collectFilePaths(m)

	assert.Equal(t, []string{"topics/b/README.md"}, paths)
}

func TestCollectFilePaths_planWithoutState(t *testing.T) {
	m := &manifest.Manifest{
		Plans: []manifest.PlanEntry{
			{ID: "d", Path: "plans/d/README.md"},
		},
	}

	paths := collectFilePaths(m)

	assert.Equal(t, []string{"plans/d/README.md"}, paths)
}

func TestCollectFilePaths_empty(t *testing.T) {
	m := &manifest.Manifest{}
	paths := collectFilePaths(m)
	assert.Nil(t, paths)
}
