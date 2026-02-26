package set

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/preferences"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupProject creates a minimal project with codectx.yml and optional
// preferences. Returns a cleanup function that restores the original cwd.
func setupProject(t *testing.T, prefs *preferences.Preferences) string {
	t.Helper()
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Write codectx.yml.
	cfg := &config.Config{Name: "test-project"}
	require.NoError(t, config.Write("codectx.yml", cfg))

	// Create .codectx directory and optional preferences.
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	if prefs != nil {
		require.NoError(t, preferences.Write(outputDir, prefs))
	}

	return dir
}

// --- parseBool ---

func TestParseBool_true(t *testing.T) {
	for _, input := range []string{"true", "True", "TRUE", "1", "yes", "Yes"} {
		v, err := parseBool(input)
		require.NoError(t, err, "input: %s", input)
		assert.True(t, v, "input: %s", input)
	}
}

func TestParseBool_false(t *testing.T) {
	for _, input := range []string{"false", "False", "FALSE", "0", "no", "No"} {
		v, err := parseBool(input)
		require.NoError(t, err, "input: %s", input)
		assert.False(t, v, "input: %s", input)
	}
}

func TestParseBool_invalid(t *testing.T) {
	_, err := parseBool("maybe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected true or false")
}

// --- formatBoolPtr ---

func TestFormatBoolPtr_nil(t *testing.T) {
	result := formatBoolPtr(nil)
	assert.Contains(t, result, "unset")
}

func TestFormatBoolPtr_true(t *testing.T) {
	v := true
	assert.Equal(t, "true", formatBoolPtr(&v))
}

func TestFormatBoolPtr_false(t *testing.T) {
	v := false
	assert.Equal(t, "false", formatBoolPtr(&v))
}

// --- readValue ---

func TestReadValue_emptyPrefs(t *testing.T) {
	prefs := &preferences.Preferences{}
	for _, key := range []string{"compression", "auto_compile", "ai.provider", "ai.model", "ai.class"} {
		result := readValue(prefs, key)
		assert.Contains(t, result, "unset", "key: %s", key)
	}
}

func TestReadValue_boolSet(t *testing.T) {
	prefs := &preferences.Preferences{
		Compression: preferences.BoolPtr(true),
		AutoCompile: preferences.BoolPtr(false),
	}
	assert.Equal(t, "true", readValue(prefs, "compression"))
	assert.Equal(t, "false", readValue(prefs, "auto_compile"))
}

func TestReadValue_aiSet(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{
			Provider: "claude",
			Model:    "sonnet",
			Class:    "gpt-4o-class",
		},
	}
	assert.Equal(t, "claude", readValue(prefs, "ai.provider"))
	assert.Equal(t, "sonnet", readValue(prefs, "ai.model"))
	assert.Equal(t, "gpt-4o-class", readValue(prefs, "ai.class"))
}

// --- setKeyValue ---

func TestSetKeyValue_compression(t *testing.T) {
	setupProject(t, nil)

	require.NoError(t, setKeyValue("compression=true"))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	prefs, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	require.NotNil(t, prefs.Compression)
	assert.True(t, *prefs.Compression)
}

func TestSetKeyValue_autoCompile(t *testing.T) {
	setupProject(t, nil)

	require.NoError(t, setKeyValue("auto_compile=false"))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	prefs, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	require.NotNil(t, prefs.AutoCompile)
	assert.False(t, *prefs.AutoCompile)
}

func TestSetKeyValue_aiModel(t *testing.T) {
	setupProject(t, nil)

	require.NoError(t, setKeyValue("ai.model=llama3"))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	prefs, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	require.NotNil(t, prefs.AI)
	assert.Equal(t, "llama3", prefs.AI.Model)
}

func TestSetKeyValue_aiProviderClear(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{Provider: "claude", Model: "sonnet"},
	}
	setupProject(t, prefs)

	require.NoError(t, setKeyValue("ai.provider="))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	loaded, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	assert.Nil(t, loaded.AI)
}

func TestSetKeyValue_unknownKey(t *testing.T) {
	setupProject(t, nil)

	err := setKeyValue("unknown_key=value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown preference")
}

func TestSetKeyValue_noEquals(t *testing.T) {
	setupProject(t, nil)

	err := setKeyValue("compression")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected key=value")
}

func TestSetKeyValue_invalidBool(t *testing.T) {
	setupProject(t, nil)

	err := setKeyValue("compression=maybe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected true or false")
}

func TestSetKeyValue_aiProviderUnknown(t *testing.T) {
	setupProject(t, nil)

	err := setKeyValue("ai.provider=chatgpt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AI provider")
}

func TestSetKeyValue_preservesExisting(t *testing.T) {
	// Set compression, then set auto_compile — compression should remain.
	prefs := &preferences.Preferences{
		Compression: preferences.BoolPtr(true),
	}
	setupProject(t, prefs)

	require.NoError(t, setKeyValue("auto_compile=true"))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	loaded, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	require.NotNil(t, loaded.Compression)
	assert.True(t, *loaded.Compression)
	require.NotNil(t, loaded.AutoCompile)
	assert.True(t, *loaded.AutoCompile)
}

func TestSetKeyValue_overwrite(t *testing.T) {
	prefs := &preferences.Preferences{
		Compression: preferences.BoolPtr(true),
	}
	setupProject(t, prefs)

	require.NoError(t, setKeyValue("compression=false"))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	loaded, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	require.NotNil(t, loaded.Compression)
	assert.False(t, *loaded.Compression)
}

// --- showAll ---

func TestShowAll_noError(t *testing.T) {
	setupProject(t, nil)

	// Should not error even with no preferences set.
	err := showAll()
	assert.NoError(t, err)
}

func TestShowAll_withPrefs(t *testing.T) {
	prefs := &preferences.Preferences{
		Compression: preferences.BoolPtr(true),
		AutoCompile: preferences.BoolPtr(false),
		AI: &preferences.AIConfig{
			Provider: "claude",
		},
	}
	setupProject(t, prefs)

	err := showAll()
	assert.NoError(t, err)
}

// --- applyValue ---

func TestApplyValue_compressionTrue(t *testing.T) {
	prefs := &preferences.Preferences{}
	entry := &preferenceKey{Key: "compression", Type: "bool"}
	require.NoError(t, applyValue(prefs, entry, "true"))
	require.NotNil(t, prefs.Compression)
	assert.True(t, *prefs.Compression)
}

func TestApplyValue_compressionFalse(t *testing.T) {
	prefs := &preferences.Preferences{}
	entry := &preferenceKey{Key: "compression", Type: "bool"}
	require.NoError(t, applyValue(prefs, entry, "false"))
	require.NotNil(t, prefs.Compression)
	assert.False(t, *prefs.Compression)
}

func TestApplyValue_autoCompile(t *testing.T) {
	prefs := &preferences.Preferences{}
	entry := &preferenceKey{Key: "auto_compile", Type: "bool"}
	require.NoError(t, applyValue(prefs, entry, "yes"))
	require.NotNil(t, prefs.AutoCompile)
	assert.True(t, *prefs.AutoCompile)
}

func TestApplyValue_aiModel_createsAIConfig(t *testing.T) {
	prefs := &preferences.Preferences{}
	entry := &preferenceKey{Key: "ai.model", Type: "string"}
	require.NoError(t, applyValue(prefs, entry, "llama3"))
	require.NotNil(t, prefs.AI)
	assert.Equal(t, "llama3", prefs.AI.Model)
}

func TestApplyValue_aiProviderEmpty_clearsAI(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{Provider: "claude", Model: "sonnet"},
	}
	entry := &preferenceKey{Key: "ai.provider", Type: "string"}
	require.NoError(t, applyValue(prefs, entry, ""))
	assert.Nil(t, prefs.AI)
}

func TestApplyValue_boolInvalid(t *testing.T) {
	prefs := &preferences.Preferences{}
	entry := &preferenceKey{Key: "compression", Type: "bool"}
	err := applyValue(prefs, entry, "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected true or false")
}

// --- validateAIProvider ---

func TestValidateAIProvider_unknownID(t *testing.T) {
	err := validateAIProvider("chatgpt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AI provider")
	assert.Contains(t, err.Error(), "claude")
}

// Note: testing validateAIProvider with known IDs (e.g., "claude") depends
// on PATH availability. We test the unknown case exhaustively and leave
// binary detection to integration tests.

// --- registry ---

func TestRegistry_allKeysUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, k := range registry {
		assert.False(t, seen[k.Key], "duplicate key: %s", k.Key)
		seen[k.Key] = true
	}
}

func TestRegistry_allKeysNonEmpty(t *testing.T) {
	for _, k := range registry {
		assert.NotEmpty(t, k.Key)
		assert.NotEmpty(t, k.Description)
		assert.NotEmpty(t, k.Type)
	}
}

func TestRegistry_containsExpectedKeys(t *testing.T) {
	expected := []string{"compression", "auto_compile", "ai.provider", "ai.model", "ai.class"}
	keys := make([]string, len(registry))
	for i, k := range registry {
		keys[i] = k.Key
	}
	assert.Equal(t, expected, keys)
}

// --- showAll error paths ---

func TestShowAll_noConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	err := showAll()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestSetKeyValue_noConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	err := setKeyValue("compression=true")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

// --- ai.class ---

func TestSetKeyValue_aiClass(t *testing.T) {
	setupProject(t, nil)

	require.NoError(t, setKeyValue("ai.class=gpt-4o-class"))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	prefs, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	require.NotNil(t, prefs.AI)
	assert.Equal(t, "gpt-4o-class", prefs.AI.Class)
}

func TestSetKeyValue_aiClassUnknown(t *testing.T) {
	setupProject(t, nil)

	err := setKeyValue("ai.class=gpt-3-class")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model class")
}

func TestSetKeyValue_aiClassClear(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{Provider: "claude", Class: "gpt-4o-class"},
	}
	setupProject(t, prefs)

	require.NoError(t, setKeyValue("ai.class="))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	loaded, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	// AI config should still exist (provider is set), but class should be empty.
	require.NotNil(t, loaded.AI)
	assert.Equal(t, "claude", loaded.AI.Provider)
	assert.Empty(t, loaded.AI.Class)
}

func TestSetKeyValue_aiClass_createsAIConfig(t *testing.T) {
	setupProject(t, nil)

	require.NoError(t, setKeyValue("ai.class=o1-class"))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	prefs, err := preferences.Load(cfg.OutputDir())
	require.NoError(t, err)
	require.NotNil(t, prefs.AI)
	assert.Equal(t, "o1-class", prefs.AI.Class)
}

func TestApplyValue_aiClass_valid(t *testing.T) {
	prefs := &preferences.Preferences{}
	entry := &preferenceKey{Key: "ai.class", Type: "string"}
	require.NoError(t, applyValue(prefs, entry, "claude-sonnet-class"))
	require.NotNil(t, prefs.AI)
	assert.Equal(t, "claude-sonnet-class", prefs.AI.Class)
}

func TestApplyValue_aiClass_invalid(t *testing.T) {
	prefs := &preferences.Preferences{}
	entry := &preferenceKey{Key: "ai.class", Type: "string"}
	err := applyValue(prefs, entry, "unknown-class")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model class")
}

func TestApplyValue_aiClass_empty_clearsClass(t *testing.T) {
	prefs := &preferences.Preferences{
		AI: &preferences.AIConfig{Class: "gpt-4o-class"},
	}
	entry := &preferenceKey{Key: "ai.class", Type: "string"}
	require.NoError(t, applyValue(prefs, entry, ""))
	require.NotNil(t, prefs.AI)
	assert.Empty(t, prefs.AI.Class)
}

func TestValidateAIClass_unknownID(t *testing.T) {
	err := validateAIClass("chatgpt-class")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model class")
	assert.Contains(t, err.Error(), "gpt-4o-class")
}

func TestValidateAIClass_knownIDs(t *testing.T) {
	for _, id := range []string{"gpt-4o-class", "claude-sonnet-class", "o1-class"} {
		assert.NoError(t, validateAIClass(id), "should accept: %s", id)
	}
}
