package ide

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPhase_String(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected string
	}{
		{PhaseDiscover, "discover"},
		{PhaseClassify, "classify"},
		{PhaseScope, "scope"},
		{PhaseDraft, "draft"},
		{PhaseReview, "review"},
		{PhaseFinalize, "finalize"},
		{PhaseComplete, "complete"},
		{Phase(99), "unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.phase.String())
	}
}

func TestParsePhase(t *testing.T) {
	for _, name := range []string{
		"discover", "classify", "scope", "draft",
		"review", "finalize", "complete",
	} {
		p, err := ParsePhase(name)
		require.NoError(t, err)
		assert.Equal(t, name, p.String())
	}

	_, err := ParsePhase("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown phase")
}

func TestPhase_MarshalYAML(t *testing.T) {
	type wrapper struct {
		Phase Phase `yaml:"phase"`
	}
	w := wrapper{Phase: PhaseDraft}
	data, err := yaml.Marshal(w)
	require.NoError(t, err)
	assert.Contains(t, string(data), "phase: draft")
}

func TestPhase_UnmarshalYAML(t *testing.T) {
	type wrapper struct {
		Phase Phase `yaml:"phase"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte("phase: review"), &w)
	require.NoError(t, err)
	assert.Equal(t, PhaseReview, w.Phase)
}

func TestPhase_UnmarshalYAML_invalid(t *testing.T) {
	type wrapper struct {
		Phase Phase `yaml:"phase"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte("phase: bogus"), &w)
	assert.Error(t, err)
}

func TestPhase_CanTransition_forward(t *testing.T) {
	// All forward transitions are valid.
	assert.True(t, PhaseDiscover.CanTransition(PhaseClassify))
	assert.True(t, PhaseClassify.CanTransition(PhaseScope))
	assert.True(t, PhaseScope.CanTransition(PhaseDraft))
	assert.True(t, PhaseDraft.CanTransition(PhaseReview))
	assert.True(t, PhaseReview.CanTransition(PhaseFinalize))
	assert.True(t, PhaseFinalize.CanTransition(PhaseComplete))

	// Skipping phases is also valid.
	assert.True(t, PhaseDiscover.CanTransition(PhaseDraft))
	assert.True(t, PhaseDiscover.CanTransition(PhaseComplete))
}

func TestPhase_CanTransition_samePhase(t *testing.T) {
	assert.True(t, PhaseDraft.CanTransition(PhaseDraft))
	assert.True(t, PhaseReview.CanTransition(PhaseReview))
}

func TestPhase_CanTransition_allowedBackward(t *testing.T) {
	// Review -> Draft (revisions)
	assert.True(t, PhaseReview.CanTransition(PhaseDraft))
	// Finalize -> Draft (preview rejection)
	assert.True(t, PhaseFinalize.CanTransition(PhaseDraft))
}

func TestPhase_CanTransition_disallowedBackward(t *testing.T) {
	assert.False(t, PhaseDraft.CanTransition(PhaseDiscover))
	assert.False(t, PhaseScope.CanTransition(PhaseClassify))
	assert.False(t, PhaseComplete.CanTransition(PhaseDraft))
	assert.False(t, PhaseReview.CanTransition(PhaseScope))
}
