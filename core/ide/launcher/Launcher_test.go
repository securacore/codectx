package launcher

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/preferences"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_withBinPreference(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{Bin: "claude"},
	}
	l, err := Resolve(prefs)
	// Succeeds if claude is on PATH, fails with "not found" if not.
	if err != nil {
		assert.Contains(t, err.Error(), "not found")
	} else {
		assert.Equal(t, "claude", l.ID())
	}
}

func TestResolve_nilPreferences(t *testing.T) {
	l, err := Resolve(nil)
	// Auto-detect: either finds claude/opencode or returns "no supported AI binary".
	if err != nil {
		assert.Contains(t, err.Error(), "no supported AI binary")
	} else {
		require.NotNil(t, l)
		assert.Contains(t, []string{"claude", "opencode"}, l.ID())
	}
}

func TestResolve_emptyBin(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{Bin: ""},
	}
	l, err := Resolve(prefs)
	// Falls through to auto-detect.
	if err != nil {
		assert.Contains(t, err.Error(), "no supported AI binary")
	} else {
		require.NotNil(t, l)
	}
}

func TestResolve_unknownBin(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{Bin: "chatgpt"},
	}
	_, err := Resolve(prefs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AI binary")
}

func TestResolve_ollama(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{Bin: "ollama"},
	}
	_, err := Resolve(prefs)
	// Ollama is a known provider but not supported for interactive sessions.
	// On systems where ollama is on PATH, this hits the "not supported" default branch.
	// On systems where it's not on PATH, this hits the "not found" error.
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "not supported for interactive sessions") ||
			strings.Contains(err.Error(), "not found"),
		"expected 'not supported' or 'not found', got: %s", err.Error(),
	)
}
