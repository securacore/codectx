package shared

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureGitignoreEntry_createsNew(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	err := EnsureGitignoreEntry(path, ".codectx/")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, ".codectx/\n", string(data))
}

func TestEnsureGitignoreEntry_alreadyPresent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(path, []byte(".codectx/\n"), 0o644))

	err := EnsureGitignoreEntry(path, ".codectx/")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, ".codectx/\n", string(data))
}

func TestEnsureGitignoreEntry_appendsToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(path, []byte("node_modules/\n"), 0o644))

	err := EnsureGitignoreEntry(path, ".codectx/")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "node_modules/")
	assert.Contains(t, string(data), ".codectx/")
}

func TestEnsureGitignoreEntry_appendsNewlineIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(path, []byte("node_modules/"), 0o644))

	err := EnsureGitignoreEntry(path, ".codectx/")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "node_modules/\n.codectx/")
}

func TestEnsureGitignoreEntry_emptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	err := EnsureGitignoreEntry(path, ".codectx/")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), ".codectx/")
}
