package tui_test

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/tui"
)

func TestRenderTree_SingleNode(t *testing.T) {
	nodes := []tui.TreeNode{
		{Name: "file.txt"},
	}

	result := tui.RenderTree(nodes)

	if !strings.Contains(result, "file.txt") {
		t.Error("expected file name in tree output")
	}
	if !strings.Contains(result, tui.TreeCorner) {
		t.Error("expected corner connector for single/last node")
	}
}

func TestRenderTree_MultipleNodes(t *testing.T) {
	nodes := []tui.TreeNode{
		{Name: "first.txt"},
		{Name: "second.txt"},
		{Name: "third.txt"},
	}

	result := tui.RenderTree(nodes)

	if !strings.Contains(result, "first.txt") {
		t.Error("expected first file")
	}
	if !strings.Contains(result, "second.txt") {
		t.Error("expected second file")
	}
	if !strings.Contains(result, "third.txt") {
		t.Error("expected third file")
	}

	// First two should use branch connector, last should use corner.
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "├") {
		t.Error("expected branch connector for first node")
	}
	if !strings.Contains(lines[1], "├") {
		t.Error("expected branch connector for second node")
	}
	if !strings.Contains(lines[2], "└") {
		t.Error("expected corner connector for last node")
	}
}

func TestRenderTree_NestedNodes(t *testing.T) {
	nodes := []tui.TreeNode{
		{
			Name: "parent/",
			Children: []tui.TreeNode{
				{Name: "child1.txt"},
				{Name: "child2.txt"},
			},
		},
	}

	result := tui.RenderTree(nodes)

	if !strings.Contains(result, "parent/") {
		t.Error("expected parent directory")
	}
	if !strings.Contains(result, "child1.txt") {
		t.Error("expected first child")
	}
	if !strings.Contains(result, "child2.txt") {
		t.Error("expected second child")
	}

	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestRenderTree_DeeplyNested(t *testing.T) {
	nodes := []tui.TreeNode{
		{
			Name: "a/",
			Children: []tui.TreeNode{
				{
					Name: "b/",
					Children: []tui.TreeNode{
						{Name: "c.txt"},
					},
				},
			},
		},
	}

	result := tui.RenderTree(nodes)

	if !strings.Contains(result, "a/") {
		t.Error("expected level 1")
	}
	if !strings.Contains(result, "b/") {
		t.Error("expected level 2")
	}
	if !strings.Contains(result, "c.txt") {
		t.Error("expected level 3")
	}
}

func TestRenderTree_EmptySlice(t *testing.T) {
	result := tui.RenderTree(nil)
	if result != "" {
		t.Errorf("expected empty string for nil nodes, got %q", result)
	}
}

func TestInitSummary_ContainsAllSections(t *testing.T) {
	tree := []tui.TreeNode{
		{Name: "codectx.yml"},
		{Name: "docs/", Children: []tui.TreeNode{
			{Name: "foundation/"},
			{Name: "topics/"},
		}},
	}
	steps := []string{
		"Add foundation documentation",
		"Run codectx compile",
	}

	result := tui.InitSummary("my-project", tree, steps)

	if !strings.Contains(result, "my-project") {
		t.Error("expected project name in summary")
	}
	if !strings.Contains(result, "codectx.yml") {
		t.Error("expected tree content")
	}
	if !strings.Contains(result, "foundation/") {
		t.Error("expected nested tree content")
	}
	if !strings.Contains(result, "Add foundation documentation") {
		t.Error("expected first next step")
	}
	if !strings.Contains(result, "codectx compile") {
		t.Error("expected second next step")
	}
	if !strings.Contains(result, "Created:") {
		t.Error("expected 'Created:' section header")
	}
	if !strings.Contains(result, "Next steps:") {
		t.Error("expected 'Next steps:' section header")
	}
}

func TestInitSummary_NoNextSteps(t *testing.T) {
	tree := []tui.TreeNode{{Name: "file.txt"}}

	result := tui.InitSummary("test", tree, nil)

	if strings.Contains(result, "Next steps:") {
		t.Error("should not show next steps when none provided")
	}
}

func TestDetectedTool_Format(t *testing.T) {
	result := tui.DetectedTool("Claude Code", "v2.1.63")

	if !strings.Contains(result, "Claude Code") {
		t.Error("expected tool name")
	}
	if !strings.Contains(result, "v2.1.63") {
		t.Error("expected version")
	}
}

func TestNotDetectedTool_Format(t *testing.T) {
	result := tui.NotDetectedTool("Cursor")

	if !strings.Contains(result, "Cursor") {
		t.Error("expected tool name")
	}
}

func TestKeyValue_Format(t *testing.T) {
	result := tui.KeyValue("Model", "claude-sonnet-4")

	if !strings.Contains(result, "Model:") {
		t.Error("expected key with colon")
	}
	if !strings.Contains(result, "claude-sonnet-4") {
		t.Error("expected value")
	}
}
