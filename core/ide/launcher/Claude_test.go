package launcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaude_implementsLauncher(t *testing.T) {
	var _ Launcher = (*Claude)(nil)
}

func TestNewClaude(t *testing.T) {
	c := NewClaude("/usr/bin/claude")
	assert.Equal(t, "claude", c.ID())
	assert.Equal(t, "/usr/bin/claude", c.Binary())
}

func TestClaude_SupportsSessionID(t *testing.T) {
	c := NewClaude("/usr/bin/claude")
	assert.True(t, c.SupportsSessionID())
}

func TestClaude_NewSessionArgs(t *testing.T) {
	c := NewClaude("/usr/bin/claude")
	args := c.NewSessionArgs("abc-123", "You are a docs assistant")
	assert.Equal(t, []string{
		"--session-id", "abc-123",
		"--append-system-prompt", "You are a docs assistant",
	}, args)
}

func TestClaude_ResumeArgs(t *testing.T) {
	c := NewClaude("/usr/bin/claude")
	args := c.ResumeArgs("abc-123", "You are a docs assistant")
	assert.Equal(t, []string{
		"--resume", "abc-123",
		"--append-system-prompt", "You are a docs assistant",
	}, args)
}

func TestClaude_GenerateSessionID(t *testing.T) {
	c := NewClaude("/usr/bin/claude")
	id := c.GenerateSessionID()
	assert.NotEmpty(t, id)
	assert.Len(t, id, 36) // UUID format: 8-4-4-4-12
}

func TestClaude_GenerateSessionID_unique(t *testing.T) {
	c := NewClaude("/usr/bin/claude")
	id1 := c.GenerateSessionID()
	id2 := c.GenerateSessionID()
	assert.NotEqual(t, id1, id2)
}

func TestClaude_FindLatestSession_returnsEmpty(t *testing.T) {
	c := NewClaude("/usr/bin/claude")
	id, err := c.FindLatestSession("/some/project")
	assert.NoError(t, err)
	assert.Empty(t, id)
}
