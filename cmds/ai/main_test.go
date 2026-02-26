package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_structure(t *testing.T) {
	assert.Equal(t, "ai", Command.Name)
	assert.Equal(t, "Manage AI tool integration", Command.Usage)
	require.Len(t, Command.Commands, 2)
}

func TestCommand_subcommands(t *testing.T) {
	names := make(map[string]string)
	for _, sub := range Command.Commands {
		names[sub.Name] = sub.Usage
	}

	assert.Contains(t, names, "setup")
	assert.Contains(t, names, "status")
	assert.Equal(t, "Detect and configure AI tool integration", names["setup"])
	assert.Equal(t, "Show AI integration status and detected tools", names["status"])
}

func TestCommand_setupRequiresConfig(t *testing.T) {
	// runSetup should fail when no codectx.yml is present.
	err := runSetup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestCommand_statusRequiresConfig(t *testing.T) {
	// runStatus should fail when no codectx.yml is present.
	err := runStatus()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}
