package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_minimal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	content := `name: test-pkg
author: test-author
version: "1.0.0"
description: A test package
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "test-pkg", m.Name)
	assert.Equal(t, "test-author", m.Author)
	assert.Equal(t, "1.0.0", m.Version)
	assert.Equal(t, "A test package", m.Description)
}

func TestLoad_withEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	content := `name: full-pkg
author: org
version: "2.0.0"
description: Full package
foundation:
  - id: philosophy
    path: foundation/philosophy.md
    description: Core philosophy
    load: always
topics:
  - id: react
    path: topics/react/README.md
    description: React conventions
    spec: topics/react/spec/README.md
    files:
      - topics/react/hooks.md
prompts:
  - id: review
    path: prompts/review/README.md
    description: Code review prompt
plans:
  - id: migration
    path: plans/migration/README.md
    state: plans/migration/state.yml
    description: Migration plan
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)

	require.Len(t, m.Foundation, 1)
	assert.Equal(t, "philosophy", m.Foundation[0].ID)
	assert.Equal(t, "always", m.Foundation[0].Load)

	require.Len(t, m.Topics, 1)
	assert.Equal(t, "react", m.Topics[0].ID)
	assert.Equal(t, "topics/react/spec/README.md", m.Topics[0].Spec)
	assert.Equal(t, []string{"topics/react/hooks.md"}, m.Topics[0].Files)

	require.Len(t, m.Prompts, 1)
	assert.Equal(t, "review", m.Prompts[0].ID)

	require.Len(t, m.Plans, 1)
	assert.Equal(t, "migration", m.Plans[0].ID)
	assert.Equal(t, "plans/migration/state.yml", m.Plans[0].State)
}

func TestLoad_schemaValidationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	// Missing required fields.
	content := `name: bad-pkg
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	_, err = Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

func TestLoad_nonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/manifest.yml")
	assert.Error(t, err)
}

// --- Write + Load round-trip ---

func TestLoad_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	err := os.WriteFile(path, []byte(":\n  :\n- {\n  invalid:\n"), 0o644)
	require.NoError(t, err)

	_, err = Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestLoad_dependsOnRequiredBy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	content := `name: dep-pkg
author: test-author
version: "1.0.0"
description: Package with depends_on and required_by
foundation:
  - id: conventions
    path: foundation/conventions.md
    description: Coding conventions
  - id: philosophy
    path: foundation/philosophy.md
    description: Core philosophy
    depends_on:
      - conventions
    required_by:
      - react
topics:
  - id: react
    path: topics/react/README.md
    description: React conventions
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)

	require.Len(t, m.Foundation, 2)
	assert.Equal(t, []string{"conventions"}, m.Foundation[1].DependsOn)
	assert.Equal(t, []string{"react"}, m.Foundation[1].RequiredBy)
}

func TestLoad_multipleEntriesPerSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	content := `name: multi-pkg
author: test-author
version: "1.0.0"
description: Package with multiple entries
foundation:
  - id: philosophy
    path: foundation/philosophy.md
    description: Core philosophy
  - id: markdown
    path: foundation/markdown.md
    description: Markdown conventions
  - id: documentation
    path: foundation/documentation.md
    description: Documentation management
topics:
  - id: react
    path: topics/react/README.md
    description: React conventions
  - id: go
    path: topics/go/README.md
    description: Go conventions
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)

	require.Len(t, m.Foundation, 3)
	assert.Equal(t, "philosophy", m.Foundation[0].ID)
	assert.Equal(t, "markdown", m.Foundation[1].ID)
	assert.Equal(t, "documentation", m.Foundation[2].ID)

	require.Len(t, m.Topics, 2)
	assert.Equal(t, "react", m.Topics[0].ID)
	assert.Equal(t, "go", m.Topics[1].ID)
}

func TestLoad_withPromptsAndPlans(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	content := `name: prompts-plans-pkg
author: test-author
version: "1.0.0"
description: Package with prompts and plans
prompts:
  - id: review
    path: prompts/review/README.md
    description: Code review prompt
  - id: refactor
    path: prompts/refactor/README.md
    description: Refactoring prompt
plans:
  - id: migration
    path: plans/migration/README.md
    state: plans/migration/state.yml
    description: Database migration plan
  - id: redesign
    path: plans/redesign/README.md
    state: plans/redesign/state.yml
    description: UI redesign plan
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)

	require.Len(t, m.Prompts, 2)
	assert.Equal(t, "review", m.Prompts[0].ID)
	assert.Equal(t, "prompts/review/README.md", m.Prompts[0].Path)
	assert.Equal(t, "Code review prompt", m.Prompts[0].Description)
	assert.Equal(t, "refactor", m.Prompts[1].ID)

	require.Len(t, m.Plans, 2)
	assert.Equal(t, "migration", m.Plans[0].ID)
	assert.Equal(t, "plans/migration/README.md", m.Plans[0].Path)
	assert.Equal(t, "plans/migration/state.yml", m.Plans[0].State)
	assert.Equal(t, "Database migration plan", m.Plans[0].Description)
	assert.Equal(t, "redesign", m.Plans[1].ID)
	assert.Equal(t, "plans/redesign/state.yml", m.Plans[1].State)
}

// --- Write + Load round-trip ---

func TestWrite_standalone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	m := &Manifest{
		Name:        "standalone-pkg",
		Author:      "test-author",
		Version:     "1.0.0",
		Description: "Standalone write test",
	}

	err := Write(path, m)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)
	assert.Contains(t, string(data), "name: standalone-pkg")
	assert.Contains(t, string(data), "author: test-author")
	assert.Contains(t, string(data), "version:")
}

func TestWrite_withAllSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	m := &Manifest{
		Name:        "all-sections-pkg",
		Author:      "test-author",
		Version:     "2.0.0",
		Description: "All sections test",
		Foundation: []FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Core philosophy", Load: "always"},
		},
		Topics: []TopicEntry{
			{ID: "react", Path: "topics/react/README.md", Description: "React conventions"},
		},
		Prompts: []PromptEntry{
			{ID: "review", Path: "prompts/review/README.md", Description: "Code review prompt"},
		},
		Plans: []PlanEntry{
			{ID: "migration", Path: "plans/migration/README.md", State: "plans/migration/state.yml", Description: "Migration plan"},
		},
	}

	err := Write(path, m)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "all-sections-pkg", loaded.Name)
	require.Len(t, loaded.Foundation, 1)
	assert.Equal(t, "philosophy", loaded.Foundation[0].ID)
	assert.Equal(t, "always", loaded.Foundation[0].Load)
	require.Len(t, loaded.Topics, 1)
	assert.Equal(t, "react", loaded.Topics[0].ID)
	require.Len(t, loaded.Prompts, 1)
	assert.Equal(t, "review", loaded.Prompts[0].ID)
	require.Len(t, loaded.Plans, 1)
	assert.Equal(t, "migration", loaded.Plans[0].ID)
	assert.Equal(t, "plans/migration/state.yml", loaded.Plans[0].State)
}

func TestWrite_invalidPath(t *testing.T) {
	m := &Manifest{
		Name:        "bad-path-pkg",
		Author:      "test-author",
		Version:     "1.0.0",
		Description: "Invalid path test",
	}

	err := Write("/nonexistent/deep/manifest.yml", m)
	assert.Error(t, err)
}

func TestWrite_overwriteExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	first := &Manifest{
		Name:        "first-pkg",
		Author:      "first-author",
		Version:     "1.0.0",
		Description: "First manifest",
	}

	second := &Manifest{
		Name:        "second-pkg",
		Author:      "second-author",
		Version:     "2.0.0",
		Description: "Second manifest",
	}

	err := Write(path, first)
	require.NoError(t, err)

	err = Write(path, second)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "second-pkg", loaded.Name)
	assert.Equal(t, "second-author", loaded.Author)
	assert.Equal(t, "2.0.0", loaded.Version)
	assert.Equal(t, "Second manifest", loaded.Description)
}

func TestWriteAndLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	original := &Manifest{
		Name:        "round-trip-pkg",
		Author:      "test-author",
		Version:     "1.0.0",
		Description: "Round trip test",
		Foundation: []FoundationEntry{
			{ID: "doc", Path: "foundation/doc.md", Description: "A doc"},
		},
		Topics: []TopicEntry{
			{ID: "topic", Path: "topics/topic/README.md", Description: "A topic"},
		},
	}

	err := Write(path, original)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Author, loaded.Author)
	assert.Equal(t, original.Version, loaded.Version)
	assert.Equal(t, original.Description, loaded.Description)
	require.Len(t, loaded.Foundation, 1)
	assert.Equal(t, "doc", loaded.Foundation[0].ID)
	require.Len(t, loaded.Topics, 1)
	assert.Equal(t, "topic", loaded.Topics[0].ID)
}

func TestLoad_topicDependsOnRequiredBy(t *testing.T) {
	// Write YAML with topic that has depends_on and required_by
	dir := t.TempDir()
	yaml := `name: test
author: tester
version: "1.0.0"
description: Test package
topics:
  - id: conventions
    path: topics/conventions.md
    description: Coding conventions
    depends_on: [core-principles]
    required_by: [react-patterns]
`
	path := filepath.Join(dir, "manifest.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))

	m, err := Load(path)
	require.NoError(t, err)
	require.Len(t, m.Topics, 1)
	assert.Equal(t, []string{"core-principles"}, m.Topics[0].DependsOn)
	assert.Equal(t, []string{"react-patterns"}, m.Topics[0].RequiredBy)
}

func TestLoad_promptDependsOnRequiredBy(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: test
author: tester
version: "1.0.0"
description: Test package
prompts:
  - id: commit
    path: prompts/commit.md
    description: Commit prompt
    depends_on: [conventions]
    required_by: [review]
`
	path := filepath.Join(dir, "manifest.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))

	m, err := Load(path)
	require.NoError(t, err)
	require.Len(t, m.Prompts, 1)
	assert.Equal(t, []string{"conventions"}, m.Prompts[0].DependsOn)
	assert.Equal(t, []string{"review"}, m.Prompts[0].RequiredBy)
}

func TestLoad_planDependsOnRequiredBy(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: test
author: tester
version: "1.0.0"
description: Test package
plans:
  - id: migration
    path: plans/migration/README.md
    state: plans/migration/state.yml
    description: Database migration plan
    depends_on: [schema-design]
    required_by: [deployment]
`
	path := filepath.Join(dir, "manifest.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))

	m, err := Load(path)
	require.NoError(t, err)
	require.Len(t, m.Plans, 1)
	assert.Equal(t, []string{"schema-design"}, m.Plans[0].DependsOn)
	assert.Equal(t, []string{"deployment"}, m.Plans[0].RequiredBy)
}
