package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAll_createsAllSchemas(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schemas")

	err := WriteAll(schemaDir)
	require.NoError(t, err)

	for _, name := range schemaFiles {
		path := filepath.Join(schemaDir, name)
		info, err := os.Stat(path)
		require.NoError(t, err, "schema file %s should exist", name)
		assert.Greater(t, info.Size(), int64(0), "schema file %s should not be empty", name)
	}
}

func TestWriteAll_contentMatchesEmbedded(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir)
	require.NoError(t, err)

	for _, name := range schemaFiles {
		// Read the embedded version.
		embedded, err := schemas.ReadFile(name)
		require.NoError(t, err)

		// Read the written version.
		written, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)

		assert.Equal(t, embedded, written, "written schema %s should match embedded", name)
	}
}

func TestWriteAll_createsDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "schemas")

	err := WriteAll(nested)
	require.NoError(t, err)

	info, err := os.Stat(nested)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestWriteAll_failsOnReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denial as root")
	}
	// Use a path nested under a regular file, which makes MkdirAll fail.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocked")
	require.NoError(t, os.WriteFile(blocker, []byte("not a dir"), 0o644))

	err := WriteAll(filepath.Join(blocker, "schemas"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create schema directory")
}

func TestWriteAll_idempotent(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schemas")

	// First call.
	err := WriteAll(schemaDir)
	require.NoError(t, err)

	// Capture file contents after first write.
	firstWrite := make(map[string][]byte)
	for _, name := range schemaFiles {
		data, err := os.ReadFile(filepath.Join(schemaDir, name))
		require.NoError(t, err)
		firstWrite[name] = data
	}

	// Second call should succeed.
	err = WriteAll(schemaDir)
	require.NoError(t, err)

	// Verify files match embedded content and first write.
	for _, name := range schemaFiles {
		data, err := os.ReadFile(filepath.Join(schemaDir, name))
		require.NoError(t, err)
		assert.Equal(t, firstWrite[name], data, "schema file %s should be identical after second write", name)

		embedded, err := schemas.ReadFile(name)
		require.NoError(t, err)
		assert.Equal(t, embedded, data, "schema file %s should match embedded after second write", name)
	}
}
