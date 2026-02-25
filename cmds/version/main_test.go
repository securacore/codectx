package version

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion_defaultValue(t *testing.T) {
	assert.Equal(t, "dev", Version)
}

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "version", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}

func TestCommand_actionPrintsVersion(t *testing.T) {
	// Capture stdout.
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	err = Command.Action(t.Context(), nil)

	_ = w.Close()
	os.Stdout = old

	require.NoError(t, err)

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), Version)
}

func TestCommand_actionPrintsCustomVersion(t *testing.T) {
	original := Version
	Version = "1.2.3"
	t.Cleanup(func() { Version = original })

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	err = Command.Action(t.Context(), nil)

	_ = w.Close()
	os.Stdout = old

	require.NoError(t, err)

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "1.2.3")
}
