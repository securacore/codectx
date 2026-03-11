package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/project"
)

// mustWriteFile creates a file with the given content, creating parent directories as needed.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), project.DirPerm); err != nil {
		t.Fatalf("creating directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), project.FilePerm); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestDiscoverSources_FindsMarkdownFiles(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "overview.md"), "# Overview")
	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth")
	mustWriteFile(t, filepath.Join(root, "topics", "auth.spec.md"), "# Auth Spec")

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}

	// Should be sorted by path.
	paths := make([]string, len(sources))
	for i, s := range sources {
		paths[i] = s.Path
	}

	expected := []string{
		"foundation/overview.md",
		"topics/auth.md",
		"topics/auth.spec.md",
	}
	for i, p := range expected {
		if paths[i] != p {
			t.Errorf("source[%d]: expected %q, got %q", i, p, paths[i])
		}
	}
}

func TestDiscoverSources_ClassifiesSpecFiles(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth")
	mustWriteFile(t, filepath.Join(root, "topics", "auth.spec.md"), "# Auth Spec")
	mustWriteFile(t, filepath.Join(root, "system", "compiler.md"), "# Compiler")

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	specCount := 0
	for _, s := range sources {
		if s.IsSpec {
			specCount++
			if s.Path != "topics/auth.spec.md" {
				t.Errorf("unexpected spec file: %s", s.Path)
			}
		}
	}

	if specCount != 1 {
		t.Errorf("expected 1 spec file, got %d", specCount)
	}
}

func TestDiscoverSources_SkipsHiddenDirectories(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth")
	mustWriteFile(t, filepath.Join(root, ".hidden", "secret.md"), "# Secret")
	mustWriteFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/main")

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source (hidden dirs skipped), got %d", len(sources))
	}

	if sources[0].Path != "topics/auth.md" {
		t.Errorf("expected topics/auth.md, got %s", sources[0].Path)
	}
}

func TestDiscoverSources_SkipsNonMarkdownFiles(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth")
	mustWriteFile(t, filepath.Join(root, "topics", "diagram.png"), "PNG data")
	mustWriteFile(t, filepath.Join(root, "topics", "notes.txt"), "notes")
	mustWriteFile(t, filepath.Join(root, "codectx.yml"), "name: test")

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
}

func TestDiscoverSources_EmptyDirectory(t *testing.T) {
	root := t.TempDir()

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	if len(sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(sources))
	}
}

func TestDiscoverSources_AbsPathIsPopulated(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth")

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].AbsPath != filepath.Join(root, "topics", "auth.md") {
		t.Errorf("expected AbsPath %q, got %q",
			filepath.Join(root, "topics", "auth.md"), sources[0].AbsPath)
	}
}

func TestDiscoverSources_IncludesActivePackages(t *testing.T) {
	root := t.TempDir()

	// Local doc.
	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth")

	// Active package.
	mustWriteFile(t, filepath.Join(root, project.CodectxDir, project.PackagesDir, "react-patterns", "topics", "hooks.md"), "# Hooks")
	// Inactive package.
	mustWriteFile(t, filepath.Join(root, project.CodectxDir, project.PackagesDir, "vue-patterns", "topics", "refs.md"), "# Refs")

	activeDeps := map[string]bool{
		"react-patterns": true,
		"vue-patterns":   false,
	}

	sources, err := compile.DiscoverSources(root, activeDeps)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	// Should have local doc + active package doc, but not inactive.
	paths := make(map[string]bool, len(sources))
	for _, s := range sources {
		paths[s.Path] = true
	}

	if !paths["topics/auth.md"] {
		t.Error("expected topics/auth.md")
	}
	if !paths[project.CodectxDir+"/"+project.PackagesDir+"/react-patterns/topics/hooks.md"] {
		t.Error("expected active package hooks.md")
	}
	if paths[project.CodectxDir+"/"+project.PackagesDir+"/vue-patterns/topics/refs.md"] {
		t.Error("expected inactive package refs.md to be excluded")
	}
}

func TestDiscoverSources_NilActiveDepsSkipsAllPackages(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth")
	mustWriteFile(t, filepath.Join(root, project.CodectxDir, project.PackagesDir, "react-patterns", "topics", "hooks.md"), "# Hooks")

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	// Only local docs, packages skipped since nil activeDeps means nothing is active.
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
}

func TestDiscoverSources_SystemDirIncluded(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "topics", "auth.md"), "# Auth")
	mustWriteFile(t, filepath.Join(root, project.SystemDir, "topics", "taxonomy-generation", "README.md"), "# Taxonomy")
	mustWriteFile(t, filepath.Join(root, project.SystemDir, "foundation", "compiler.md"), "# Compiler")

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}

	paths := make(map[string]bool, len(sources))
	for _, s := range sources {
		paths[s.Path] = true
	}

	if !paths["system/topics/taxonomy-generation/README.md"] {
		t.Error("expected system taxonomy README")
	}
	if !paths["system/foundation/compiler.md"] {
		t.Error("expected system compiler doc")
	}
}

func TestDiscoverSources_DeterministicOrder(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "z-last.md"), "# Z")
	mustWriteFile(t, filepath.Join(root, "a-first.md"), "# A")
	mustWriteFile(t, filepath.Join(root, "m-middle.md"), "# M")

	sources, err := compile.DiscoverSources(root, nil)
	if err != nil {
		t.Fatalf("DiscoverSources: %v", err)
	}

	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}

	if sources[0].Path != "a-first.md" {
		t.Errorf("expected first source a-first.md, got %s", sources[0].Path)
	}
	if sources[1].Path != "m-middle.md" {
		t.Errorf("expected second source m-middle.md, got %s", sources[1].Path)
	}
	if sources[2].Path != "z-last.md" {
		t.Errorf("expected third source z-last.md, got %s", sources[2].Path)
	}
}

func TestDiscoverSources_NonexistentRootReturnsError(t *testing.T) {
	_, err := compile.DiscoverSources("/nonexistent/path", nil)
	if err == nil {
		t.Error("expected error for nonexistent root directory")
	}
}
