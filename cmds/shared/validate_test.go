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

// --- ValidateAIProvider ---

func TestValidateAIProvider_unknown(t *testing.T) {
	err := ValidateAIProvider("chatgpt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AI provider")
	assert.Contains(t, err.Error(), "claude")
	assert.Contains(t, err.Error(), "opencode")
	assert.Contains(t, err.Error(), "ollama")
}

func TestValidateAIProvider_empty(t *testing.T) {
	err := ValidateAIProvider("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AI provider")
}

func TestValidateAIProvider_knownProviders(t *testing.T) {
	// Test each known provider. If the binary is available, it should
	// succeed. If not, it should fail with "binary not found" (not
	// "unknown provider").
	for _, id := range []string{"claude", "opencode", "ollama"} {
		err := ValidateAIProvider(id)
		if err != nil {
			// It's a known provider but binary not found on PATH — that's
			// a legitimate runtime state, not a test failure.
			assert.Contains(t, err.Error(), "binary")
			assert.Contains(t, err.Error(), "not found")
			assert.NotContains(t, err.Error(), "unknown AI provider")
		}
		// If no error, the binary was found on PATH.
	}
}

func TestValidateAIProvider_errorFormat(t *testing.T) {
	// Unknown provider error should list all known providers.
	err := ValidateAIProvider("nonexistent")
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "unknown AI provider")
	assert.Contains(t, msg, "nonexistent")
	assert.Contains(t, msg, "claude")
	assert.Contains(t, msg, "opencode")
	assert.Contains(t, msg, "ollama")
}
