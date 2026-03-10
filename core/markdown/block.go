package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// BlockType identifies the kind of semantic block extracted from markdown.
type BlockType int

const (
	// BlockParagraph is a prose paragraph.
	BlockParagraph BlockType = iota

	// BlockHeading is a heading (H1-H6).
	BlockHeading

	// BlockCodeBlock is a fenced or indented code block. Atomic — never split.
	BlockCodeBlock

	// BlockList is a complete list (ordered or unordered) with all items. Atomic.
	BlockList

	// BlockTable is a GFM table with all rows. Atomic.
	BlockTable

	// BlockBlockquote is a blockquote with all content.
	BlockBlockquote

	// BlockThematicBreak is a horizontal rule (---, ***, ___).
	// These are typically removed during stripping as decorative.
	BlockThematicBreak

	// BlockHTMLBlock is a raw HTML block.
	// HTML comments are removed during stripping; other HTML is preserved.
	BlockHTMLBlock
)

// String returns a human-readable name for the block type.
func (t BlockType) String() string {
	switch t {
	case BlockParagraph:
		return "paragraph"
	case BlockHeading:
		return "heading"
	case BlockCodeBlock:
		return "code_block"
	case BlockList:
		return "list"
	case BlockTable:
		return "table"
	case BlockBlockquote:
		return "blockquote"
	case BlockThematicBreak:
		return "thematic_break"
	case BlockHTMLBlock:
		return "html_block"
	default:
		return "unknown"
	}
}

// Block is the smallest meaningful unit of content extracted from a markdown
// document. Blocks are the input to the chunking stage — the chunker
// accumulates blocks until reaching a token target, never splitting within
// a single block.
type Block struct {
	// Type identifies the kind of block.
	Type BlockType

	// Content is the cleaned text content of this block.
	// For headings, this is the heading text without the # prefix.
	// For code blocks, this is the code content (without fences).
	// For lists, this is the full list rendered as plain text with markers.
	// For tables, this is the pipe-delimited table text.
	Content string

	// Level is the heading level (1-6) for BlockHeading, 0 for all other types.
	Level int

	// Heading is the heading hierarchy at this block's position in the document.
	// Example: ["Authentication", "JWT Tokens", "Refresh Flow"]
	// This is the "breadcrumb" path from H1 down to the most recent heading
	// above this block.
	Heading []string

	// Position is the zero-based ordinal position of this block in the source file.
	Position int

	// Language is the language tag for fenced code blocks (e.g., "go", "python").
	// Empty for non-code blocks and indented code blocks.
	Language string

	// Tokens is the token count of Content using the configured encoding.
	// Zero until populated by tokens.CountBlocks().
	Tokens int
}

// Document represents a parsed and block-extracted markdown file.
type Document struct {
	// Source is the original markdown source bytes.
	Source []byte

	// Blocks is the ordered sequence of semantic blocks.
	Blocks []Block

	// MinLevel is the minimum heading level found in the document.
	// Used for heading level normalization (e.g., if MinLevel is 3,
	// all headings can be shifted down by 2 to start at H1).
	// Zero if the document has no headings.
	MinLevel int

	// TotalTokens is the sum of all block token counts.
	// Zero until populated by tokens.CountBlocks().
	TotalTokens int
}

// Parse parses markdown source bytes into a Document with semantic blocks.
// The blocks preserve heading hierarchy context and are ordered as they
// appear in the source.
func Parse(source []byte) *Document {
	root := parseAST(source)

	doc := &Document{
		Source: source,
	}

	extractor := &blockExtractor{
		source:  source,
		heading: make([]string, 7), // index 0 unused; 1-6 for heading levels
	}

	// Walk direct children of the document node.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		extractor.extract(child, doc)
	}

	// Compute MinLevel.
	doc.MinLevel = computeMinLevel(doc.Blocks)

	return doc
}

// blockExtractor maintains state during block extraction.
type blockExtractor struct {
	source  []byte
	heading []string // heading text at each level (1-6)
	pos     int      // current block position counter
}

// extract processes a single top-level AST node and appends the resulting
// block(s) to the document.
func (e *blockExtractor) extract(node ast.Node, doc *Document) {
	switch n := node.(type) {
	case *ast.Heading:
		text := renderInlineText(n, e.source)

		// Update heading hierarchy.
		e.heading[n.Level] = text
		// Clear deeper levels.
		for i := n.Level + 1; i <= 6; i++ {
			e.heading[i] = ""
		}

		doc.Blocks = append(doc.Blocks, Block{
			Type:     BlockHeading,
			Content:  text,
			Level:    n.Level,
			Heading:  e.currentHeading(),
			Position: e.pos,
		})
		e.pos++

	case *ast.Paragraph:
		doc.Blocks = append(doc.Blocks, Block{
			Type:     BlockParagraph,
			Content:  renderInlineText(n, e.source),
			Heading:  e.currentHeading(),
			Position: e.pos,
		})
		e.pos++

	case *ast.FencedCodeBlock:
		lang := ""
		if n.Language(e.source) != nil {
			lang = string(n.Language(e.source))
		}
		doc.Blocks = append(doc.Blocks, Block{
			Type:     BlockCodeBlock,
			Content:  strings.TrimRight(renderBlockLines(n, e.source), "\n"),
			Heading:  e.currentHeading(),
			Position: e.pos,
			Language: lang,
		})
		e.pos++

	case *ast.CodeBlock:
		doc.Blocks = append(doc.Blocks, Block{
			Type:     BlockCodeBlock,
			Content:  strings.TrimRight(renderBlockLines(n, e.source), "\n"),
			Heading:  e.currentHeading(),
			Position: e.pos,
		})
		e.pos++

	case *ast.List:
		doc.Blocks = append(doc.Blocks, Block{
			Type:     BlockList,
			Content:  strings.TrimRight(renderListText(n, e.source), "\n"),
			Heading:  e.currentHeading(),
			Position: e.pos,
		})
		e.pos++

	case *ast.Blockquote:
		doc.Blocks = append(doc.Blocks, Block{
			Type:     BlockBlockquote,
			Content:  strings.TrimRight(renderBlockquoteText(n, e.source), "\n"),
			Heading:  e.currentHeading(),
			Position: e.pos,
		})
		e.pos++

	case *ast.ThematicBreak:
		doc.Blocks = append(doc.Blocks, Block{
			Type:     BlockThematicBreak,
			Heading:  e.currentHeading(),
			Position: e.pos,
		})
		e.pos++

	case *ast.HTMLBlock:
		doc.Blocks = append(doc.Blocks, Block{
			Type:     BlockHTMLBlock,
			Content:  strings.TrimRight(renderBlockLines(n, e.source), "\n"),
			Heading:  e.currentHeading(),
			Position: e.pos,
		})
		e.pos++

	default:
		// Check for GFM table extension nodes.
		if tbl, ok := node.(*east.Table); ok {
			doc.Blocks = append(doc.Blocks, Block{
				Type:     BlockTable,
				Content:  strings.TrimRight(renderTableText(tbl, e.source), "\n"),
				Heading:  e.currentHeading(),
				Position: e.pos,
			})
			e.pos++
		}
		// Other unknown block types are silently skipped.
	}
}

// currentHeading returns a snapshot of the current heading hierarchy,
// excluding empty levels. The result is a fresh slice safe from mutation.
func (e *blockExtractor) currentHeading() []string {
	return headingSnapshot(e.heading)
}

// headingSnapshot builds a heading hierarchy from a 7-element tracker array
// (index 0 unused, 1-6 for heading levels). Stops at the first gap to avoid
// orphaned deep headings without a parent.
//
// Example: ["", "Auth", "", "Flow", "", "", ""] → ["Auth"]
// (heading[2] is empty, so "Flow" at level 3 is excluded)
//
// Used by both blockExtractor during parsing and Strip during normalization.
func headingSnapshot(heading []string) []string {
	var h []string
	for i := 1; i <= 6; i++ {
		if heading[i] != "" {
			h = append(h, heading[i])
		} else {
			break
		}
	}
	if h == nil {
		return []string{}
	}
	return h
}

// computeMinLevel finds the minimum heading level in the block list.
// Returns 0 if there are no headings.
func computeMinLevel(blocks []Block) int {
	min := 0
	for _, b := range blocks {
		if b.Type == BlockHeading {
			if min == 0 || b.Level < min {
				min = b.Level
			}
		}
	}
	return min
}
