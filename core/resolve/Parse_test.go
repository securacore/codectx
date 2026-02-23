package resolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_nameOnly(t *testing.T) {
	ref, err := Parse("react")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Empty(t, ref.Author)
	assert.Empty(t, ref.Version)
}

func TestParse_nameAndVersion(t *testing.T) {
	ref, err := Parse("react:^1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Empty(t, ref.Author)
	assert.Equal(t, "^1.0.0", ref.Version)
}

func TestParse_nameAndAuthor(t *testing.T) {
	ref, err := Parse("react@facebook")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Equal(t, "facebook", ref.Author)
	assert.Empty(t, ref.Version)
}

func TestParse_fullyQualified(t *testing.T) {
	ref, err := Parse("react@facebook:^1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "react", ref.Name)
	assert.Equal(t, "facebook", ref.Author)
	assert.Equal(t, "^1.0.0", ref.Version)
}

func TestParse_tildeVersion(t *testing.T) {
	ref, err := Parse("utils@org:~2.1.0")
	require.NoError(t, err)
	assert.Equal(t, "utils", ref.Name)
	assert.Equal(t, "org", ref.Author)
	assert.Equal(t, "~2.1.0", ref.Version)
}

func TestParse_exactVersion(t *testing.T) {
	ref, err := Parse("lib@author:1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", ref.Version)
}

func TestParse_errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty input", ""},
		{"empty version after colon", "react:"},
		{"empty author after at", "react@"},
		{"empty name before at", "@author"},
		{"empty name before colon", ":1.0.0"},
		{"empty author with version", "@author:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			assert.Error(t, err)
		})
	}
}
