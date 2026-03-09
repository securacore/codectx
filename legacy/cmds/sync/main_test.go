package sync

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Command metadata ---

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "sync", Command.Name)
	assert.NotEmpty(t, Command.Usage)
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
	require.NoError(t, config.Write(filepath.Join(dir, shared.ConfigFile), cfg))

	// Write docs/manifest.yml.
	m := &manifest.Manifest{
		Name:        "test-project",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test project for sync command",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	return dir
}

// captureStdout runs fn and returns whatever it writes to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

// --- run() tests ---

func TestRun_emptyProject(t *testing.T) {
	setupProject(t)

	err := run()
	require.NoError(t, err)

	// Manifest should still exist and be valid.
	m, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	assert.Equal(t, "test-project", m.Name)
	assert.Nil(t, m.Foundation)
	assert.Nil(t, m.Topics)
}

func TestRun_discoversNewEntries(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Create foundation and topic files on disk.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	topicDir := filepath.Join(docsDir, "topics", "react")
	require.NoError(t, os.MkdirAll(topicDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(topicDir, "README.md"),
		[]byte("# React\n"), 0o644))

	err := run()
	require.NoError(t, err)

	// Verify entries were discovered and written.
	m, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, m.Foundation, 1)
	assert.Equal(t, "philosophy", m.Foundation[0].ID)
	require.Len(t, m.Topics, 1)
	assert.Equal(t, "react", m.Topics[0].ID)
}

func TestRun_discoversApplicationEntries(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	appDir := filepath.Join(docsDir, "application", "architecture")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(appDir, "README.md"),
		[]byte("# Architecture\n"), 0o644))

	err := run()
	require.NoError(t, err)

	m, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, m.Application, 1)
	assert.Equal(t, "architecture", m.Application[0].ID)
}

func TestRun_removesStaleEntries(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Create a foundation file on disk.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	// Write manifest with a stale entry that doesn't exist on disk.
	m := &manifest.Manifest{
		Name:    "test-project",
		Author:  "tester",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md", Description: "Philosophy"},
			{ID: "removed", Path: "foundation/removed/README.md", Description: "Gone"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	err := run()
	require.NoError(t, err)

	result, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)
}

func TestRun_infersRelationships(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Foundation doc.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	// Topic that links to foundation.
	topicDir := filepath.Join(docsDir, "topics", "react")
	require.NoError(t, os.MkdirAll(topicDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(topicDir, "README.md"),
		[]byte("# React\nSee [philosophy](../../foundation/philosophy/README.md).\n"), 0o644))

	err := run()
	require.NoError(t, err)

	result, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)

	// react depends_on philosophy.
	require.Len(t, result.Topics, 1)
	assert.Equal(t, []string{"philosophy"}, result.Topics[0].DependsOn)

	// philosophy required_by react.
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, []string{"react"}, result.Foundation[0].RequiredBy)
}

func TestRun_preservesManifestMetadata(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Write manifest with custom metadata.
	m := &manifest.Manifest{
		Name:        "my-docs",
		Author:      "org",
		Version:     "2.0.0",
		Description: "Custom docs package",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	err := run()
	require.NoError(t, err)

	result, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	assert.Equal(t, "my-docs", result.Name)
	assert.Equal(t, "org", result.Author)
	assert.Equal(t, "2.0.0", result.Version)
	assert.Equal(t, "Custom docs package", result.Description)
}

func TestRun_preservesEntryDescriptions(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Create a foundation file.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	// Write manifest with a custom description for the entry.
	m := &manifest.Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy/README.md",
				Description: "Custom description", Load: "always"},
		},
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	err := run()
	require.NoError(t, err)

	result, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "Custom description", result.Foundation[0].Description)
	assert.Equal(t, "always", result.Foundation[0].Load)
}

func TestRun_fullSync(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Create foundation doc.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	// Create application doc.
	appDir := filepath.Join(docsDir, "application", "arch")
	require.NoError(t, os.MkdirAll(appDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(appDir, "README.md"),
		[]byte("# Architecture\nSee [philosophy](../../foundation/philosophy/README.md).\n"), 0o644))

	// Create two topics with cross-references.
	for _, topic := range []string{"react", "typescript"} {
		tDir := filepath.Join(docsDir, "topics", topic)
		require.NoError(t, os.MkdirAll(tDir, 0o755))
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "react", "README.md"),
		[]byte("# React\nSee [TypeScript](../typescript/README.md).\nSee [philosophy](../../foundation/philosophy/README.md).\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "topics", "typescript", "README.md"),
		[]byte("# TypeScript\nSee [philosophy](../../foundation/philosophy/README.md).\n"), 0o644))

	// Create a prompt.
	promptDir := filepath.Join(docsDir, "prompts", "audit")
	require.NoError(t, os.MkdirAll(promptDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(promptDir, "README.md"),
		[]byte("# Audit\nLoad [philosophy](../../foundation/philosophy/README.md).\n"), 0o644))

	// Create a plan.
	planDir := filepath.Join(docsDir, "plans", "migrate")
	require.NoError(t, os.MkdirAll(planDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(planDir, "README.md"),
		[]byte("# Migrate\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(planDir, "plan.yml"),
		[]byte("status: in-progress\n"), 0o644))

	err := run()
	require.NoError(t, err)

	result, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)

	// All sections discovered.
	assert.Len(t, result.Foundation, 1)
	assert.Len(t, result.Application, 1)
	assert.Len(t, result.Topics, 2)
	assert.Len(t, result.Prompts, 1)
	assert.Len(t, result.Plans, 1)

	// philosophy is required_by multiple entries.
	assert.True(t, len(result.Foundation[0].RequiredBy) >= 3)
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

func TestRun_missingManifestCreatesOne(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	for _, sub := range []string{"foundation", "topics"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	// Write codectx.yml but no manifest.yml.
	cfg := &config.Config{
		Name:     "new-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, shared.ConfigFile), cfg))

	// Create a foundation file on disk.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	err = run()
	require.NoError(t, err)

	// Manifest should now exist with the discovered entry.
	result, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	assert.Equal(t, "new-project", result.Name)
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)
}

func TestRun_idempotent(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Create a foundation file.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	// First sync.
	err := run()
	require.NoError(t, err)

	first, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)

	// Second sync should produce the same result.
	err = run()
	require.NoError(t, err)

	second, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)

	assert.Equal(t, len(first.Foundation), len(second.Foundation))
	assert.Equal(t, first.Foundation[0].ID, second.Foundation[0].ID)
}

func TestRun_customDocsDir(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "custom-docs")
	for _, sub := range []string{"foundation", "topics"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			DocsDir: docsDir,
		},
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, shared.ConfigFile), cfg))

	// Write manifest and a foundation doc.
	m := &manifest.Manifest{Name: "test-project", Version: "1.0.0"}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	err = run()
	require.NoError(t, err)

	result, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)
}

// --- sectionCounts ---

func TestSectionCounts_empty(t *testing.T) {
	m := &manifest.Manifest{}
	c := sectionCounts(m)
	assert.Equal(t, 0, c.total())
}

func TestSectionCounts_allSections(t *testing.T) {
	m := &manifest.Manifest{
		Foundation:  []manifest.FoundationEntry{{ID: "a"}, {ID: "b"}},
		Application: []manifest.ApplicationEntry{{ID: "c"}},
		Topics:      []manifest.TopicEntry{{ID: "d"}, {ID: "e"}, {ID: "f"}},
		Prompts:     []manifest.PromptEntry{{ID: "g"}},
		Plans:       []manifest.PlanEntry{{ID: "h"}},
	}
	c := sectionCounts(m)
	assert.Equal(t, 2, c.foundation)
	assert.Equal(t, 1, c.application)
	assert.Equal(t, 3, c.topics)
	assert.Equal(t, 1, c.prompts)
	assert.Equal(t, 1, c.plans)
	assert.Equal(t, 8, c.total())
}

// --- countRelationships ---

func TestCountRelationships_none(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "a"}},
		Topics:     []manifest.TopicEntry{{ID: "b"}},
	}
	assert.Equal(t, 0, countRelationships(m))
}

func TestCountRelationships_mixed(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", DependsOn: []string{"b"}},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "arch", DependsOn: []string{"a", "b"}},
		},
		Topics: []manifest.TopicEntry{
			{ID: "b", DependsOn: []string{"a"}},
			{ID: "c"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "d", DependsOn: []string{"a"}},
		},
	}
	assert.Equal(t, 5, countRelationships(m))
}

// --- printSummary ---

func TestPrintSummary_discovered(t *testing.T) {
	before := counts{foundation: 1}
	after := counts{foundation: 3}
	result := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", DependsOn: []string{"b"}},
			{ID: "b"},
			{ID: "c"},
		},
	}

	out := captureStdout(t, func() {
		printSummary(before, after, result)
	})

	assert.Contains(t, out, "3 entries")
	assert.Contains(t, out, "Discovered")
	assert.Contains(t, out, "Foundation")
	assert.Contains(t, out, "Relationships")
}

func TestPrintSummary_removed(t *testing.T) {
	before := counts{foundation: 5}
	after := counts{foundation: 2}
	result := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a"},
			{ID: "b"},
		},
	}

	out := captureStdout(t, func() {
		printSummary(before, after, result)
	})

	assert.Contains(t, out, "Removed")
}

func TestPrintSummary_noChange(t *testing.T) {
	c := counts{foundation: 2}
	result := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a"},
			{ID: "b"},
		},
	}

	out := captureStdout(t, func() {
		printSummary(c, c, result)
	})

	assert.Contains(t, out, "2 entries")
	assert.NotContains(t, out, "Discovered")
	assert.NotContains(t, out, "Removed")
	assert.NotContains(t, out, "Relationships")
}

// --- run(): error paths and integration scenarios ---

func TestRun_corruptManifestRecovery(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation"), 0o755))

	cfg := &config.Config{
		Name:     "recovery-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, shared.ConfigFile), cfg))

	// Write a corrupt manifest.yml.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "manifest.yml"),
		[]byte("{{{{not valid yaml"), 0o644))

	// Create a foundation file on disk.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	err = run()
	require.NoError(t, err)

	// Manifest should be recreated with config name and discovered entry.
	result, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	assert.Equal(t, "recovery-project", result.Name)
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)
}

func TestRun_addFileBetweenSyncs(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// First sync: empty project.
	err := run()
	require.NoError(t, err)

	first, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	assert.Nil(t, first.Foundation)

	// Add a foundation file.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "philosophy"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "philosophy", "README.md"),
		[]byte("# Philosophy\n"), 0o644))

	// Second sync: discovers the new file.
	err = run()
	require.NoError(t, err)

	second, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, second.Foundation, 1)
	assert.Equal(t, "philosophy", second.Foundation[0].ID)
}

func TestRun_deleteFileBetweenSyncs(t *testing.T) {
	dir := setupProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Create two foundation files.
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "a"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "a", "README.md"),
		[]byte("# A\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "foundation", "b"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "b", "README.md"),
		[]byte("# B\n"), 0o644))

	// First sync discovers both.
	err := run()
	require.NoError(t, err)

	first, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, first.Foundation, 2)

	// Delete one file.
	require.NoError(t, os.RemoveAll(filepath.Join(docsDir, "foundation", "b")))

	// Second sync removes the stale entry.
	err = run()
	require.NoError(t, err)

	second, err := manifest.Load(filepath.Join("docs", "manifest.yml"))
	require.NoError(t, err)
	require.Len(t, second.Foundation, 1)
	assert.Equal(t, "a", second.Foundation[0].ID)
}

// --- countRelationships: additional coverage ---

func TestCountRelationships_withPlans(t *testing.T) {
	m := &manifest.Manifest{
		Plans: []manifest.PlanEntry{
			{ID: "migrate", DependsOn: []string{"a", "b"}},
		},
	}
	assert.Equal(t, 2, countRelationships(m))
}

func TestCountRelationships_emptyManifest(t *testing.T) {
	assert.Equal(t, 0, countRelationships(&manifest.Manifest{}))
}

// --- printSummary: additional coverage ---

func TestPrintSummary_zeroEntries(t *testing.T) {
	zero := counts{}
	result := &manifest.Manifest{}

	out := captureStdout(t, func() {
		printSummary(zero, zero, result)
	})

	assert.Contains(t, out, "0 entries")
	assert.NotContains(t, out, "Foundation")
	assert.NotContains(t, out, "Application")
	assert.NotContains(t, out, "Topics")
	assert.NotContains(t, out, "Prompts")
	assert.NotContains(t, out, "Plans")
	assert.NotContains(t, out, "Discovered")
	assert.NotContains(t, out, "Removed")
	assert.NotContains(t, out, "Relationships")
}

func TestPrintSummary_applicationAndPlans(t *testing.T) {
	before := counts{}
	after := counts{application: 2, plans: 1}
	result := &manifest.Manifest{
		Application: []manifest.ApplicationEntry{
			{ID: "arch", DependsOn: []string{"philosophy"}},
			{ID: "design"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "migrate"},
		},
	}

	out := captureStdout(t, func() {
		printSummary(before, after, result)
	})

	assert.Contains(t, out, "Application")
	assert.Contains(t, out, "Plans")
	assert.Contains(t, out, "Relationships")
	assert.NotContains(t, out, "Foundation")
	assert.NotContains(t, out, "Topics")
	assert.NotContains(t, out, "Prompts")
}

func TestPrintSummary_allSections(t *testing.T) {
	before := counts{}
	after := counts{foundation: 1, application: 1, topics: 1, prompts: 1, plans: 1}
	result := &manifest.Manifest{
		Foundation:  []manifest.FoundationEntry{{ID: "a"}},
		Application: []manifest.ApplicationEntry{{ID: "b"}},
		Topics:      []manifest.TopicEntry{{ID: "c"}},
		Prompts:     []manifest.PromptEntry{{ID: "d"}},
		Plans:       []manifest.PlanEntry{{ID: "e"}},
	}

	out := captureStdout(t, func() {
		printSummary(before, after, result)
	})

	assert.Contains(t, out, "Foundation")
	assert.Contains(t, out, "Application")
	assert.Contains(t, out, "Topics")
	assert.Contains(t, out, "Prompts")
	assert.Contains(t, out, "Plans")
}
