package query

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/tui"
)

// ShortHash truncates a full hash to 12 characters for display.
// If the hash has a "sha256:" prefix, that prefix is stripped first.
func ShortHash(hash string) string {
	h := strings.TrimPrefix(hash, "sha256:")
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

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

	if len(r.Unified) > 0 {
		// BM25F unified mode — single ranked list with count header.
		fmt.Fprintf(&b, "\n%s %s\n",
			tui.StyleBold.Render("Results"),
			tui.StyleMuted.Render(fmt.Sprintf("(%d, bm25f + rrf)", len(r.Unified))),
		)
		formatUnifiedEntries(&b, r.Unified)

		// Summary line: total tokens across all results.
		totalTokens := 0
		for _, e := range r.Unified {
			totalTokens += e.Tokens
		}
		fmt.Fprintf(&b, "\n%s%s\n",
			tui.Indent(1),
			tui.KeyValue("Total", fmt.Sprintf("%s tokens across %d results",
				tui.FormatNumber(totalTokens),
				len(r.Unified),
			)),
		)
	} else {
		// BM25 mode — separate per-type lists.
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

	if len(r.Instructions) == 0 && len(r.Reasoning) == 0 && len(r.System) == 0 && len(r.Unified) == 0 {
		fmt.Fprintf(&b, "\n%s No results found.\n", tui.Warning())
	} else {
		fmt.Fprintf(&b, "%s\n%s\n",
			tui.StyleMuted.Render("Run \"codectx generate\" with the top chunk IDs above to read their full content before proceeding."),
			tui.StyleMuted.Render("Try additional queries with different terms to explore related areas before deciding."),
		)
	}

	b.WriteString("\n")
	return b.String()
}

// PromptSummary holds metadata for the combined prompt output header and footer.
type PromptSummary struct {
	RawQuery      string // original search string
	ExpandedQuery string // post-expansion display string
	SelectedCount int    // number of chunks auto-selected
	SelectedTotal int    // total tokens of selected chunks
	QueryTotal    int    // total number of query results
	Budget        int    // computed token budget
	BudgetFormula string // human-readable formula, e.g. "450 × 3 × 1.0"
}

// FormatPromptHeader renders the header shown before the generated document
// content in prompt output.
//
// Output format (styled):
//
//	-> Prompt: "How do I create a React component"
//	   Expanded: react compon ui widget
//	   Selected: 3 chunks (1,247 tokens) from 8 query results
//	   Budget: 1,350 tokens (450 × 3 × 1.0)
func FormatPromptHeader(s *PromptSummary) string {
	var b strings.Builder

	// Header: mirrors the query command's "-> Results for:" pattern.
	fmt.Fprintf(&b, "\n%s Prompt: %s\n",
		tui.Arrow(),
		tui.StyleBold.Render(fmt.Sprintf("%q", s.RawQuery)),
	)

	// Expanded query: mirrors FormatQueryResults — only shown when different.
	if s.ExpandedQuery != "" && s.ExpandedQuery != s.RawQuery {
		fmt.Fprintf(&b, "%s%s\n",
			tui.Indent(1),
			tui.KeyValue("Expanded", tui.StyleMuted.Render(s.ExpandedQuery)),
		)
	}

	// Selection summary.
	fmt.Fprintf(&b, "%s%s\n",
		tui.Indent(1),
		tui.KeyValue("Selected", fmt.Sprintf("%s chunks (%s tokens) from %s query results",
			tui.StyleBold.Render(fmt.Sprintf("%d", s.SelectedCount)),
			tui.FormatNumber(s.SelectedTotal),
			tui.FormatNumber(s.QueryTotal),
		)),
	)

	// Budget with formula.
	fmt.Fprintf(&b, "%s%s\n",
		tui.Indent(1),
		tui.KeyValue("Budget", fmt.Sprintf("%s tokens (%s)",
			tui.FormatNumber(s.Budget),
			tui.StyleMuted.Render(s.BudgetFormula),
		)),
	)

	b.WriteString("\n")
	return b.String()
}

// FormatPromptNoResults renders the no-results message for the prompt command.
// Mirrors the FormatQueryResults empty-results pattern.
func FormatPromptNoResults(rawQuery string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s No results found for %s\n\n",
		tui.Warning(),
		tui.StyleBold.Render(fmt.Sprintf("%q", rawQuery)),
	)
	return b.String()
}

// FormatPromptFooter renders the footer shown after the generated document
// content in prompt output. Reuses FormatGenerateSummary for the generate
// metadata and related chunks.
func FormatPromptFooter(r *GenerateResult, historyPath, filePath string, cacheHit bool) string {
	return FormatGenerateSummary(r, historyPath, filePath, cacheHit)
}

// formatUnifiedEntries writes numbered entries from the RRF-fused result list,
// showing the index sources that contributed to each result.
func formatUnifiedEntries(b *strings.Builder, entries []ResultEntry) {
	formatEntriesWithPrecision(b, entries, "%.4f", true)
}

// formatEntries writes numbered result entries to the builder with TUI styling.
func formatEntries(b *strings.Builder, entries []ResultEntry) {
	formatEntriesWithPrecision(b, entries, "%.2f", false)
}

// formatEntriesWithPrecision writes numbered result entries with configurable
// score precision and optional index source annotations.
func formatEntriesWithPrecision(b *strings.Builder, entries []ResultEntry, scoreFmt string, showSources bool) {
	for i, entry := range entries {
		fmt.Fprintf(b, "%s%d. %s %s \u2014 %s\n",
			tui.Indent(1),
			i+1,
			tui.StyleMuted.Render(fmt.Sprintf("[score: "+scoreFmt+"]", entry.Score)),
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
		if showSources {
			if sources := formatSourceAnnotation(entry.IndexSources); sources != "" {
				fmt.Fprintf(b, "%s%s\n",
					tui.Indent(2),
					tui.KeyValue("Indexes", tui.StyleMuted.Render(sources)),
				)
			}
		}
	}
}

// formatSourceAnnotation builds a human-readable string showing which
// indexes contributed to a result and at what rank.
func formatSourceAnnotation(sources map[string]int) string {
	if len(sources) == 0 {
		return ""
	}
	order := []string{"objects", "specs", "system"}
	var parts []string
	for _, name := range order {
		if rank, ok := sources[name]; ok {
			parts = append(parts, fmt.Sprintf("%s:#%d", name, rank))
		}
	}
	return strings.Join(parts, ", ")
}

// FormatGenerateSummary renders the summary for a generate operation with
// full TUI styling. historyPath is the path to the saved history document
// (always shown). filePath is the --file output path (empty when stdout mode).
// cacheHit appends "[from cache]" to the header when true.
//
// Output format:
//
//	✓ Generated (1,772 tokens, hash: a1b2c3d4e5f6)
//	  History: docs/.codectx/history/docs/1741532400000000000.a1b2c3d4e5f6.md
//	  Written to: /path/to/output.md   (only when --file is used)
//	  Contains: obj:a1b2c3.03, obj:a1b2c3.04, spec:f7g8h9.02
//
//	  Related chunks not included:
//	    obj:a1b2c3.02 — Authentication > JWT Tokens > Token Structure (488 tokens)
func FormatGenerateSummary(r *GenerateResult, historyPath, filePath string, cacheHit bool) string {
	var b strings.Builder

	// Build the header: tokens + short hash.
	shortHash := ShortHash(r.ContentHash)

	cacheTag := ""
	if cacheHit {
		cacheTag = " [from cache]"
	}

	fmt.Fprintf(&b, "\n%s %s\n\n",
		tui.Success(),
		tui.StyleBold.Render(fmt.Sprintf("Generated (%s tokens, hash: %s)%s",
			tui.FormatNumber(r.TotalTokens),
			shortHash,
			cacheTag,
		)),
	)

	if filePath != "" {
		fmt.Fprintf(&b, "%s%s\n",
			tui.Indent(1),
			tui.KeyValue("Written to", tui.StylePath.Render(filePath)),
		)
	}

	if historyPath != "" {
		fmt.Fprintf(&b, "%s%s\n",
			tui.Indent(1),
			tui.KeyValue("History", tui.StylePath.Render(historyPath)),
		)
	}

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
