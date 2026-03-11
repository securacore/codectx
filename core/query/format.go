package query

import (
	"fmt"
	"strings"
)

// FormatQueryResults renders a QueryResult into the spec-defined output format.
//
// Output format:
//
//	Results for: "jwt refresh token validation"
//
//	Instructions:
//	1. [score: 8.42] obj:a1b2c3.03 - Authentication > JWT Tokens > Refresh Flow
//	   Source: docs/topics/auth/jwt.md (chunk 3/7, 462 tokens)
//
//	Reasoning:
//	...
//
//	Related chunks (adjacent to top results, not scored):
//	  obj:a1b2c3.02 - Authentication > JWT Tokens > Token Structure
func FormatQueryResults(r *QueryResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Results for: %q\n", r.RawQuery)

	if len(r.Instructions) > 0 {
		b.WriteString("\nInstructions:\n")
		formatEntries(&b, r.Instructions)
	}

	if len(r.Reasoning) > 0 {
		b.WriteString("\nReasoning:\n")
		formatEntries(&b, r.Reasoning)
	}

	if len(r.System) > 0 {
		b.WriteString("\nSystem:\n")
		formatEntries(&b, r.System)
	}

	if len(r.Related) > 0 {
		b.WriteString("\nRelated chunks (adjacent to top results, not scored):\n")
		for _, rel := range r.Related {
			fmt.Fprintf(&b, "  %s \u2014 %s (%d tokens)\n",
				rel.ChunkID, rel.Heading, rel.Tokens)
		}
	}

	if len(r.Instructions) == 0 && len(r.Reasoning) == 0 && len(r.System) == 0 {
		b.WriteString("\nNo results found.\n")
	}

	return b.String()
}

// formatEntries writes numbered result entries to the builder.
func formatEntries(b *strings.Builder, entries []ResultEntry) {
	for i, entry := range entries {
		fmt.Fprintf(b, "%d. [score: %.2f] %s \u2014 %s\n",
			i+1, entry.Score, entry.ChunkID, entry.Heading)
		fmt.Fprintf(b, "   Source: %s (chunk %d/%d, %d tokens)\n",
			entry.Source, entry.Sequence, entry.TotalInFile, entry.Tokens)
	}
}

// FormatGenerateSummary renders the stdout summary for a generate operation.
//
// Output format:
//
//	Generated: /tmp/codectx/auth-jwt.1741532400.md (1,772 tokens)
//	Contains: obj:a1b2c3.03, obj:a1b2c3.04, spec:f7g8h9.02
//
//	Related chunks not included:
//	  obj:a1b2c3.02 - Authentication > JWT Tokens > Token Structure (488 tokens)
func FormatGenerateSummary(r *GenerateResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Generated: %s (%d tokens)\n", r.FilePath, r.TotalTokens)
	fmt.Fprintf(&b, "Contains: %s\n", strings.Join(r.ChunkIDs, ", "))

	if len(r.Related) > 0 {
		b.WriteString("\nRelated chunks not included:\n")
		for _, rel := range r.Related {
			fmt.Fprintf(&b, "  %s \u2014 %s (%d tokens)\n",
				rel.ChunkID, rel.Heading, rel.Tokens)
		}
	}

	return b.String()
}
