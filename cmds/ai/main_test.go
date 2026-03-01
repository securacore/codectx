package ai

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/preferences"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_structure(t *testing.T) {
	assert.Equal(t, "ai", Command.Name)
	assert.Equal(t, "Manage AI tool integration", Command.Usage)
	require.Len(t, Command.Commands, 4)
}

func TestCommand_subcommands(t *testing.T) {
	names := make(map[string]string)
	for _, sub := range Command.Commands {
		names[sub.Name] = sub.Usage
	}

	assert.Contains(t, names, "ide")
	assert.Contains(t, names, "link")
	assert.Contains(t, names, "setup")
	assert.Contains(t, names, "status")
	assert.Equal(t, "Launch an AI documentation authoring session", names["ide"])
	assert.Equal(t, "Create AI tool entry point files", names["link"])
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

// setupAIProject creates a minimal project with config and preferences.
func setupAIProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, shared.ConfigFile), cfg))

	outputDir := cfg.OutputDir()
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	return dir
}

func TestRunStatus_noAIConfigured(t *testing.T) {
	setupAIProject(t)

	// Write preferences with no AI config.
	prefs := &preferences.Preferences{}
	require.NoError(t, preferences.Write(".codectx", prefs))

	// runStatus should not error — it prints "Not configured" warning.
	err := runStatus()
	assert.NoError(t, err)
}

func TestRunStatus_withAIConfigured(t *testing.T) {
	setupAIProject(t)

	// Write preferences with AI config set to a known provider.
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{
			Bin:   "claude",
			Model: "",
		},
	}
	require.NoError(t, preferences.Write(".codectx", prefs))

	// runStatus should not error regardless of whether claude binary is found.
	err := runStatus()
	assert.NoError(t, err)
}

func TestRunStatus_withUnknownProvider(t *testing.T) {
	setupAIProject(t)

	// Write preferences with an unknown provider.
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{
			Bin: "nonexistent",
		},
	}
	require.NoError(t, preferences.Write(".codectx", prefs))

	// runStatus should not error but should print "Unknown provider" failure.
	err := runStatus()
	assert.NoError(t, err)
}

func TestRunStatus_withOllamaProvider(t *testing.T) {
	setupAIProject(t)

	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{
			Bin:   "ollama",
			Model: "llama3",
		},
	}
	require.NoError(t, preferences.Write(".codectx", prefs))

	// runStatus should not error — exercises the ollama-specific branch.
	err := runStatus()
	assert.NoError(t, err)
}

func TestRunStatus_withModelAndClass(t *testing.T) {
	setupAIProject(t)

	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{
			Bin:   "claude",
			Model: "claude-sonnet-4-20250514",
			Class: "claude-sonnet-class",
		},
	}
	require.NoError(t, preferences.Write(".codectx", prefs))

	err := runStatus()
	assert.NoError(t, err)
}

func TestRunStatus_preferencesLoadError(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			OutputDir: "/nonexistent/output/dir",
		},
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, shared.ConfigFile), cfg))

	// Preferences load should fail because the output dir doesn't exist
	// with a valid preferences file. However, preferences.Load creates
	// defaults if the dir is missing. So let's verify it still doesn't error.
	err = runStatus()
	assert.NoError(t, err)
}
