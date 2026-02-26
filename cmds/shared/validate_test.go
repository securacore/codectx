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
