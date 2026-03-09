package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWrite_validPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	cfg := &Config{
		Name: "test-project",
		Config: &BuildConfig{
			DocsDir:   "documentation",
			OutputDir: "build",
		},
		Packages: []PackageDep{
			{
				Name:    "react-docs",
				Author:  "acme",
				Version: "1.0.0",
			},
		},
	}

	err := Write(path, cfg)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var loaded Config
	err = yaml.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, "test-project", loaded.Name)
	require.NotNil(t, loaded.Config)
	assert.Equal(t, "documentation", loaded.Config.DocsDir)
	assert.Equal(t, "build", loaded.Config.OutputDir)
	require.Len(t, loaded.Packages, 1)
	assert.Equal(t, "react-docs", loaded.Packages[0].Name)
	assert.Equal(t, "acme", loaded.Packages[0].Author)
	assert.Equal(t, "1.0.0", loaded.Packages[0].Version)
}

func TestWrite_invalidPath(t *testing.T) {
	cfg := &Config{
		Name:     "test",
		Packages: []PackageDep{},
	}

	err := Write("/nonexistent/deep/path/codectx.yml", cfg)
	assert.Error(t, err)
}

func TestWrite_omitemptyBehavior(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.yml")

	cfg := &Config{
		Name:     "minimal",
		Config:   nil,
		Packages: []PackageDep{},
	}

	err := Write(path, cfg)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// With Config as nil and omitempty tag, the "config:" key should not appear.
	assert.NotContains(t, string(data), "config:")
}
