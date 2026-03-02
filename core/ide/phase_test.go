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
		p, err := parsePhase(name)
		require.NoError(t, err)
		assert.Equal(t, name, p.String())
	}

	_, err := parsePhase("invalid")
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
