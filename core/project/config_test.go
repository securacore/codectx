package project_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/project"
)

func TestDefaultConfig_HasSensibleDefaults(t *testing.T) {
	cfg := project.DefaultConfig("my-project", "")

	if cfg.Root != project.DefaultRoot {
		t.Errorf("expected root %q, got %q", project.DefaultRoot, cfg.Root)
	}
	if cfg.Name != "my-project" {
		t.Errorf("expected name %q, got %q", "my-project", cfg.Name)
	}
	if cfg.Version != "0.1.0" {
		t.Errorf("expected version %q, got %q", "0.1.0", cfg.Version)
	}
	if cfg.Session == nil {
		t.Fatal("expected session config to be set")
	}
	if cfg.Session.Budget != project.DefaultSessionBudget {
		t.Errorf("expected budget %d, got %d", project.DefaultSessionBudget, cfg.Session.Budget)
	}
	if cfg.Registry != project.DefaultRegistry {
		t.Errorf("expected registry %q, got %q", project.DefaultRegistry, cfg.Registry)
	}
}

func TestDefaultConfig_CustomRoot(t *testing.T) {
	cfg := project.DefaultConfig("test", "ai-docs")
	if cfg.Root != "ai-docs" {
		t.Errorf("expected root %q, got %q", "ai-docs", cfg.Root)
	}
}

func TestConfig_WriteAndLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, project.ConfigFileName)

	original := project.DefaultConfig("roundtrip-test", "docs")
	original.Org = "testorg"
	original.Description = "A test project"
	original.Dependencies = map[string]*project.DependencyConfig{
		"react-patterns@community:latest": {Active: true},
	}

	if err := original.WriteToFile(path); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	loaded, err := project.LoadConfig(path)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("name: expected %q, got %q", original.Name, loaded.Name)
	}
	if loaded.Org != original.Org {
		t.Errorf("org: expected %q, got %q", original.Org, loaded.Org)
	}
	if loaded.Root != original.Root {
		t.Errorf("root: expected %q, got %q", original.Root, loaded.Root)
	}
	if loaded.Version != original.Version {
		t.Errorf("version: expected %q, got %q", original.Version, loaded.Version)
	}
	if loaded.Description != original.Description {
		t.Errorf("description: expected %q, got %q", original.Description, loaded.Description)
	}
	if loaded.Session == nil {
		t.Fatal("expected session to be loaded")
	}
	if loaded.Session.Budget != original.Session.Budget {
		t.Errorf("budget: expected %d, got %d", original.Session.Budget, loaded.Session.Budget)
	}
	if loaded.Registry != original.Registry {
		t.Errorf("registry: expected %q, got %q", original.Registry, loaded.Registry)
	}

	dep, ok := loaded.Dependencies["react-patterns@community:latest"]
	if !ok {
		t.Fatal("expected dependency react-patterns@community:latest")
	}
	if !dep.Active {
		t.Error("expected dependency to be active")
	}
}

func TestConfig_WriteToFile_HasHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, project.ConfigFileName)

	cfg := project.DefaultConfig("test", "")
	if err := cfg.WriteToFile(path); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "# codectx project configuration") {
		t.Error("expected config file to start with header comment")
	}
}

func TestConfig_WriteToFile_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	path := filepath.Join(nested, project.ConfigFileName)

	cfg := project.DefaultConfig("test", "")
	if err := cfg.WriteToFile(path); err != nil {
		t.Fatalf("writing config to nested path: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist at nested path: %v", err)
	}
}

func TestConfig_WriteToFile_ErrorOnInvalidPath(t *testing.T) {
	cfg := project.DefaultConfig("test", "")
	// Writing to /dev/null/impossible is invalid on all platforms.
	err := cfg.WriteToFile("/dev/null/impossible/codectx.yml")
	if err == nil {
		t.Error("expected error writing to invalid path")
	}
}

func TestLoadConfig_InvalidPath(t *testing.T) {
	_, err := project.LoadConfig("/nonexistent/path/codectx.yml")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, project.ConfigFileName)

	if err := os.WriteFile(path, []byte("{{invalid yaml"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	_, err := project.LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestDefaultAIConfig_HasSensibleDefaults(t *testing.T) {
	cfg := project.DefaultAIConfig()

	if cfg.Compilation.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected compilation model %q, got %q", "claude-sonnet-4-20250514", cfg.Compilation.Model)
	}
	if cfg.Compilation.Encoding != "cl100k_base" {
		t.Errorf("expected encoding %q, got %q", "cl100k_base", cfg.Compilation.Encoding)
	}
	if cfg.Consumption.ContextWindow != project.DefaultContextWindow {
		t.Errorf("expected context window %d, got %d", project.DefaultContextWindow, cfg.Consumption.ContextWindow)
	}
	if cfg.Consumption.ResultsCount != project.DefaultResultsCount {
		t.Errorf("expected results count %d, got %d", project.DefaultResultsCount, cfg.Consumption.ResultsCount)
	}
	if cfg.OutputFormat != "markdown" {
		t.Errorf("expected output format %q, got %q", "markdown", cfg.OutputFormat)
	}
}

func TestAIConfig_WriteToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai.yml")

	cfg := project.DefaultAIConfig()
	if err := cfg.WriteToFile(path); err != nil {
		t.Fatalf("writing ai config: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "# codectx AI configuration") {
		t.Error("expected ai config file to start with header comment")
	}
	if !strings.Contains(content, "cl100k_base") {
		t.Error("expected ai config to contain encoding")
	}
}

func TestAIConfig_WriteAndLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai.yml")

	original := project.DefaultAIConfig()
	original.Compilation.Model = "gpt-4o"
	original.Compilation.Encoding = "o200k_base"
	original.Consumption.ContextWindow = 128000
	original.OutputFormat = "xml_tags"

	if err := original.WriteToFile(path); err != nil {
		t.Fatalf("writing ai config: %v", err)
	}

	loaded, err := project.LoadAIConfig(path)
	if err != nil {
		t.Fatalf("loading ai config: %v", err)
	}

	if loaded.Compilation.Model != original.Compilation.Model {
		t.Errorf("compilation model: expected %q, got %q", original.Compilation.Model, loaded.Compilation.Model)
	}
	if loaded.Compilation.Encoding != original.Compilation.Encoding {
		t.Errorf("compilation encoding: expected %q, got %q", original.Compilation.Encoding, loaded.Compilation.Encoding)
	}
	if loaded.Consumption.Model != original.Consumption.Model {
		t.Errorf("consumption model: expected %q, got %q", original.Consumption.Model, loaded.Consumption.Model)
	}
	if loaded.Consumption.ContextWindow != original.Consumption.ContextWindow {
		t.Errorf("context window: expected %d, got %d", original.Consumption.ContextWindow, loaded.Consumption.ContextWindow)
	}
	if loaded.Consumption.ResultsCount != original.Consumption.ResultsCount {
		t.Errorf("results count: expected %d, got %d", original.Consumption.ResultsCount, loaded.Consumption.ResultsCount)
	}
	if loaded.OutputFormat != original.OutputFormat {
		t.Errorf("output format: expected %q, got %q", original.OutputFormat, loaded.OutputFormat)
	}
}

func TestLoadAIConfig_InvalidPath(t *testing.T) {
	_, err := project.LoadAIConfig("/nonexistent/path/ai.yml")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestLoadAIConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai.yml")

	if err := os.WriteFile(path, []byte("{{invalid yaml"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	_, err := project.LoadAIConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadAIConfigForProject(t *testing.T) {
	dir := t.TempDir()

	// Set up project structure: dir/docs/.codectx/ai.yml
	cfg := project.DefaultConfig("test", "docs")
	codectxDir := filepath.Join(dir, "docs", project.CodectxDir)
	if err := os.MkdirAll(codectxDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}

	aiCfg := project.DefaultAIConfig()
	aiCfg.Compilation.Model = "custom-model"
	if err := aiCfg.WriteToFile(filepath.Join(codectxDir, project.AIConfigFile)); err != nil {
		t.Fatal(err)
	}

	loaded, err := project.LoadAIConfigForProject(dir, &cfg)
	if err != nil {
		t.Fatalf("LoadAIConfigForProject: %v", err)
	}
	if loaded.Compilation.Model != "custom-model" {
		t.Errorf("model = %q, want %q", loaded.Compilation.Model, "custom-model")
	}
}

func TestLoadAIConfigForProject_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs")

	_, err := project.LoadAIConfigForProject(dir, &cfg)
	if err == nil {
		t.Error("expected error for missing AI config file")
	}
}

func TestPreferencesConfig_WriteAndLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "preferences.yml")

	original := project.DefaultPreferencesConfig()
	original.Chunking.TargetTokens = 600
	original.BM25.K1 = 1.5
	original.Taxonomy.MaxAliasCount = 20
	original.Validation.RequireSpec = true

	if err := original.WriteToFile(path); err != nil {
		t.Fatalf("writing preferences config: %v", err)
	}

	loaded, err := project.LoadPreferencesConfig(path)
	if err != nil {
		t.Fatalf("loading preferences config: %v", err)
	}

	if loaded.Chunking.TargetTokens != original.Chunking.TargetTokens {
		t.Errorf("target tokens: expected %d, got %d", original.Chunking.TargetTokens, loaded.Chunking.TargetTokens)
	}
	if loaded.Chunking.MinTokens != original.Chunking.MinTokens {
		t.Errorf("min tokens: expected %d, got %d", original.Chunking.MinTokens, loaded.Chunking.MinTokens)
	}
	if loaded.BM25.K1 != original.BM25.K1 {
		t.Errorf("bm25 k1: expected %f, got %f", original.BM25.K1, loaded.BM25.K1)
	}
	if loaded.BM25.B != original.BM25.B {
		t.Errorf("bm25 b: expected %f, got %f", original.BM25.B, loaded.BM25.B)
	}
	if loaded.Taxonomy.MaxAliasCount != original.Taxonomy.MaxAliasCount {
		t.Errorf("max alias count: expected %d, got %d", original.Taxonomy.MaxAliasCount, loaded.Taxonomy.MaxAliasCount)
	}
	if loaded.Taxonomy.POSExtraction != original.Taxonomy.POSExtraction {
		t.Errorf("pos extraction: expected %v, got %v", original.Taxonomy.POSExtraction, loaded.Taxonomy.POSExtraction)
	}
	if loaded.Validation.RequireSpec != original.Validation.RequireSpec {
		t.Errorf("require spec: expected %v, got %v", original.Validation.RequireSpec, loaded.Validation.RequireSpec)
	}
	if loaded.Validation.RequireHeadings != original.Validation.RequireHeadings {
		t.Errorf("require headings: expected %v, got %v", original.Validation.RequireHeadings, loaded.Validation.RequireHeadings)
	}
}

func TestLoadPreferencesConfig_InvalidPath(t *testing.T) {
	_, err := project.LoadPreferencesConfig("/nonexistent/path/preferences.yml")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestLoadPreferencesConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "preferences.yml")

	if err := os.WriteFile(path, []byte("{{invalid yaml"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	_, err := project.LoadPreferencesConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestPreferencesConfig_WriteAndLoad_HashLength(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "preferences.yml")

	original := project.DefaultPreferencesConfig()
	original.Chunking.HashLength = 32 // Non-default value.

	if err := original.WriteToFile(path); err != nil {
		t.Fatalf("writing preferences config: %v", err)
	}

	loaded, err := project.LoadPreferencesConfig(path)
	if err != nil {
		t.Fatalf("loading preferences config: %v", err)
	}

	if loaded.Chunking.HashLength != 32 {
		t.Errorf("hash length: expected 32, got %d", loaded.Chunking.HashLength)
	}
}

func TestDefaultPreferencesConfig_HasSensibleDefaults(t *testing.T) {
	cfg := project.DefaultPreferencesConfig()

	if cfg.Chunking.TargetTokens != 450 {
		t.Errorf("expected target tokens 450, got %d", cfg.Chunking.TargetTokens)
	}
	if cfg.Chunking.MinTokens != 200 {
		t.Errorf("expected min tokens 200, got %d", cfg.Chunking.MinTokens)
	}
	if cfg.Chunking.MaxTokens != 800 {
		t.Errorf("expected max tokens 800, got %d", cfg.Chunking.MaxTokens)
	}
	if cfg.Chunking.FlexibilityWindow != 0.8 {
		t.Errorf("expected flexibility window 0.8, got %f", cfg.Chunking.FlexibilityWindow)
	}
	if cfg.BM25.K1 != 1.2 {
		t.Errorf("expected BM25 k1 1.2, got %f", cfg.BM25.K1)
	}
	if cfg.BM25.B != 0.75 {
		t.Errorf("expected BM25 b 0.75, got %f", cfg.BM25.B)
	}
	if cfg.Taxonomy.MinTermFrequency != 2 {
		t.Errorf("expected min term frequency 2, got %d", cfg.Taxonomy.MinTermFrequency)
	}
	if cfg.Taxonomy.MaxAliasCount != 10 {
		t.Errorf("expected max alias count 10, got %d", cfg.Taxonomy.MaxAliasCount)
	}
	if !cfg.Taxonomy.POSExtraction {
		t.Error("expected POS extraction to be enabled")
	}
	if !cfg.Taxonomy.LLMAliasGeneration {
		t.Error("expected LLM alias generation to be enabled")
	}
	if !cfg.Validation.RequireReadme {
		t.Error("expected require_readme to be true")
	}
	if cfg.Validation.RequireSpec {
		t.Error("expected require_spec to be false")
	}
	if cfg.Validation.MaxFileTokens != 10000 {
		t.Errorf("expected max file tokens 10000, got %d", cfg.Validation.MaxFileTokens)
	}
	if !cfg.Validation.RequireHeadings {
		t.Error("expected require_headings to be true")
	}
}

func TestPreferencesConfig_WriteToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "preferences.yml")

	cfg := project.DefaultPreferencesConfig()
	if err := cfg.WriteToFile(path); err != nil {
		t.Fatalf("writing preferences config: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "# codectx compiler preferences") {
		t.Error("expected preferences config file to start with header comment")
	}
	if !strings.Contains(content, "target_tokens") {
		t.Error("expected preferences config to contain chunking settings")
	}
}

func TestYAMLFiles_Use2SpaceIndent(t *testing.T) {
	dir := t.TempDir()

	// Write all three config types.
	cfgPath := filepath.Join(dir, project.ConfigFileName)
	cfg := project.DefaultConfig("indent-test", "docs")
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	aiPath := filepath.Join(dir, "ai.yml")
	ai := project.DefaultAIConfig()
	if err := ai.WriteToFile(aiPath); err != nil {
		t.Fatalf("writing ai config: %v", err)
	}

	prefsPath := filepath.Join(dir, "preferences.yml")
	prefs := project.DefaultPreferencesConfig()
	if err := prefs.WriteToFile(prefsPath); err != nil {
		t.Fatalf("writing preferences config: %v", err)
	}

	files := map[string]string{
		"codectx.yml":     cfgPath,
		"ai.yml":          aiPath,
		"preferences.yml": prefsPath,
	}

	for name, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		content := string(data)

		// Must not contain tabs.
		if strings.Contains(content, "\t") {
			t.Errorf("%s: contains tab characters", name)
		}

		// Verify 2-space indent: find lines that start with spaces and
		// check that the indent is a multiple of 2 (not 4-only).
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if indent > 0 && indent%2 != 0 {
				t.Errorf("%s line %d: indent %d is not a multiple of 2: %q", name, i+1, indent, line)
			}
		}

		// Verify that nested keys use 2-space indent (not 4).
		// The preferences.yml has nested keys under "chunking:" that must
		// be indented 2 spaces.
		if name == "preferences.yml" {
			if !strings.Contains(content, "  target_tokens:") {
				t.Errorf("%s: expected 2-space indent for target_tokens", name)
			}
			if strings.Contains(content, "    target_tokens:") {
				t.Errorf("%s: found 4-space indent for target_tokens, expected 2-space", name)
			}
		}
	}
}

func TestDefaultPreferencesConfig_HasHashLength(t *testing.T) {
	cfg := project.DefaultPreferencesConfig()

	if cfg.Chunking.HashLength != project.DefaultHashLength {
		t.Errorf("expected hash length %d, got %d", project.DefaultHashLength, cfg.Chunking.HashLength)
	}
}

func TestResolveRoot(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", project.DefaultRoot},
		{"docs", "docs"},
		{"ai-docs", "ai-docs"},
		{".codectx-docs", ".codectx-docs"},
		{"custom/path", "custom/path"},
	}

	for _, tt := range tests {
		got := project.ResolveRoot(tt.input)
		if got != tt.expected {
			t.Errorf("ResolveRoot(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestClampHashLength(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, project.DefaultHashLength},
		{-1, project.DefaultHashLength},
		{4, project.MinHashLength},
		{7, project.MinHashLength},
		{8, 8},
		{16, 16},
		{64, 64},
		{100, project.MaxHashLength},
		{1000, project.MaxHashLength},
	}

	for _, tt := range tests {
		got := project.ClampHashLength(tt.input)
		if got != tt.expected {
			t.Errorf("ClampHashLength(%d) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// SessionConfig.EffectiveBudget
// ---------------------------------------------------------------------------

func TestEffectiveBudget_NilReceiver(t *testing.T) {
	var s *project.SessionConfig
	if got := s.EffectiveBudget(); got != project.DefaultSessionBudget {
		t.Errorf("nil.EffectiveBudget() = %d, want %d", got, project.DefaultSessionBudget)
	}
}

func TestEffectiveBudget_ZeroBudget(t *testing.T) {
	s := &project.SessionConfig{Budget: 0}
	if got := s.EffectiveBudget(); got != project.DefaultSessionBudget {
		t.Errorf("EffectiveBudget() with zero = %d, want %d", got, project.DefaultSessionBudget)
	}
}

func TestEffectiveBudget_NegativeBudget(t *testing.T) {
	s := &project.SessionConfig{Budget: -100}
	if got := s.EffectiveBudget(); got != project.DefaultSessionBudget {
		t.Errorf("EffectiveBudget() with negative = %d, want %d", got, project.DefaultSessionBudget)
	}
}

func TestEffectiveBudget_CustomBudget(t *testing.T) {
	s := &project.SessionConfig{Budget: 50000}
	if got := s.EffectiveBudget(); got != 50000 {
		t.Errorf("EffectiveBudget() = %d, want 50000", got)
	}
}
