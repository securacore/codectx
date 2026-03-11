package tui

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/markdown"
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
	for _, line := range markdown.SplitLines(treeStr) {
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
