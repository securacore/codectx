package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyze_apiDocs(t *testing.T) {
	input := readTestdata("api_docs.md")
	stats, err := Analyze(input)
	require.NoError(t, err)

	assert.Greater(t, stats.OriginalBytes, 0)
	assert.Greater(t, stats.CompressedBytes, 0)
	assert.Greater(t, stats.EstTokensBefore, 0)
	assert.Greater(t, stats.EstTokensAfter, 0)
	// Compact markdown normalizes the document; savings vary by content.
	// Just verify the counts are reasonable and non-zero.
}

func TestAnalyze_emptyDocument(t *testing.T) {
	stats, err := Analyze([]byte(""))
	require.NoError(t, err)

	assert.Equal(t, 0, stats.OriginalBytes)
	assert.Equal(t, 0, stats.EstTokensBefore)
}

func TestAnalyze_tokenSavingsSign(t *testing.T) {
	// Verify that TokenSavings is computed correctly regardless of sign.
	input := readTestdata("api_docs.md")
	stats, err := Analyze(input)
	require.NoError(t, err)

	// Manually verify: savings = (before - after) / before * 100
	if stats.EstTokensBefore > 0 {
		expected := float64(stats.EstTokensBefore-stats.EstTokensAfter) /
			float64(stats.EstTokensBefore) * 100
		assert.InDelta(t, expected, stats.TokenSavings, 0.001,
			"token savings should match manual calculation")
	}
}

func TestAnalyze_proseDocument(t *testing.T) {
	input := readTestdata("prose.md")
	stats, err := Analyze(input)
	require.NoError(t, err)

	// Prose-heavy content should still produce valid stats.
	assert.Greater(t, stats.OriginalBytes, 0)
	assert.Greater(t, stats.CompressedBytes, 0)
}

func TestAnalyze_byteSavingsCalculation(t *testing.T) {
	input := readTestdata("api_docs.md")
	stats, err := Analyze(input)
	require.NoError(t, err)

	// Verify byte savings calculation.
	if stats.OriginalBytes > 0 {
		expected := float64(stats.OriginalBytes-stats.CompressedBytes) /
			float64(stats.OriginalBytes) * 100
		assert.InDelta(t, expected, stats.ByteSavings, 0.001,
			"byte savings should match manual calculation")
	}
}
