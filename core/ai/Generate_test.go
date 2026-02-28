package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- truncateResponse ---

func TestTruncateResponse_shortSentence(t *testing.T) {
	input := "This is a complete sentence. And more text follows after it."
	result := truncateResponse(input)
	assert.Equal(t, "This is a complete sentence.", result)
}

func TestTruncateResponse_noSentenceBoundary(t *testing.T) {
	input := "No period here"
	result := truncateResponse(input)
	assert.Equal(t, "No period here", result)
}

func TestTruncateResponse_veryLongNoPeriod(t *testing.T) {
	// 350 chars, no period — should truncate at 300.
	long := ""
	for len(long) < 350 {
		long += "abcdefghij"
	}
	result := truncateResponse(long)
	assert.Len(t, result, 300)
}

func TestTruncateResponse_periodAfter300(t *testing.T) {
	// Period exists but after the 300-char boundary — should truncate at 300.
	long := ""
	for len(long) < 310 {
		long += "abcdefghij"
	}
	long += "."
	result := truncateResponse(long)
	assert.Len(t, result, 300)
}

func TestTruncateResponse_periodTooEarly(t *testing.T) {
	// Period at position 5 (i <= 10) — should NOT be treated as sentence end.
	input := "Hi. This is a longer description without another period"
	result := truncateResponse(input)
	assert.Equal(t, input, result)
}

func TestTruncateResponse_emptyString(t *testing.T) {
	result := truncateResponse("")
	assert.Equal(t, "", result)
}

func TestTruncateResponse_exactlyOneSentence(t *testing.T) {
	input := "A codectx package for React documentation."
	result := truncateResponse(input)
	assert.Equal(t, "A codectx package for React documentation.", result)
}

// --- Generate: error cases ---

func TestGenerate_unsupportedBinary(t *testing.T) {
	_, err := Generate("unsupported-binary", "test prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found on PATH")
}

func TestGenerate_missingBinary(t *testing.T) {
	// Use a binary name that definitely doesn't exist on PATH.
	_, err := Generate("claude", "test prompt")
	// This will fail with either "not found on PATH" or "unsupported"
	// depending on whether claude is installed. In CI it won't be.
	if err != nil {
		// Either the binary isn't found or generation fails — both are acceptable.
		assert.Error(t, err)
	}
}

func TestGenerate_emptyBinary(t *testing.T) {
	_, err := Generate("", "test prompt")
	require.Error(t, err)
}

func TestGenerate_existsOnPathButUnsupported(t *testing.T) {
	// "bash" exists on PATH but is not a supported AI binary.
	// This should hit the default case in the switch, not LookPath.
	_, err := Generate("bash", "test prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported AI binary")
}
