package add

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseActivateFlag_all(t *testing.T) {
	a, err := parseActivateFlag("all")
	require.NoError(t, err)
	assert.Equal(t, "all", a.Mode)
	assert.True(t, a.IsAll())
	assert.Nil(t, a.Map)
}

func TestParseActivateFlag_none(t *testing.T) {
	a, err := parseActivateFlag("none")
	require.NoError(t, err)
	assert.Equal(t, "none", a.Mode)
	assert.True(t, a.IsNone())
}

func TestParseActivateFlag_singleGranular(t *testing.T) {
	a, err := parseActivateFlag("topics:react")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
}

func TestParseActivateFlag_multipleGranular(t *testing.T) {
	a, err := parseActivateFlag("foundation:philosophy,topics:react,topics:go,plans:migration")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"philosophy"}, a.Map.Foundation)
	assert.Equal(t, []string{"react", "go"}, a.Map.Topics)
	assert.Nil(t, a.Map.Prompts)
	assert.Equal(t, []string{"migration"}, a.Map.Plans)
}

func TestParseActivateFlag_allSections(t *testing.T) {
	a, err := parseActivateFlag("foundation:a,topics:b,prompts:c,plans:d")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"a"}, a.Map.Foundation)
	assert.Equal(t, []string{"b"}, a.Map.Topics)
	assert.Equal(t, []string{"c"}, a.Map.Prompts)
	assert.Equal(t, []string{"d"}, a.Map.Plans)
}

func TestParseActivateFlag_errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		msg   string
	}{
		{"no colon", "topicsreact", "expected section:id"},
		{"empty id", "topics:", "empty id"},
		{"unknown section", "widgets:foo", "unknown section"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseActivateFlag(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}
