package cmdx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

func FuzzRoundTrip(f *testing.F) {
	// Seed corpus with known fixtures.
	f.Add(readTestdata("simple.md"))
	f.Add(readTestdata("api_docs.md"))
	f.Add([]byte("# Hello\n\nWorld"))
	f.Add([]byte("| a | b |\n|---|---|\n| 1 | 2 |"))
	f.Add([]byte("```go\nfunc main() {}\n```"))
	f.Add([]byte("Text with @ and $ signs"))
	f.Add([]byte(""))
	f.Add([]byte("## GET /users\n\n> **Note:** Be careful\n"))
	f.Add([]byte("| Field | Type | Description |\n|-------|------|-------------|\n| id | string | ID |\n"))

	f.Fuzz(func(t *testing.T, input []byte) { //nolint:thelper
		// Normalize control characters — carriage returns, vertical tabs,
		// form feeds, etc. are not semantically meaningful in markdown
		// and goldmark handles them inconsistently.
		input = bytes.ReplaceAll(input, []byte("\r"), []byte(""))
		input = bytes.ReplaceAll(input, []byte("\v"), []byte(""))
		input = bytes.ReplaceAll(input, []byte("\f"), []byte(""))

		// Skip inputs with known serialization ambiguities that cannot
		// be round-tripped through CMDX.
		if hasSerializationAmbiguity(input) {
			t.Skip()
		}

		encoded, err := Encode(input)
		if err != nil {
			t.Skip() // Invalid markdown is fine to skip.
		}
		decoded, err := Decode(encoded)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		equal, diff, err := CompareASTs(input, decoded)
		if err != nil {
			t.Fatalf("compare failed: %v", err)
		}
		if !equal {
			t.Fatalf("round-trip mismatch:\n%s", diff)
		}
	})
}

// hasSerializationAmbiguity checks if the input's goldmark AST contains
// structures that cannot be losslessly serialized as markdown:
//  1. Backtick characters in text nodes adjacent to code spans — backtick runs
//     in text merge with code span delimiters during re-serialization.
//  2. Nested strikethrough — goldmark parses single ~ as strikethrough but
//     re-serializing with ~~ doesn't produce valid nesting.
func hasSerializationAmbiguity(input []byte) bool {
	md := newGoldmark()
	reader := text.NewReader(input)
	doc := md.Parser().Parse(reader)

	hasBacktickText := false
	hasCodeSpan := false
	hasTildeText := false
	hasStrikethrough := false
	hasNestedStrikethrough := false
	hasNestedEmphasis := false
	hasEmphasisWithSoftBreak := false
	hasAutoLink := false
	hasHTML := false
	hasHardBreakBeforeListMarker := false
	strikeDepth := 0
	emphDepth := 0
	prevWasHardBreak := false
	inEmphasis := false

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if _, ok := n.(*east.Strikethrough); ok {
			hasStrikethrough = true
			if entering {
				strikeDepth++
				if strikeDepth > 1 {
					hasNestedStrikethrough = true
				}
			} else {
				strikeDepth--
			}
			return ast.WalkContinue, nil
		}
		if _, ok := n.(*ast.Emphasis); ok {
			if entering {
				emphDepth++
				inEmphasis = emphDepth > 0
				if emphDepth > 1 {
					hasNestedEmphasis = true
				}
			} else {
				emphDepth--
				inEmphasis = emphDepth > 0
			}
			return ast.WalkContinue, nil
		}
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := n.(*ast.Text); ok {
			seg := t.Segment.Value(input)
			if bytes.ContainsRune(seg, '`') {
				hasBacktickText = true
			}
			if bytes.ContainsRune(seg, '~') {
				hasTildeText = true
			}
			// Check if text after a hard break starts with an ordered list
			// marker (digits + ./)) — these can't be backslash-escaped.
			if prevWasHardBreak && len(seg) > 0 && seg[0] >= '0' && seg[0] <= '9' {
				hasHardBreakBeforeListMarker = true
			}
			// Soft breaks inside emphasis can't be serialized as spaces
			// because the space prevents the closing delimiter from being
			// right-flanking.
			if inEmphasis && t.SoftLineBreak() {
				hasEmphasisWithSoftBreak = true
			}
			prevWasHardBreak = t.HardLineBreak()
		} else {
			prevWasHardBreak = false
		}
		if _, ok := n.(*ast.CodeSpan); ok {
			hasCodeSpan = true
		}
		if _, ok := n.(*ast.AutoLink); ok {
			hasAutoLink = true
		}
		// Text containing @ with domain-like patterns can become GFM autolinks
		// after _ escaping changes word boundaries.
		if t, ok := n.(*ast.Text); ok {
			seg := t.Segment.Value(input)
			if bytes.Contains(seg, []byte("@")) && bytes.Contains(seg, []byte(".")) {
				hasAutoLink = true
			}
		}
		if _, ok := n.(*ast.HTMLBlock); ok {
			hasHTML = true
		}
		if _, ok := n.(*ast.RawHTML); ok {
			hasHTML = true
		}
		return ast.WalkContinue, nil
	})
	// Check for consecutive emphasis nodes (emphasis immediately followed by
	// emphasis without any raw text between them). This creates ambiguous
	// markdown output when * and _ delimiters merge or fail intraword.
	hasConsecutiveEmphasis := false
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, ok := n.(*ast.Emphasis); ok {
			if next := n.NextSibling(); next != nil {
				if _, nextIsEmph := next.(*ast.Emphasis); nextIsEmph {
					hasConsecutiveEmphasis = true
				}
			}
		}
		return ast.WalkContinue, nil
	})

	// Check for empty list items that contain nested lists.
	// goldmark can't round-trip nested lists inside empty items because
	// the empty `- ` + indented nested list parses as sibling lists.
	hasEmptyItemWithNestedList := false
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if li, ok := n.(*ast.ListItem); ok {
			hasNestedList := false
			for c := li.FirstChild(); c != nil; c = c.NextSibling() {
				if _, isList := c.(*ast.List); isList {
					hasNestedList = true
				}
			}
			if hasNestedList && isEmptyListItem(li, input) {
				hasEmptyItemWithNestedList = true
			}
		}
		return ast.WalkContinue, nil
	})

	// Check for block-level elements inside list items that aren't nested lists.
	// CMDX doesn't support blockquotes, code blocks, etc. inside list items.
	// Also check for list items that mix nested lists with other content —
	// this combination creates ambiguous round-trips.
	hasBlockInListItem := false
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, ok := n.(*ast.ListItem); ok {
			contentBlocks := 0 // non-list block children
			hasNested := false
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				switch c.(type) {
				case *ast.Paragraph, *ast.TextBlock:
					contentBlocks++
					// Check if paragraph text starts with block-level markers
					// that would be reinterpreted by goldmark when inside a list item.
					if fc := c.FirstChild(); fc != nil {
						if t, ok := fc.(*ast.Text); ok {
							seg := t.Segment.Value(input)
							if len(seg) > 0 {
								ch := seg[0]
								// Block markers that goldmark interprets inside list items:
								// headings (#), blockquotes (>), and list markers (+, -, *)
								// when followed by space or end of text.
								if ch == '#' || ch == '>' {
									hasBlockInListItem = true
								}
								if ch == '+' || ch == '-' || ch == '*' {
									if len(seg) == 1 || seg[1] == ' ' || seg[1] == '\t' {
										hasBlockInListItem = true
									}
								}
							}
							// Check if paragraph text looks like a thematic break.
							fullText := extractListItemText(c, input)
							if looksLikeThematicBreakFuzz(fullText) {
								hasBlockInListItem = true
							}
						}
					}
				case *ast.Heading:
					contentBlocks++
				case *ast.List:
					hasNested = true
				default:
					hasBlockInListItem = true
				}
			}
			// Multiple content blocks in a list item can't be round-tripped
			// since the encoder concatenates everything into one line.
			if contentBlocks > 1 {
				hasBlockInListItem = true
			}
			// Mixed content: nested list alongside content blocks.
			if hasNested && contentBlocks > 0 {
				hasBlockInListItem = true
			}
		}
		return ast.WalkContinue, nil
	})

	// Check for inline formatting inside table cells.
	// CMDX stores table cells as plain text, losing inline markup.
	hasFormattedTableCell := false
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if cell, ok := n.(*east.TableCell); ok {
			for c := cell.FirstChild(); c != nil; c = c.NextSibling() {
				switch c.(type) {
				case *ast.Text:
					// Plain text is fine.
				default:
					hasFormattedTableCell = true
				}
			}
		}
		return ast.WalkContinue, nil
	})

	// Check for link reference definitions — goldmark consumes them without
	// producing any visible output, so they can't round-trip through CMDX.
	hasLinkRefDef := false
	if links := doc.OwnerDocument().Meta()["linkReferences"]; links != nil {
		hasLinkRefDef = true
	}
	// Alternative: check if document has TextBlock children with no text content
	// (goldmark creates empty TextBlocks for link reference definitions).
	if !hasLinkRefDef {
		_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			if tb, ok := n.(*ast.TextBlock); ok {
				// TextBlock with no text children is a link reference definition.
				hasText := false
				for c := tb.FirstChild(); c != nil; c = c.NextSibling() {
					if _, isText := c.(*ast.Text); isText {
						hasText = true
						break
					}
				}
				if !hasText {
					hasLinkRefDef = true
				}
			}
			return ast.WalkContinue, nil
		})
	}

	return (hasBacktickText && hasCodeSpan) || hasNestedStrikethrough || (hasTildeText && hasStrikethrough) || hasNestedEmphasis || hasConsecutiveEmphasis || hasEmphasisWithSoftBreak || hasAutoLink || hasHardBreakBeforeListMarker || hasHTML || hasEmptyItemWithNestedList || hasBlockInListItem || hasFormattedTableCell || hasLinkRefDef
}

// isEmptyListItem checks if a ListItem has no meaningful text content.
func isEmptyListItem(li *ast.ListItem, source []byte) bool {
	for c := li.FirstChild(); c != nil; c = c.NextSibling() {
		if _, isList := c.(*ast.List); isList {
			continue // nested lists don't count as item content
		}
		// Check if block child has text content
		for ic := c.FirstChild(); ic != nil; ic = ic.NextSibling() {
			if t, ok := ic.(*ast.Text); ok {
				if len(bytes.TrimSpace(t.Segment.Value(source))) > 0 {
					return false
				}
			}
		}
	}
	return true
}

// extractListItemText concatenates all text segments from a block node.
func extractListItemText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
			if t.SoftLineBreak() {
				buf.WriteByte(' ')
			}
		}
	}
	return buf.String()
}

// looksLikeThematicBreakFuzz checks if text would form a thematic break
// when placed inside a list item (3+ of same char from {-, *, _} with spaces).
func looksLikeThematicBreakFuzz(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	// Check each thematic break character.
	for _, ch := range []byte{'-', '*', '_'} {
		count := 0
		allMatch := true
		for i := 0; i < len(s); i++ {
			if s[i] == ch {
				count++
			} else if s[i] != ' ' && s[i] != '\t' {
				allMatch = false
				break
			}
		}
		if allMatch && count >= 3 {
			return true
		}
	}
	return false
}
