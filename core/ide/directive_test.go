package ide

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirective_notEmpty(t *testing.T) {
	d := Directive()
	assert.NotEmpty(t, d)
	assert.Contains(t, d, "documentation authoring assistant")
}

func TestAssemblePrompt_includesDirective(t *testing.T) {
	prompt := AssemblePrompt("", "")
	assert.Contains(t, prompt, "documentation authoring assistant")
	// No extra sections when both are empty.
	assert.NotContains(t, prompt, "Existing Documentation")
	assert.NotContains(t, prompt, "Project Preferences")
}

func TestAssemblePrompt_includesManifest(t *testing.T) {
	prompt := AssemblePrompt("### Foundation\n\n- **philosophy**: Guiding principles\n", "")
	assert.Contains(t, prompt, "Existing Documentation")
	assert.Contains(t, prompt, "philosophy")
	assert.NotContains(t, prompt, "Project Preferences")
}

func TestAssemblePrompt_includesPreferences(t *testing.T) {
	prompt := AssemblePrompt("", "- Compression is **enabled**")
	assert.Contains(t, prompt, "Project Preferences")
	assert.Contains(t, prompt, "Compression is **enabled**")
}

func TestAssemblePrompt_includesBoth(t *testing.T) {
	prompt := AssemblePrompt("### Topics\n\n- **go**: Go conventions\n", "- Model class: gpt-4o-class")
	assert.Contains(t, prompt, "Existing Documentation")
	assert.Contains(t, prompt, "Project Preferences")
	assert.Contains(t, prompt, "go")
	assert.Contains(t, prompt, "gpt-4o-class")
}

func TestFormatPhaseHint(t *testing.T) {
	tests := []struct {
		phase    Phase
		contains string
	}{
		{PhaseDiscover, "Discovering"},
		{PhaseClassify, "Classifying"},
		{PhaseScope, "scope"},
		{PhaseDraft, "Drafting"},
		{PhaseReview, "Reviewing"},
		{PhaseFinalize, "Preparing"},
		{PhaseComplete, "complete"},
	}
	for _, tt := range tests {
		hint := FormatPhaseHint(tt.phase)
		assert.Contains(t, hint, tt.contains)
	}
}
