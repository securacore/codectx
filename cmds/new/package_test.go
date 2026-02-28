package new

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- packageCommand metadata ---

func TestPackageCommand_metadata(t *testing.T) {
	assert.Equal(t, "package", packageCommand.Name)
	assert.NotEmpty(t, packageCommand.Usage)
	assert.Equal(t, "<name>", packageCommand.ArgsUsage)
}

// --- runPackage: name validation ---

func TestRunPackage_invalidName(t *testing.T) {
	err := runPackage("INVALID")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid name")
}

func TestRunPackage_invalidNameUnderscore(t *testing.T) {
	err := runPackage("hello_world")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid name")
}

func TestRunPackage_invalidNameHyphenPrefix(t *testing.T) {
	err := runPackage("-leading")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid name")
}

// --- runPackage: full scaffolding ---

func TestRunPackage_createsProjectDirectory(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("react-hooks")
	require.NoError(t, err)

	// Should create codectx-react-hooks/ directory.
	projectDir := filepath.Join(dir, "codectx-react-hooks")
	info, err := os.Stat(projectDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestRunPackage_prependsCodectxPrefix(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("my-lib")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-my-lib")

	// codectx.yml should have the full name.
	cfgData, err := os.ReadFile(filepath.Join(projectDir, "codectx.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(cfgData), "codectx-my-lib")
}

func TestRunPackage_createsDocsStructure(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	// Core docs directories.
	dirs := []string{
		"docs",
		"docs/foundation",
		"docs/topics",
		"docs/prompts",
		"docs/plans",
		"docs/schemas",
	}
	for _, d := range dirs {
		info, err := os.Stat(filepath.Join(projectDir, d))
		require.NoError(t, err, "directory %s should exist", d)
		assert.True(t, info.IsDir(), "%s should be a directory", d)
	}
}

func TestRunPackage_createsPackageDir(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	// Package directory structure.
	pkgDirs := []string{
		"package",
		"package/foundation",
		"package/topics",
		"package/prompts",
		"package/schemas",
		"package/packages",
		"package/plans",
	}
	for _, d := range pkgDirs {
		info, err := os.Stat(filepath.Join(projectDir, d))
		require.NoError(t, err, "directory %s should exist", d)
		assert.True(t, info.IsDir(), "%s should be a directory", d)
	}
}

func TestRunPackage_writesPackageManifest(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	// package/manifest.yml should exist.
	pkgManifest, err := manifest.Load(filepath.Join(projectDir, "package", "manifest.yml"))
	require.NoError(t, err)
	assert.Equal(t, "codectx-test-pkg", pkgManifest.Name)
	assert.Equal(t, "0.1.0", pkgManifest.Version)
}

func TestRunPackage_writesTemplateFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	// Key template files from packagetpl.
	templateFiles := []string{
		".justfile",
		"devbox.json",
		"bin/release",
		"bin/just/settings.just",
		".github/workflows/release.yml",
		"docs/prompts/save/README.md",
	}
	for _, f := range templateFiles {
		_, err := os.Stat(filepath.Join(projectDir, f))
		assert.NoError(t, err, "template file %s should exist", f)
	}
}

func TestRunPackage_binReleaseIsExecutable(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	info, err := os.Stat(filepath.Join(projectDir, "bin", "release"))
	require.NoError(t, err)
	assert.True(t, info.Mode().Perm()&0o111 != 0,
		"bin/release should be executable, got %o", info.Mode().Perm())
}

func TestRunPackage_packageFoundationHasDefaults(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	// package/foundation/ should have default docs written by scaffoldPackageDir.
	expectedDocs := []string{"philosophy", "documentation", "markdown", "specs", "ai-authoring", "prompts", "plans"}
	for _, doc := range expectedDocs {
		readme := filepath.Join(projectDir, "package", "foundation", doc, "README.md")
		_, err := os.Stat(readme)
		assert.NoError(t, err, "package/foundation/%s/README.md should exist", doc)
	}
}

func TestRunPackage_packageSchemasWritten(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	// package/schemas/ should have schemas.
	schemas := []string{
		"package/schemas/codectx.schema.json",
		"package/schemas/manifest.schema.json",
		"package/schemas/plan.schema.json",
	}
	for _, s := range schemas {
		_, err := os.Stat(filepath.Join(projectDir, s))
		assert.NoError(t, err, "%s should exist", s)
	}
}

func TestRunPackage_packageEmptyDirsHaveGitkeep(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	// Directories that should have .gitkeep (empty content dirs in package/).
	gitkeepDirs := []string{
		"package/topics",
		"package/prompts",
		"package/packages",
		"package/plans",
	}
	for _, d := range gitkeepDirs {
		path := filepath.Join(projectDir, d, ".gitkeep")
		_, err := os.Stat(path)
		assert.NoError(t, err, "%s/.gitkeep should exist", d)
	}
}

func TestRunPackage_docsManifestReSynced(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	// docs/manifest.yml should contain the save prompt from the template.
	m, err := manifest.Load(filepath.Join(projectDir, "docs", "manifest.yml"))
	require.NoError(t, err)

	// The template adds docs/prompts/save/README.md which should be discovered.
	found := false
	for _, p := range m.Prompts {
		if p.ID == "save" {
			found = true
			break
		}
	}
	assert.True(t, found, "docs/manifest.yml should contain the 'save' prompt after re-sync")
}

func TestRunPackage_preferencesWritten(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = runPackage("test-pkg")
	require.NoError(t, err)

	projectDir := filepath.Join(dir, "codectx-test-pkg")

	prefs, err := preferences.Load(filepath.Join(projectDir, ".codectx"))
	require.NoError(t, err)
	require.NotNil(t, prefs.AutoCompile)
	assert.True(t, *prefs.AutoCompile)
}

// --- scaffoldPackageDir ---

func TestScaffoldPackageDir_createsDirectories(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Create a minimal docs dir so the function has something to reference.
	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	err = scaffoldPackageDir(docsDir)
	require.NoError(t, err)

	expected := []string{
		"package",
		"package/foundation",
		"package/topics",
		"package/prompts",
		"package/schemas",
		"package/packages",
		"package/plans",
	}
	for _, d := range expected {
		info, err := os.Stat(filepath.Join(dir, d))
		require.NoError(t, err, "directory %s should exist", d)
		assert.True(t, info.IsDir())
	}
}

func TestScaffoldPackageDir_writesFoundationDefaults(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	err = scaffoldPackageDir(docsDir)
	require.NoError(t, err)

	// All 7 default foundation docs should exist in package/foundation/.
	expectedDocs := []string{"philosophy", "documentation", "markdown", "specs", "ai-authoring", "prompts", "plans"}
	for _, doc := range expectedDocs {
		readme := filepath.Join(dir, "package", "foundation", doc, "README.md")
		info, err := os.Stat(readme)
		require.NoError(t, err, "package/foundation/%s/README.md should exist", doc)
		assert.Greater(t, info.Size(), int64(0))

		spec := filepath.Join(dir, "package", "foundation", doc, "spec", "README.md")
		info, err = os.Stat(spec)
		require.NoError(t, err, "package/foundation/%s/spec/README.md should exist", doc)
		assert.Greater(t, info.Size(), int64(0))
	}
}

func TestScaffoldPackageDir_writesSchemas(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	err = scaffoldPackageDir(docsDir)
	require.NoError(t, err)

	schemas := []string{
		"package/schemas/codectx.schema.json",
		"package/schemas/manifest.schema.json",
		"package/schemas/plan.schema.json",
	}
	for _, s := range schemas {
		_, err := os.Stat(filepath.Join(dir, s))
		assert.NoError(t, err, "%s should exist", s)
	}
}

func TestScaffoldPackageDir_gitkeepInEmptyDirs(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	err = scaffoldPackageDir(docsDir)
	require.NoError(t, err)

	gitkeepDirs := []string{"package/topics", "package/prompts", "package/packages", "package/plans"}
	for _, d := range gitkeepDirs {
		_, err := os.Stat(filepath.Join(dir, d, ".gitkeep"))
		assert.NoError(t, err, "%s/.gitkeep should exist", d)
	}

	// foundation and schemas should NOT have .gitkeep (they have content).
	for _, d := range []string{"package/foundation", "package/schemas"} {
		_, err := os.Stat(filepath.Join(dir, d, ".gitkeep"))
		assert.True(t, os.IsNotExist(err), "%s/.gitkeep should NOT exist (has content)", d)
	}
}
