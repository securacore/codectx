package compile

import (
	"strings"
	"testing"

	"securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
)

func TestGenerateReadme_noSections(t *testing.T) {
	m := &manifest.Manifest{
		Name: "empty-project",
	}

	result := generateReadme(m)

	assert.Contains(t, result, "# empty-project")
	assert.Contains(t, result, "## Loading Protocol")
	assert.NotContains(t, result, "## Sections")
}

func TestGenerateReadme_allSections(t *testing.T) {
	m := &manifest.Manifest{
		Name: "full-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md", Description: "A"},
			{ID: "b", Path: "foundation/b.md", Description: "B"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "c", Path: "topics/c/README.md", Description: "C"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "d", Path: "prompts/d/README.md", Description: "D"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "e", Path: "plans/e/README.md", State: "plans/e/state.yml", Description: "E"},
		},
	}

	result := generateReadme(m)

	assert.Contains(t, result, "# full-project")
	assert.Contains(t, result, "## Loading Protocol")
	assert.Contains(t, result, "## Sections")
	assert.Contains(t, result, "**Foundation**: 2 documents")
	assert.Contains(t, result, "**Topics**: 1 entry")
	assert.Contains(t, result, "**Prompts**: 1 entry")
	assert.Contains(t, result, "**Plans**: 1 entry")
	assert.Contains(t, result, "Read `state.yml` before loading full plans")
}

func TestGenerateReadme_partialSections(t *testing.T) {
	m := &manifest.Manifest{
		Name: "partial-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md", Description: "A"},
		},
	}

	result := generateReadme(m)

	assert.Contains(t, result, "## Sections")
	assert.Contains(t, result, "**Foundation**: 1 document")
	assert.NotContains(t, result, "**Topics**")
	assert.NotContains(t, result, "**Prompts**")
	assert.NotContains(t, result, "**Plans**")
}

func TestGenerateReadme_loadingProtocol(t *testing.T) {
	m := &manifest.Manifest{Name: "test"}
	result := generateReadme(m)

	// Verify all 4 loading steps.
	assert.Contains(t, result, "1. Load this file (done).")
	assert.Contains(t, result, "2. Load [package.yml](package.yml)")
	assert.Contains(t, result, "3. Load all foundation entries")
	assert.Contains(t, result, "4. As the task progresses")
}

func TestGenerateReadme_startsWithH1(t *testing.T) {
	m := &manifest.Manifest{Name: "my-project"}
	result := generateReadme(m)
	assert.True(t, strings.HasPrefix(result, "# my-project\n"))
}

// --- pluralize ---

func TestPluralize(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		singular string
		plural   string
		expected string
	}{
		{"zero uses plural", 0, "entry", "entries", "entries"},
		{"one uses singular", 1, "entry", "entries", "entry"},
		{"two uses plural", 2, "entry", "entries", "entries"},
		{"large number uses plural", 100, "document", "documents", "documents"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pluralize(tt.n, tt.singular, tt.plural))
		})
	}
}
