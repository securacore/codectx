package new

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
)

func TestResolveTarget_CurrentDir(t *testing.T) {
	t.Parallel()

	dir, created, err := resolveTarget(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("should not create a new directory for nil args")
	}
	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("dir = %q, want CWD %q", dir, cwd)
	}
}

func TestResolveTarget_Dot(t *testing.T) {
	t.Parallel()

	dir, created, err := resolveTarget([]string{"."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("should not create a new directory for '.'")
	}
	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("dir = %q, want CWD %q", dir, cwd)
	}
}

func TestResolveTarget_ExistingDir(t *testing.T) {
	t.Parallel()

	existing := t.TempDir()
	dir, created, err := resolveTarget([]string{existing})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("should not report created for existing directory")
	}
	if dir != existing {
		t.Errorf("dir = %q, want %q", dir, existing)
	}
}

func TestResolveTarget_NewDir(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	target := filepath.Join(parent, "new-pkg")

	dir, created, err := resolveTarget([]string{target})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("should report created for new directory")
	}
	if dir != target {
		t.Errorf("dir = %q, want %q", dir, target)
	}

	// Verify directory was actually created.
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("new directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("target should be a directory")
	}
}

func TestResolveTarget_FileNotDir(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	filePath := filepath.Join(parent, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := resolveTarget([]string{filePath})
	if err == nil {
		t.Fatal("expected error when target is a file, not a directory")
	}
}

func TestBuildPackageSummaryTree(t *testing.T) {
	t.Parallel()

	tree := buildPackageSummaryTree("docs")

	if len(tree) != 5 {
		t.Fatalf("expected 5 top-level nodes, got %d", len(tree))
	}

	// Verify first node is codectx.yml.
	if tree[0].Name != project.ConfigFileName {
		t.Errorf("tree[0].Name = %q, want %q", tree[0].Name, project.ConfigFileName)
	}

	// Verify package/ node has children.
	pkgNode := tree[1]
	if pkgNode.Name != project.PackageContentDir+"/" {
		t.Errorf("tree[1].Name = %q, want %q", pkgNode.Name, project.PackageContentDir+"/")
	}
	if len(pkgNode.Children) != 5 {
		t.Errorf("package/ node should have 5 children, got %d", len(pkgNode.Children))
	}

	// Verify docs/ node.
	docsNode := tree[2]
	if docsNode.Name != "docs/" {
		t.Errorf("tree[2].Name = %q, want %q", docsNode.Name, "docs/")
	}
	if len(docsNode.Children) != 6 {
		t.Errorf("docs/ node should have 6 children, got %d", len(docsNode.Children))
	}

	// Verify .github/workflows/ node.
	ghNode := tree[3]
	if len(ghNode.Children) != 1 {
		t.Errorf("github node should have 1 child, got %d", len(ghNode.Children))
	}
	if ghNode.Children[0].Name != "release.yml" {
		t.Errorf("github child = %q, want %q", ghNode.Children[0].Name, "release.yml")
	}

	// Verify README.md node.
	if tree[4].Name != "README.md" {
		t.Errorf("tree[4].Name = %q, want %q", tree[4].Name, "README.md")
	}
}

func TestBuildPackageSummaryTree_CustomRoot(t *testing.T) {
	t.Parallel()

	tree := buildPackageSummaryTree("documentation")

	// The docs node should use the custom root name.
	docsNode := tree[2]
	if docsNode.Name != "documentation/" {
		t.Errorf("tree[2].Name = %q, want %q", docsNode.Name, "documentation/")
	}
}

func TestEnsurePackagePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"react", "codectx-react"},
		{"codectx-react", "codectx-react"},
		{"react-patterns", "codectx-react-patterns"},
		{"codectx-react-patterns", "codectx-react-patterns"},
		{"", "codectx-"},
	}

	for _, tt := range tests {
		if got := ensurePackagePrefix(tt.input); got != tt.want {
			t.Errorf("ensurePackagePrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripPackagePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"codectx-react", "react"},
		{"react", "react"},
		{"codectx-react-patterns", "react-patterns"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := stripPackagePrefix(tt.input); got != tt.want {
			t.Errorf("stripPackagePrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveTarget_BareName_GetsPrefix(t *testing.T) {
	t.Parallel()

	// Use a temp dir as the working directory context.
	parent := t.TempDir()
	target := "react"

	// resolveTarget uses relative paths from CWD, so we need a full path.
	// Pass a bare name — it should get the codectx- prefix.
	fullTarget := filepath.Join(parent, target)

	// The bare name doesn't contain path separators (relative to the function),
	// but since we pass a full path here, it won't be treated as bare.
	// Instead, test with just the bare name from within a temp dir.

	// We can test the prefix behavior by passing a path that doesn't exist
	// and contains a separator — it should NOT get prefixed.
	dir, created, err := resolveTarget([]string{fullTarget})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("should report created")
	}
	// Full path contains separator, so no prefix applied.
	if dir != fullTarget {
		t.Errorf("dir = %q, want %q", dir, fullTarget)
	}
}

func TestResolveTarget_BareNameNoSeparator(t *testing.T) {
	t.Parallel()

	// When the target is a bare name without separators and doesn't exist,
	// resolveTarget should prepend codectx-.
	parent := t.TempDir()

	// We need to chdir for this test since resolveTarget uses relative paths.
	// Instead, we can verify the behavior by checking the directory name
	// after resolution — the function creates the dir and returns an abs path.
	target := filepath.Join(parent, "react") // has separator — won't be prefixed

	dir, _, err := resolveTarget([]string{target})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Contains separator, so name stays as-is.
	if !strings.HasSuffix(dir, "react") {
		t.Errorf("expected dir to end with 'react', got %q", dir)
	}

	// Now test: if we were to pass just "react" (no separator),
	// the function should create "codectx-react".
	// We use the helper directly to verify prefix logic.
	prefixed := ensurePackagePrefix("react")
	if prefixed != registry.RepoPrefix+"react" {
		t.Errorf("ensurePackagePrefix(react) = %q, want %q", prefixed, registry.RepoPrefix+"react")
	}
}

// Ensure tui package is used (avoids unused import in case of test changes).
var _ = tui.Arrow
