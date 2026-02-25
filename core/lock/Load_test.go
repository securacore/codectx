package lock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_fileNotFound(t *testing.T) {
	l, err := Load("/nonexistent/codectx.lock")
	assert.NoError(t, err)
	assert.Nil(t, l)
}

func TestLoad_validLockFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")

	content := `compiled_at: "2025-01-15"
packages:
    - name: react
      author: org
      version: "1.2.0"
      active: all
    - name: go
      author: org
      version: "2.0.0"
      active: none
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	l, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, l)

	assert.Equal(t, "2025-01-15", l.CompiledAt)
	require.Len(t, l.Packages, 2)

	assert.Equal(t, "react", l.Packages[0].Name)
	assert.Equal(t, "org", l.Packages[0].Author)
	assert.Equal(t, "1.2.0", l.Packages[0].Version)
	assert.True(t, l.Packages[0].Active.IsAll())

	assert.Equal(t, "go", l.Packages[1].Name)
	assert.True(t, l.Packages[1].Active.IsNone())
}

func TestLoad_invalidYaml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")
	// Tabs are not valid YAML indentation in mapping context.
	require.NoError(t, os.WriteFile(path, []byte("compiled_at:\n\t- invalid\n\t  indentation"), 0o644))

	_, err := Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse lock")
}

func TestLoad_emptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	l, err := Load(path)
	require.NoError(t, err)
	// Empty YAML unmarshals to zero value.
	require.NotNil(t, l)
	assert.Empty(t, l.Packages)
}

func TestLoad_permissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denied as root")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")
	require.NoError(t, os.WriteFile(path, []byte("compiled_at: now"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	l, err := Load(path)
	assert.Error(t, err)
	assert.Nil(t, l)
	assert.Contains(t, err.Error(), "read lock")
}

func TestLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")

	original := &Lock{
		CompiledAt: "2025-03-01",
		Packages: []LockedPackage{
			{Name: "ts", Author: "org", Version: "3.0.0"},
		},
	}

	require.NoError(t, Write(path, original))

	loaded, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, original.CompiledAt, loaded.CompiledAt)
	require.Len(t, loaded.Packages, 1)
	assert.Equal(t, "ts", loaded.Packages[0].Name)
	assert.Equal(t, "org", loaded.Packages[0].Author)
	assert.Equal(t, "3.0.0", loaded.Packages[0].Version)
}
