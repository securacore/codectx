package tui

import (
	"fmt"
	"strings"
)

// TreeNode represents a node in a directory tree for display purposes.
type TreeNode struct {
	// Name is the display name of this node (e.g., "docs/", "codectx.yml").
	Name string

	// Children are sub-nodes. Empty for leaf nodes.
	Children []TreeNode
}

// RenderTree formats a list of tree nodes as an indented directory tree
// with box-drawing characters. Example output:
//
//	docs/
//	├── codectx.yml
//	├── .codectx/
//	│   ├── ai.yml
//	│   └── preferences.yml
//	├── foundation/
//	└── topics/
func RenderTree(nodes []TreeNode) string {
	var b strings.Builder
	renderNodes(&b, nodes, "")
	return b.String()
}

// renderNodes recursively renders tree nodes with proper box-drawing prefixes.
func renderNodes(b *strings.Builder, nodes []TreeNode, prefix string) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		connector := TreeBranch
		if isLast {
			connector = TreeCorner
		}

		fmt.Fprintf(b, "%s%s%s\n", prefix, connector, StylePath.Render(node.Name))

		if len(node.Children) > 0 {
			childPrefix := prefix + TreePipe
			if isLast {
				childPrefix = prefix + TreeSpace
			}
			renderNodes(b, node.Children, childPrefix)
		}
	}
}

// InitSummary formats the post-initialization summary with a success header,
// directory tree, and next steps.
func InitSummary(projectName string, tree []TreeNode, nextSteps []string) string {
	var b strings.Builder

	// Success header.
	fmt.Fprintf(&b, "\n%s %s\n\n",
		Success(),
		StyleBold.Render(fmt.Sprintf("Project initialized: %s", projectName)),
	)

	// Directory tree.
	fmt.Fprintf(&b, "%s%s\n\n", Indent(1), StyleMuted.Render("Created:"))
	treeStr := RenderTree(tree)
	for _, line := range strings.Split(strings.TrimRight(treeStr, "\n"), "\n") {
		fmt.Fprintf(&b, "%s%s\n", Indent(1), line)
	}

	// Next steps.
	if len(nextSteps) > 0 {
		fmt.Fprintf(&b, "\n%s%s\n\n", Indent(1), StyleMuted.Render("Next steps:"))
		for i, step := range nextSteps {
			fmt.Fprintf(&b, "%s%s %s\n", Indent(2), StyleMuted.Render(fmt.Sprintf("%d.", i+1)), step)
		}
	}

	b.WriteString("\n")
	return b.String()
}

// DetectedTool formats a single detected tool line for the detection summary.
func DetectedTool(name, version string) string {
	return fmt.Sprintf("%s %s %s",
		Success(),
		StyleBold.Render(name),
		StyleMuted.Render(version),
	)
}

// KeyValue formats a key-value pair with the key in muted and value in default color.
func KeyValue(key, value string) string {
	return fmt.Sprintf("%s %s", StyleMuted.Render(key+":"), value)
}

// FormatBudget formats a token count against a budget as "X / Y tokens (Z%)".
func FormatBudget(used, total int) string {
	if total <= 0 {
		return fmt.Sprintf("%s tokens", FormatNumber(used))
	}
	utilization := float64(used) / float64(total) * 100.0
	return fmt.Sprintf("%s / %s tokens (%.1f%%)",
		FormatNumber(used), FormatNumber(total), utilization)
}

// FormatNumber adds comma separators to large numbers for display.
// E.g., 1438 -> "1,438", 42 -> "42".
func FormatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

// FormatDuration formats seconds into a human-readable duration string.
// E.g., 0.099 -> "99ms", 2.3 -> "2.3s", 75.2 -> "1m15.2s".
func FormatDuration(seconds float64) string {
	if seconds < 1.0 {
		return fmt.Sprintf("%.0fms", seconds*1000)
	}
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	mins := int(seconds) / 60
	secs := seconds - float64(mins*60)
	return fmt.Sprintf("%dm%.1fs", mins, secs)
}
