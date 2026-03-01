// Package normalize provides the terminology normalization directive and
// prompt assembly for AI-driven documentation normalization.
package normalize

import (
	_ "embed"
	"strings"
)

//go:embed content/directive.md
var directive string

// Directive returns the raw embedded normalization directive.
func Directive() string {
	return directive
}

// AssemblePrompt builds the full system prompt by combining the embedded
// directive with dynamic context about the documentation corpus.
func AssemblePrompt(docsDir, manifestSummary string) string {
	var b strings.Builder
	b.WriteString(directive)

	b.WriteString("\n\n---\n\n")
	b.WriteString("## Documentation Directory\n\n")
	b.WriteString("The documentation to normalize is in: `")
	b.WriteString(docsDir)
	b.WriteString("`\n")

	if manifestSummary != "" {
		b.WriteString("\n\n---\n\n")
		b.WriteString("## Documentation Map\n\n")
		b.WriteString("The following documentation entries exist in this project. ")
		b.WriteString("Use this to understand the corpus structure before reading files.\n\n")
		b.WriteString(manifestSummary)
	}

	return b.String()
}
