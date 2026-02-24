package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_defaultValue(t *testing.T) {
	assert.Equal(t, "dev", Version)
}

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "version", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}
