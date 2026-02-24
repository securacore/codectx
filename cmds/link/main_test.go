package link

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "link", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}

func TestRun_missingConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// No codectx.yml exists.
	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_invalidConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Write invalid YAML.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, configFile),
		[]byte("{{{{not valid"), 0o644))

	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_missingCompiledOutput(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Write valid codectx.yml but no compiled output.
	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compiled output not found")
	assert.Contains(t, err.Error(), "codectx compile")
}

func TestRun_missingCompiledOutputCustomDir(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	customOutput := filepath.Join(dir, "custom-output")
	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			OutputDir: customOutput,
		},
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compiled output not found")
	assert.Contains(t, err.Error(), customOutput)
}
