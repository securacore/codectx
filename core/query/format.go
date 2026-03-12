package query

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/tui"
)

// FormatQueryResults renders a QueryResult into the spec-defined output format
// with full TUI styling.
//
// Output format:
//
//	-> Results for: "jwt refresh token validation"
//
//	Instructions:
//	1. [score: 8.42] obj:a1b2c3.03 — Authentication > JWT Tokens > Refresh Flow
//	   Source: docs/topics/auth/jwt.md (chunk 3/7, 462 tokens)
//
//	Reasoning:
//	...
//
//	Related chunks (adjacent to top results, not scored):
//	  obj:a1b2c3.02 — Authentication > JWT Tokens > Token Structure
func FormatQueryResults(r *QueryResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s Results for: %s\n",
		tui.Arrow(),
		tui.StyleBold.Render(fmt.Sprintf("%q", r.RawQuery)),
	)

	// Show expanded query if expansion added any tokens.
	if r.ExpandedQuery != "" && r.ExpandedQuery != r.RawQuery {
		fmt.Fprintf(&b, "%sExpanded: %s\n",
			tui.Indent(1),
			tui.StyleMuted.Render(r.ExpandedQuery),
		)
	}

	if len(r.Instructions) > 0 {
		fmt.Fprintf(&b, "\n%s\n", tui.StyleBold.Render("Instructions:"))
		formatEntries(&b, r.Instructions)
	}

	if len(r.Reasoning) > 0 {
		fmt.Fprintf(&b, "\n%s\n", tui.StyleBold.Render("Reasoning:"))
		formatEntries(&b, r.Reasoning)
	}

	if len(r.System) > 0 {
		fmt.Fprintf(&b, "\n%s\n", tui.StyleBold.Render("System:"))
		formatEntries(&b, r.System)
	}

	if len(r.Related) > 0 {
		fmt.Fprintf(&b, "\n%s\n", tui.StyleMuted.Render("Related chunks (adjacent to top results, not scored):"))
		for _, rel := range r.Related {
			fmt.Fprintf(&b, "%s%s \u2014 %s %s\n",
				tui.Indent(1),
				tui.StyleAccent.Render(rel.ChunkID),
				rel.Heading,
				tui.StyleMuted.Render(fmt.Sprintf("(%s tokens)", tui.FormatNumber(rel.Tokens))),
			)
		}
	}

	if len(r.Instructions) == 0 && len(r.Reasoning) == 0 && len(r.System) == 0 {
		fmt.Fprintf(&b, "\n%s No results found.\n", tui.Warning())
	}

	b.WriteString("\n")
	return b.String()
}

// formatEntries writes numbered result entries to the builder with TUI styling.
func formatEntries(b *strings.Builder, entries []ResultEntry) {
	for i, entry := range entries {
		fmt.Fprintf(b, "%s%d. %s %s \u2014 %s\n",
			tui.Indent(1),
			i+1,
			tui.StyleMuted.Render(fmt.Sprintf("[score: %.2f]", entry.Score)),
			tui.StyleAccent.Render(entry.ChunkID),
			entry.Heading,
		)
		fmt.Fprintf(b, "%s%s\n",
			tui.Indent(2),
			tui.KeyValue("Source", fmt.Sprintf("%s (chunk %d/%d, %s tokens)",
				tui.StylePath.Render(entry.Source),
				entry.Sequence, entry.TotalInFile,
				tui.FormatNumber(entry.Tokens),
			)),
		)
	}
}

// FormatGenerateSummary renders the stdout summary for a generate operation
// with full TUI styling.
//
// Output format:
//
//	✓ Generated: /tmp/codectx/auth-jwt.1741532400.md (1,772 tokens)
//	  Contains: obj:a1b2c3.03, obj:a1b2c3.04, spec:f7g8h9.02
//
//	  Related chunks not included:
//	    obj:a1b2c3.02 — Authentication > JWT Tokens > Token Structure (488 tokens)
func FormatGenerateSummary(r *GenerateResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s %s\n",
		tui.Success(),
		tui.KeyValue("Generated", fmt.Sprintf("%s (%s tokens)",
			tui.StylePath.Render(r.FilePath),
			tui.FormatNumber(r.TotalTokens),
		)),
	)

	styledIDs := make([]string, len(r.ChunkIDs))
	for i, id := range r.ChunkIDs {
		styledIDs[i] = tui.StyleAccent.Render(id)
	}
	fmt.Fprintf(&b, "%s%s\n",
		tui.Indent(1),
		tui.KeyValue("Contains", strings.Join(styledIDs, ", ")),
	)

	if len(r.Related) > 0 {
		fmt.Fprintf(&b, "\n%s%s\n",
			tui.Indent(1),
			tui.StyleMuted.Render("Related chunks not included:"),
		)
		for _, rel := range r.Related {
			fmt.Fprintf(&b, "%s%s \u2014 %s %s\n",
				tui.Indent(2),
				tui.StyleAccent.Render(rel.ChunkID),
				rel.Heading,
				tui.StyleMuted.Render(fmt.Sprintf("(%s tokens)", tui.FormatNumber(rel.Tokens))),
			)
		}
	}

	b.WriteString("\n")
	return b.String()
}
