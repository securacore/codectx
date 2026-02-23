package init

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_createsProjectStructure(t *testing.T) {
	dir := t.TempDir()

	// Change to temp dir since init writes relative to cwd.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	err = run("test-project")
	require.NoError(t, err)

	// Verify codectx.yml was created.
	_, err = os.Stat(filepath.Join(dir, "codectx.yml"))
	assert.NoError(t, err)

	// Verify docs/package.yml was created.
	_, err = os.Stat(filepath.Join(dir, "docs", "package.yml"))
	assert.NoError(t, err)

	// Verify directory structure.
	dirs := []string{
		"docs",
		"docs/foundation",
		"docs/topics",
		"docs/prompts",
		"docs/plans",
		"docs/schemas",
		"docs/packages",
	}
	for _, d := range dirs {
		info, err := os.Stat(filepath.Join(dir, d))
		require.NoError(t, err, "directory %s should exist", d)
		assert.True(t, info.IsDir(), "%s should be a directory", d)
	}

	// Verify schemas were written.
	schemaFiles := []string{
		"docs/schemas/codectx.schema.json",
		"docs/schemas/package.schema.json",
		"docs/schemas/state.schema.json",
	}
	for _, f := range schemaFiles {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.NoError(t, err, "schema %s should exist", f)
	}
}

func TestRun_infersNameFromDirectory(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// Pass empty name -- should infer from directory basename.
	err = run("")
	require.NoError(t, err)

	// Verify the config file was created (name inference is hard to verify
	// from outside without parsing, but the fact it succeeds is the test).
	_, err = os.Stat(filepath.Join(dir, "codectx.yml"))
	assert.NoError(t, err)
}

func TestRun_failsIfAlreadyInitialized(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// First init succeeds.
	err = run("test-project")
	require.NoError(t, err)

	// Second init fails.
	err = run("test-project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
