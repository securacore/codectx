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

// TypeProject is the default project type — a full documentation environment.
const TypeProject = "project"

// TypePackage indicates a documentation package authoring project.
// Package projects have a package/ directory containing publishable content
// alongside the docs/ authoring workspace.
const TypePackage = "package"

// PackageContentDir is the directory name under the project root where
// publishable package content lives (foundation/, topics/, plans/, prompts/).
const PackageContentDir = "package"

// DefaultSessionBudget is the default token budget for always-loaded context.
const DefaultSessionBudget = 30000

// DefaultRegistry is the default package registry.
const DefaultRegistry = "github.com"

// DefaultContextWindow is the default AI model context window size in tokens.
const DefaultContextWindow = 200000

// DefaultResultsCount is the default number of results returned by codectx query.
const DefaultResultsCount = 10

// DefaultEncoding is the fallback tokenizer encoding.
const DefaultEncoding = "cl100k_base"

// DirPerm is the standard directory permission mode used throughout the project.
const DirPerm = 0755

// FilePerm is the standard file permission mode used throughout the project.
const FilePerm = 0644

// CodectxDir is the hidden directory name under the documentation root
// that holds all tooling state (.codectx/).
const CodectxDir = ".codectx"

// CompiledDir is the directory name under .codectx/ for compiled output.
const CompiledDir = "compiled"

// BM25Dir is the directory name under compiled/ for flat BM25 index files.
const BM25Dir = "bm25"

// BM25FDir is the directory name under compiled/ for field-weighted BM25F index files.
const BM25FDir = "bm25f"

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
	// Type distinguishes project types: "project" (default) is a full
	// documentation environment; "package" is a documentation package
	// authoring project with a package/ directory for publishable content.
	Type string `yaml:"type,omitempty"`

	// Root is the documentation root directory relative to the project root.
	// Defaults to "docs". All documentation paths are relative to this root.
	Root string `yaml:"root"`

	// Name is the package/project name.
	Name string `yaml:"name"`

	// Author is the GitHub username or organization that owns this package.
	Author string `yaml:"author"`

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

// IsPackage reports whether this project is a documentation package
// authoring project (type: "package").
func (c *Config) IsPackage() bool {
	return c.Type == TypePackage
}

// PackageConfigPath returns the absolute path to the package content
// codectx.yml file (e.g., /path/to/project/package/codectx.yml).
// Only meaningful when IsPackage() is true.
func PackageConfigPath(projectDir string) string {
	return filepath.Join(projectDir, PackageContentDir, ConfigFileName)
}

// PackageContentPath returns the absolute path to the package content
// directory (e.g., /path/to/project/package/).
func PackageContentPath(projectDir string) string {
	return filepath.Join(projectDir, PackageContentDir)
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
// The projectType parameter controls the type field: use TypeProject for a
// standard documentation project, or TypePackage for a package authoring project.
// An empty string defaults to TypeProject (omitted from YAML output).
func DefaultConfig(name string, root string, projectType string) Config {
	root = ResolveRoot(root)

	cfg := Config{
		Root:        root,
		Name:        name,
		Author:      "",
		Version:     "0.1.0",
		Description: "",
		Session: &SessionConfig{
			AlwaysLoaded: []string{
				"system/foundation/documentation-protocol",
				"system/foundation/history",
			},
			Budget: DefaultSessionBudget,
		},
		Dependencies: map[string]*DependencyConfig{},
		Registry:     DefaultRegistry,
	}

	if projectType == TypePackage {
		cfg.Type = TypePackage
	}

	return cfg
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

// AICompilationConfig configures compilation settings.
type AICompilationConfig struct {
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
			Encoding: DefaultEncoding,
		},
		Consumption: AIConsumptionConfig{
			Model:         "claude-sonnet-4-20250514",
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

	// Search configures package search and discovery settings.
	Search SearchConfig `yaml:"search"`

	// Chunking configures chunk compilation settings.
	Chunking ChunkingConfig `yaml:"chunking"`

	// Indexer selects which BM25 implementation is used at query time.
	// Both indexes are always built during compilation for instant switching.
	// Valid values: "bm25" (flat), "bm25f" (field-weighted). Defaults to "bm25f".
	Indexer IndexerType `yaml:"indexer,omitempty"`

	// BM25 configures flat BM25 index parameters.
	BM25 BM25Config `yaml:"bm25"`

	// BM25F configures field-weighted BM25F index parameters.
	BM25F BM25FConfig `yaml:"bm25f"`

	// Query configures the query pipeline (expansion, RRF, graph re-ranking).
	Query QueryConfig `yaml:"query"`

	// Taxonomy configures taxonomy extraction settings.
	Taxonomy TaxonomyConfig `yaml:"taxonomy"`

	// Validation configures documentation linting and validation.
	Validation ValidationConfig `yaml:"validation"`
}

// SearchConfig controls package search and discovery behavior.
type SearchConfig struct {
	// ShowUninstallable controls whether packages without a release archive
	// are included in search and add results.
	// Defaults to false when absent (nil → hide uninstallable packages).
	// Uses a pointer to distinguish "not set" from "explicitly set to true".
	ShowUninstallable *bool `yaml:"show_uninstallable,omitempty"`
}

// EffectiveShowUninstallable returns whether uninstallable packages should
// be shown in search results. Returns false when ShowUninstallable is nil
// (field absent from config file), hiding packages without release archives.
func (c *SearchConfig) EffectiveShowUninstallable() bool {
	if c.ShowUninstallable == nil {
		return false
	}
	return *c.ShowUninstallable
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

// IndexerType selects which BM25 scoring implementation is used at query time.
type IndexerType string

const (
	// IndexerBM25 selects the flat bag-of-words BM25 scorer.
	IndexerBM25 IndexerType = "bm25"

	// IndexerBM25F selects the field-weighted BM25F scorer with per-field
	// weights and length normalization.
	IndexerBM25F IndexerType = "bm25f"
)

// BM25Config controls flat BM25 index parameters.
type BM25Config struct {
	// K1 is the term frequency saturation parameter.
	K1 float64 `yaml:"k1"`

	// B is the document length normalization parameter.
	B float64 `yaml:"b"`
}

// BM25FConfig controls BM25F field-weighted index parameters.
type BM25FConfig struct {
	// K1 is the term frequency saturation parameter.
	K1 float64 `yaml:"k1"`

	// Fields defines per-field scoring weights and length normalization.
	Fields map[string]BM25FFieldConfig `yaml:"fields"`
}

// BM25FFieldConfig controls scoring for a single BM25F field.
type BM25FFieldConfig struct {
	// Weight is the field importance multiplier.
	Weight float64 `yaml:"weight"`

	// B is the per-field document length normalization parameter.
	B float64 `yaml:"b"`
}

// QueryConfig controls the query pipeline configuration.
type QueryConfig struct {
	// Expansion controls taxonomy-based query expansion.
	Expansion ExpansionConfig `yaml:"expansion"`

	// RRF controls Reciprocal Rank Fusion parameters.
	RRF RRFConfig `yaml:"rrf"`

	// GraphRerank controls graph-based re-ranking after RRF.
	GraphRerank GraphRerankConfig `yaml:"graph_rerank"`
}

// ExpansionConfig controls taxonomy query expansion behavior.
type ExpansionConfig struct {
	// Enabled controls whether query expansion is active.
	Enabled *bool `yaml:"enabled,omitempty"`

	// AliasWeight is the weight multiplier for taxonomy alias matches.
	AliasWeight float64 `yaml:"alias_weight"`

	// NarrowerWeight is the weight multiplier for narrower term matches.
	NarrowerWeight float64 `yaml:"narrower_weight"`

	// RelatedWeight is the weight multiplier for related term matches.
	RelatedWeight float64 `yaml:"related_weight"`

	// MaxExpansionTerms caps the total expanded terms to prevent query explosion.
	MaxExpansionTerms int `yaml:"max_expansion_terms"`
}

// EffectiveEnabled returns whether query expansion is enabled.
// Defaults to true when Enabled is nil (field absent from config).
func (c *ExpansionConfig) EffectiveEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// RRFConfig controls Reciprocal Rank Fusion parameters.
type RRFConfig struct {
	// K is the smoothing constant for RRF scoring. Standard default is 60.
	K float64 `yaml:"k"`

	// IndexWeights assigns importance multipliers per index type.
	IndexWeights map[string]float64 `yaml:"index_weights"`
}

// GraphRerankConfig controls graph-based re-ranking after RRF fusion.
type GraphRerankConfig struct {
	// Enabled controls whether graph re-ranking is active.
	Enabled *bool `yaml:"enabled,omitempty"`

	// AdjacentBoost is the score multiplier for chunks with adjacent
	// chunks that also scored in the top window.
	AdjacentBoost float64 `yaml:"adjacent_boost"`

	// SpecBoost is the score multiplier for chunks whose paired
	// spec/object counterpart also scored.
	SpecBoost float64 `yaml:"spec_boost"`

	// CrossRefBoost is the score multiplier for chunks from documents
	// that cross-reference other scored documents.
	CrossRefBoost float64 `yaml:"cross_ref_boost"`
}

// EffectiveEnabled returns whether graph re-ranking is enabled.
// Defaults to true when Enabled is nil (field absent from config).
func (c *GraphRerankConfig) EffectiveEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// TaxonomyConfig controls taxonomy extraction settings.
type TaxonomyConfig struct {
	// MinTermFrequency is the minimum corpus-wide frequency to include a term.
	MinTermFrequency int `yaml:"min_term_frequency"`

	// MaxAliasCount is the maximum aliases per canonical term.
	MaxAliasCount int `yaml:"max_alias_count"`

	// POSExtraction enables POS-based term extraction.
	POSExtraction bool `yaml:"pos_extraction"`
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

// EffectiveIndexer returns the active indexer type.
// Defaults to IndexerBM25F when empty or unrecognized.
func (c *PreferencesConfig) EffectiveIndexer() IndexerType {
	switch c.Indexer {
	case IndexerBM25:
		return IndexerBM25
	case IndexerBM25F:
		return IndexerBM25F
	default:
		return IndexerBM25F
	}
}

// BoolPtr returns a pointer to the given bool value.
// Used when setting *bool fields like AutoCompile in config structs.
func BoolPtr(v bool) *bool {
	return &v
}

// DefaultBM25FConfig returns sensible defaults for BM25F field-weighted scoring.
func DefaultBM25FConfig() BM25FConfig {
	return BM25FConfig{
		K1: 1.2,
		Fields: map[string]BM25FFieldConfig{
			"heading": {Weight: 3.0, B: 0.3},
			"terms":   {Weight: 2.0, B: 0.0},
			"body":    {Weight: 1.0, B: 0.75},
			"code":    {Weight: 0.6, B: 0.5},
		},
	}
}

// DefaultQueryConfig returns sensible defaults for the query pipeline.
func DefaultQueryConfig() QueryConfig {
	return QueryConfig{
		Expansion: ExpansionConfig{
			Enabled:           BoolPtr(true),
			AliasWeight:       1.0,
			NarrowerWeight:    0.7,
			RelatedWeight:     0.4,
			MaxExpansionTerms: 20,
		},
		RRF: RRFConfig{
			K: 60,
			IndexWeights: map[string]float64{
				"objects": 1.0,
				"specs":   0.7,
				"system":  0.3,
			},
		},
		GraphRerank: GraphRerankConfig{
			Enabled:       BoolPtr(true),
			AdjacentBoost: 0.15,
			SpecBoost:     0.20,
			CrossRefBoost: 0.10,
		},
	}
}

// DefaultPreferencesConfig returns a PreferencesConfig with sensible defaults.
func DefaultPreferencesConfig() PreferencesConfig {
	return PreferencesConfig{
		AutoCompile:         BoolPtr(true),
		ScaffoldMaintenance: BoolPtr(true),
		Search:              SearchConfig{},
		Chunking: ChunkingConfig{
			TargetTokens:      450,
			MinTokens:         200,
			MaxTokens:         800,
			FlexibilityWindow: 0.8,
			HashLength:        DefaultHashLength,
		},
		Indexer: IndexerBM25F,
		BM25: BM25Config{
			K1: 1.2,
			B:  0.75,
		},
		BM25F: DefaultBM25FConfig(),
		Query: DefaultQueryConfig(),
		Taxonomy: TaxonomyConfig{
			MinTermFrequency: 2,
			MaxAliasCount:    10,
			POSExtraction:    true,
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

// PackageManifest represents the minimal package-only codectx.yml file
// used inside the package/ directory. This is a subset of Config containing
// only identity fields and semver-range dependencies.
type PackageManifest struct {
	// Name is the package name.
	Name string `yaml:"name"`

	// Author is the GitHub username or organization that owns this package.
	Author string `yaml:"author"`

	// Version is the package version in semver format.
	Version string `yaml:"version"`

	// Description is a one-line package description.
	Description string `yaml:"description,omitempty"`

	// Dependencies maps "name@author" to semver range constraints.
	// Published packages use ranges (e.g., ">=1.0.0") instead of the
	// project-level active/inactive format.
	Dependencies map[string]string `yaml:"dependencies,omitempty"`
}

// WriteToFile marshals the package manifest to YAML and writes it to the given path.
func (m *PackageManifest) WriteToFile(path string) error {
	return WriteYAMLFile(path, "# codectx package manifest\n# See: https://github.com/securacore/codectx\n\n", m)
}

// LoadPackageManifest reads and parses a package-only codectx.yml from the given path.
func LoadPackageManifest(path string) (*PackageManifest, error) {
	return loadYAMLFile[PackageManifest](path)
}

// DefaultPackageManifest returns a PackageManifest with sensible defaults.
func DefaultPackageManifest(name, author, description string) PackageManifest {
	return PackageManifest{
		Name:         name,
		Author:       author,
		Version:      "0.1.0",
		Description:  description,
		Dependencies: map[string]string{},
	}
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
