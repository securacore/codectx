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

func TestParse_hyphenatedName(t *testing.T) {
	ref, err := Parse("my-lib@org:^1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "my-lib", ref.Name)
	assert.Equal(t, "org", ref.Author)
	assert.Equal(t, "^1.0.0", ref.Version)
}

func TestParse_multipleAtSigns(t *testing.T) {
	// lastIndex finds the LAST @, so "a@b@c" -> name="a@b", author="c"
	ref, err := Parse("a@b@c")
	require.NoError(t, err)
	assert.Equal(t, "a@b", ref.Name)
	assert.Equal(t, "c", ref.Author)
}

func TestParse_multipleColons(t *testing.T) {
	// lastIndex finds the LAST :, so "a:b:c" -> nameAuthor="a:b", version="c"
	ref, err := Parse("a:b:c")
	require.NoError(t, err)
	assert.Equal(t, "a:b", ref.Name)
	assert.Equal(t, "c", ref.Version)
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

// --- lastIndex ---

func TestLastIndex_found(t *testing.T) {
	assert.Equal(t, 5, lastIndex("hello@world", '@'))
}

func TestLastIndex_multiple(t *testing.T) {
	assert.Equal(t, 3, lastIndex("a@b@c", '@'))
}

func TestLastIndex_notFound(t *testing.T) {
	assert.Equal(t, -1, lastIndex("hello", '@'))
}

func TestLastIndex_atStart(t *testing.T) {
	assert.Equal(t, 0, lastIndex("@start", '@'))
}

func TestLastIndex_atEnd(t *testing.T) {
	assert.Equal(t, 3, lastIndex("end@", '@'))
}

func TestLastIndex_emptyString(t *testing.T) {
	assert.Equal(t, -1, lastIndex("", '@'))
}
