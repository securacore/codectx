package new

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command metadata ---

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "new", Command.Name)
	assert.NotEmpty(t, Command.Usage)
	assert.Len(t, Command.Commands, 6)
}

func TestSubcommand_names(t *testing.T) {
	names := make([]string, len(Command.Commands))
	for i, c := range Command.Commands {
		names[i] = c.Name
	}
	assert.Contains(t, names, "foundation")
	assert.Contains(t, names, "topic")
	assert.Contains(t, names, "prompt")
	assert.Contains(t, names, "plan")
	assert.Contains(t, names, "application")
	assert.Contains(t, names, "package")
}

// --- setupProject ---

// setupProject creates a minimal project in a temp directory and
// changes cwd into it. Returns the project root. Cleanup restores cwd.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	for _, sub := range []string{"foundation", "application", "topics", "prompts", "plans"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	// Write codectx.yml.
	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	// Write docs/manifest.yml.
	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project for new command",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	return dir
}

// --- kebabCase validation ---

func TestKebabCase_valid(t *testing.T) {
	valid := []string{"hello", "hello-world", "my-long-name", "a1", "test-123", "a"}
	for _, v := range valid {
		assert.True(t, kebabCase.MatchString(v), "expected %q to be valid kebab-case", v)
	}
}

func TestKebabCase_invalid(t *testing.T) {
	invalid := []string{"", "Hello", "HELLO", "hello_world", "hello world", "-hello", "hello-", "hello--world", "Hello-World"}
	for _, v := range invalid {
		assert.False(t, kebabCase.MatchString(v), "expected %q to be invalid kebab-case", v)
	}
}

// --- kebabToTitle ---

func TestKebabToTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"hello-world", "Hello World"},
		{"my-long-name", "My Long Name"},
		{"a", "A"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, kebabToTitle(tt.input))
	}
}

// --- sectionDir ---

func TestSectionDir(t *testing.T) {
	assert.Equal(t, "foundation", sectionDir(kindFoundation))
	assert.Equal(t, "topics", sectionDir(kindTopic))
	assert.Equal(t, "prompts", sectionDir(kindPrompt))
	assert.Equal(t, "plans", sectionDir(kindPlan))
	assert.Equal(t, "application", sectionDir(kindApplication))
}

// --- scaffold: missing name ---

func TestScaffold_invalidName(t *testing.T) {
	setupProject(t)

	err := scaffold(kindFoundation, "INVALID", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid name")
}

func TestScaffold_invalidNameUnderscore(t *testing.T) {
	setupProject(t)

	err := scaffold(kindFoundation, "hello_world", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid name")
}

// --- scaffold: missing config ---

func TestScaffold_missingConfig(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = scaffold(kindFoundation, "test", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

// --- scaffold: duplicate ---

func TestScaffold_duplicate(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Pre-create the directory to simulate an existing entry.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "existing"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "existing", "README.md"),
		[]byte("# Existing\n"), 0o644))

	err := scaffold(kindFoundation, "existing", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// --- scaffold: foundation ---

func TestScaffold_foundation(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	err := scaffold(kindFoundation, "philosophy", false)
	require.NoError(t, err)

	// README.md should exist.
	readme := filepath.Join(docsDir, "foundation", "philosophy", "README.md")
	data, err := os.ReadFile(readme)
	require.NoError(t, err)
	assert.Equal(t, "# Philosophy\n", string(data))

	// No spec or plan.yml should exist.
	_, err = os.Stat(filepath.Join(docsDir, "foundation", "philosophy", "spec"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(docsDir, "foundation", "philosophy", "plan.yml"))
	assert.True(t, os.IsNotExist(err))

	// Manifest should contain the entry.
	m, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, m.Foundation, 1)
	assert.Equal(t, "philosophy", m.Foundation[0].ID)
	assert.Equal(t, "foundation/philosophy/README.md", m.Foundation[0].Path)
}

// --- scaffold: topic ---

func TestScaffold_topic(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	err := scaffold(kindTopic, "react", false)
	require.NoError(t, err)

	// README.md should exist.
	readme := filepath.Join(docsDir, "topics", "react", "README.md")
	data, err := os.ReadFile(readme)
	require.NoError(t, err)
	assert.Equal(t, "# React\n", string(data))

	// spec/README.md should exist.
	specReadme := filepath.Join(docsDir, "topics", "react", "spec", "README.md")
	data, err = os.ReadFile(specReadme)
	require.NoError(t, err)
	assert.Equal(t, "# React Spec\n", string(data))

	// Manifest should contain the entry.
	m, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, m.Topics, 1)
	assert.Equal(t, "react", m.Topics[0].ID)
	assert.Equal(t, "topics/react/README.md", m.Topics[0].Path)
	assert.Equal(t, "topics/react/spec/README.md", m.Topics[0].Spec)
}

// --- scaffold: prompt ---

func TestScaffold_prompt(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	err := scaffold(kindPrompt, "audit", false)
	require.NoError(t, err)

	// README.md should exist.
	readme := filepath.Join(docsDir, "prompts", "audit", "README.md")
	data, err := os.ReadFile(readme)
	require.NoError(t, err)
	assert.Equal(t, "# Audit\n", string(data))

	// No spec or plan.yml.
	_, err = os.Stat(filepath.Join(docsDir, "prompts", "audit", "spec"))
	assert.True(t, os.IsNotExist(err))

	// Manifest should contain the entry.
	m, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, m.Prompts, 1)
	assert.Equal(t, "audit", m.Prompts[0].ID)
	assert.Equal(t, "prompts/audit/README.md", m.Prompts[0].Path)
}

// --- scaffold: plan ---

func TestScaffold_plan(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	err := scaffold(kindPlan, "migrate", false)
	require.NoError(t, err)

	// README.md should exist.
	readme := filepath.Join(docsDir, "plans", "migrate", "README.md")
	data, err := os.ReadFile(readme)
	require.NoError(t, err)
	assert.Equal(t, "# Migrate\n", string(data))

	// plan.yml should exist with initial state.
	planYML := filepath.Join(docsDir, "plans", "migrate", "plan.yml")
	data, err = os.ReadFile(planYML)
	require.NoError(t, err)
	assert.Contains(t, string(data), "plan: migrate")
	assert.Contains(t, string(data), "status: not_started")

	// Manifest should contain the entry with plan_state.
	m, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, m.Plans, 1)
	assert.Equal(t, "migrate", m.Plans[0].ID)
	assert.Equal(t, "plans/migrate/README.md", m.Plans[0].Path)
	assert.Equal(t, "plans/migrate/plan.yml", m.Plans[0].PlanState)
}

// --- scaffold: application ---

func TestScaffold_application(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	err := scaffold(kindApplication, "architecture", false)
	require.NoError(t, err)

	// README.md should exist.
	readme := filepath.Join(docsDir, "application", "architecture", "README.md")
	data, err := os.ReadFile(readme)
	require.NoError(t, err)
	assert.Equal(t, "# Architecture\n", string(data))

	// spec/README.md should exist.
	specReadme := filepath.Join(docsDir, "application", "architecture", "spec", "README.md")
	data, err = os.ReadFile(specReadme)
	require.NoError(t, err)
	assert.Equal(t, "# Architecture Spec\n", string(data))

	// Manifest should contain the entry.
	m, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, m.Application, 1)
	assert.Equal(t, "architecture", m.Application[0].ID)
	assert.Equal(t, "application/architecture/README.md", m.Application[0].Path)
	assert.Equal(t, "application/architecture/spec/README.md", m.Application[0].Spec)
}

// --- scaffold: multi-word names ---

func TestScaffold_multiWordName(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	err := scaffold(kindFoundation, "coding-standards", false)
	require.NoError(t, err)

	readme := filepath.Join(docsDir, "foundation", "coding-standards", "README.md")
	data, err := os.ReadFile(readme)
	require.NoError(t, err)
	assert.Equal(t, "# Coding Standards\n", string(data))

	m, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, m.Foundation, 1)
	assert.Equal(t, "coding-standards", m.Foundation[0].ID)
}

// --- scaffold: preserves existing entries ---

func TestScaffold_preservesExistingEntries(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Create an existing foundation entry on disk.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "existing"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "existing", "README.md"),
		[]byte("# Existing\n"), 0o644))

	// Write manifest with the existing entry.
	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "existing", Path: "foundation/existing/README.md", Description: "Existing"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Scaffold a new foundation entry.
	err := scaffold(kindFoundation, "new-entry", false)
	require.NoError(t, err)

	// Both entries should be in the manifest.
	result, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, result.Foundation, 2)

	ids := []string{result.Foundation[0].ID, result.Foundation[1].ID}
	assert.Contains(t, ids, "existing")
	assert.Contains(t, ids, "new-entry")
}
