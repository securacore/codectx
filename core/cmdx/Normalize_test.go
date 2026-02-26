package cmdx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDumpAST_stripsPositions(t *testing.T) {
	input := []byte("# Hello\n\nWorld\n")
	dump, err := DumpAST(input)
	require.NoError(t, err)
	// Should contain structural info but no source positions.
	assert.Contains(t, dump, "Heading[level=1]")
	assert.Contains(t, dump, "Paragraph")
	assert.Contains(t, dump, `Text "Hello"`)
	assert.Contains(t, dump, `Text "World"`)
	// Source positions should not appear.
	assert.NotContains(t, dump, "Start=")
	assert.NotContains(t, dump, "Stop=")
}

func TestDumpAST_stripsAlignment(t *testing.T) {
	input := []byte("| a | b |\n|:--|---:|\n| 1 | 2 |\n")
	dump, err := DumpAST(input)
	require.NoError(t, err)
	// Should contain table structure.
	assert.Contains(t, dump, "Table")
	// Should NOT contain alignment info.
	assert.NotContains(t, dump, "left")
	assert.NotContains(t, dump, "right")
	assert.NotContains(t, dump, "center")
	assert.NotContains(t, dump, "Alignment")
}

func TestCompareASTs_identical(t *testing.T) {
	input := []byte("# Hello\n\nSome **bold** text.\n")
	equal, diff, err := CompareASTs(input, input)
	require.NoError(t, err)
	assert.True(t, equal, "same input should be equal: %s", diff)
}

func TestCompareASTs_different(t *testing.T) {
	a := []byte("# Hello\n\nWorld\n")
	b := []byte("# Hello\n\nDifferent\n")
	equal, diff, err := CompareASTs(a, b)
	require.NoError(t, err)
	assert.False(t, equal)
	assert.NotEmpty(t, diff)
}

func TestCompareASTs_whitespaceNormalized(t *testing.T) {
	a := []byte("# Hello\n\n\n\nWorld\n")
	b := []byte("# Hello\n\nWorld\n")
	equal, _, err := CompareASTs(a, b)
	require.NoError(t, err)
	assert.True(t, equal, "extra blank lines should not matter")
}
