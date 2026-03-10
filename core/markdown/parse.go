// Package markdown provides the parsing, stripping, and normalization layer
// for the codectx compilation pipeline. It transforms raw markdown files into
// ordered sequences of semantic blocks — the units that flow into chunking,
// indexing, and taxonomy extraction in later stages.
//
// The package uses goldmark (CommonMark + GFM extensions) for parsing and
// operates entirely on the AST — it never renders back to HTML. The key
// output is a Document containing an ordered []Block with heading hierarchy
// context, ready for the chunker to consume.
package markdown

import (
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	// GFM extension for tables, strikethrough, task lists.
	"github.com/yuin/goldmark/extension"
)

// md is the shared goldmark instance configured with GFM extensions.
// It is safe for concurrent use.
var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
)

// mdParser is the shared parser extracted from the goldmark instance.
var mdParser = md.Parser()

// parseAST parses markdown source bytes into a goldmark AST Document node.
func parseAST(source []byte) ast.Node {
	reader := text.NewReader(source)
	return mdParser.Parse(reader)
}

// renderInlineText extracts the plain text content from an inline node tree.
// It walks all child inline nodes recursively, concatenating text segments.
//
// Handling by node type:
//   - Text nodes: extract content from source via segment
//   - Emphasis/Bold: include child text (markdown syntax is already stripped by AST)
//   - CodeSpan: wrap in backticks to preserve code identity
//   - Link: render link text only (destination is metadata, not content)
//   - Image: render alt text
//   - Soft line break: space
//   - Hard line break: newline
//   - AutoLink: render the URL text
func renderInlineText(node ast.Node, source []byte) string {
	var b strings.Builder
	renderInlineInto(&b, node, source)
	return b.String()
}

// renderInlineInto recursively writes inline text content into the builder.
func renderInlineInto(b *strings.Builder, node ast.Node, source []byte) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			b.Write(n.Segment.Value(source))
			if n.SoftLineBreak() {
				b.WriteByte(' ')
			}
			if n.HardLineBreak() {
				b.WriteByte('\n')
			}

		case *ast.CodeSpan:
			// Preserve code spans with backticks — code identifiers matter.
			b.WriteByte('`')
			renderCodeSpanText(b, n, source)
			b.WriteByte('`')

		case *ast.Emphasis:
			// Walk children to get text (emphasis markers are syntax, not in the AST text).
			renderInlineInto(b, n, source)

		case *ast.Link:
			// Render link text content only.
			renderInlineInto(b, n, source)

		case *ast.AutoLink:
			b.Write(n.URL(source))

		case *ast.Image:
			// Render alt text.
			renderInlineInto(b, n, source)

		case *ast.RawHTML:
			// Raw inline HTML — include as-is from source segments.
			for i := 0; i < n.Segments.Len(); i++ {
				seg := n.Segments.At(i)
				b.Write(seg.Value(source))
			}

		default:
			// For any other inline nodes, recurse into children.
			if child.HasChildren() {
				renderInlineInto(b, child, source)
			}
		}
	}
}

// renderCodeSpanText extracts the text content from a CodeSpan node.
func renderCodeSpanText(b *strings.Builder, node *ast.CodeSpan, source []byte) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			b.Write(t.Segment.Value(source))
			if t.SoftLineBreak() {
				b.WriteByte(' ')
			}
		}
	}
}

// renderBlockLines extracts the raw text content from a block node's line segments.
// Used for code blocks and HTML blocks that store content in Lines().
func renderBlockLines(node ast.Node, source []byte) string {
	lines := node.Lines()
	if lines == nil || lines.Len() == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	return b.String()
}

// nodeText returns the inline text content of a block node by rendering
// all its inline children. For block nodes that store content in Lines()
// (like code blocks), use renderBlockLines instead.
func nodeText(node ast.Node, source []byte) string {
	return renderInlineText(node, source)
}

// renderTableText extracts a plain text representation of a GFM table.
// Preserves the tabular structure using pipe-delimited format.
func renderTableText(node *east.Table, source []byte) string {
	var b strings.Builder

	for row := node.FirstChild(); row != nil; row = row.NextSibling() {
		first := true
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if !first {
				b.WriteString(" | ")
			}
			first = false
			b.WriteString(renderInlineText(cell, source))
		}
		b.WriteByte('\n')
	}

	return b.String()
}

// renderListText extracts a plain text representation of a list.
// Preserves list structure with appropriate markers.
func renderListText(node *ast.List, source []byte) string {
	var b strings.Builder
	itemNum := node.Start
	if itemNum == 0 && node.IsOrdered() {
		itemNum = 1
	}

	for item := node.FirstChild(); item != nil; item = item.NextSibling() {
		if node.IsOrdered() {
			b.WriteString(formatOrderedMarker(itemNum, node.Marker))
			itemNum++
		} else {
			b.WriteByte(node.Marker)
			b.WriteByte(' ')
		}

		// Render item content (may contain multiple block children).
		renderListItemContent(&b, item, source, node.IsOrdered())
	}

	return b.String()
}

// formatOrderedMarker produces "1. " or "1) " depending on the marker byte.
func formatOrderedMarker(num int, marker byte) string {
	return strconv.Itoa(num) + string(marker) + " "
}

// renderListItemContent renders the content blocks within a list item.
func renderListItemContent(b *strings.Builder, item ast.Node, source []byte, ordered bool) {
	firstBlock := true
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		if !firstBlock {
			// Continuation blocks in a list item get indented.
			b.WriteString("  ")
		}

		switch n := child.(type) {
		case *ast.Paragraph:
			b.WriteString(renderInlineText(n, source))
			b.WriteByte('\n')

		case *ast.FencedCodeBlock:
			if !firstBlock {
				b.WriteByte('\n')
			}
			lang := string(n.Language(source))
			b.WriteString("  ```")
			if lang != "" {
				b.WriteString(lang)
			}
			b.WriteByte('\n')
			lines := renderBlockLines(n, source)
			for _, line := range strings.Split(strings.TrimRight(lines, "\n"), "\n") {
				b.WriteString("  ")
				b.WriteString(line)
				b.WriteByte('\n')
			}
			b.WriteString("  ```\n")

		case *ast.CodeBlock:
			lines := renderBlockLines(n, source)
			for _, line := range strings.Split(strings.TrimRight(lines, "\n"), "\n") {
				b.WriteString("      ")
				b.WriteString(line)
				b.WriteByte('\n')
			}

		case *ast.List:
			// Nested list.
			nested := renderListText(n, source)
			for _, line := range strings.Split(strings.TrimRight(nested, "\n"), "\n") {
				b.WriteString("  ")
				b.WriteString(line)
				b.WriteByte('\n')
			}

		case *ast.Blockquote:
			content := renderBlockquoteText(n, source)
			for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
				b.WriteString("  > ")
				b.WriteString(line)
				b.WriteByte('\n')
			}

		default:
			// Fallback: render any block lines or inline text.
			if child.HasChildren() {
				b.WriteString(renderInlineText(child, source))
				b.WriteByte('\n')
			} else {
				lines := renderBlockLines(child, source)
				if lines != "" {
					b.WriteString(lines)
				}
			}
		}

		firstBlock = false
	}
}

// renderBlockquoteText extracts text from a blockquote node.
func renderBlockquoteText(node *ast.Blockquote, source []byte) string {
	var b strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Paragraph:
			b.WriteString(renderInlineText(n, source))
			b.WriteByte('\n')
		default:
			if n.HasChildren() {
				b.WriteString(renderInlineText(n, source))
				b.WriteByte('\n')
			} else {
				lines := renderBlockLines(n, source)
				if lines != "" {
					b.WriteString(lines)
				}
			}
		}
	}
	return b.String()
}
