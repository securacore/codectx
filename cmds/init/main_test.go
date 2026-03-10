package init

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/tui"
)

// ---------------------------------------------------------------------------
// resolveTarget
// ---------------------------------------------------------------------------

func TestResolveTarget_NoArgs(t *testing.T) {
	dir, created, err := resolveTarget(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected created=false for no args")
	}

	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("expected CWD %q, got %q", cwd, dir)
	}
}

func TestResolveTarget_Dot(t *testing.T) {
	dir, created, err := resolveTarget([]string{"."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected created=false for '.'")
	}

	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("expected CWD %q, got %q", cwd, dir)
	}
}

func TestResolveTarget_ExistingDir(t *testing.T) {
	existing := t.TempDir()

	dir, created, err := resolveTarget([]string{existing})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected created=false for existing directory")
	}
	if dir != existing {
		t.Errorf("expected %q, got %q", existing, dir)
	}
}

func TestResolveTarget_NewDir(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "newproject")

	dir, created, err := resolveTarget([]string{target})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected created=true for new directory")
	}

	// Verify directory was actually created.
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("should be a directory")
	}
}

func TestResolveTarget_ExistingFile(t *testing.T) {
	parent := t.TempDir()
	filePath := filepath.Join(parent, "notadir")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := resolveTarget([]string{filePath})
	if err == nil {
		t.Error("expected error when target is a file, not a directory")
	}
}

func TestResolveTarget_EmptySlice(t *testing.T) {
	dir, created, err := resolveTarget([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected created=false for empty args")
	}

	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("expected CWD %q, got %q", cwd, dir)
	}
}

func TestResolveTarget_NestedNewDirs(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "a", "b", "c")

	dir, created, err := resolveTarget([]string{target})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected created=true for nested new directories")
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("nested directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("should be a directory")
	}
}

// ---------------------------------------------------------------------------
// encodingForModel
// ---------------------------------------------------------------------------

func TestEncodingForModel_GPT4o(t *testing.T) {
	if enc := encodingForModel("gpt-4o"); enc != "o200k_base" {
		t.Errorf("expected o200k_base for gpt-4o, got %q", enc)
	}
}

func TestEncodingForModel_O1(t *testing.T) {
	if enc := encodingForModel("o1"); enc != "o200k_base" {
		t.Errorf("expected o200k_base for o1, got %q", enc)
	}
}

func TestEncodingForModel_O3Mini(t *testing.T) {
	if enc := encodingForModel("o3-mini"); enc != "o200k_base" {
		t.Errorf("expected o200k_base for o3-mini, got %q", enc)
	}
}

func TestEncodingForModel_Claude(t *testing.T) {
	if enc := encodingForModel("claude-sonnet-4-20250514"); enc != "cl100k_base" {
		t.Errorf("expected cl100k_base for Claude, got %q", enc)
	}
}

func TestEncodingForModel_Gemini(t *testing.T) {
	if enc := encodingForModel("gemini-2.0-flash"); enc != "cl100k_base" {
		t.Errorf("expected cl100k_base for Gemini, got %q", enc)
	}
}

func TestEncodingForModel_Unknown(t *testing.T) {
	if enc := encodingForModel("some-custom-model"); enc != "cl100k_base" {
		t.Errorf("expected cl100k_base for unknown model, got %q", enc)
	}
}

func TestEncodingForModel_Empty(t *testing.T) {
	if enc := encodingForModel(""); enc != "cl100k_base" {
		t.Errorf("expected cl100k_base for empty string, got %q", enc)
	}
}

// ---------------------------------------------------------------------------
// buildSummaryTree
// ---------------------------------------------------------------------------

func TestBuildSummaryTree_DefaultRoot(t *testing.T) {
	tree := buildSummaryTree("docs", nil)

	if len(tree) != 2 {
		t.Fatalf("expected 2 top-level nodes, got %d", len(tree))
	}

	if tree[0].Name != "codectx.yml" {
		t.Errorf("first node should be codectx.yml, got %q", tree[0].Name)
	}

	if tree[1].Name != "docs/" {
		t.Errorf("second node should be docs/, got %q", tree[1].Name)
	}

	// Check children of docs/.
	docsChildren := tree[1].Children
	if len(docsChildren) != 6 {
		t.Fatalf("expected 6 children of docs/, got %d", len(docsChildren))
	}

	expectedNames := []string{"foundation/", "topics/", "plans/", "prompts/", "system/", ".codectx/"}
	for i, expected := range expectedNames {
		if docsChildren[i].Name != expected {
			t.Errorf("child %d: expected %q, got %q", i, expected, docsChildren[i].Name)
		}
	}
}

func TestBuildSummaryTree_CustomRoot(t *testing.T) {
	tree := buildSummaryTree("ai-docs", nil)

	if tree[1].Name != "ai-docs/" {
		t.Errorf("expected root name 'ai-docs/', got %q", tree[1].Name)
	}
}

func TestBuildSummaryTree_SystemChildren(t *testing.T) {
	tree := buildSummaryTree("docs", nil)

	// Find system/ node.
	var systemNode *tui.TreeNode
	for i := range tree[1].Children {
		if tree[1].Children[i].Name == "system/" {
			systemNode = &tree[1].Children[i]
			break
		}
	}

	if systemNode == nil {
		t.Fatal("system/ node not found")
	}

	if len(systemNode.Children) != 4 {
		t.Errorf("expected 4 system/ children, got %d", len(systemNode.Children))
	}
}

func TestBuildSummaryTree_CodectxChildren(t *testing.T) {
	tree := buildSummaryTree("docs", nil)

	// Find .codectx/ node.
	var codectxNode *tui.TreeNode
	for i := range tree[1].Children {
		if tree[1].Children[i].Name == ".codectx/" {
			codectxNode = &tree[1].Children[i]
			break
		}
	}

	if codectxNode == nil {
		t.Fatal(".codectx/ node not found")
	}

	expectedChildren := []string{"ai.yml", "preferences.yml", "compiled/", "packages/"}
	if len(codectxNode.Children) != len(expectedChildren) {
		t.Fatalf("expected %d .codectx/ children, got %d", len(expectedChildren), len(codectxNode.Children))
	}

	for i, expected := range expectedChildren {
		if codectxNode.Children[i].Name != expected {
			t.Errorf(".codectx/ child %d: expected %q, got %q", i, expected, codectxNode.Children[i].Name)
		}
	}
}
