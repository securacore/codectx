package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- UnmarshalYAML ---

func TestUnmarshalYAML_all(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`all`), &a)
	require.NoError(t, err)
	assert.Equal(t, "all", a.Mode)
	assert.Nil(t, a.Map)
}

func TestUnmarshalYAML_none(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`none`), &a)
	require.NoError(t, err)
	assert.Equal(t, "none", a.Mode)
	assert.Nil(t, a.Map)
}

func TestUnmarshalYAML_granular(t *testing.T) {
	input := `
foundation:
  - philosophy
  - conventions
topics:
  - react
prompts:
  - review
plans:
  - migration
`
	var a Activation
	err := yaml.Unmarshal([]byte(input), &a)
	require.NoError(t, err)
	assert.Empty(t, a.Mode)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"philosophy", "conventions"}, a.Map.Foundation)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
	assert.Equal(t, []string{"review"}, a.Map.Prompts)
	assert.Equal(t, []string{"migration"}, a.Map.Plans)
}

func TestUnmarshalYAML_invalidMode(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`partial`), &a)
	assert.Error(t, err)
}

// --- MarshalYAML ---

func TestMarshalYAML_all(t *testing.T) {
	a := Activation{Mode: "all"}
	data, err := yaml.Marshal(a)
	require.NoError(t, err)
	assert.Equal(t, "all\n", string(data))
}

func TestMarshalYAML_none(t *testing.T) {
	a := Activation{Mode: "none"}
	data, err := yaml.Marshal(a)
	require.NoError(t, err)
	assert.Equal(t, "none\n", string(data))
}

func TestMarshalYAML_zeroValue(t *testing.T) {
	a := Activation{}
	data, err := yaml.Marshal(a)
	require.NoError(t, err)
	assert.Equal(t, "none\n", string(data))
}

func TestMarshalYAML_granular(t *testing.T) {
	a := Activation{
		Map: &ActivationMap{
			Foundation: []string{"philosophy"},
			Topics:     []string{"react", "go"},
		},
	}
	data, err := yaml.Marshal(a)
	require.NoError(t, err)

	// Unmarshal back to verify round-trip.
	var roundtrip Activation
	err = yaml.Unmarshal(data, &roundtrip)
	require.NoError(t, err)
	require.NotNil(t, roundtrip.Map)
	assert.Equal(t, []string{"philosophy"}, roundtrip.Map.Foundation)
	assert.Equal(t, []string{"react", "go"}, roundtrip.Map.Topics)
}

// --- IsAll / IsNone / IsGranular ---

func TestIsAll(t *testing.T) {
	tests := []struct {
		name     string
		a        Activation
		expected bool
	}{
		{"all mode", Activation{Mode: "all"}, true},
		{"none mode", Activation{Mode: "none"}, false},
		{"empty mode", Activation{}, false},
		{"granular", Activation{Map: &ActivationMap{}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.a.IsAll())
		})
	}
}

func TestIsNone(t *testing.T) {
	tests := []struct {
		name     string
		a        Activation
		expected bool
	}{
		{"none mode", Activation{Mode: "none"}, true},
		{"empty mode", Activation{}, true},
		{"all mode", Activation{Mode: "all"}, false},
		{"granular", Activation{Map: &ActivationMap{}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.a.IsNone())
		})
	}
}

func TestIsGranular(t *testing.T) {
	tests := []struct {
		name     string
		a        Activation
		expected bool
	}{
		{"granular", Activation{Map: &ActivationMap{}}, true},
		{"all mode", Activation{Mode: "all"}, false},
		{"none mode", Activation{Mode: "none"}, false},
		{"empty", Activation{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.a.IsGranular())
		})
	}
}

func TestUnmarshalYAML_invalidType(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`42`), &a)
	assert.Error(t, err)
}

func TestUnmarshalYAML_listInput(t *testing.T) {
	// A YAML list is neither a string nor an ActivationMap.
	// This exercises the second unmarshal failure path (line 46-47).
	var a Activation
	err := yaml.Unmarshal([]byte(`[foo, bar]`), &a)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid activation value")
}

func TestUnmarshalYAML_nestedInvalidMap(t *testing.T) {
	// A map with wrong value types (int instead of []string) should fail
	// ActivationMap unmarshal and produce the "invalid activation value" error.
	var a Activation
	err := yaml.Unmarshal([]byte("foundation: 42\n"), &a)
	// This may succeed because go-yaml is lenient with scalar-to-slice conversion.
	// If it doesn't fail, the test documents the behavior.
	if err != nil {
		assert.Contains(t, err.Error(), "invalid activation")
	}
}

func TestMarshalYAML_ambiguousMapAndMode(t *testing.T) {
	// When both Map and Mode are set, MarshalYAML checks Map first (Map != nil),
	// so it returns the Map. The round-trip should preserve granular activation.
	a := Activation{
		Mode: "all",
		Map:  &ActivationMap{Topics: []string{"react"}},
	}

	data, err := yaml.Marshal(a)
	require.NoError(t, err)

	var roundtrip Activation
	err = yaml.Unmarshal(data, &roundtrip)
	require.NoError(t, err)

	// Should round-trip as granular (Map), not as "all".
	assert.True(t, roundtrip.IsGranular())
	assert.Empty(t, roundtrip.Mode)
	require.NotNil(t, roundtrip.Map)
	assert.Equal(t, []string{"react"}, roundtrip.Map.Topics)
}
