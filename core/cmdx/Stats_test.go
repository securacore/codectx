package cmdx

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
	assert.Less(t, stats.CompressedBytes, stats.OriginalBytes, "compressed should be smaller than original")
	assert.Greater(t, stats.ByteSavings, 0.0, "should have positive byte savings")
	assert.Greater(t, stats.DictEntries, 0, "should have dictionary entries for repetitive content")
	assert.Greater(t, stats.DictSavings, 0, "dictionary should save bytes")
	assert.Greater(t, stats.EstTokensBefore, 0)
	assert.Greater(t, stats.EstTokensAfter, 0)
	// Note: token savings may be negative for some documents because CMDX
	// dictionary syntax (@DICT{, $N=, etc.) can tokenize into more BPE
	// tokens than the text it replaces, even when byte savings are positive.
	// This is a known property of the format — compression targets bytes
	// not tokens. We just verify token counts are reasonable.
	assert.NotEqual(t, stats.EstTokensBefore, stats.EstTokensAfter,
		"token counts should differ between original and compressed")
}

// TestStats_apiDocs is the plan-specified test case (Phase 2).
// It validates the same functionality as TestAnalyze_apiDocs with explicit Stats naming.
func TestStats_apiDocs(t *testing.T) {
	input := readTestdata("api_docs.md")
	stats, err := Analyze(input)
	require.NoError(t, err)

	assert.Greater(t, stats.OriginalBytes, 0)
	assert.Greater(t, stats.CompressedBytes, 0)
	assert.Less(t, stats.CompressedBytes, stats.OriginalBytes)
	assert.Greater(t, stats.ByteSavings, 0.0)
	assert.Greater(t, stats.DictEntries, 0)
	assert.Greater(t, stats.DictSavings, 0)
	assert.Greater(t, stats.EstTokensBefore, 0)
	assert.Greater(t, stats.EstTokensAfter, 0)
	// Token savings can be negative with real BPE tokenization — see
	// TestAnalyze_apiDocs for explanation. Just verify counts are non-zero.
}

func TestAnalyze_emptyDocument(t *testing.T) {
	stats, err := Analyze([]byte(""))
	require.NoError(t, err)

	assert.Equal(t, 0, stats.OriginalBytes)
	assert.Equal(t, 0.0, stats.ByteSavings)
	assert.Equal(t, 0, stats.DictEntries)
	assert.Equal(t, 0, stats.DictSavings)
	assert.Equal(t, 0, stats.EstTokensBefore)
}

func TestAnalyze_tokenSavingsSign(t *testing.T) {
	// With real BPE tokenization, CMDX compression may produce negative
	// token savings because dictionary syntax tokenizes poorly. Verify
	// that TokenSavings is computed correctly regardless of sign.
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
	apiInput := readTestdata("api_docs.md")

	proseStats, err := Analyze(input)
	require.NoError(t, err)

	apiStats, err := Analyze(apiInput)
	require.NoError(t, err)

	// Prose-heavy content should have lower savings than API docs.
	assert.Less(t, proseStats.ByteSavings, apiStats.ByteSavings,
		"prose should have lower savings than API docs (prose=%.1f%%, api=%.1f%%)",
		proseStats.ByteSavings, apiStats.ByteSavings)
}
