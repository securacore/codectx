package context

import (
	"fmt"
	"os"
	"strings"

	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/tokens"
)

// EntryResult holds assembly statistics for a single always_loaded entry.
type EntryResult struct {
	// Reference is the original always_loaded string.
	Reference string

	// Title is the display title used as the H2 heading in context.md.
	Title string

	// Tokens is the total token count for this entry.
	Tokens int

	// FileCount is the number of markdown files in this entry.
	FileCount int
}

// AssemblyResult holds the complete result of context assembly.
type AssemblyResult struct {
	// Content is the assembled markdown content for context.md
	// (excluding the metadata header — that is added by the writer).
	Content string

	// TotalTokens is the total token count of the assembled content.
	TotalTokens int

	// Budget is the token budget from configuration.
	Budget int

	// Utilization is the percentage of budget used (0-100+).
	Utilization float64

	// Entries holds per-entry statistics.
	Entries []EntryResult

	// Warnings are any issues encountered during assembly.
	Warnings []string
}

// Assemble processes resolved entries through the strip/normalize pipeline
// and concatenates them into a single context document.
//
// Each entry becomes an H2 section. Internal headings within entry files
// are shifted to H3+ to maintain consistent hierarchy. Files within an
// entry are concatenated in the order they were resolved.
//
// The encoding parameter is the tokenizer encoding name (e.g., "cl100k_base").
// The budget parameter is the max token budget; if the total exceeds it, a
// warning is emitted.
func Assemble(entries []ResolvedEntry, encoding string, budget int) (*AssemblyResult, error) {
	if len(entries) == 0 {
		return &AssemblyResult{Budget: budget}, nil
	}

	counter, err := tokens.New(encoding)
	if err != nil {
		return nil, fmt.Errorf("creating token counter: %w", err)
	}

	result := &AssemblyResult{
		Budget:  budget,
		Entries: make([]EntryResult, 0, len(entries)),
	}

	var sections []string

	for _, entry := range entries {
		entryContent, entryTokens, err := assembleEntry(entry, counter)
		if err != nil {
			return nil, fmt.Errorf("assembling %q: %w", entry.Reference, err)
		}

		sections = append(sections, entryContent)

		result.Entries = append(result.Entries, EntryResult{
			Reference: entry.Reference,
			Title:     entry.Title,
			Tokens:    entryTokens,
			FileCount: len(entry.Files),
		})

		result.TotalTokens += entryTokens
	}

	result.Content = strings.Join(sections, "\n\n")

	// Compute budget utilization.
	if budget > 0 {
		result.Utilization = float64(result.TotalTokens) / float64(budget) * 100.0

		if result.TotalTokens > budget {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("session context exceeds budget: %d / %d tokens (%.1f%%)",
					result.TotalTokens, budget, result.Utilization))

			// Identify largest entries.
			result.Warnings = append(result.Warnings, identifyLargestEntries(result.Entries)...)
		}
	}

	return result, nil
}

// assembleEntry processes a single resolved entry into markdown content.
// Returns the rendered content and its token count.
func assembleEntry(entry ResolvedEntry, counter *tokens.Counter) (string, int, error) {
	var parts []string

	// H2 heading for this entry.
	parts = append(parts, fmt.Sprintf("## %s", entry.Title))

	totalTokens := 0

	for _, f := range entry.Files {
		data, err := os.ReadFile(f.AbsPath)
		if err != nil {
			return "", 0, fmt.Errorf("reading %s: %w", f.RelPath, err)
		}

		// Parse and strip the markdown.
		doc := markdown.Parse(data)
		stripped := markdown.Strip(doc)

		// Count tokens on the stripped blocks.
		if err := tokens.CountBlocks(stripped, counter); err != nil {
			return "", 0, fmt.Errorf("counting tokens for %s: %w", f.RelPath, err)
		}

		totalTokens += stripped.TotalTokens

		// Render blocks as flowing prose with heading level shifts.
		// Entry title is H2, so internal headings start at H3.
		rendered := renderBlocks(stripped.Blocks, 2)
		if rendered != "" {
			parts = append(parts, rendered)
		}
	}

	return strings.Join(parts, "\n\n"), totalTokens, nil
}

// renderBlocks renders a slice of markdown blocks back into markdown text.
// The baseLevel parameter controls the heading level shift: internal headings
// are shifted so that the minimum heading level becomes baseLevel+1.
//
// For context assembly, baseLevel is 2 (the entry title is H2),
// so internal H1s become H3, H2s become H4, etc.
func renderBlocks(blocks []markdown.Block, baseLevel int) string {
	if len(blocks) == 0 {
		return ""
	}

	// Find the minimum heading level in these blocks.
	minLevel := 0
	for _, b := range blocks {
		if b.Type == markdown.BlockHeading {
			if minLevel == 0 || b.Level < minLevel {
				minLevel = b.Level
			}
		}
	}

	// Compute the shift needed: internal headings start at baseLevel+1.
	shift := 0
	if minLevel > 0 {
		shift = (baseLevel + 1) - minLevel
	}

	var parts []string

	for _, b := range blocks {
		switch b.Type {
		case markdown.BlockHeading:
			level := b.Level + shift
			if level < 1 {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			parts = append(parts, strings.Repeat("#", level)+" "+b.Content)

		case markdown.BlockCodeBlock:
			if b.Language != "" {
				parts = append(parts, "```"+b.Language+"\n"+b.Content+"\n```")
			} else {
				parts = append(parts, "```\n"+b.Content+"\n```")
			}

		default:
			// Paragraphs, lists, tables, blockquotes, etc. — content is already rendered.
			if b.Content != "" {
				parts = append(parts, b.Content)
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

// identifyLargestEntries returns warnings listing the top token consumers.
func identifyLargestEntries(entries []EntryResult) []string {
	if len(entries) <= 1 {
		return nil
	}

	// Find the largest entry.
	var largest EntryResult
	for _, e := range entries {
		if e.Tokens > largest.Tokens {
			largest = e
		}
	}

	return []string{
		fmt.Sprintf("largest entry: %q (%d tokens)", largest.Reference, largest.Tokens),
	}
}
