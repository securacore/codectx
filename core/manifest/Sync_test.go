package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSync_discoversNewEntries(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "topics/react/README.md", "# React\n")
	createFile(t, dir, "application/architecture/README.md", "# Architecture\n")

	result := Sync(dir, emptyManifest())

	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)
	require.Len(t, result.Application, 1)
	assert.Equal(t, "architecture", result.Application[0].ID)
	require.Len(t, result.Topics, 1)
	assert.Equal(t, "react", result.Topics[0].ID)
}

func TestSync_removesStaleEntries(t *testing.T) {
	dir := t.TempDir()
	// Only philosophy.md exists on disk, not markdown.md
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")

	existing := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Philosophy"},
			{ID: "markdown", Path: "foundation/markdown.md", Description: "Markdown"},
		},
	}

	result := Sync(dir, existing)

	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)
}

func TestSync_removesStaleTopics(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", "# React\n")
	// typescript directory does not exist

	existing := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React"},
			{ID: "typescript", Path: "topics/typescript/README.md", Description: "TypeScript"},
		},
	}

	result := Sync(dir, existing)

	require.Len(t, result.Topics, 1)
	assert.Equal(t, "react", result.Topics[0].ID)
}

func TestSync_removesStaleApplication(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "application/architecture/README.md", "# Architecture\n")

	existing := &Manifest{
		Application: []ApplicationEntry{
			{ID: "architecture", Path: "application/architecture/README.md"},
			{ID: "removed", Path: "application/removed/README.md"},
		},
	}

	result := Sync(dir, existing)

	require.Len(t, result.Application, 1)
	assert.Equal(t, "architecture", result.Application[0].ID)
}

func TestSync_removesStalePrompts(t *testing.T) {
	dir := t.TempDir()

	existing := &Manifest{
		Prompts: []PromptEntry{
			{ID: "audit", Path: "prompts/audit/README.md"},
		},
	}

	result := Sync(dir, existing)

	assert.Nil(t, result.Prompts)
}

func TestSync_removesStalePlans(t *testing.T) {
	dir := t.TempDir()

	existing := &Manifest{
		Plans: []PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md"},
		},
	}

	result := Sync(dir, existing)

	assert.Nil(t, result.Plans)
}

func TestSync_infersRelationships(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "topics/react/README.md", `# React
See [philosophy](../../foundation/philosophy.md).
See [TypeScript](../typescript/README.md).
`)
	createFile(t, dir, "topics/typescript/README.md", "# TypeScript\n")

	result := Sync(dir, emptyManifest())

	// react depends on philosophy and typescript
	require.Len(t, result.Topics, 2)
	reactIdx := 0
	if result.Topics[0].ID != "react" {
		reactIdx = 1
	}
	assert.Equal(t, []string{"philosophy", "typescript"}, result.Topics[reactIdx].DependsOn)

	// philosophy is required_by react
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, []string{"react"}, result.Foundation[0].RequiredBy)
}

func TestSync_discoversAndRemovesAndInfers(t *testing.T) {
	dir := t.TempDir()
	// Create new entries on disk
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "topics/react/README.md", `# React
See [philosophy](../../foundation/philosophy.md).
`)

	// Existing manifest has a stale entry
	existing := &Manifest{
		Name:    "test-project",
		Version: "1.0.0",
		Foundation: []FoundationEntry{
			{ID: "stale", Path: "foundation/stale.md", Description: "Stale entry"},
		},
	}

	result := Sync(dir, existing)

	// Metadata preserved
	assert.Equal(t, "test-project", result.Name)
	assert.Equal(t, "1.0.0", result.Version)

	// Stale removed, new discovered
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)

	// Relationships inferred
	require.Len(t, result.Topics, 1)
	assert.Equal(t, []string{"philosophy"}, result.Topics[0].DependsOn)
	assert.Equal(t, []string{"react"}, result.Foundation[0].RequiredBy)
}

func TestSync_preservesEntryMetadata(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "application/arch/README.md", "# Architecture\n")

	existing := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md",
				Description: "Custom description", Load: "always"},
		},
		Application: []ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md",
				Description: "Architecture overview", Load: "documentation"},
		},
	}

	result := Sync(dir, existing)

	// Preserved fields from existing entries
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "Custom description", result.Foundation[0].Description)
	assert.Equal(t, "always", result.Foundation[0].Load)

	require.Len(t, result.Application, 1)
	assert.Equal(t, "Architecture overview", result.Application[0].Description)
	assert.Equal(t, "documentation", result.Application[0].Load)
}

func TestSync_emptyDir(t *testing.T) {
	dir := t.TempDir()
	result := Sync(dir, emptyManifest())

	assert.Nil(t, result.Foundation)
	assert.Nil(t, result.Application)
	assert.Nil(t, result.Topics)
	assert.Nil(t, result.Prompts)
	assert.Nil(t, result.Plans)
}

func TestSync_fullPackage(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "application/architecture/README.md", `# Architecture
See [philosophy](../../foundation/philosophy.md).
`)
	createFile(t, dir, "topics/react/README.md", `# React
See [philosophy](../../foundation/philosophy.md).
See [TypeScript](../typescript/README.md).
`)
	createFile(t, dir, "topics/typescript/README.md", `# TypeScript
See [philosophy](../../foundation/philosophy.md).
`)
	createFile(t, dir, "prompts/audit/README.md", `# Audit
Load [philosophy](../../foundation/philosophy.md).
`)
	createFile(t, dir, "plans/migrate/README.md", "# Migration\n")
	createFile(t, dir, "plans/migrate/state.yml", "status: in-progress\n")

	result := Sync(dir, emptyManifest())

	// All sections discovered
	require.Len(t, result.Foundation, 1)
	require.Len(t, result.Application, 1)
	require.Len(t, result.Topics, 2)
	require.Len(t, result.Prompts, 1)
	require.Len(t, result.Plans, 1)

	// philosophy is required_by architecture, react, typescript, audit
	assert.Len(t, result.Foundation[0].RequiredBy, 4)

	// architecture depends on philosophy
	assert.Equal(t, []string{"philosophy"}, result.Application[0].DependsOn)
}

func TestSync_nilSections(t *testing.T) {
	dir := t.TempDir()
	existing := &Manifest{
		Name: "test",
	}
	result := Sync(dir, existing)
	assert.Equal(t, "test", result.Name)
}

// --- Sync: pipeline ordering ---

func TestSync_staleEntryRemovedBeforeRelationshipInference(t *testing.T) {
	dir := t.TempDir()
	// Create react and philosophy on disk. typescript is stale (no file).
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "topics/react/README.md", `# React
See [philosophy](../../foundation/philosophy.md).
See [TypeScript](../typescript/README.md).
`)

	existing := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
			{ID: "typescript", Path: "topics/typescript/README.md"}, // stale
		},
	}

	result := Sync(dir, existing)

	// typescript must be removed (stale) before relationship inference.
	require.Len(t, result.Topics, 1)
	assert.Equal(t, "react", result.Topics[0].ID)

	// react depends on philosophy only (not typescript, which was removed).
	assert.Equal(t, []string{"philosophy"}, result.Topics[0].DependsOn)

	// philosophy required_by react only.
	assert.Equal(t, []string{"react"}, result.Foundation[0].RequiredBy)
}

func TestSync_doesNotMutateInput(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "topics/react/README.md", `# React
See [philosophy](../../foundation/philosophy.md).
`)

	existing := &Manifest{
		Name:    "original",
		Version: "1.0.0",
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Original desc"},
		},
	}

	// Capture original state.
	origName := existing.Name
	origFoundLen := len(existing.Foundation)
	origDesc := existing.Foundation[0].Description
	origDeps := existing.Foundation[0].DependsOn

	_ = Sync(dir, existing)

	// Input must not be mutated.
	assert.Equal(t, origName, existing.Name)
	assert.Len(t, existing.Foundation, origFoundLen)
	assert.Equal(t, origDesc, existing.Foundation[0].Description)
	assert.Equal(t, origDeps, existing.Foundation[0].DependsOn)
	assert.Nil(t, existing.Topics) // Topics not added to input.
}

func TestSync_replacesExistingRelationships(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/a.md", "# A\n")
	createFile(t, dir, "foundation/b.md", `# B
See [a](a.md).
`)

	existing := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "a", Path: "foundation/a.md",
				DependsOn: []string{"stale-dep"}, RequiredBy: []string{"stale-rev"}},
			{ID: "b", Path: "foundation/b.md",
				DependsOn: []string{"old-dep"}, RequiredBy: []string{"old-rev"}},
		},
	}

	result := Sync(dir, existing)

	// Old relationships cleared, rebuilt from links.
	assert.Nil(t, result.Foundation[0].DependsOn) // a has no outbound links
	assert.Equal(t, []string{"b"}, result.Foundation[0].RequiredBy)
	assert.Equal(t, []string{"a"}, result.Foundation[1].DependsOn)
	assert.Nil(t, result.Foundation[1].RequiredBy) // b is not linked to by anyone
}

func TestSync_nonexistentDir(t *testing.T) {
	existing := &Manifest{
		Name:    "test",
		Version: "1.0.0",
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	result := Sync("/nonexistent/dir/12345", existing)

	// All entries removed (files don't exist), metadata preserved.
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Nil(t, result.Foundation)
	assert.Nil(t, result.Topics)
}

func TestSync_allMetadataPreserved(t *testing.T) {
	dir := t.TempDir()

	existing := &Manifest{
		Name:        "my-pkg",
		Author:      "my-author",
		Version:     "2.5.0",
		Description: "A comprehensive docs package",
	}

	result := Sync(dir, existing)

	assert.Equal(t, "my-pkg", result.Name)
	assert.Equal(t, "my-author", result.Author)
	assert.Equal(t, "2.5.0", result.Version)
	assert.Equal(t, "A comprehensive docs package", result.Description)
}

func TestSync_mergesNewEntriesIntoExistingSection(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "foundation/markdown.md", "# Markdown\n")

	existing := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md",
				Description: "Custom desc", Load: "always"},
		},
	}

	result := Sync(dir, existing)

	require.Len(t, result.Foundation, 2)
	// Existing entry preserves metadata.
	var philosophy, markdown *FoundationEntry
	for i := range result.Foundation {
		if result.Foundation[i].ID == "philosophy" {
			philosophy = &result.Foundation[i]
		}
		if result.Foundation[i].ID == "markdown" {
			markdown = &result.Foundation[i]
		}
	}
	require.NotNil(t, philosophy)
	require.NotNil(t, markdown)
	assert.Equal(t, "Custom desc", philosophy.Description)
	assert.Equal(t, "always", philosophy.Load)
	// New entry discovered with auto-description.
	assert.Equal(t, "Markdown", markdown.Description)
}

func TestSync_preservesTopicSpecAndFiles(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", "# React\n")
	createFile(t, dir, "topics/react/spec/README.md", "# Spec\n")
	createFile(t, dir, "topics/react/hooks.md", "# Hooks\n")

	existing := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md",
				Description: "React conventions",
				Spec:        "topics/react/spec/README.md",
				Files:       []string{"topics/react/hooks.md"}},
		},
	}

	result := Sync(dir, existing)

	require.Len(t, result.Topics, 1)
	assert.Equal(t, "React conventions", result.Topics[0].Description)
	assert.Equal(t, "topics/react/spec/README.md", result.Topics[0].Spec)
	assert.Equal(t, []string{"topics/react/hooks.md"}, result.Topics[0].Files)
}

func TestSync_mixedStaleAcrossAllSections(t *testing.T) {
	dir := t.TempDir()
	// Create one valid file per section, leave the stale ones missing.
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "application/arch/README.md", "# Architecture\n")
	createFile(t, dir, "topics/react/README.md", "# React\n")
	createFile(t, dir, "prompts/audit/README.md", "# Audit\n")
	createFile(t, dir, "plans/migrate/README.md", "# Migrate\n")

	existing := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
			{ID: "stale-f", Path: "foundation/stale.md"},
		},
		Application: []ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md"},
			{ID: "stale-a", Path: "application/stale/README.md"},
		},
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
			{ID: "stale-t", Path: "topics/stale/README.md"},
		},
		Prompts: []PromptEntry{
			{ID: "audit", Path: "prompts/audit/README.md"},
			{ID: "stale-p", Path: "prompts/stale/README.md"},
		},
		Plans: []PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md"},
			{ID: "stale-pl", Path: "plans/stale/README.md"},
		},
	}

	result := Sync(dir, existing)

	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "philosophy", result.Foundation[0].ID)
	require.Len(t, result.Application, 1)
	assert.Equal(t, "arch", result.Application[0].ID)
	require.Len(t, result.Topics, 1)
	assert.Equal(t, "react", result.Topics[0].ID)
	require.Len(t, result.Prompts, 1)
	assert.Equal(t, "audit", result.Prompts[0].ID)
	require.Len(t, result.Plans, 1)
	assert.Equal(t, "migrate", result.Plans[0].ID)
}

func TestSync_deterministicOutput(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/a.md", "# A\n")
	createFile(t, dir, "foundation/b.md", `# B
See [a](a.md).
`)
	createFile(t, dir, "topics/x/README.md", `# X
See [a](../../foundation/a.md).
`)

	existing := emptyManifest()

	result1 := Sync(dir, existing)
	result2 := Sync(dir, existing)

	assert.Equal(t, len(result1.Foundation), len(result2.Foundation))
	assert.Equal(t, len(result1.Topics), len(result2.Topics))
	for i := range result1.Foundation {
		assert.Equal(t, result1.Foundation[i].ID, result2.Foundation[i].ID)
		assert.Equal(t, result1.Foundation[i].DependsOn, result2.Foundation[i].DependsOn)
		assert.Equal(t, result1.Foundation[i].RequiredBy, result2.Foundation[i].RequiredBy)
	}
}

// --- fileExists ---

func TestFileExists_exists(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "test.md", "content")
	assert.True(t, fileExists(dir+"/test.md"))
}

func TestFileExists_notExists(t *testing.T) {
	assert.False(t, fileExists("/nonexistent/path.md"))
}

func TestFileExists_directory(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, fileExists(dir))
}

func TestFileExists_emptyPath(t *testing.T) {
	assert.False(t, fileExists(""))
}

func TestFileExists_symlink(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "real.md", "content")
	require.NoError(t, os.Symlink(filepath.Join(dir, "real.md"), filepath.Join(dir, "link.md")))
	assert.True(t, fileExists(filepath.Join(dir, "link.md")))
}

func TestFileExists_brokenSymlink(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Symlink(filepath.Join(dir, "missing.md"), filepath.Join(dir, "broken.md")))
	assert.False(t, fileExists(filepath.Join(dir, "broken.md")))
}
