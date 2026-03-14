package compile_test

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/project"
)

func TestBuildConfig_BasicMapping(t *testing.T) {
	cfg := &project.Config{
		Name: "test-project",
		Root: "docs",
	}
	aiCfg := &project.AIConfig{
		Compilation: project.AICompilationConfig{
			Encoding: "cl100k_base",
		},
	}
	prefsCfg := &project.PreferencesConfig{
		Chunking: project.ChunkingConfig{
			MinTokens: 100,
			MaxTokens: 1500,
		},
	}

	got := compile.BuildConfig("/projects/test", "/projects/test/docs", cfg, aiCfg, prefsCfg)

	if got.ProjectDir != "/projects/test" {
		t.Errorf("ProjectDir = %q, want /projects/test", got.ProjectDir)
	}
	if got.RootDir != "/projects/test/docs" {
		t.Errorf("RootDir = %q, want /projects/test/docs", got.RootDir)
	}
	if got.Encoding != "cl100k_base" {
		t.Errorf("Encoding = %q, want cl100k_base", got.Encoding)
	}
	if got.Chunking.MinTokens != 100 {
		t.Errorf("Chunking.MinTokens = %d, want 100", got.Chunking.MinTokens)
	}
	if got.SystemDir != "system" {
		t.Errorf("SystemDir = %q, want system", got.SystemDir)
	}
}

func TestBuildConfig_ActiveDeps(t *testing.T) {
	cfg := &project.Config{
		Dependencies: map[string]*project.DependencyConfig{
			"pkg-a": {Active: true},
			"pkg-b": {Active: false},
			"pkg-c": {Active: true},
			"pkg-d": nil,
		},
	}
	aiCfg := &project.AIConfig{}
	prefsCfg := &project.PreferencesConfig{}

	got := compile.BuildConfig("/p", "/r", cfg, aiCfg, prefsCfg)

	if len(got.ActiveDeps) != 2 {
		t.Fatalf("expected 2 active deps, got %d", len(got.ActiveDeps))
	}
	if !got.ActiveDeps["pkg-a"] {
		t.Error("expected pkg-a to be active")
	}
	if !got.ActiveDeps["pkg-c"] {
		t.Error("expected pkg-c to be active")
	}
	if got.ActiveDeps["pkg-b"] {
		t.Error("expected pkg-b to NOT be active")
	}
}

func TestBuildConfig_NilDependencies(t *testing.T) {
	cfg := &project.Config{}
	aiCfg := &project.AIConfig{}
	prefsCfg := &project.PreferencesConfig{}

	got := compile.BuildConfig("/p", "/r", cfg, aiCfg, prefsCfg)

	if got.ActiveDeps == nil {
		t.Fatal("ActiveDeps should not be nil")
	}
	if len(got.ActiveDeps) != 0 {
		t.Errorf("expected 0 active deps, got %d", len(got.ActiveDeps))
	}
}

func TestBuildConfig_SessionPassthrough(t *testing.T) {
	session := &project.SessionConfig{
		AlwaysLoaded: []string{"foundation/standards"},
		Budget:       25000,
	}
	cfg := &project.Config{Session: session}
	aiCfg := &project.AIConfig{}
	prefsCfg := &project.PreferencesConfig{}

	got := compile.BuildConfig("/p", "/r", cfg, aiCfg, prefsCfg)

	if got.Session == nil {
		t.Fatal("expected Session to be passed through")
	}
	if got.Session.Budget != 25000 {
		t.Errorf("Session.Budget = %d, want 25000", got.Session.Budget)
	}
}

func TestBuildConfig_CompiledDir(t *testing.T) {
	cfg := &project.Config{}
	aiCfg := &project.AIConfig{}
	prefsCfg := &project.PreferencesConfig{}

	got := compile.BuildConfig("/p", "/p/docs", cfg, aiCfg, prefsCfg)

	if !strings.HasSuffix(got.CompiledDir, ".codectx/compiled") {
		t.Errorf("CompiledDir = %q, expected to end with .codectx/compiled", got.CompiledDir)
	}
}
