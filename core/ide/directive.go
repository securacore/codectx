package ide

import (
	_ "embed"
	"strings"
)

//go:embed content/directive.md
var directive string

// AssemblePrompt builds the full system prompt by combining the embedded
// directive with dynamic context (package mode, manifest summary, and preferences).
func AssemblePrompt(manifestSummary, prefsContext, packageContext string) string {
	var b strings.Builder
	b.WriteString(directive)

	if packageContext != "" {
		b.WriteString("\n\n---\n\n")
		b.WriteString("## Package Authoring\n\n")
		b.WriteString(packageContext)
	}

	if manifestSummary != "" {
		b.WriteString("\n\n---\n\n")
		b.WriteString("## Existing Documentation\n\n")
		b.WriteString("The following documentation already exists in this project. ")
		b.WriteString("Use this to avoid duplication, identify dependencies, and ensure cross-references resolve correctly.\n\n")
		b.WriteString(manifestSummary)
	}

	if prefsContext != "" {
		b.WriteString("\n\n---\n\n")
		b.WriteString("## Project Preferences\n\n")
		b.WriteString(prefsContext)
	}

	return b.String()
}
