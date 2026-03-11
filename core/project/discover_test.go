package project_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/project"
)

func TestDiscover_FindsConfigInCurrentDir(t *testing.T) {
	dir := t.TempDir()

	// Create a codectx.yml in the temp dir.
	configPath := filepath.Join(dir, project.ConfigFileName)
	if err := os.WriteFile(configPath, []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := project.Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if found != dir {
		t.Errorf("expected %s, got %s", dir, found)
	}
}

func TestDiscover_FindsConfigInParentDir(t *testing.T) {
	dir := t.TempDir()

	// Create codectx.yml in the root temp dir.
	configPath := filepath.Join(dir, project.ConfigFileName)
	if err := os.WriteFile(configPath, []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a child directory and search from there.
	child := filepath.Join(dir, "sub", "nested")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}

	found, err := project.Discover(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if found != dir {
		t.Errorf("expected %s, got %s", dir, found)
	}
}

func TestDiscover_ReturnsErrNotFoundAtRoot(t *testing.T) {
	// Use a deep temp dir that has no codectx.yml anywhere above it.
	dir := t.TempDir()

	_, err := project.Discover(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, project.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestDiscover_FindsClosestConfig(t *testing.T) {
	// Create nested project structure: outer has codectx.yml, inner has codectx.yml.
	outer := t.TempDir()
	inner := filepath.Join(outer, "inner")
	if err := os.MkdirAll(inner, 0755); err != nil {
		t.Fatal(err)
	}

	// Both outer and inner have codectx.yml.
	for _, dir := range []string{outer, inner} {
		configPath := filepath.Join(dir, project.ConfigFileName)
		if err := os.WriteFile(configPath, []byte("name: test\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Searching from inner should find inner, not outer.
	found, err := project.Discover(inner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if found != inner {
		t.Errorf("expected %s (inner), got %s", inner, found)
	}
}

func TestRootDir_DefaultRoot(t *testing.T) {
	cfg := &project.Config{Root: ""}
	result := project.RootDir("/some/project", cfg)
	expected := filepath.Join("/some/project", project.DefaultRoot)

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRootDir_CustomRoot(t *testing.T) {
	cfg := &project.Config{Root: "ai-docs"}
	result := project.RootDir("/some/project", cfg)
	expected := "/some/project/ai-docs"

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

// ---------------------------------------------------------------------------
// DiscoverAndLoad
// ---------------------------------------------------------------------------

func TestDiscoverAndLoad_Success(t *testing.T) {
	dir := t.TempDir()

	cfg := project.DefaultConfig("test-project", "")
	if err := cfg.WriteToFile(filepath.Join(dir, project.ConfigFileName)); err != nil {
		t.Fatal(err)
	}

	projectDir, loadedCfg, err := project.DiscoverAndLoad(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if projectDir != dir {
		t.Errorf("expected %s, got %s", dir, projectDir)
	}
	if loadedCfg.Name != "test-project" {
		t.Errorf("expected name 'test-project', got %q", loadedCfg.Name)
	}
}

func TestDiscoverAndLoad_NoProject(t *testing.T) {
	dir := t.TempDir()

	_, _, err := project.DiscoverAndLoad(dir)
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestDiscoverAndLoad_InvalidConfig(t *testing.T) {
	dir := t.TempDir()

	// Write invalid YAML.
	if err := os.WriteFile(filepath.Join(dir, project.ConfigFileName), []byte(":\n\t\tinvalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := project.DiscoverAndLoad(dir)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

// ---------------------------------------------------------------------------
// ContextRelPath
// ---------------------------------------------------------------------------

func TestContextRelPath_DefaultRoot(t *testing.T) {
	got := project.ContextRelPath("")
	want := "docs/.codectx/compiled/context.md"
	if got != want {
		t.Errorf("ContextRelPath('') = %q, want %q", got, want)
	}
}

func TestContextRelPath_CustomRoot(t *testing.T) {
	got := project.ContextRelPath("ai-docs")
	want := "ai-docs/.codectx/compiled/context.md"
	if got != want {
		t.Errorf("ContextRelPath('ai-docs') = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// PackagesPath
// ---------------------------------------------------------------------------

func TestPackagesPath(t *testing.T) {
	got := project.PackagesPath("/some/root")
	want := filepath.Join("/some/root", project.CodectxDir, project.PackagesDir)
	if got != want {
		t.Errorf("PackagesPath('/some/root') = %q, want %q", got, want)
	}
}
