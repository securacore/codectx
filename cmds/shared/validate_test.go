package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ValidateAIClass ---

func TestValidateAIClass_knownIDs(t *testing.T) {
	for _, id := range []string{"gpt-4o-class", "claude-sonnet-class", "o1-class"} {
		err := ValidateAIClass(id)
		assert.NoError(t, err, "should accept known class: %s", id)
	}
}

func TestValidateAIClass_unknown(t *testing.T) {
	err := ValidateAIClass("gpt-3-class")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model class")
	assert.Contains(t, err.Error(), "gpt-4o-class")
	assert.Contains(t, err.Error(), "claude-sonnet-class")
	assert.Contains(t, err.Error(), "o1-class")
}

func TestValidateAIClass_empty(t *testing.T) {
	err := ValidateAIClass("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model class")
}

// --- ValidateAIBin ---

func TestValidateAIBin_unknown(t *testing.T) {
	err := ValidateAIBin("chatgpt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AI provider")
	assert.Contains(t, err.Error(), "claude")
	assert.Contains(t, err.Error(), "opencode")
	assert.Contains(t, err.Error(), "ollama")
}

func TestValidateAIBin_empty(t *testing.T) {
	err := ValidateAIBin("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AI provider")
}

func TestValidateAIBin_knownBinaries(t *testing.T) {
	// Test each known binary. If available on PATH, it should succeed.
	// If not, it should fail with "binary not found" (not "unknown").
	for _, id := range []string{"claude", "opencode", "ollama"} {
		err := ValidateAIBin(id)
		if err != nil {
			// Known binary but not found on PATH — legitimate runtime state.
			assert.Contains(t, err.Error(), "binary")
			assert.Contains(t, err.Error(), "not found")
			assert.NotContains(t, err.Error(), "unknown AI provider")
		}
	}
}

func TestValidateAIBin_errorFormat(t *testing.T) {
	// Unknown binary error should list all known providers.
	err := ValidateAIBin("nonexistent")
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "unknown AI provider")
	assert.Contains(t, msg, "nonexistent")
	assert.Contains(t, msg, "claude")
	assert.Contains(t, msg, "opencode")
	assert.Contains(t, msg, "ollama")
}
