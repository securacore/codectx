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
	path := filepath.Join(dir, "package.yml")

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
	path := filepath.Join(dir, "package.yml")

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
	path := filepath.Join(dir, "package.yml")

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
	_, err := Load("/nonexistent/package.yml")
	assert.Error(t, err)
}

// --- Write + Load round-trip ---

func TestWriteAndLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.yml")

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
