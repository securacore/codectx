package manifest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- extractLinks ---

func TestExtractLinks_simpleLinks(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `# Test
See [philosophy](foundation/philosophy.md) for details.
Also check [react](../topics/react/README.md).
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	assert.Equal(t, []string{"foundation/philosophy.md", "../topics/react/README.md"}, links)
}

func TestExtractLinks_skipsHttpLinks(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `
[Google](https://google.com)
[HTTP](http://example.com/page.md)
[Local](local.md)
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	assert.Equal(t, []string{"local.md"}, links)
}

func TestExtractLinks_skipsFragmentOnly(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `
[Section](#section)
[Other](other.md#section)
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	// #section is skipped (fragment-only), other.md#section keeps the file part
	assert.Equal(t, []string{"other.md"}, links)
}

func TestExtractLinks_deduplicates(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `
[A](target.md)
[B](target.md)
[C](other.md)
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	assert.Equal(t, []string{"target.md", "other.md"}, links)
}

func TestExtractLinks_multiplePerLine(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `See [A](a.md) and [B](b.md) plus [C](c.md).`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	assert.Equal(t, []string{"a.md", "b.md", "c.md"}, links)
}

func TestExtractLinks_onlyMdFiles(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `
[Schema](schema.json)
[Config](codectx.yml)
[Doc](doc.md)
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	// Only .md files are matched
	assert.Equal(t, []string{"doc.md"}, links)
}

func TestExtractLinks_missingFile(t *testing.T) {
	links := extractLinks("/nonexistent/path/file.md")
	assert.Nil(t, links)
}

func TestExtractLinks_emptyFile(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "empty.md", "")
	links := extractLinks(filepath.Join(dir, "empty.md"))
	assert.Nil(t, links)
}

func TestExtractLinks_relativePathPatterns(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `
[Sibling](sibling.md)
[Parent](../parent.md)
[Grandparent](../../grandparent.md)
[Deep](../../../foundation/philosophy.md)
[Child](child/doc.md)
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	assert.Equal(t, []string{
		"sibling.md",
		"../parent.md",
		"../../grandparent.md",
		"../../../foundation/philosophy.md",
		"child/doc.md",
	}, links)
}

// --- resolveLink ---

func TestResolveLink_sameDirectory(t *testing.T) {
	result := resolveLink("foundation/philosophy.md", "markdown.md")
	assert.Equal(t, filepath.Join("foundation", "markdown.md"), result)
}

func TestResolveLink_parentDirectory(t *testing.T) {
	result := resolveLink("topics/react/README.md", "../typescript/README.md")
	assert.Equal(t, filepath.Join("topics", "typescript", "README.md"), result)
}

func TestResolveLink_twoLevelsUp(t *testing.T) {
	result := resolveLink("topics/react/README.md", "../../foundation/philosophy.md")
	assert.Equal(t, filepath.Join("foundation", "philosophy.md"), result)
}

func TestResolveLink_threeLevelsUp(t *testing.T) {
	result := resolveLink("topics/react/spec/README.md", "../../../foundation/philosophy.md")
	assert.Equal(t, filepath.Join("foundation", "philosophy.md"), result)
}

func TestResolveLink_childPath(t *testing.T) {
	result := resolveLink("topics/react/README.md", "spec/README.md")
	assert.Equal(t, filepath.Join("topics", "react", "spec", "README.md"), result)
}

func TestResolveLink_escapesRoot(t *testing.T) {
	result := resolveLink("foundation/philosophy.md", "../../outside.md")
	assert.Equal(t, "", result)
}

func TestResolveLink_fromFoundationToTopic(t *testing.T) {
	result := resolveLink("foundation/review-standards.md", "../prompts/docs-audit/README.md")
	assert.Equal(t, filepath.Join("prompts", "docs-audit", "README.md"), result)
}

func TestResolveLink_fromSpecToPeerTopic(t *testing.T) {
	result := resolveLink("topics/nextjs/spec/README.md", "../../react/memoization.md")
	assert.Equal(t, filepath.Join("topics", "react", "memoization.md"), result)
}

// --- pathToEntryID ---

func TestPathToEntryID_allSections(t *testing.T) {
	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Application: []ApplicationEntry{
			{ID: "architecture", Path: "application/architecture/README.md",
				Spec:  "application/architecture/spec/README.md",
				Files: []string{"application/architecture/decisions.md"}},
		},
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md",
				Spec:  "topics/react/spec/README.md",
				Files: []string{"topics/react/hooks.md"}},
		},
		Prompts: []PromptEntry{
			{ID: "docs-audit", Path: "prompts/docs-audit/README.md"},
		},
		Plans: []PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md"},
		},
	}

	index := pathToEntryID(m)

	assert.Equal(t, "philosophy", index["foundation/philosophy.md"])
	assert.Equal(t, "architecture", index["application/architecture/README.md"])
	assert.Equal(t, "architecture", index["application/architecture/spec/README.md"])
	assert.Equal(t, "architecture", index["application/architecture/decisions.md"])
	assert.Equal(t, "react", index["topics/react/README.md"])
	assert.Equal(t, "react", index["topics/react/spec/README.md"])
	assert.Equal(t, "react", index["topics/react/hooks.md"])
	assert.Equal(t, "docs-audit", index["prompts/docs-audit/README.md"])
	assert.Equal(t, "migrate", index["plans/migrate/README.md"])
}

func TestPathToEntryID_empty(t *testing.T) {
	m := &Manifest{}
	index := pathToEntryID(m)
	assert.Empty(t, index)
}

// --- inferRelationships ---

func TestInferRelationships_crossTopicLinks(t *testing.T) {
	dir := t.TempDir()
	// react links to typescript
	createFile(t, dir, "topics/react/README.md", `# React
See [TypeScript](../typescript/README.md) for type conventions.
`)
	// typescript links back to react (bidirectional links in source)
	createFile(t, dir, "topics/typescript/README.md", `# TypeScript
Used by [React](../react/README.md).
`)

	m := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
			{ID: "typescript", Path: "topics/typescript/README.md"},
		},
	}

	inferRelationships(dir, m)

	// react depends_on typescript, typescript depends_on react (both link to each other)
	assert.Equal(t, []string{"typescript"}, m.Topics[0].DependsOn)
	assert.Equal(t, []string{"typescript"}, m.Topics[0].RequiredBy)
	assert.Equal(t, []string{"react"}, m.Topics[1].DependsOn)
	assert.Equal(t, []string{"react"}, m.Topics[1].RequiredBy)
}

func TestInferRelationships_topicToFoundation(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "topics/react/README.md", `# React
Follow [philosophy](../../foundation/philosophy.md).
`)

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	inferRelationships(dir, m)

	assert.Equal(t, []string{"philosophy"}, m.Topics[0].DependsOn)
	assert.Nil(t, m.Topics[0].RequiredBy)
	assert.Nil(t, m.Foundation[0].DependsOn)
	assert.Equal(t, []string{"react"}, m.Foundation[0].RequiredBy)
}

func TestInferRelationships_intraEntryLinksIgnored(t *testing.T) {
	dir := t.TempDir()
	// Topic sub-file links to its own spec (intra-entry)
	createFile(t, dir, "topics/react/README.md", `# React
See [spec](spec/README.md).
See [hooks](hooks.md).
`)
	createFile(t, dir, "topics/react/spec/README.md", "# Spec\n")
	createFile(t, dir, "topics/react/hooks.md", "# Hooks\n")

	m := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md",
				Spec:  "topics/react/spec/README.md",
				Files: []string{"topics/react/hooks.md"}},
		},
	}

	inferRelationships(dir, m)

	// No relationships — all links are intra-entry
	assert.Nil(t, m.Topics[0].DependsOn)
	assert.Nil(t, m.Topics[0].RequiredBy)
}

func TestInferRelationships_subFileLinksCreateDependency(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", "# React\n")
	createFile(t, dir, "topics/react/hooks.md", `# Hooks
See [TypeScript](../typescript/README.md).
`)
	createFile(t, dir, "topics/typescript/README.md", "# TypeScript\n")

	m := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md",
				Files: []string{"topics/react/hooks.md"}},
			{ID: "typescript", Path: "topics/typescript/README.md"},
		},
	}

	inferRelationships(dir, m)

	// react (via hooks.md sub-file) depends_on typescript
	assert.Equal(t, []string{"typescript"}, m.Topics[0].DependsOn)
	assert.Nil(t, m.Topics[0].RequiredBy)
	assert.Nil(t, m.Topics[1].DependsOn)
	assert.Equal(t, []string{"react"}, m.Topics[1].RequiredBy)
}

func TestInferRelationships_specLinksCreateDependency(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/nextjs/README.md", "# Next.js\n")
	createFile(t, dir, "topics/nextjs/spec/README.md", `# Spec
See [philosophy](../../../foundation/philosophy.md).
See [react](../../react/README.md).
`)
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "topics/react/README.md", "# React\n")

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Topics: []TopicEntry{
			{ID: "nextjs", Path: "topics/nextjs/README.md",
				Spec: "topics/nextjs/spec/README.md"},
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	inferRelationships(dir, m)

	// nextjs (via spec) depends_on philosophy and react
	assert.Equal(t, []string{"philosophy", "react"}, m.Topics[0].DependsOn)
	assert.Equal(t, []string{"nextjs"}, m.Foundation[0].RequiredBy)
	assert.Equal(t, []string{"nextjs"}, m.Topics[1].RequiredBy)
}

func TestInferRelationships_clearsExistingRelationships(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/a.md", "# A\n")
	createFile(t, dir, "foundation/b.md", "# B\n")

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "a", Path: "foundation/a.md",
				DependsOn: []string{"stale"}, RequiredBy: []string{"old"}},
			{ID: "b", Path: "foundation/b.md",
				DependsOn: []string{"gone"}, RequiredBy: []string{"removed"}},
		},
	}

	inferRelationships(dir, m)

	// No links exist, so all relationships should be cleared
	assert.Nil(t, m.Foundation[0].DependsOn)
	assert.Nil(t, m.Foundation[0].RequiredBy)
	assert.Nil(t, m.Foundation[1].DependsOn)
	assert.Nil(t, m.Foundation[1].RequiredBy)
}

func TestInferRelationships_multipleDependencies(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", `# React
See [TypeScript](../typescript/README.md).
See [Go](../go/README.md).
`)
	createFile(t, dir, "topics/typescript/README.md", "# TypeScript\n")
	createFile(t, dir, "topics/go/README.md", "# Go\n")

	m := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
			{ID: "typescript", Path: "topics/typescript/README.md"},
			{ID: "go", Path: "topics/go/README.md"},
		},
	}

	inferRelationships(dir, m)

	// react depends on both go and typescript (sorted)
	assert.Equal(t, []string{"go", "typescript"}, m.Topics[0].DependsOn)
	// go and typescript are required_by react
	assert.Equal(t, []string{"react"}, m.Topics[1].RequiredBy)
	assert.Equal(t, []string{"react"}, m.Topics[2].RequiredBy)
}

func TestInferRelationships_applicationEntries(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "application/architecture/README.md", `# Architecture
See [philosophy](../../foundation/philosophy.md).
`)
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Application: []ApplicationEntry{
			{ID: "architecture", Path: "application/architecture/README.md"},
		},
	}

	inferRelationships(dir, m)

	assert.Equal(t, []string{"philosophy"}, m.Application[0].DependsOn)
	assert.Equal(t, []string{"architecture"}, m.Foundation[0].RequiredBy)
}

func TestInferRelationships_promptToFoundation(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "prompts/audit/README.md", `# Audit
Load [review-standards](../../foundation/review-standards.md).
`)
	createFile(t, dir, "foundation/review-standards.md", "# Review Standards\n")

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "review-standards", Path: "foundation/review-standards.md"},
		},
		Prompts: []PromptEntry{
			{ID: "audit", Path: "prompts/audit/README.md"},
		},
	}

	inferRelationships(dir, m)

	assert.Equal(t, []string{"review-standards"}, m.Prompts[0].DependsOn)
	assert.Equal(t, []string{"audit"}, m.Foundation[0].RequiredBy)
}

func TestInferRelationships_noFiles(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{}
	inferRelationships(dir, m)
	// No panic, no error
}

func TestInferRelationships_linkToUnknownEntry(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", `# React
See [unknown](../unknown/README.md).
`)

	m := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	inferRelationships(dir, m)

	// Link to unknown entry is silently ignored
	assert.Nil(t, m.Topics[0].DependsOn)
}

// --- inferRelationships: plans section ---

func TestInferRelationships_planEntries(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "plans/migrate/README.md", `# Migrate
See [philosophy](../../foundation/philosophy.md).
`)

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Plans: []PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md"},
		},
	}

	inferRelationships(dir, m)

	assert.Equal(t, []string{"philosophy"}, m.Plans[0].DependsOn)
	assert.Nil(t, m.Plans[0].RequiredBy)
	assert.Nil(t, m.Foundation[0].DependsOn)
	assert.Equal(t, []string{"migrate"}, m.Foundation[0].RequiredBy)
}

// --- inferRelationships: application spec and files ---

func TestInferRelationships_applicationSpecAndFiles(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "application/arch/README.md", "# Architecture\n")
	createFile(t, dir, "application/arch/spec/README.md", `# Spec
See [philosophy](../../../foundation/philosophy.md).
`)
	createFile(t, dir, "application/arch/decisions.md", `# Decisions
See [philosophy](../../foundation/philosophy.md).
`)
	createFile(t, dir, "topics/react/README.md", "# React\n")

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Application: []ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md",
				Spec:  "application/arch/spec/README.md",
				Files: []string{"application/arch/decisions.md"}},
		},
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	inferRelationships(dir, m)

	// arch depends on philosophy via both spec and files
	assert.Equal(t, []string{"philosophy"}, m.Application[0].DependsOn)
	assert.Equal(t, []string{"arch"}, m.Foundation[0].RequiredBy)
}

func TestInferRelationships_missingFileOnDisk(t *testing.T) {
	dir := t.TempDir()
	// Don't create the file on disk — only declare it in the manifest.
	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
	}

	inferRelationships(dir, m)

	// No panic, no relationships (file doesn't exist so no links parsed).
	assert.Nil(t, m.Foundation[0].DependsOn)
	assert.Nil(t, m.Foundation[0].RequiredBy)
}

func TestInferRelationships_linkEscapesRoot(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", `# Philosophy
See [outside](../../outside.md).
`)

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
	}

	inferRelationships(dir, m)

	// Link escapes root, silently ignored.
	assert.Nil(t, m.Foundation[0].DependsOn)
}

func TestInferRelationships_selfLink(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "topics/react/README.md", `# React
See [self](README.md).
`)

	m := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}

	inferRelationships(dir, m)

	// Self-link resolves to own entry Path, filtered as intra-entry.
	assert.Nil(t, m.Topics[0].DependsOn)
}

func TestInferRelationships_crossSectionChain(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/philosophy.md", "# Philosophy\n")
	createFile(t, dir, "topics/react/README.md", `# React
See [philosophy](../../foundation/philosophy.md).
`)
	createFile(t, dir, "prompts/audit/README.md", `# Audit
See [react](../../topics/react/README.md).
See [philosophy](../../foundation/philosophy.md).
`)

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md"},
		},
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
		Prompts: []PromptEntry{
			{ID: "audit", Path: "prompts/audit/README.md"},
		},
	}

	inferRelationships(dir, m)

	// audit depends on both philosophy and react
	assert.Equal(t, []string{"philosophy", "react"}, m.Prompts[0].DependsOn)
	// react depends on philosophy
	assert.Equal(t, []string{"philosophy"}, m.Topics[0].DependsOn)
	// philosophy required_by audit and react
	assert.Equal(t, []string{"audit", "react"}, m.Foundation[0].RequiredBy)
	// react required_by audit
	assert.Equal(t, []string{"audit"}, m.Topics[0].RequiredBy)
}

func TestInferRelationships_diamondDependency(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "foundation/base.md", "# Base\n")
	createFile(t, dir, "topics/a/README.md", `# A
See [base](../../foundation/base.md).
See [b](../b/README.md).
`)
	createFile(t, dir, "topics/b/README.md", `# B
See [base](../../foundation/base.md).
`)

	m := &Manifest{
		Foundation: []FoundationEntry{
			{ID: "base", Path: "foundation/base.md"},
		},
		Topics: []TopicEntry{
			{ID: "a", Path: "topics/a/README.md"},
			{ID: "b", Path: "topics/b/README.md"},
		},
	}

	inferRelationships(dir, m)

	// base required_by both a and b
	assert.Equal(t, []string{"a", "b"}, m.Foundation[0].RequiredBy)
	// a depends on base and b
	assert.Equal(t, []string{"b", "base"}, m.Topics[0].DependsOn)
	// b depends on base only
	assert.Equal(t, []string{"base"}, m.Topics[1].DependsOn)
	// b required_by a
	assert.Equal(t, []string{"a"}, m.Topics[1].RequiredBy)
}

// --- extractLinks: edge cases ---

func TestExtractLinks_caseSensitiveMd(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `[upper](FILE.MD)
[lower](file.md)
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	// Only lowercase .md matches the regex.
	assert.Equal(t, []string{"file.md"}, links)
}

func TestExtractLinks_imageLinks(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `![diagram](diagram.md)
![img](image.png)
[normal](normal.md)
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	// Image link syntax ![alt](url) is also matched by the regex since it
	// matches [text](url). The ! is before the [, not part of the capture.
	assert.Equal(t, []string{"diagram.md", "normal.md"}, links)
}

func TestExtractLinks_noLinksInContent(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `# Heading

Some plain text without any links.
Code: `+"`func foo() {}`"+`
More text.
`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	assert.Nil(t, links)
}

func TestExtractLinks_mdInDirectoryName(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `[x](docs.md-old/README.md)`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	assert.Equal(t, []string{"docs.md-old/README.md"}, links)
}

func TestExtractLinks_trailingHashNoFragment(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "doc.md", `[x](file.md#)`)
	links := extractLinks(filepath.Join(dir, "doc.md"))
	// Fragment is empty after #, stripped to "file.md"
	assert.Equal(t, []string{"file.md"}, links)
}

// --- resolveLink: edge cases ---

func TestResolveLink_emptyTarget(t *testing.T) {
	result := resolveLink("foundation/a.md", "")
	// filepath.Join("foundation", "") = "foundation", then Clean = "foundation"
	assert.Equal(t, "foundation", result)
}

func TestResolveLink_emptySource(t *testing.T) {
	result := resolveLink("", "foo.md")
	assert.Equal(t, "foo.md", result)
}

func TestResolveLink_dotSegment(t *testing.T) {
	result := resolveLink("topics/react/README.md", "./spec/README.md")
	assert.Equal(t, filepath.Join("topics", "react", "spec", "README.md"), result)
}

func TestResolveLink_rootLevelSource(t *testing.T) {
	result := resolveLink("README.md", "other.md")
	assert.Equal(t, "other.md", result)
}

func TestResolveLink_upToRoot(t *testing.T) {
	result := resolveLink("foundation/a.md", "../README.md")
	assert.Equal(t, "README.md", result)
}

// --- pathToEntryID: edge cases ---

func TestPathToEntryID_noSpec(t *testing.T) {
	m := &Manifest{
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md"},
		},
	}
	index := pathToEntryID(m)
	assert.Equal(t, "react", index["topics/react/README.md"])
	// Empty spec should not be in the index.
	_, hasEmpty := index[""]
	assert.False(t, hasEmpty)
}

func TestPathToEntryID_noFiles(t *testing.T) {
	m := &Manifest{
		Application: []ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md"},
		},
	}
	index := pathToEntryID(m)
	assert.Equal(t, "arch", index["application/arch/README.md"])
	assert.Len(t, index, 1)
}

// --- sortedKeys ---

func TestSortedKeys(t *testing.T) {
	s := map[string]bool{"c": true, "a": true, "b": true}
	assert.Equal(t, []string{"a", "b", "c"}, sortedKeys(s))
}

func TestSortedKeys_empty(t *testing.T) {
	s := map[string]bool{}
	result := sortedKeys(s)
	assert.Empty(t, result)
}

func TestSortedKeys_nil(t *testing.T) {
	result := sortedKeys(nil)
	assert.Empty(t, result)
}

func TestSortedKeys_single(t *testing.T) {
	s := map[string]bool{"only": true}
	assert.Equal(t, []string{"only"}, sortedKeys(s))
}
