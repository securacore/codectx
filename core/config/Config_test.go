package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocsDir_default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, "docs", cfg.DocsDir())
}

func TestDocsDir_nilConfig(t *testing.T) {
	cfg := &Config{Config: nil}
	assert.Equal(t, "docs", cfg.DocsDir())
}

func TestDocsDir_emptyString(t *testing.T) {
	cfg := &Config{Config: &BuildConfig{DocsDir: ""}}
	assert.Equal(t, "docs", cfg.DocsDir())
}

func TestDocsDir_custom(t *testing.T) {
	cfg := &Config{Config: &BuildConfig{DocsDir: "documentation"}}
	assert.Equal(t, "documentation", cfg.DocsDir())
}

func TestOutputDir_default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, ".codectx", cfg.OutputDir())
}

func TestOutputDir_nilConfig(t *testing.T) {
	cfg := &Config{Config: nil}
	assert.Equal(t, ".codectx", cfg.OutputDir())
}

func TestOutputDir_emptyString(t *testing.T) {
	cfg := &Config{Config: &BuildConfig{OutputDir: ""}}
	assert.Equal(t, ".codectx", cfg.OutputDir())
}

func TestOutputDir_custom(t *testing.T) {
	cfg := &Config{Config: &BuildConfig{OutputDir: "dist"}}
	assert.Equal(t, "dist", cfg.OutputDir())
}

// --- IsPackage ---

func TestIsPackage_true(t *testing.T) {
	cfg := &Config{Type: "package"}
	assert.True(t, cfg.IsPackage())
}

func TestIsPackage_project(t *testing.T) {
	cfg := &Config{Type: "project"}
	assert.False(t, cfg.IsPackage())
}

func TestIsPackage_empty(t *testing.T) {
	cfg := &Config{Type: ""}
	assert.False(t, cfg.IsPackage())
}

func TestIsPackage_zeroValue(t *testing.T) {
	cfg := &Config{}
	assert.False(t, cfg.IsPackage())
}
