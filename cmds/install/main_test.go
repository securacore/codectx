package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command metadata ---

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "install", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}

func TestCommand_flags(t *testing.T) {
	flagNames := make(map[string]bool)
	for _, f := range Command.Flags {
		flagNames[f.Names()[0]] = true
	}
	assert.True(t, flagNames["activate"])
}

// --- parseActivateFlag ---

func TestParseActivateFlag_all(t *testing.T) {
	a, err := parseActivateFlag("all")
	require.NoError(t, err)
	assert.True(t, a.IsAll())
}

func TestParseActivateFlag_none(t *testing.T) {
	a, err := parseActivateFlag("none")
	require.NoError(t, err)
	assert.True(t, a.IsNone())
}

func TestParseActivateFlag_granular(t *testing.T) {
	a, err := parseActivateFlag("topics:react,prompts:commit")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"react"}, a.Map.Topics)
	assert.Equal(t, []string{"commit"}, a.Map.Prompts)
}

func TestParseActivateFlag_unknownSection(t *testing.T) {
	_, err := parseActivateFlag("bad:id")
	assert.Error(t, err)
}

// --- setupDocsDir ---

func TestSetupDocsDir_createsNewDir(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{Name: "test"}
	docsDir := "docs"

	// Write a valid config so setupDocsDir can write back if needed.
	require.NoError(t, config.Write(configFile, cfg))

	err = setupDocsDir(cfg, &docsDir)
	require.NoError(t, err)

	// Verify structure was created.
	assert.DirExists(t, filepath.Join(dir, "docs"))
	assert.DirExists(t, filepath.Join(dir, "docs", "packages"))
	assert.DirExists(t, filepath.Join(dir, "docs", "schemas"))
	assert.FileExists(t, filepath.Join(dir, "docs", "package.yml"))
}

func TestSetupDocsDir_existingCompatible(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Create a valid docs dir.
	docsPath := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsPath, 0o755))
	pkgYml := `name: test
author: org
version: "1.0.0"
description: "Test"
`
	require.NoError(t, os.WriteFile(filepath.Join(docsPath, "package.yml"), []byte(pkgYml), 0o644))

	cfg := &config.Config{Name: "test"}
	require.NoError(t, config.Write(configFile, cfg))

	docsDir := "docs"
	err = setupDocsDir(cfg, &docsDir)
	require.NoError(t, err)
	assert.Equal(t, "docs", docsDir) // should not have changed
}

func TestSetupDocsDir_preservesExistingPackageYml(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsPath := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsPath, 0o755))

	pkgYml := `name: existing
author: org
version: "2.0.0"
description: "Existing package"
`
	require.NoError(t, os.WriteFile(filepath.Join(docsPath, "package.yml"), []byte(pkgYml), 0o644))

	cfg := &config.Config{Name: "test"}
	require.NoError(t, config.Write(configFile, cfg))

	docsDir := "docs"
	err = setupDocsDir(cfg, &docsDir)
	require.NoError(t, err)

	// Existing package.yml should not be overwritten.
	data, err := os.ReadFile(filepath.Join(docsPath, "package.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "existing")
}

// --- ensureGitignore ---

func TestEnsureGitignore_createsNew(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), ".codectx/")
}

func TestEnsureGitignore_alreadyPresent(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(".gitignore", []byte(".codectx/\n"), 0o644))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	// Should not duplicate the entry.
	assert.Equal(t, ".codectx/\n", string(data))
}

func TestEnsureGitignore_appendsToExisting(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(".gitignore", []byte("node_modules/\n"), 0o644))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), "node_modules/")
	assert.Contains(t, string(data), ".codectx/")
}

func TestEnsureGitignore_appendsNewlineIfMissing(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// File without trailing newline.
	require.NoError(t, os.WriteFile(".gitignore", []byte("node_modules/"), 0o644))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), "node_modules/\n.codectx/")
}

// --- run ---

func TestRun_noConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = run("all")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_noPackages(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{Name: "test", Packages: []config.PackageDep{}}
	require.NoError(t, config.Write(configFile, cfg))

	err = run("all")
	assert.NoError(t, err)
}

// --- parseActivateFlag error branches ---

func TestParseActivateFlag_missingColon(t *testing.T) {
	_, err := parseActivateFlag("topicsreact")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected section:id")
}

func TestParseActivateFlag_emptyId(t *testing.T) {
	_, err := parseActivateFlag("topics:")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty id")
}

func TestParseActivateFlag_allSections(t *testing.T) {
	a, err := parseActivateFlag("foundation:a,topics:b,prompts:c,plans:d")
	require.NoError(t, err)
	require.NotNil(t, a.Map)
	assert.Equal(t, []string{"a"}, a.Map.Foundation)
	assert.Equal(t, []string{"b"}, a.Map.Topics)
	assert.Equal(t, []string{"c"}, a.Map.Prompts)
	assert.Equal(t, []string{"d"}, a.Map.Plans)
}

// --- ensureGitignore edge cases ---

func TestEnsureGitignore_emptyExistingFile(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(".gitignore", []byte(""), 0o644))

	err = ensureGitignore(".codectx")
	require.NoError(t, err)

	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), ".codectx/")
}

// --- setupDocsDir edge cases ---

func TestSetupDocsDir_deeplyNestedDir(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{Name: "test"}
	require.NoError(t, config.Write(configFile, cfg))

	docsDir := filepath.Join("a", "b", "c", "docs")
	err = setupDocsDir(cfg, &docsDir)
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(dir, "a", "b", "c", "docs"))
	assert.DirExists(t, filepath.Join(dir, "a", "b", "c", "docs", "packages"))
	assert.DirExists(t, filepath.Join(dir, "a", "b", "c", "docs", "schemas"))
}

// --- run with invalid activate flag ---

func TestRun_invalidActivateFlag(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	cfg := &config.Config{
		Name: "test",
		Packages: []config.PackageDep{
			{Name: "pkg", Author: "org", Active: config.Activation{Mode: "none"}},
		},
	}
	require.NoError(t, config.Write(configFile, cfg))

	// Create docs directory structure for setupDocsDir.
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "packages"), 0o755))

	// Create output directory.
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	err = run("badformat")
	// Should fail trying to resolve the package (no git source), but
	// the activate flag error only fires if packages have inactive status
	// and we get past the install phase. Since there's no actual git source,
	// install will fail for all packages first.
	assert.Error(t, err)
}
