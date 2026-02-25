package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// createFile creates a file with the given content under dir, creating
// intermediate directories as needed.
func createFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	abs := filepath.Join(dir, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
}

// emptyManifest returns a manifest with only metadata fields.
func emptyManifest() *Manifest {
	return &Manifest{
		Name:        "test-pkg",
		Author:      "test-author",
		Version:     "1.0.0",
		Description: "A test package",
	}
}

// --- extractDescription ---

func TestExtractDescription_heading(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(path, []byte("# React Conventions\n\nSome content."), 0o644))

	desc := extractDescription(path, "fallback")
	assert.Equal(t, "React Conventions", desc)
}

func TestExtractDescription_headingAfterBlankLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(path, []byte("\n\n# My Title\n\nBody."), 0o644))

	desc := extractDescription(path, "fallback")
	assert.Equal(t, "My Title", desc)
}

func TestExtractDescription_noHeading(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(path, []byte("No heading here.\nJust text."), 0o644))

	desc := extractDescription(path, "my-fallback")
	assert.Equal(t, "my-fallback", desc)
}

func TestExtractDescription_missingFile(t *testing.T) {
	desc := extractDescription("/nonexistent/doc.md", "fb")
	assert.Equal(t, "fb", desc)
}

func TestExtractDescription_emptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	desc := extractDescription(path, "fallback")
	assert.Equal(t, "fallback", desc)
}

func TestExtractDescription_h2Ignored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(path, []byte("## Not H1\n\n# Actual Title"), 0o644))

	// The first # heading found is "Actual Title" (## is skipped).
	desc := extractDescription(path, "fallback")
	assert.Equal(t, "Actual Title", desc)
}

// --- Discover: empty package ---

func TestDiscover_emptyDir(t *testing.T) {
	dir := t.TempDir()
	m := emptyManifest()

	result := Discover(dir, m)

	assert.Equal(t, "test-pkg", result.Name)
	assert.Equal(t, "test-author", result.Author)
	assert.Nil(t, result.Foundation)
	assert.Nil(t, result.Application)
	assert.Nil(t, result.Topics)
	assert.Nil(t, result.Prompts)
	assert.Nil(t, result.Plans)
}

func TestDiscover_nonexistentDir(t *testing.T) {
	m := emptyManifest()
	result := Discover("/nonexistent/pkg", m)

	// Should not panic, just return the existing manifest unchanged.
	assert.Equal(t, "test-pkg", result.Name)
}

// --- Discover: foundation ---

func TestDiscover_foundation(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n\nGuiding principles.")
	createFile(t, dir, "foundation/markdown.md", "# Markdown\n\nConventions.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Foundation, 2)
	assert.Equal(t, "markdown", result.Foundation[0].ID)
	assert.Equal(t, "foundation/markdown.md", result.Foundation[0].Path)
	assert.Equal(t, "Markdown", result.Foundation[0].Description)
	assert.Equal(t, "philosophy", result.Foundation[1].ID)
	assert.Equal(t, "Philosophy", result.Foundation[1].Description)
}

func TestDiscover_foundationSkipsNonMd(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/notes.txt", "not markdown")
	createFile(t, dir, "foundation/data.json", "{}")
	createFile(t, dir, "foundation/actual.md", "# Actual\n\nContent.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "actual", result.Foundation[0].ID)
}

func TestDiscover_foundationSkipsDirs(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/subdir/nested.md", "# Nested\n\nContent.")
	createFile(t, dir, "foundation/top.md", "# Top\n\nContent.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "top", result.Foundation[0].ID)
}

// --- Discover: topics ---

func TestDiscover_topics(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", "# React\n\nComponent conventions.")
	createFile(t, dir, "topics/react/hooks.md", "# Hooks\n\nHook patterns.")
	createFile(t, dir, "topics/react/state.md", "# State\n\nState management.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Topics, 1)
	assert.Equal(t, "react", result.Topics[0].ID)
	assert.Equal(t, "topics/react/README.md", result.Topics[0].Path)
	assert.Equal(t, "React", result.Topics[0].Description)
	assert.Equal(t, []string{
		"topics/react/hooks.md",
		"topics/react/state.md",
	}, result.Topics[0].Files)
	assert.Empty(t, result.Topics[0].Spec)
}

func TestDiscover_topicWithSpec(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/go/README.md", "# Go\n\nGo conventions.")
	createFile(t, dir, "topics/go/spec/README.md", "# Spec\n\nDesign decisions.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Topics, 1)
	assert.Equal(t, "topics/go/spec/README.md", result.Topics[0].Spec)
}

func TestDiscover_topicNoReadme(t *testing.T) {
	dir := t.TempDir()
	// Directory exists but has no README.md — should be skipped.
	createFile(t, dir, "topics/orphan/hooks.md", "# Hooks\n\nOrphan content.")

	result := Discover(dir, emptyManifest())

	assert.Nil(t, result.Topics)
}

func TestDiscover_multipleTopics(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", "# React\n\nReact content.")
	createFile(t, dir, "topics/go/README.md", "# Go\n\nGo content.")
	createFile(t, dir, "topics/typescript/README.md", "# TypeScript\n\nTS content.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Topics, 3)
	assert.Equal(t, "go", result.Topics[0].ID)
	assert.Equal(t, "react", result.Topics[1].ID)
	assert.Equal(t, "typescript", result.Topics[2].ID)
}

// --- Discover: application ---

func TestDiscover_application(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "application/architecture/README.md", "# Architecture\n\nSystem architecture.")
	createFile(t, dir, "application/architecture/decisions.md", "# Decisions\n\nKey decisions.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Application, 1)
	assert.Equal(t, "architecture", result.Application[0].ID)
	assert.Equal(t, "application/architecture/README.md", result.Application[0].Path)
	assert.Equal(t, "Architecture", result.Application[0].Description)
	assert.Equal(t, []string{
		"application/architecture/decisions.md",
	}, result.Application[0].Files)
}

func TestDiscover_applicationWithSpec(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "application/api/README.md", "# API\n\nAPI documentation.")
	createFile(t, dir, "application/api/spec/README.md", "# Spec\n\nDesign decisions.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Application, 1)
	assert.Equal(t, "application/api/spec/README.md", result.Application[0].Spec)
}

func TestDiscover_applicationNoReadme(t *testing.T) {
	dir := t.TempDir()
	// Directory exists but has no README.md — should be skipped.
	createFile(t, dir, "application/orphan/notes.md", "# Notes\n\nOrphan content.")

	result := Discover(dir, emptyManifest())

	assert.Nil(t, result.Application)
}

func TestDiscover_multipleApplications(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "application/api/README.md", "# API\n\nAPI docs.")
	createFile(t, dir, "application/architecture/README.md", "# Architecture\n\nArch docs.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Application, 2)
	assert.Equal(t, "api", result.Application[0].ID)
	assert.Equal(t, "architecture", result.Application[1].ID)
}

// --- Discover: prompts ---

func TestDiscover_prompts(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "prompts/review/README.md", "# Code Review\n\nReview prompt.")
	createFile(t, dir, "prompts/refactor/README.md", "# Refactor\n\nRefactor prompt.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Prompts, 2)
	assert.Equal(t, "refactor", result.Prompts[0].ID)
	assert.Equal(t, "Refactor", result.Prompts[0].Description)
	assert.Equal(t, "review", result.Prompts[1].ID)
	assert.Equal(t, "Code Review", result.Prompts[1].Description)
}

func TestDiscover_promptNoReadme(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "prompts/orphan/notes.md", "# Notes")

	result := Discover(dir, emptyManifest())

	assert.Nil(t, result.Prompts)
}

// --- Discover: plans ---

func TestDiscover_plans(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "plans/migration/README.md", "# Migration\n\nMigration plan.")
	createFile(t, dir, "plans/migration/state.yml", "id: migration\nstatus: not_started")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Plans, 1)
	assert.Equal(t, "migration", result.Plans[0].ID)
	assert.Equal(t, "plans/migration/README.md", result.Plans[0].Path)
	assert.Equal(t, "Migration", result.Plans[0].Description)
	assert.Equal(t, "plans/migration/state.yml", result.Plans[0].State)
}

func TestDiscover_planWithoutState(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "plans/redesign/README.md", "# Redesign\n\nRedesign plan.")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Plans, 1)
	assert.Equal(t, "redesign", result.Plans[0].ID)
	assert.Empty(t, result.Plans[0].State)
}

// --- Discover: merge-missing behavior ---

func TestDiscover_preservesExistingEntries(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", "# React\n\nReact conventions.")
	createFile(t, dir, "topics/go/README.md", "# Go\n\nGo conventions.")
	createFile(t, dir, "application/architecture/README.md", "# Architecture\n\nArch docs.")
	createFile(t, dir, "application/api/README.md", "# API\n\nAPI docs.")

	existing := emptyManifest()
	existing.Topics = []TopicEntry{
		{ID: "react", Path: "topics/react/README.md", Description: "Existing description"},
	}
	existing.Application = []ApplicationEntry{
		{ID: "architecture", Path: "application/architecture/README.md", Description: "Existing arch", Load: "always"},
	}

	result := Discover(dir, existing)

	require.Len(t, result.Topics, 2)
	// Existing entry preserved with original description.
	assert.Equal(t, "react", result.Topics[0].ID)
	assert.Equal(t, "Existing description", result.Topics[0].Description)
	// New entry discovered.
	assert.Equal(t, "go", result.Topics[1].ID)
	assert.Equal(t, "Go", result.Topics[1].Description)

	// Application: existing preserved, new discovered.
	require.Len(t, result.Application, 2)
	assert.Equal(t, "architecture", result.Application[0].ID)
	assert.Equal(t, "Existing arch", result.Application[0].Description)
	assert.Equal(t, "always", result.Application[0].Load)
	assert.Equal(t, "api", result.Application[1].ID)
	assert.Equal(t, "API", result.Application[1].Description)
}

func TestDiscover_mergesMissingAcrossSections(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n\nContent.")
	createFile(t, dir, "application/architecture/README.md", "# Architecture\n\nContent.")
	createFile(t, dir, "topics/react/README.md", "# React\n\nContent.")
	createFile(t, dir, "prompts/review/README.md", "# Review\n\nContent.")
	createFile(t, dir, "plans/migration/README.md", "# Migration\n\nContent.")

	// Only has foundation pre-declared.
	existing := emptyManifest()
	existing.Foundation = []FoundationEntry{
		{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Pre-existing"},
	}

	result := Discover(dir, existing)

	// Foundation: existing preserved, nothing new.
	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "Pre-existing", result.Foundation[0].Description)

	// Other sections: discovered.
	require.Len(t, result.Application, 1)
	assert.Equal(t, "architecture", result.Application[0].ID)
	require.Len(t, result.Topics, 1)
	assert.Equal(t, "react", result.Topics[0].ID)
	require.Len(t, result.Prompts, 1)
	assert.Equal(t, "review", result.Prompts[0].ID)
	require.Len(t, result.Plans, 1)
	assert.Equal(t, "migration", result.Plans[0].ID)
}

func TestDiscover_partialSection(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy")
	createFile(t, dir, "foundation/markdown.md", "# Markdown")
	createFile(t, dir, "foundation/docs.md", "# Documentation")

	existing := emptyManifest()
	existing.Foundation = []FoundationEntry{
		{ID: "markdown", Path: "foundation/markdown.md", Description: "Manual entry", Load: "always"},
	}

	result := Discover(dir, existing)

	require.Len(t, result.Foundation, 3)
	// Existing entry first (preserved with load field).
	assert.Equal(t, "markdown", result.Foundation[0].ID)
	assert.Equal(t, "Manual entry", result.Foundation[0].Description)
	assert.Equal(t, "always", result.Foundation[0].Load)
	// Discovered entries after.
	assert.Equal(t, "docs", result.Foundation[1].ID)
	assert.Equal(t, "Documentation", result.Foundation[1].Description)
	assert.Equal(t, "philosophy", result.Foundation[2].ID)
}

func TestDiscover_doesNotMutateInput(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/new.md", "# New\n\nContent.")

	existing := emptyManifest()
	existing.Foundation = []FoundationEntry{
		{ID: "old", Path: "foundation/old.md", Description: "Original"},
	}
	origLen := len(existing.Foundation)

	result := Discover(dir, existing)

	// Input is not mutated.
	assert.Len(t, existing.Foundation, origLen)
	// Result has both.
	assert.Len(t, result.Foundation, 2)
}

// --- Discover: all entry types together ---

func TestDiscover_fullPackage(t *testing.T) {
	dir := t.TempDir()

	// Foundation.
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n\nGuiding principles.")
	createFile(t, dir, "foundation/markdown.md", "# Markdown\n\nFormatting rules.")

	// Application.
	createFile(t, dir, "application/architecture/README.md", "# Architecture\n\nSystem design.")

	// Topics.
	createFile(t, dir, "topics/react/README.md", "# React\n\nComponent patterns.")
	createFile(t, dir, "topics/react/hooks.md", "# Hooks\n\nHook patterns.")
	createFile(t, dir, "topics/react/state.md", "# State Management\n\nState patterns.")
	createFile(t, dir, "topics/react/spec/README.md", "# Spec\n\nDesign decisions.")
	createFile(t, dir, "topics/go/README.md", "# Go\n\nGo conventions.")

	// Prompts.
	createFile(t, dir, "prompts/review/README.md", "# Code Review\n\nReview process.")

	// Plans.
	createFile(t, dir, "plans/migration/README.md", "# Migration\n\nMigration plan.")
	createFile(t, dir, "plans/migration/state.yml", "id: migration\nstatus: not_started")

	result := Discover(dir, emptyManifest())

	// Foundation: 2 entries.
	require.Len(t, result.Foundation, 2)
	assert.Equal(t, "markdown", result.Foundation[0].ID)
	assert.Equal(t, "philosophy", result.Foundation[1].ID)

	// Application: 1 entry.
	require.Len(t, result.Application, 1)
	assert.Equal(t, "architecture", result.Application[0].ID)

	// Topics: 2 entries.
	require.Len(t, result.Topics, 2)
	assert.Equal(t, "go", result.Topics[0].ID)
	assert.Nil(t, result.Topics[0].Files)
	assert.Equal(t, "react", result.Topics[1].ID)
	assert.Equal(t, "topics/react/spec/README.md", result.Topics[1].Spec)
	assert.Equal(t, []string{
		"topics/react/hooks.md",
		"topics/react/state.md",
	}, result.Topics[1].Files)

	// Prompts: 1 entry.
	require.Len(t, result.Prompts, 1)
	assert.Equal(t, "review", result.Prompts[0].ID)

	// Plans: 1 entry.
	require.Len(t, result.Plans, 1)
	assert.Equal(t, "migration", result.Plans[0].ID)
	assert.Equal(t, "plans/migration/state.yml", result.Plans[0].State)
}

// --- Discover: description fallback ---

func TestDiscover_descriptionFallback(t *testing.T) {
	dir := t.TempDir()
	// File with no heading.
	createFile(t, dir, "foundation/noheading.md", "Just plain text, no markdown heading.\n")

	result := Discover(dir, emptyManifest())

	require.Len(t, result.Foundation, 1)
	assert.Equal(t, "noheading", result.Foundation[0].ID)
	assert.Equal(t, "noheading", result.Foundation[0].Description)
}

// --- idSet ---

func TestIdSet(t *testing.T) {
	entries := []FoundationEntry{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}
	s := idSet(entries, func(e FoundationEntry) string { return e.ID })
	assert.True(t, s["a"])
	assert.True(t, s["b"])
	assert.True(t, s["c"])
	assert.False(t, s["d"])
}

func TestIdSet_empty(t *testing.T) {
	s := idSet([]FoundationEntry(nil), func(e FoundationEntry) string { return e.ID })
	assert.Empty(t, s)
}

// --- copy helpers ---

func TestCopyFoundation_nil(t *testing.T) {
	assert.Nil(t, copyFoundation(nil))
}

func TestCopyFoundation_nonNil(t *testing.T) {
	orig := []FoundationEntry{{ID: "a"}}
	cp := copyFoundation(orig)
	require.Len(t, cp, 1)
	assert.Equal(t, "a", cp[0].ID)
	// Mutating copy should not affect original.
	cp[0].ID = "b"
	assert.Equal(t, "a", orig[0].ID)
}

func TestCopyApplication_nil(t *testing.T) {
	assert.Nil(t, copyApplication(nil))
}

func TestCopyApplication_nonNil(t *testing.T) {
	orig := []ApplicationEntry{
		{ID: "arch", Path: "application/arch/README.md", Description: "Architecture", Load: "always"},
	}
	cp := copyApplication(orig)
	require.Len(t, cp, 1)
	assert.Equal(t, "arch", cp[0].ID)
	assert.Equal(t, "always", cp[0].Load)
	// Mutating copy should not affect original.
	cp[0].ID = "modified"
	cp[0].Load = "never"
	assert.Equal(t, "arch", orig[0].ID)
	assert.Equal(t, "always", orig[0].Load)
}

func TestCopyTopics_nil(t *testing.T) {
	assert.Nil(t, copyTopics(nil))
}

func TestCopyPrompts_nil(t *testing.T) {
	assert.Nil(t, copyPrompts(nil))
}

func TestCopyPlans_nil(t *testing.T) {
	assert.Nil(t, copyPlans(nil))
}
