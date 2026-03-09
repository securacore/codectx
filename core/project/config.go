// Package project provides configuration types and project discovery for codectx.
// It defines the structs that map to codectx.yml, ai.yml, and preferences.yml,
// and provides functions for loading and writing these files.
package project

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigFileName is the expected name of the project configuration file.
const ConfigFileName = "codectx.yml"

// DefaultRoot is the default documentation root directory.
const DefaultRoot = "docs"

// Config represents the project-level codectx.yml file.
// This is the source of truth for package identity, dependencies,
// session context, and registry configuration.
type Config struct {
	// Root is the documentation root directory relative to the project root.
	// Defaults to "docs". All documentation paths are relative to this root.
	Root string `yaml:"root"`

	// Name is the package/project name.
	Name string `yaml:"name"`

	// Org is the organization or author namespace.
	Org string `yaml:"org"`

	// Version is the package version in semver format.
	Version string `yaml:"version"`

	// Description is a one-line package description.
	Description string `yaml:"description"`

	// Session defines always-loaded context for AI sessions.
	Session *SessionConfig `yaml:"session,omitempty"`

	// Dependencies lists documentation package dependencies.
	Dependencies map[string]*DependencyConfig `yaml:"dependencies,omitempty"`

	// Registry is where to resolve packages. Defaults to "github.com".
	Registry string `yaml:"registry,omitempty"`
}

// SessionConfig defines always-loaded session context.
type SessionConfig struct {
	// AlwaysLoaded lists paths and package references that are compiled
	// into context.md and loaded at the start of every AI session.
	// Order matters — documents appear in this order in the compiled context.
	AlwaysLoaded []string `yaml:"always_loaded,omitempty"`

	// Budget is the maximum token budget for always-loaded context.
	// The compiler warns if assembled content exceeds this.
	Budget int `yaml:"budget,omitempty"`
}

// DependencyConfig represents a single package dependency entry.
type DependencyConfig struct {
	// Active controls whether this package is included in compiled output.
	// Inactive packages remain installed but are excluded from compilation.
	Active bool `yaml:"active"`
}

// DefaultConfig returns a Config with sensible defaults for a new project.
func DefaultConfig(name string, root string) Config {
	if root == "" {
		root = DefaultRoot
	}
	return Config{
		Root:        root,
		Name:        name,
		Org:         "",
		Version:     "0.1.0",
		Description: "",
		Session: &SessionConfig{
			AlwaysLoaded: []string{},
			Budget:       30000,
		},
		Dependencies: map[string]*DependencyConfig{},
		Registry:     "github.com",
	}
}

// WriteToFile marshals the config to YAML and writes it to the given path.
func (c *Config) WriteToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	header := []byte("# codectx project configuration\n# See: https://github.com/securacore/codectx\n\n")
	return os.WriteFile(path, append(header, data...), 0644)
}

// LoadConfig reads and parses a codectx.yml file from the given path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return &cfg, nil
}

// AIConfig represents the .codectx/ai.yml file.
// AI model and behavior configuration. Checked into version control (no secrets).
type AIConfig struct {
	// Compilation configures the model used during compilation for alias
	// generation and bridge summaries.
	Compilation AICompilationConfig `yaml:"compilation"`

	// Consumption configures the target model for consumption, affecting
	// context budgets and formatting.
	Consumption AIConsumptionConfig `yaml:"consumption"`

	// OutputFormat is the context formatting preference for generated output.
	// Valid values: "markdown", "xml_tags", "plain".
	OutputFormat string `yaml:"output_format"`
}

// AICompilationConfig configures the model used during compilation.
type AICompilationConfig struct {
	// Model is the AI model used for alias generation and bridge summaries.
	Model string `yaml:"model"`

	// Encoding is the tokenizer encoding for token counting.
	Encoding string `yaml:"encoding"`
}

// AIConsumptionConfig configures the target model for consumption.
type AIConsumptionConfig struct {
	// Model is the target AI model for consumption.
	Model string `yaml:"model"`

	// ContextWindow is the target model's context window size in tokens.
	ContextWindow int `yaml:"context_window"`

	// ResultsCount is the default number of results returned by codectx query.
	ResultsCount int `yaml:"results_count"`
}

// DefaultAIConfig returns an AIConfig with sensible defaults.
func DefaultAIConfig() AIConfig {
	return AIConfig{
		Compilation: AICompilationConfig{
			Model:    "claude-sonnet-4-20250514",
			Encoding: "cl100k_base",
		},
		Consumption: AIConsumptionConfig{
			Model:         "claude-sonnet-4-20250514",
			ContextWindow: 200000,
			ResultsCount:  10,
		},
		OutputFormat: "markdown",
	}
}

// WriteToFile marshals the AI config to YAML and writes it to the given path.
func (c *AIConfig) WriteToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling ai config: %w", err)
	}

	header := []byte("# codectx AI configuration\n# Model and behavior settings for compilation and consumption.\n# Checked into version control (no secrets).\n\n")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory for ai config: %w", err)
	}
	return os.WriteFile(path, append(header, data...), 0644)
}

// PreferencesConfig represents the .codectx/preferences.yml file.
// Compiler and pipeline configuration. Checked into version control.
type PreferencesConfig struct {
	// Chunking configures chunk compilation settings.
	Chunking ChunkingConfig `yaml:"chunking"`

	// BM25 configures BM25 index parameters.
	BM25 BM25Config `yaml:"bm25"`

	// Taxonomy configures taxonomy extraction settings.
	Taxonomy TaxonomyConfig `yaml:"taxonomy"`

	// Validation configures documentation linting and validation.
	Validation ValidationConfig `yaml:"validation"`
}

// ChunkingConfig controls how documents are split into chunks.
type ChunkingConfig struct {
	// TargetTokens is the target chunk size in tokens.
	TargetTokens int `yaml:"target_tokens"`

	// MinTokens is the minimum chunk size to avoid tiny fragments.
	MinTokens int `yaml:"min_tokens"`

	// MaxTokens is the maximum chunk size (hard ceiling).
	MaxTokens int `yaml:"max_tokens"`

	// FlexibilityWindow is the fraction of target at which to break
	// if the next block would exceed target. E.g., 0.8 means break
	// after 80% of target.
	FlexibilityWindow float64 `yaml:"flexibility_window"`
}

// BM25Config controls BM25 index parameters.
type BM25Config struct {
	// K1 is the term frequency saturation parameter.
	K1 float64 `yaml:"k1"`

	// B is the document length normalization parameter.
	B float64 `yaml:"b"`
}

// TaxonomyConfig controls taxonomy extraction settings.
type TaxonomyConfig struct {
	// MinTermFrequency is the minimum corpus-wide frequency to include a term.
	MinTermFrequency int `yaml:"min_term_frequency"`

	// MaxAliasCount is the maximum aliases per canonical term.
	MaxAliasCount int `yaml:"max_alias_count"`

	// POSExtraction enables POS-based term extraction.
	POSExtraction bool `yaml:"pos_extraction"`

	// LLMAliasGeneration enables LLM pass for alias generation.
	LLMAliasGeneration bool `yaml:"llm_alias_generation"`
}

// ValidationConfig controls documentation linting and validation.
type ValidationConfig struct {
	// RequireReadme requires every topic directory to have a README.md.
	RequireReadme bool `yaml:"require_readme"`

	// RequireSpec requires spec files for every documentation file.
	RequireSpec bool `yaml:"require_spec"`

	// MaxFileTokens warns if a single source file exceeds this token count.
	MaxFileTokens int `yaml:"max_file_tokens"`

	// RequireHeadings warns if a file has no heading structure.
	RequireHeadings bool `yaml:"require_headings"`
}

// DefaultPreferencesConfig returns a PreferencesConfig with sensible defaults.
func DefaultPreferencesConfig() PreferencesConfig {
	return PreferencesConfig{
		Chunking: ChunkingConfig{
			TargetTokens:      450,
			MinTokens:         200,
			MaxTokens:         800,
			FlexibilityWindow: 0.8,
		},
		BM25: BM25Config{
			K1: 1.2,
			B:  0.75,
		},
		Taxonomy: TaxonomyConfig{
			MinTermFrequency:   2,
			MaxAliasCount:      10,
			POSExtraction:      true,
			LLMAliasGeneration: true,
		},
		Validation: ValidationConfig{
			RequireReadme:   true,
			RequireSpec:     false,
			MaxFileTokens:   10000,
			RequireHeadings: true,
		},
	}
}

// WriteToFile marshals the preferences config to YAML and writes it to the given path.
func (c *PreferencesConfig) WriteToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling preferences config: %w", err)
	}

	header := []byte("# codectx compiler preferences\n# Compiler and pipeline configuration.\n# Checked into version control.\n\n")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory for preferences config: %w", err)
	}
	return os.WriteFile(path, append(header, data...), 0644)
}
