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

	result := generateReadme(m, nil, nil, "")

	assert.Contains(t, result, "# empty-project")
	assert.Contains(t, result, "## Loading Protocol")
	assert.NotContains(t, result, "## Sections")
}

func TestGenerateReadme_allSections(t *testing.T) {
	m := &manifest.Manifest{
		Name: "full-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a/README.md", Description: "A"},
			{ID: "b", Path: "foundation/b/README.md", Description: "B"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md", Description: "Architecture"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "c", Path: "topics/c/README.md", Description: "C"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "d", Path: "prompts/d/README.md", Description: "D"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "e", Path: "plans/e/README.md", PlanState: "plans/e/plan.yml", Description: "E"},
		},
	}

	result := generateReadme(m, nil, nil, "")

	assert.Contains(t, result, "# full-project")
	assert.Contains(t, result, "## Loading Protocol")
	assert.Contains(t, result, "## Sections")
	assert.Contains(t, result, "**Foundation**: 2 documents")
	assert.Contains(t, result, "**Application**: 1 entry")
	assert.Contains(t, result, "**Topics**: 1 entry")
	assert.Contains(t, result, "**Prompts**: 1 entry")
	assert.Contains(t, result, "**Plans**: 1 entry")
	assert.Contains(t, result, "Implementation plans with state tracking")
}

func TestGenerateReadme_partialSections(t *testing.T) {
	m := &manifest.Manifest{
		Name: "partial-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a/README.md", Description: "A"},
		},
	}

	result := generateReadme(m, nil, nil, "")

	assert.Contains(t, result, "## Sections")
	assert.Contains(t, result, "**Foundation**: 1 document")
	assert.NotContains(t, result, "**Application**")
	assert.NotContains(t, result, "**Topics**")
	assert.NotContains(t, result, "**Prompts**")
	assert.NotContains(t, result, "**Plans**")
}

func TestGenerateReadme_loadingProtocol(t *testing.T) {
	m := &manifest.Manifest{Name: "test"}
	result := generateReadme(m, nil, nil, "")

	// Verify blocking preamble.
	assert.Contains(t, result, "Complete steps 1-3 before responding to any user message")
	assert.Contains(t, result, "Do not answer questions, write code, or provide guidance until the required context is loaded")

	// Verify all 4 loading steps.
	assert.Contains(t, result, "1. Load this file (done).")
	assert.Contains(t, result, "2. Load [manifest.yml](manifest.yml)")
	assert.Contains(t, result, "3. Load all foundation entries")
	assert.Contains(t, result, "load both the `object` and its `spec`")
	assert.Contains(t, result, "4. Before responding to a user message, check the manifest for topics")
}

func TestGenerateReadme_startsWithH1(t *testing.T) {
	m := &manifest.Manifest{Name: "my-project"}
	result := generateReadme(m, nil, nil, "")
	assert.True(t, strings.HasPrefix(result, "# my-project\n"))
}

func TestGenerateReadme_withHeuristics(t *testing.T) {
	m := &manifest.Manifest{
		Name: "heuristics-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a/README.md", Description: "A", Load: "always"},
			{ID: "b", Path: "foundation/b/README.md", Description: "B"},
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

	result := generateReadme(m, h, nil, "")

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
			{ID: "a", Path: "foundation/a/README.md", Load: "always"},
			{ID: "b", Path: "foundation/b/README.md", Load: "always"},
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

	result := generateReadme(m, h, nil, "")
	assert.Contains(t, result, "2 are auto-loaded")
}

func TestGenerateReadme_applicationOnly(t *testing.T) {
	m := &manifest.Manifest{
		Name: "app-only",
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md", Description: "Architecture"},
		},
	}

	result := generateReadme(m, nil, nil, "")

	assert.Contains(t, result, "## Sections")
	assert.Contains(t, result, "**Application**: 1 entry")
	assert.Contains(t, result, "Product architecture and design documentation")
	assert.NotContains(t, result, "**Foundation**")
	assert.NotContains(t, result, "**Topics**")
	assert.NotContains(t, result, "**Prompts**")
	assert.NotContains(t, result, "**Plans**")
}

// --- required context ---

func TestGenerateReadme_requiredContext_withAlwaysLoad(t *testing.T) {
	m := &manifest.Manifest{
		Name: "ctx-project",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Spec: "foundation/philosophy/spec/README.md", Load: "always"},
			{ID: "markdown", Path: "foundation/markdown/README.md", Spec: "foundation/markdown/spec/README.md", Load: "documentation"},
		},
	}

	pathToHash := map[string]string{
		"foundation/philosophy/README.md":      "abc123def456",
		"foundation/philosophy/spec/README.md": "789ghi012jkl",
		"foundation/markdown/README.md":        "mno345pqr678",
		"foundation/markdown/spec/README.md":   "stu901vwx234",
	}

	result := generateReadme(m, nil, pathToHash, ".cmdx")

	assert.Contains(t, result, "## Required Context")
	assert.Contains(t, result, "Load these files now. Do not skip this step.")
	assert.Contains(t, result, "### philosophy")
	assert.Contains(t, result, "objects/abc123def456.cmdx")
	assert.Contains(t, result, "objects/789ghi012jkl.cmdx")
	// documentation-load entries should not appear in required context.
	assert.NotContains(t, result, "### markdown")
	assert.NotContains(t, result, "objects/mno345pqr678.cmdx")
}

func TestGenerateReadme_requiredContext_noAlwaysLoad(t *testing.T) {
	m := &manifest.Manifest{
		Name: "no-always",
		Foundation: []manifest.FoundationEntry{
			{ID: "markdown", Path: "foundation/markdown/README.md", Load: "documentation"},
		},
	}

	pathToHash := map[string]string{
		"foundation/markdown/README.md": "abc123def456",
	}

	result := generateReadme(m, nil, pathToHash, ".md")

	assert.NotContains(t, result, "## Required Context")
}

func TestGenerateReadme_requiredContext_nilPathToHash(t *testing.T) {
	m := &manifest.Manifest{
		Name: "nil-hash",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Load: "always"},
		},
	}

	result := generateReadme(m, nil, nil, "")

	assert.NotContains(t, result, "## Required Context")
}

func TestGenerateReadme_requiredContext_multipleAlwaysLoad(t *testing.T) {
	m := &manifest.Manifest{
		Name: "multi-ctx",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Spec: "foundation/philosophy/spec/README.md", Load: "always"},
			{ID: "architecture", Path: "foundation/architecture/README.md", Load: "always"},
		},
	}

	pathToHash := map[string]string{
		"foundation/philosophy/README.md":      "aaa111bbb222",
		"foundation/philosophy/spec/README.md": "ccc333ddd444",
		"foundation/architecture/README.md":    "eee555fff666",
	}

	result := generateReadme(m, nil, pathToHash, ".md")

	assert.Contains(t, result, "### philosophy")
	assert.Contains(t, result, "objects/aaa111bbb222.md")
	assert.Contains(t, result, "objects/ccc333ddd444.md")
	assert.Contains(t, result, "### architecture")
	assert.Contains(t, result, "objects/eee555fff666.md")
}

func TestGenerateReadme_requiredContext_noSpec(t *testing.T) {
	m := &manifest.Manifest{
		Name: "no-spec",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Load: "always"},
		},
	}

	pathToHash := map[string]string{
		"foundation/philosophy/README.md": "abc123def456",
	}

	result := generateReadme(m, nil, pathToHash, ".cmdx")

	assert.Contains(t, result, "### philosophy")
	assert.Contains(t, result, "objects/abc123def456.cmdx")
	assert.Contains(t, result, "(object)")
	assert.NotContains(t, result, "(spec)")
}

func TestGenerateReadme_requiredContext_objectHashMiss(t *testing.T) {
	m := &manifest.Manifest{
		Name: "hash-miss",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Spec: "foundation/philosophy/spec/README.md", Load: "always"},
		},
	}

	// pathToHash exists but does not contain the entry's Path.
	pathToHash := map[string]string{
		"foundation/philosophy/spec/README.md": "spechashabc123",
	}

	result := generateReadme(m, nil, pathToHash, ".cmdx")

	// Section header and spec should appear; object line should be skipped.
	assert.Contains(t, result, "## Required Context")
	assert.Contains(t, result, "### philosophy")
	assert.NotContains(t, result, "(object)")
	assert.Contains(t, result, "(spec)")
	assert.Contains(t, result, "objects/spechashabc123.cmdx")
}

func TestGenerateReadme_requiredContext_specHashMiss(t *testing.T) {
	m := &manifest.Manifest{
		Name: "spec-miss",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Spec: "foundation/philosophy/spec/README.md", Load: "always"},
		},
	}

	// pathToHash contains the object but not the spec.
	pathToHash := map[string]string{
		"foundation/philosophy/README.md": "objhashabc123",
	}

	result := generateReadme(m, nil, pathToHash, ".md")

	assert.Contains(t, result, "## Required Context")
	assert.Contains(t, result, "### philosophy")
	assert.Contains(t, result, "(object)")
	assert.Contains(t, result, "objects/objhashabc123.md")
	// Spec is declared but hash is missing, so spec line should be skipped.
	assert.NotContains(t, result, "(spec)")
}

func TestGenerateReadme_requiredContext_emptyPathToHash(t *testing.T) {
	m := &manifest.Manifest{
		Name: "empty-hash",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Load: "always"},
		},
	}

	// Non-nil but empty map — no hashes resolve.
	pathToHash := map[string]string{}

	result := generateReadme(m, nil, pathToHash, ".cmdx")

	// Section header appears (there IS an always-load entry), but no file lines.
	assert.Contains(t, result, "## Required Context")
	assert.Contains(t, result, "### philosophy")
	assert.NotContains(t, result, "(object)")
	assert.NotContains(t, result, "(spec)")
}

func TestGenerateReadme_requiredContext_withCompression(t *testing.T) {
	m := &manifest.Manifest{
		Name: "compressed-ctx",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Spec: "foundation/philosophy/spec/README.md", Load: "always"},
		},
	}

	pathToHash := map[string]string{
		"foundation/philosophy/README.md":      "abc123def456",
		"foundation/philosophy/spec/README.md": "789ghi012jkl",
	}

	result := generateReadme(m, nil, pathToHash, ".cmdx", true)

	// Both CMDX format note and Required Context should be present.
	assert.Contains(t, result, "CMDX compression")
	assert.Contains(t, result, "## Required Context")
	assert.Contains(t, result, "objects/abc123def456.cmdx")
	assert.Contains(t, result, "objects/789ghi012jkl.cmdx")
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
