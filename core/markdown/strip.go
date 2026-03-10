package markdown

import (
	"strings"
)

// Strip takes a Document and returns a new Document with human-formatting
// overhead removed. The source bytes are preserved; only the Blocks slice
// is modified.
//
// Operations performed:
//   - Remove decorative thematic breaks (horizontal rules)
//   - Remove HTML comments
//   - Normalize heading levels (shift so minimum becomes H1)
//   - Strip emphasis/bold from heading text (heading level provides emphasis)
//   - Recompute heading hierarchy after normalization
//
// Operations deliberately NOT performed (conservative approach):
//   - Emphasis within prose paragraphs is preserved (may carry semantic weight)
//   - Definition-pattern bold in list items is preserved ("**Term.** Description")
//   - Code block content is never modified
//   - Link text is preserved (link destinations are already stripped by the parser)
//
// Excessive blank lines are implicitly handled by block extraction — the parser
// only produces semantic blocks, so whitespace between them is irrelevant.
func Strip(doc *Document) *Document {
	if doc == nil {
		return nil
	}
	result := &Document{
		Source: doc.Source,
	}

	// Compute the heading level shift needed for normalization.
	levelDelta := 0
	if doc.MinLevel > 1 {
		levelDelta = doc.MinLevel - 1
	}

	// Heading hierarchy tracker for recomputing after normalization.
	heading := make([]string, 7)
	pos := 0

	for _, block := range doc.Blocks {
		// Skip decorative thematic breaks.
		if block.Type == BlockThematicBreak {
			continue
		}

		// Skip HTML comments.
		if block.Type == BlockHTMLBlock && isHTMLCommentContent(block.Content) {
			continue
		}

		b := Block{
			Type:     block.Type,
			Content:  block.Content,
			Level:    block.Level,
			Language: block.Language,
			Position: pos,
		}

		// Normalize heading levels.
		if b.Type == BlockHeading && levelDelta > 0 {
			b.Level -= levelDelta
			if b.Level < 1 {
				b.Level = 1
			}
		}

		// Note: heading emphasis markers are already stripped by the goldmark
		// AST parser during inline text rendering — no additional stripping needed.

		// Recompute heading hierarchy.
		if b.Type == BlockHeading {
			heading[b.Level] = b.Content
			for i := b.Level + 1; i <= 6; i++ {
				heading[i] = ""
			}
		}
		b.Heading = headingSnapshot(heading)

		result.Blocks = append(result.Blocks, b)
		pos++
	}

	result.MinLevel = computeMinLevel(result.Blocks)
	return result
}

// isHTMLCommentContent checks if an HTML block's content is an HTML comment.
func isHTMLCommentContent(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "<!--")
}
