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

	result := tui.RenderTreeForTest(nodes)

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

	result := tui.RenderTreeForTest(nodes)

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

	result := tui.RenderTreeForTest(nodes)

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

	result := tui.RenderTreeForTest(nodes)

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
	result := tui.RenderTreeForTest(nil)
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

func TestKeyValue_Format(t *testing.T) {
	result := tui.KeyValue("Model", "claude-sonnet-4")

	if !strings.Contains(result, "Model:") {
		t.Error("expected key with colon")
	}
	if !strings.Contains(result, "claude-sonnet-4") {
		t.Error("expected value")
	}
}

// ---------------------------------------------------------------------------
// FormatNumber
// ---------------------------------------------------------------------------

func TestFormatNumber_Small(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{999, "999"},
	}
	for _, tt := range tests {
		if got := tui.FormatNumber(tt.n); got != tt.want {
			t.Errorf("FormatNumber(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatNumber_Thousands(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{1000, "1,000"},
		{1438, "1,438"},
		{9999, "9,999"},
		{10000, "10,000"},
		{100000, "100,000"},
		{1000000, "1,000,000"},
		{12500, "12,500"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		if got := tui.FormatNumber(tt.n); got != tt.want {
			t.Errorf("FormatNumber(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// FormatDuration
// ---------------------------------------------------------------------------

func TestFormatDuration_Milliseconds(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{0.001, "1ms"},
		{0.05, "50ms"},
		{0.099, "99ms"},
		{0.5, "500ms"},
		{0.999, "999ms"},
	}
	for _, tt := range tests {
		if got := tui.FormatDuration(tt.seconds); got != tt.want {
			t.Errorf("FormatDuration(%f) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestFormatDuration_Seconds(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{1.0, "1.0s"},
		{2.3, "2.3s"},
		{59.9, "59.9s"},
	}
	for _, tt := range tests {
		if got := tui.FormatDuration(tt.seconds); got != tt.want {
			t.Errorf("FormatDuration(%f) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// FormatBudget
// ---------------------------------------------------------------------------

func TestFormatBudget_Normal(t *testing.T) {
	got := tui.FormatBudget(28450, 30000)
	if !strings.Contains(got, "28,450") {
		t.Errorf("FormatBudget missing used count: %q", got)
	}
	if !strings.Contains(got, "30,000") {
		t.Errorf("FormatBudget missing total count: %q", got)
	}
	if !strings.Contains(got, "94.8%") {
		t.Errorf("FormatBudget missing utilization: %q", got)
	}
}

func TestFormatBudget_Exceeded(t *testing.T) {
	got := tui.FormatBudget(35000, 30000)
	if !strings.Contains(got, "116.7%") {
		t.Errorf("FormatBudget should show >100%% utilization: %q", got)
	}
}

func TestFormatBudget_ZeroTotal(t *testing.T) {
	got := tui.FormatBudget(5000, 0)
	if !strings.Contains(got, "5,000 tokens") {
		t.Errorf("FormatBudget with zero total: %q", got)
	}
}

func TestFormatBudget_ZeroUsed(t *testing.T) {
	got := tui.FormatBudget(0, 30000)
	if !strings.Contains(got, "0 / 30,000") {
		t.Errorf("FormatBudget with zero used: %q", got)
	}
	if !strings.Contains(got, "0.0%") {
		t.Errorf("FormatBudget should show 0.0%%: %q", got)
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{60.0, "1m0.0s"},
		{75.2, "1m15.2s"},
		{120.0, "2m0.0s"},
		{185.5, "3m5.5s"},
	}
	for _, tt := range tests {
		if got := tui.FormatDuration(tt.seconds); got != tt.want {
			t.Errorf("FormatDuration(%f) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestFormatNumber_Negative(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{-1, "-1"},
		{-42, "-42"},
		{-999, "-999"},
		{-1000, "-1,000"},
		{-1234567, "-1,234,567"},
	}
	for _, tt := range tests {
		got := tui.FormatNumber(tt.n)
		if got != tt.want {
			t.Errorf("FormatNumber(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatDuration_Zero(t *testing.T) {
	got := tui.FormatDuration(0)
	if got != "0ms" {
		t.Errorf("FormatDuration(0) = %q, want %q", got, "0ms")
	}
}

func TestFormatDuration_ExactlyOneSecond(t *testing.T) {
	got := tui.FormatDuration(1.0)
	if got != "1.0s" {
		t.Errorf("FormatDuration(1.0) = %q, want %q", got, "1.0s")
	}
}

func TestFormatBudget_NegativeTotal(t *testing.T) {
	// Negative total should be treated the same as zero (no percentage).
	got := tui.FormatBudget(5000, -1)
	if !strings.Contains(got, "5,000 tokens") {
		t.Errorf("FormatBudget with negative total: %q", got)
	}
}
