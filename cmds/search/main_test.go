package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "search", Command.Name)
	assert.NotEmpty(t, Command.Usage)
	assert.Equal(t, "<query>", Command.ArgsUsage)
}

func TestCommand_authorFlag(t *testing.T) {
	require := assert.New(t)
	require.Len(Command.Flags, 1)
	flag := Command.Flags[0]
	assert.Equal(t, "author", flag.Names()[0])
}
