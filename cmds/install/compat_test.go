package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCompatibility_emptyDir(t *testing.T) {
	dir := t.TempDir()
	issues := checkCompatibility(dir)
	assert.Empty(t, issues)
}

func TestCheckCompatibility_validPackageYml(t *testing.T) {
	dir := t.TempDir()
	content := `name: test
author: org
version: "1.0.0"
description: "Test package"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.yml"), []byte(content), 0o644))

	issues := checkCompatibility(dir)
	assert.Empty(t, issues)
}

func TestCheckCompatibility_invalidPackageYml(t *testing.T) {
	dir := t.TempDir()
	// Missing required fields.
	content := `name: test
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.yml"), []byte(content), 0o644))

	issues := checkCompatibility(dir)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0].reason, "schema validation failed")
}

func TestCheckCompatibility_invalidPackageYmlSchema(t *testing.T) {
	dir := t.TempDir()
	// Valid YAML but missing required fields — triggers schema validation failure.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.yml"), []byte("foo: bar\n"), 0o644))

	issues := checkCompatibility(dir)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0].reason, "schema validation failed")
}

func TestCheckCompatibility_validPackagesDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "packages", "react@org"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "packages", "go@org"), 0o755))

	issues := checkCompatibility(dir)
	assert.Empty(t, issues)
}

func TestCheckCompatibility_invalidPackagesDir(t *testing.T) {
	dir := t.TempDir()
	// Not following name@author convention.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "packages", "just-a-name"), 0o755))

	issues := checkCompatibility(dir)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0].reason, "name@author")
}

func TestCheckCompatibility_packagesFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "packages"), 0o755))
	// Files (not directories) are ignored.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "packages", "README.md"), []byte("hi"), 0o644))

	issues := checkCompatibility(dir)
	assert.Empty(t, issues)
}

func TestCheckCompatibility_validSchemas(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "schemas"), 0o755))
	schemaContent := `{"$schema": "https://json-schema.org/draft/2020-12/schema"}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schemas", "codectx.schema.json"), []byte(schemaContent), 0o644))

	issues := checkCompatibility(dir)
	assert.Empty(t, issues)
}

func TestCheckCompatibility_invalidSchemas(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "schemas"), 0o755))
	// Missing $schema field.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schemas", "codectx.schema.json"), []byte(`{"title": "not a schema"}`), 0o644))

	issues := checkCompatibility(dir)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0].reason, "$schema")
}

func TestCheckCompatibility_nonCodectxSchemasIgnored(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "schemas"), 0o755))
	// A non-codectx schema file is ignored.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schemas", "custom.schema.json"), []byte(`not json`), 0o644))

	issues := checkCompatibility(dir)
	assert.Empty(t, issues)
}

func TestCheckCompatibility_schemaMissingField(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "schemas"), 0o755))
	// Valid JSON but missing $schema field.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schemas", "manifest.schema.json"), []byte(`{"title": "nope"}`), 0o644))

	issues := checkCompatibility(dir)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0].reason, "$schema")
}

func TestNameAtAuthor_regex(t *testing.T) {
	assert.True(t, nameAtAuthor.MatchString("react@org"))
	assert.True(t, nameAtAuthor.MatchString("my-pkg@my-author"))
	assert.True(t, nameAtAuthor.MatchString("pkg_name@author_name"))
	assert.False(t, nameAtAuthor.MatchString("nope"))
	assert.False(t, nameAtAuthor.MatchString("@author"))
	assert.False(t, nameAtAuthor.MatchString("name@"))
	assert.False(t, nameAtAuthor.MatchString(""))
}

// --- Malformed YAML in manifest.yml ---

func TestCheckCompatibility_malformedYaml(t *testing.T) {
	dir := t.TempDir()
	// Binary-like content that fails YAML parsing.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.yml"), []byte("{{{{not valid"), 0o644))

	issues := checkCompatibility(dir)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0].reason, "invalid YAML")
}

// --- Schema subdirectory is ignored ---

func TestCheckCompatibility_schemaSubdirIgnored(t *testing.T) {
	dir := t.TempDir()
	schemasDir := filepath.Join(dir, "schemas")
	require.NoError(t, os.MkdirAll(schemasDir, 0o755))

	// Create a subdirectory named like a codectx schema — should be skipped.
	require.NoError(t, os.MkdirAll(filepath.Join(schemasDir, "codectx.schema.json"), 0o755))

	issues := checkCompatibility(dir)
	assert.Empty(t, issues)
}

// --- Invalid JSON/YAML in schema file ---

func TestCheckCompatibility_invalidJsonInSchemaFile(t *testing.T) {
	dir := t.TempDir()
	schemasDir := filepath.Join(dir, "schemas")
	require.NoError(t, os.MkdirAll(schemasDir, 0o755))

	// Write a schema file with content that fails YAML unmarshal into map.
	require.NoError(t, os.WriteFile(
		filepath.Join(schemasDir, "codectx.schema.json"),
		[]byte("{{invalid json yaml"),
		0o644,
	))

	issues := checkCompatibility(dir)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0].reason, "not valid JSON/YAML")
}

// --- Multiple issues at once ---

func TestCheckCompatibility_multipleIssues(t *testing.T) {
	dir := t.TempDir()

	// Bad manifest.yml.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.yml"), []byte("{{bad yaml"), 0o644))

	// Bad packages/ dir.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "packages", "no-at-sign"), 0o755))

	issues := checkCompatibility(dir)
	assert.GreaterOrEqual(t, len(issues), 2)
}
