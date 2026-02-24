package compile

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
)

func TestGenerateReadme_noSections(t *testing.T) {
	m := &manifest.Manifest{
		Name: "empty-project",
	}

	result := generateReadme(m, nil)

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

	result := generateReadme(m, nil)

	assert.Contains(t, result, "# full-project")
	assert.Contains(t, result, "## Loading Protocol")
	assert.Contains(t, result, "## Sections")
	assert.Contains(t, result, "**Foundation**: 2 documents")
	assert.Contains(t, result, "**Topics**: 1 entry")
	assert.Contains(t, result, "**Prompts**: 1 entry")
	assert.Contains(t, result, "**Plans**: 1 entry")
	assert.Contains(t, result, "Implementation plans with state tracking")
}

func TestGenerateReadme_partialSections(t *testing.T) {
	m := &manifest.Manifest{
		Name: "partial-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md", Description: "A"},
		},
	}

	result := generateReadme(m, nil)

	assert.Contains(t, result, "## Sections")
	assert.Contains(t, result, "**Foundation**: 1 document")
	assert.NotContains(t, result, "**Topics**")
	assert.NotContains(t, result, "**Prompts**")
	assert.NotContains(t, result, "**Plans**")
}

func TestGenerateReadme_loadingProtocol(t *testing.T) {
	m := &manifest.Manifest{Name: "test"}
	result := generateReadme(m, nil)

	// Verify all 4 loading steps.
	assert.Contains(t, result, "1. Load this file (done).")
	assert.Contains(t, result, "2. Load [manifest.yml](manifest.yml)")
	assert.Contains(t, result, "3. Load all foundation entries")
	assert.Contains(t, result, "4. As the task progresses")
}

func TestGenerateReadme_startsWithH1(t *testing.T) {
	m := &manifest.Manifest{Name: "my-project"}
	result := generateReadme(m, nil)
	assert.True(t, strings.HasPrefix(result, "# my-project\n"))
}

func TestGenerateReadme_withHeuristics(t *testing.T) {
	m := &manifest.Manifest{
		Name: "heuristics-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md", Description: "A", Load: "always"},
			{ID: "b", Path: "foundation/b.md", Description: "B"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "c", Path: "topics/c/README.md", Description: "C"},
		},
	}

	h := &Heuristics{
		Totals: HeuristicsTotals{
			Entries:         3,
			Objects:         3,
			SizeBytes:       12000,
			EstimatedTokens: 3000,
			AlwaysLoad:      1,
		},
		Sections: HeuristicsSections{
			Foundation: &SectionStats{
				Entries:         2,
				SizeBytes:       8000,
				EstimatedTokens: 2000,
				AlwaysLoad:      1,
			},
			Topics: &SectionStats{
				Entries:         1,
				SizeBytes:       4000,
				EstimatedTokens: 1000,
			},
		},
	}

	result := generateReadme(m, h)

	// Token estimates appear in section lines.
	assert.Contains(t, result, "~2k tokens")
	assert.Contains(t, result, "~1k tokens")
	assert.Contains(t, result, "1 is auto-loaded")
	assert.Contains(t, result, "Total documentation: ~3k tokens across 3 objects")
}

func TestGenerateReadme_withHeuristicsMultipleAlwaysLoad(t *testing.T) {
	m := &manifest.Manifest{
		Name: "multi-always",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md", Load: "always"},
			{ID: "b", Path: "foundation/b.md", Load: "always"},
		},
	}

	h := &Heuristics{
		Totals: HeuristicsTotals{
			Entries:         2,
			Objects:         2,
			SizeBytes:       4000,
			EstimatedTokens: 1000,
			AlwaysLoad:      2,
		},
		Sections: HeuristicsSections{
			Foundation: &SectionStats{
				Entries:         2,
				SizeBytes:       4000,
				EstimatedTokens: 1000,
				AlwaysLoad:      2,
			},
		},
	}

	result := generateReadme(m, h)
	assert.Contains(t, result, "2 are auto-loaded")
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
