package ide

import (
	"testing"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"
	"github.com/stretchr/testify/assert"
)

func TestBuildManifestSummary_allSections(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Load: "always", Description: "Guiding principles"},
			{ID: "markdown", Load: "documentation", Description: "Markdown conventions", DependsOn: []string{"philosophy"}},
		},
		Topics: []manifest.TopicEntry{
			{ID: "go", Description: "Go conventions", DependsOn: []string{"philosophy", "specs"}},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "save", Description: "Session state persistence"},
		},
	}

	summary := BuildManifestSummary(m)
	assert.Contains(t, summary, "### Foundation")
	assert.Contains(t, summary, "**philosophy** (load:always)")
	assert.Contains(t, summary, "**markdown** (load:documentation)")
	assert.Contains(t, summary, "depends_on: philosophy")
	assert.Contains(t, summary, "### Topics")
	assert.Contains(t, summary, "**go**: Go conventions")
	assert.Contains(t, summary, "depends_on: philosophy, specs")
	assert.Contains(t, summary, "### Prompts")
	assert.Contains(t, summary, "**save**: Session state persistence")
}

func TestBuildManifestSummary_showsDependencies(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "docs", Load: "documentation", Description: "Docs", DependsOn: []string{"philosophy", "markdown"}},
		},
	}

	summary := BuildManifestSummary(m)
	assert.Contains(t, summary, "depends_on: philosophy, markdown")
}

func TestBuildManifestSummary_emptyManifest(t *testing.T) {
	m := &manifest.Manifest{}
	summary := BuildManifestSummary(m)
	assert.Equal(t, "No existing documentation.", summary)
}

func TestBuildManifestSummary_nilManifest(t *testing.T) {
	summary := BuildManifestSummary(nil)
	assert.Equal(t, "No existing documentation.", summary)
}

func TestBuildPreferencesContext_compressionEnabled(t *testing.T) {
	enabled := true
	p := &preferences.Preferences{Compression: &enabled}
	ctx := BuildPreferencesContext(p)
	assert.Contains(t, ctx, "Compression is **enabled**")
}

func TestBuildPreferencesContext_compressionDisabled(t *testing.T) {
	disabled := false
	p := &preferences.Preferences{Compression: &disabled}
	ctx := BuildPreferencesContext(p)
	assert.Contains(t, ctx, "Compression is **disabled**")
}

func TestBuildPreferencesContext_modelClass(t *testing.T) {
	p := &preferences.Preferences{
		AI: &preferences.AIConfig{Class: "gpt-4o-class"},
	}
	ctx := BuildPreferencesContext(p)
	assert.Contains(t, ctx, "gpt-4o-class")
}

func TestBuildPreferencesContext_nil(t *testing.T) {
	ctx := BuildPreferencesContext(nil)
	assert.Empty(t, ctx)
}

func TestBuildPreferencesContext_empty(t *testing.T) {
	p := &preferences.Preferences{}
	ctx := BuildPreferencesContext(p)
	assert.Empty(t, ctx)
}
