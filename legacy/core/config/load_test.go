package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_minimal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	content := `name: test-project
packages: []
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "test-project", cfg.Name)
	assert.Empty(t, cfg.Packages)
}

func TestLoad_withConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	content := `name: test-project
config:
  docs_dir: documentation
  output_dir: dist
packages: []
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "documentation", cfg.DocsDir())
	assert.Equal(t, "dist", cfg.OutputDir())
}

func TestLoad_withPackages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	content := `name: test-project
packages:
  - name: react
    author: facebook
    version: "^1.0.0"
    active: all
  - name: go
    author: google
    version: "~2.0.0"
    active: none
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 2)
	assert.Equal(t, "react", cfg.Packages[0].Name)
	assert.True(t, cfg.Packages[0].Active.IsAll())
	assert.Equal(t, "go", cfg.Packages[1].Name)
	assert.True(t, cfg.Packages[1].Active.IsNone())
}

func TestLoad_withGranularActivation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	content := `name: test-project
packages:
  - name: docs
    author: org
    version: "1.0.0"
    active:
      foundation:
        - philosophy
      topics:
        - react
        - go
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)

	pkg := cfg.Packages[0]
	assert.True(t, pkg.Active.IsGranular())
	require.NotNil(t, pkg.Active.Map)
	assert.Equal(t, []string{"philosophy"}, pkg.Active.Map.Foundation)
	assert.Equal(t, []string{"react", "go"}, pkg.Active.Map.Topics)
}

func TestLoad_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	err := os.WriteFile(path, []byte(`{invalid yaml`), 0o644)
	require.NoError(t, err)

	_, err = Load(path)
	assert.Error(t, err)
}

func TestLoad_schemaValidationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	// Valid YAML but missing required "packages" field.
	content := `name: test-project
`
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	_, err = Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

func TestLoad_nonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/codectx.yml")
	assert.Error(t, err)
}

// --- Write + Load round-trip ---

func TestWriteAndLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	original := &Config{
		Name: "round-trip-project",
		Packages: []PackageDep{
			{
				Name:    "react",
				Author:  "facebook",
				Version: "^1.0.0",
				Active:  Activation{Mode: "all"},
			},
		},
	}

	err := Write(path, original)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, original.Name, loaded.Name)
	require.Len(t, loaded.Packages, 1)
	assert.Equal(t, "react", loaded.Packages[0].Name)
	assert.True(t, loaded.Packages[0].Active.IsAll())
}

func TestLoad_withPackageSource(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: test-project
packages:
  - name: custom
    author: org
    version: "^1.0.0"
    source: https://github.com/custom-org/codectx-custom.git
    active: all
`
	path := filepath.Join(dir, "codectx.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Len(t, cfg.Packages, 1)
	assert.Equal(t, "https://github.com/custom-org/codectx-custom.git", cfg.Packages[0].Source)
}
