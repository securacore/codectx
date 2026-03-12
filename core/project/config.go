// Package project provides configuration types and project discovery for codectx.
// It defines the structs that map to codectx.yml, ai.yml, and preferences.yml,
// and provides functions for loading and writing these files.
package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Version is the codectx version string, set at build time via ldflags.
// Defaults to "dev" for local development builds.
var Version = "dev"

// ConfigFileName is the expected name of the project configuration file.
const ConfigFileName = "codectx.yml"

// DefaultRoot is the default documentation root directory.
const DefaultRoot = "docs"

// DefaultSessionBudget is the default token budget for always-loaded context.
const DefaultSessionBudget = 30000

// DefaultRegistry is the default package registry.
const DefaultRegistry = "github.com"

// DefaultContextWindow is the default AI model context window size in tokens.
const DefaultContextWindow = 200000

// DefaultResultsCount is the default number of results returned by codectx query.
const DefaultResultsCount = 10

// DefaultModel is the fallback AI model when nothing is detected.
const DefaultModel = "claude-sonnet-4-20250514"

// DefaultEncoding is the fallback tokenizer encoding.
const DefaultEncoding = "cl100k_base"

// ProviderCLI indicates the local Claude CLI binary for LLM compilation tasks.
const ProviderCLI = "cli"

// ProviderAPI indicates the Anthropic Messages API for LLM compilation tasks.
const ProviderAPI = "api"

// DirPerm is the standard directory permission mode used throughout the project.
const DirPerm = 0755

// FilePerm is the standard file permission mode used throughout the project.
const FilePerm = 0644

// CodectxDir is the hidden directory name under the documentation root
// that holds all tooling state (.codectx/).
const CodectxDir = ".codectx"

// CompiledDir is the directory name under .codectx/ for compiled output.
const CompiledDir = "compiled"

// BM25Dir is the directory name under compiled/ for BM25 index files.
const BM25Dir = "bm25"

// SystemDir is the directory name for system/compiler documentation
// under the documentation root.
const SystemDir = "system"

// PackagesDir is the directory name under .codectx/ for installed packages.
const PackagesDir = "packages"

// AIConfigFile is the filename for the AI configuration file in .codectx/.
const AIConfigFile = "ai.yml"

// PreferencesFile is the filename for the preferences configuration file in .codectx/.
const PreferencesFile = "preferences.yml"

// ResolveRoot returns root if non-empty, otherwise DefaultRoot.
// This is the single place for the "if root == "" { root = DefaultRoot }" pattern.
func ResolveRoot(root string) string {
	if root == "" {
		return DefaultRoot
	}
	return root
}

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

// EffectiveRegistry returns the registry URL, falling back to
// DefaultRegistry ("github.com") if the config value is empty.
func (c *Config) EffectiveRegistry() string {
	if c.Registry != "" {
		return c.Registry
	}
	return DefaultRegistry
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

// EffectiveBudget returns the session token budget, falling back to
// DefaultSessionBudget if the budget is unset or non-positive.
// Safe to call on a nil receiver.
func (s *SessionConfig) EffectiveBudget() int {
	if s == nil || s.Budget <= 0 {
		return DefaultSessionBudget
	}
	return s.Budget
}

// ContextRelPath returns the path to context.md relative to the project root
// for the given root configuration (e.g., "docs/.codectx/compiled/context.md").
func ContextRelPath(root string) string {
	return filepath.ToSlash(filepath.Join(
		ResolveRoot(root),
		CodectxDir,
		CompiledDir,
		"context.md",
	))
}

// PackagesPath returns the absolute path to the packages directory under
// the documentation root (e.g., /path/to/docs/.codectx/packages).
func PackagesPath(rootDir string) string {
	return filepath.Join(rootDir, CodectxDir, PackagesDir)
}

// DependencyConfig represents a single package dependency entry.
type DependencyConfig struct {
	// Active controls whether this package is included in compiled output.
	// Inactive packages remain installed but are excluded from compilation.
	Active bool `yaml:"active"`
}

// DefaultConfig returns a Config with sensible defaults for a new project.
func DefaultConfig(name string, root string) Config {
	root = ResolveRoot(root)
	return Config{
		Root:        root,
		Name:        name,
		Org:         "",
		Version:     "0.1.0",
		Description: "",
		Session: &SessionConfig{
			AlwaysLoaded: []string{
				"system/foundation/cli-usage",
				"system/foundation/history",
			},
			Budget: DefaultSessionBudget,
		},
		Dependencies: map[string]*DependencyConfig{},
		Registry:     DefaultRegistry,
	}
}

// WriteToFile marshals the config to YAML and writes it to the given path.
func (c *Config) WriteToFile(path string) error {
	return WriteYAMLFile(path, "# codectx project configuration\n# See: https://github.com/securacore/codectx\n\n", c)
}

// LoadConfig reads and parses a codectx.yml file from the given path.
func LoadConfig(path string) (*Config, error) {
	return loadYAMLFile[Config](path)
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
	// Provider is the LLM provider for compilation tasks ("cli" or "api").
	// "cli" invokes the local Claude CLI binary; "api" uses the Anthropic
	// Messages API directly. Empty string means auto-detect at compile time.
	Provider string `yaml:"provider,omitempty"`

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
// The default model and encoding are sourced from the detect package
// to ensure consistency across detection, configuration, and scaffolding.
func DefaultAIConfig() AIConfig {
	return AIConfig{
		Compilation: AICompilationConfig{
			Model:    DefaultModel,
			Encoding: DefaultEncoding,
		},
		Consumption: AIConsumptionConfig{
			Model:         DefaultModel,
			ContextWindow: DefaultContextWindow,
			ResultsCount:  DefaultResultsCount,
		},
		OutputFormat: "markdown",
	}
}

// WriteToFile marshals the AI config to YAML and writes it to the given path.
func (c *AIConfig) WriteToFile(path string) error {
	return WriteYAMLFile(path, "# codectx AI configuration\n# Model and behavior settings for compilation and consumption.\n# Checked into version control (no secrets).\n\n", c)
}

// LoadAIConfig reads and parses an ai.yml file from the given path.
func LoadAIConfig(path string) (*AIConfig, error) {
	return loadYAMLFile[AIConfig](path)
}

// LoadAIConfigForProject loads the AI configuration for a project, given the
// project directory and config. This is a convenience wrapper that constructs
// the full path to ai.yml from RootDir / .codectx / ai.yml.
func LoadAIConfigForProject(projectDir string, cfg *Config) (*AIConfig, error) {
	rootDir := RootDir(projectDir, cfg)
	codectxDir := filepath.Join(rootDir, CodectxDir)
	return LoadAIConfig(filepath.Join(codectxDir, AIConfigFile))
}

// ResolveEncoding returns the tokenizer encoding for a project.
// It attempts to load ai.yml and use the configured compilation encoding.
// Falls back to DefaultEncoding if ai.yml is missing or encoding is unset.
func ResolveEncoding(projectDir string, cfg *Config) string {
	aiCfg, err := LoadAIConfigForProject(projectDir, cfg)
	if err == nil && aiCfg.Compilation.Encoding != "" {
		return aiCfg.Compilation.Encoding
	}
	return DefaultEncoding
}

// PreferencesConfig represents the .codectx/preferences.yml file.
// Compiler and pipeline configuration. Checked into version control.
type PreferencesConfig struct {
	// AutoCompile controls whether commands that change project state
	// (init, update) automatically run the compilation pipeline.
	// Defaults to true when absent. Uses a pointer to distinguish
	// "not set" (nil → default true) from "explicitly set to false".
	AutoCompile *bool `yaml:"auto_compile,omitempty"`

	// ScaffoldMaintenance controls whether the compile command automatically
	// regenerates missing directories, restores system default files, and
	// manages .gitkeep files in content directories.
	// Defaults to true when absent. Uses a pointer to distinguish
	// "not set" (nil → default true) from "explicitly set to false".
	ScaffoldMaintenance *bool `yaml:"scaffold_maintenance,omitempty"`

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

	// HashLength is the number of hex characters to use from the SHA-256
	// content hash when generating chunk IDs. Clamped to [8, 64].
	HashLength int `yaml:"hash_length"`
}

const (
	// MinHashLength is the minimum allowed hash length for chunk IDs.
	MinHashLength = 8

	// MaxHashLength is the maximum allowed hash length for chunk IDs (full SHA-256).
	MaxHashLength = 64

	// DefaultHashLength is the default hash length for chunk IDs.
	DefaultHashLength = 16
)

// ClampHashLength ensures a hash length value falls within [MinHashLength, MaxHashLength].
// Values of zero are treated as the default.
func ClampHashLength(n int) int {
	if n <= 0 {
		return DefaultHashLength
	}
	if n < MinHashLength {
		return MinHashLength
	}
	if n > MaxHashLength {
		return MaxHashLength
	}
	return n
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

// EffectiveAutoCompile returns whether auto-compilation is enabled.
// Returns true when AutoCompile is nil (field absent from config file),
// preserving backwards compatibility for existing preferences.yml files.
func (c *PreferencesConfig) EffectiveAutoCompile() bool {
	if c.AutoCompile == nil {
		return true
	}
	return *c.AutoCompile
}

// AutoCompileIsDefault reports whether the auto_compile setting is
// unset (nil), meaning the effective value comes from the default.
// Commands use this to print an informational message when the
// default behavior is applied implicitly.
func (c *PreferencesConfig) AutoCompileIsDefault() bool {
	return c.AutoCompile == nil
}

// EffectiveScaffoldMaintenance returns whether scaffold maintenance is enabled.
// Returns true when ScaffoldMaintenance is nil (field absent from config file),
// preserving backwards compatibility for existing preferences.yml files.
func (c *PreferencesConfig) EffectiveScaffoldMaintenance() bool {
	if c.ScaffoldMaintenance == nil {
		return true
	}
	return *c.ScaffoldMaintenance
}

// BoolPtr returns a pointer to the given bool value.
// Used when setting *bool fields like AutoCompile in config structs.
func BoolPtr(v bool) *bool {
	return &v
}

// DefaultPreferencesConfig returns a PreferencesConfig with sensible defaults.
func DefaultPreferencesConfig() PreferencesConfig {
	return PreferencesConfig{
		AutoCompile:         BoolPtr(true),
		ScaffoldMaintenance: BoolPtr(true),
		Chunking: ChunkingConfig{
			TargetTokens:      450,
			MinTokens:         200,
			MaxTokens:         800,
			FlexibilityWindow: 0.8,
			HashLength:        DefaultHashLength,
		},
		BM25: BM25Config{
			K1: 1.2,
			B:  0.75,
		},
		Taxonomy: TaxonomyConfig{
			MinTermFrequency:   2,
			MaxAliasCount:      10,
			POSExtraction:      true,
			LLMAliasGeneration: false,
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
	return WriteYAMLFile(path, "# codectx compiler preferences\n# Compiler and pipeline configuration.\n# Checked into version control.\n\n", c)
}

// LoadPreferencesConfig reads and parses a preferences.yml file from the given path.
func LoadPreferencesConfig(path string) (*PreferencesConfig, error) {
	return loadYAMLFile[PreferencesConfig](path)
}

// LoadPreferencesConfigForProject loads the preferences configuration for a
// project, given the project directory and config. This is a convenience
// wrapper that constructs the full path to preferences.yml from
// RootDir / .codectx / preferences.yml.
func LoadPreferencesConfigForProject(projectDir string, cfg *Config) (*PreferencesConfig, error) {
	rootDir := RootDir(projectDir, cfg)
	codectxDir := filepath.Join(rootDir, CodectxDir)
	return LoadPreferencesConfig(filepath.Join(codectxDir, PreferencesFile))
}

// loadYAMLFile reads a YAML file from disk and unmarshals it into a new
// instance of type T. Used by all LoadXConfig functions.
func loadYAMLFile[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg T
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &cfg, nil
}

// WriteYAMLFile marshals a value to YAML with 2-space indentation, prepends a
// header comment, ensures the parent directory exists, and writes the file.
// Used by all config WriteToFile methods and the manifest package.
func WriteYAMLFile(path string, header string, v any) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := enc.Close(); err != nil {
		return fmt.Errorf("closing encoder: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), DirPerm); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	return os.WriteFile(path, append([]byte(header), buf.Bytes()...), FilePerm)
}
