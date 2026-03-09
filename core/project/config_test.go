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
	if cfg.Session.Budget != 30000 {
		t.Errorf("expected budget 30000, got %d", cfg.Session.Budget)
	}
	if cfg.Registry != "github.com" {
		t.Errorf("expected registry %q, got %q", "github.com", cfg.Registry)
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

func TestLoadConfig_InvalidPath(t *testing.T) {
	_, err := project.LoadConfig("/nonexistent/path/codectx.yml")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, project.ConfigFileName)

	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
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
	if cfg.Consumption.ContextWindow != 200000 {
		t.Errorf("expected context window 200000, got %d", cfg.Consumption.ContextWindow)
	}
	if cfg.Consumption.ResultsCount != 10 {
		t.Errorf("expected results count 10, got %d", cfg.Consumption.ResultsCount)
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
