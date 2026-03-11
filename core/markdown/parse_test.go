package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// IsMarkdown
// ---------------------------------------------------------------------------

func TestIsMarkdown_ValidExtensions(t *testing.T) {
	for _, name := range []string{"file.md", "README.md", "docs/guide.md", "FILE.MD", "test.Md"} {
		if !IsMarkdown(name) {
			t.Errorf("IsMarkdown(%q) = false, want true", name)
		}
	}
}

func TestIsMarkdown_InvalidExtensions(t *testing.T) {
	for _, name := range []string{"file.txt", "file.go", "file.yaml", "file.mdx", "file.markdown", "md", ""} {
		if IsMarkdown(name) {
			t.Errorf("IsMarkdown(%q) = true, want false", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Basic parsing: simple inputs
// ---------------------------------------------------------------------------

func TestParse_SimpleParagraph(t *testing.T) {
	doc := Parse([]byte("Hello world.\n"))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}
	b := doc.Blocks[0]
	if b.Type != BlockParagraph {
		t.Errorf("expected paragraph, got %s", b.Type)
	}
	if b.Content != "Hello world." {
		t.Errorf("expected %q, got %q", "Hello world.", b.Content)
	}
}

func TestParse_MultipleParagraphs(t *testing.T) {
	input := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(doc.Blocks))
	}
	for i, b := range doc.Blocks {
		if b.Type != BlockParagraph {
			t.Errorf("block %d: expected paragraph, got %s", i, b.Type)
		}
		if b.Position != i {
			t.Errorf("block %d: expected position %d, got %d", i, i, b.Position)
		}
	}
}

func TestParse_EmptyDocument(t *testing.T) {
	doc := Parse([]byte(""))
	if len(doc.Blocks) != 0 {
		t.Errorf("expected 0 blocks for empty doc, got %d", len(doc.Blocks))
	}
}

func TestParse_WhitespaceOnlyDocument(t *testing.T) {
	doc := Parse([]byte("   \n\n   \n"))
	if len(doc.Blocks) != 0 {
		t.Errorf("expected 0 blocks for whitespace-only doc, got %d", len(doc.Blocks))
	}
}

// ---------------------------------------------------------------------------
// Headings
// ---------------------------------------------------------------------------

func TestParse_ATXHeadings(t *testing.T) {
	input := "# H1\n## H2\n### H3\n#### H4\n##### H5\n###### H6\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 6 {
		t.Fatalf("expected 6 blocks, got %d", len(doc.Blocks))
	}

	for i, b := range doc.Blocks {
		if b.Type != BlockHeading {
			t.Errorf("block %d: expected heading, got %s", i, b.Type)
		}
		expectedLevel := i + 1
		if b.Level != expectedLevel {
			t.Errorf("block %d: expected level %d, got %d", i, expectedLevel, b.Level)
		}
	}
}

func TestParse_SetextHeadings(t *testing.T) {
	input := "H1 Heading\n==========\n\nH2 Heading\n----------\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(doc.Blocks))
	}

	if doc.Blocks[0].Level != 1 {
		t.Errorf("expected H1, got H%d", doc.Blocks[0].Level)
	}
	if doc.Blocks[0].Content != "H1 Heading" {
		t.Errorf("expected %q, got %q", "H1 Heading", doc.Blocks[0].Content)
	}

	if doc.Blocks[1].Level != 2 {
		t.Errorf("expected H2, got H%d", doc.Blocks[1].Level)
	}
}

func TestParse_HeadingWithInlineFormatting(t *testing.T) {
	input := "# **Bold** and *italic* heading\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	// Emphasis markers should be stripped in the text extraction.
	b := doc.Blocks[0]
	if b.Type != BlockHeading {
		t.Errorf("expected heading, got %s", b.Type)
	}
	if b.Content != "Bold and italic heading" {
		t.Errorf("expected %q, got %q", "Bold and italic heading", b.Content)
	}
}

func TestParse_HeadingWithInlineCode(t *testing.T) {
	input := "## The `CreateUser` function\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if !strings.Contains(b.Content, "`CreateUser`") {
		t.Errorf("expected heading to contain backtick-wrapped code, got %q", b.Content)
	}
}

// ---------------------------------------------------------------------------
// Code blocks
// ---------------------------------------------------------------------------

func TestParse_FencedCodeBlock(t *testing.T) {
	input := "```go\nfunc main() {}\n```\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockCodeBlock {
		t.Errorf("expected code_block, got %s", b.Type)
	}
	if b.Language != "go" {
		t.Errorf("expected language %q, got %q", "go", b.Language)
	}
	if !strings.Contains(b.Content, "func main()") {
		t.Errorf("expected code content, got %q", b.Content)
	}
}

func TestParse_FencedCodeBlockNoLanguage(t *testing.T) {
	input := "```\nplain text\n```\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockCodeBlock {
		t.Errorf("expected code_block, got %s", b.Type)
	}
	if b.Language != "" {
		t.Errorf("expected empty language, got %q", b.Language)
	}
}

func TestParse_IndentedCodeBlock(t *testing.T) {
	input := "Some text.\n\n    indented code\n    more code\n\nAfter code.\n"
	doc := Parse([]byte(input))

	// Should have: paragraph, code block, paragraph.
	found := false
	for _, b := range doc.Blocks {
		if b.Type == BlockCodeBlock {
			found = true
			if b.Language != "" {
				t.Errorf("indented code should have no language, got %q", b.Language)
			}
			if !strings.Contains(b.Content, "indented code") {
				t.Errorf("expected indented code content, got %q", b.Content)
			}
		}
	}
	if !found {
		t.Error("expected to find an indented code block")
	}
}

func TestParse_TildeFencedCodeBlock(t *testing.T) {
	input := "~~~python\nprint('hello')\n~~~\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}
	if doc.Blocks[0].Language != "python" {
		t.Errorf("expected language %q, got %q", "python", doc.Blocks[0].Language)
	}
}

// ---------------------------------------------------------------------------
// Lists
// ---------------------------------------------------------------------------

func TestParse_UnorderedList(t *testing.T) {
	input := "- Item one\n- Item two\n- Item three\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block (whole list), got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockList {
		t.Errorf("expected list, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "Item one") {
		t.Errorf("expected list content to contain 'Item one', got %q", b.Content)
	}
	if !strings.Contains(b.Content, "Item three") {
		t.Errorf("expected list content to contain 'Item three', got %q", b.Content)
	}
}

func TestParse_OrderedList(t *testing.T) {
	input := "1. First\n2. Second\n3. Third\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockList {
		t.Errorf("expected list, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "1.") {
		t.Errorf("expected ordered list markers, got %q", b.Content)
	}
}

func TestParse_ListIsAtomicBlock(t *testing.T) {
	input := "Before.\n\n- A\n- B\n- C\n\nAfter.\n"
	doc := Parse([]byte(input))

	// Should be: paragraph, list, paragraph — 3 blocks.
	if len(doc.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(doc.Blocks))
	}
	if doc.Blocks[0].Type != BlockParagraph {
		t.Errorf("block 0: expected paragraph, got %s", doc.Blocks[0].Type)
	}
	if doc.Blocks[1].Type != BlockList {
		t.Errorf("block 1: expected list, got %s", doc.Blocks[1].Type)
	}
	if doc.Blocks[2].Type != BlockParagraph {
		t.Errorf("block 2: expected paragraph, got %s", doc.Blocks[2].Type)
	}
}

// ---------------------------------------------------------------------------
// Tables (GFM)
// ---------------------------------------------------------------------------

func TestParse_GFMTable(t *testing.T) {
	input := "| A | B |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockTable {
		t.Errorf("expected table, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "A") {
		t.Errorf("expected table to contain header text, got %q", b.Content)
	}
	if !strings.Contains(b.Content, "3") {
		t.Errorf("expected table to contain cell text, got %q", b.Content)
	}
}

func TestParse_TableIsAtomicBlock(t *testing.T) {
	input := "Before.\n\n| X | Y |\n|---|---|\n| 1 | 2 |\n\nAfter.\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(doc.Blocks))
	}
	if doc.Blocks[1].Type != BlockTable {
		t.Errorf("block 1: expected table, got %s", doc.Blocks[1].Type)
	}
}

// ---------------------------------------------------------------------------
// Blockquotes
// ---------------------------------------------------------------------------

func TestParse_Blockquote(t *testing.T) {
	input := "> This is a quote.\n> Second line.\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockBlockquote {
		t.Errorf("expected blockquote, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "This is a quote.") {
		t.Errorf("expected blockquote content, got %q", b.Content)
	}
}

func TestParse_MultiParagraphBlockquote(t *testing.T) {
	input := "> First paragraph.\n>\n> Second paragraph.\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}
	if doc.Blocks[0].Type != BlockBlockquote {
		t.Errorf("expected blockquote, got %s", doc.Blocks[0].Type)
	}
}

func TestParse_BlockquoteWithCodeBlock(t *testing.T) {
	input := "> Example:\n>\n>     indented code line\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}
	b := doc.Blocks[0]
	if b.Type != BlockBlockquote {
		t.Errorf("expected blockquote, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "Example") {
		t.Errorf("expected blockquote to contain 'Example', got %q", b.Content)
	}
}

func TestParse_BlockquoteWithList(t *testing.T) {
	input := "> Items:\n>\n> - first\n> - second\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}
	b := doc.Blocks[0]
	if b.Type != BlockBlockquote {
		t.Errorf("expected blockquote, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "first") {
		t.Errorf("expected blockquote to contain list items, got %q", b.Content)
	}
}

// ---------------------------------------------------------------------------
// Thematic breaks
// ---------------------------------------------------------------------------

func TestParse_ThematicBreak(t *testing.T) {
	input := "Before.\n\n---\n\nAfter.\n"
	doc := Parse([]byte(input))

	found := false
	for _, b := range doc.Blocks {
		if b.Type == BlockThematicBreak {
			found = true
		}
	}
	if !found {
		t.Error("expected to find a thematic break block")
	}
}

// ---------------------------------------------------------------------------
// HTML blocks
// ---------------------------------------------------------------------------

func TestParse_HTMLComment(t *testing.T) {
	input := "<!-- This is a comment -->\n\nSome text.\n"
	doc := Parse([]byte(input))

	found := false
	for _, b := range doc.Blocks {
		if b.Type == BlockHTMLBlock {
			found = true
			if !strings.Contains(b.Content, "<!--") {
				t.Errorf("expected HTML comment content, got %q", b.Content)
			}
		}
	}
	if !found {
		t.Error("expected to find an HTML block")
	}
}

// ---------------------------------------------------------------------------
// Inline formatting in paragraphs
// ---------------------------------------------------------------------------

func TestParse_ParagraphWithBold(t *testing.T) {
	doc := Parse([]byte("This has **bold** text.\n"))
	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}
	// Bold markers should be stripped; text content preserved.
	if !strings.Contains(doc.Blocks[0].Content, "bold") {
		t.Errorf("expected 'bold' in content, got %q", doc.Blocks[0].Content)
	}
	if strings.Contains(doc.Blocks[0].Content, "**") {
		t.Errorf("bold markers should be stripped, got %q", doc.Blocks[0].Content)
	}
}

func TestParse_ParagraphWithItalic(t *testing.T) {
	doc := Parse([]byte("This has *italic* text.\n"))
	if !strings.Contains(doc.Blocks[0].Content, "italic") {
		t.Errorf("expected 'italic' in content, got %q", doc.Blocks[0].Content)
	}
	if strings.Contains(doc.Blocks[0].Content, "*italic*") {
		t.Errorf("italic markers should be stripped, got %q", doc.Blocks[0].Content)
	}
}

func TestParse_ParagraphWithInlineCode(t *testing.T) {
	doc := Parse([]byte("Use the `CreateUser` function.\n"))
	content := doc.Blocks[0].Content
	if !strings.Contains(content, "`CreateUser`") {
		t.Errorf("inline code should be preserved with backticks, got %q", content)
	}
}

func TestParse_ParagraphWithLink(t *testing.T) {
	doc := Parse([]byte("See [the docs](https://example.com) for details.\n"))
	content := doc.Blocks[0].Content
	if !strings.Contains(content, "the docs") {
		t.Errorf("link text should be preserved, got %q", content)
	}
	// URL should NOT be in the text content.
	if strings.Contains(content, "https://example.com") {
		t.Errorf("link URL should not be in text content, got %q", content)
	}
}

func TestParse_ParagraphWithAutoLink(t *testing.T) {
	doc := Parse([]byte("Visit https://example.com for details.\n"))
	content := doc.Blocks[0].Content
	if !strings.Contains(content, "https://example.com") {
		t.Errorf("autolink URL should be preserved, got %q", content)
	}
}

func TestParse_ParagraphWithImage(t *testing.T) {
	doc := Parse([]byte("Here is ![alt text](image.png) inline.\n"))
	content := doc.Blocks[0].Content
	if !strings.Contains(content, "alt text") {
		t.Errorf("image alt text should be preserved, got %q", content)
	}
}

func TestParse_ParagraphSoftLineBreak(t *testing.T) {
	input := "First line\nsecond line\n"
	doc := Parse([]byte(input))
	content := doc.Blocks[0].Content
	// Soft line breaks should be spaces.
	if !strings.Contains(content, "First line second line") {
		t.Errorf("soft line break should become space, got %q", content)
	}
}

// ---------------------------------------------------------------------------
// List items with nested block content
// ---------------------------------------------------------------------------

func TestParse_ListWithFencedCodeBlock(t *testing.T) {
	input := "- Item one\n\n  ```go\n  fmt.Println(\"hello\")\n  ```\n\n- Item two\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block (atomic list), got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockList {
		t.Errorf("expected list, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "Item one") {
		t.Errorf("expected list to contain 'Item one', got %q", b.Content)
	}
	if !strings.Contains(b.Content, "```go") {
		t.Errorf("expected list to contain fenced code markers, got %q", b.Content)
	}
	if !strings.Contains(b.Content, "fmt.Println") {
		t.Errorf("expected list to contain code content, got %q", b.Content)
	}
	if !strings.Contains(b.Content, "Item two") {
		t.Errorf("expected list to contain 'Item two', got %q", b.Content)
	}
}

func TestParse_ListWithIndentedCodeBlock(t *testing.T) {
	input := "- Item one\n\n      indented code line\n      more code\n\n- Item two\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block (atomic list), got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockList {
		t.Errorf("expected list, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "indented code line") {
		t.Errorf("expected list to contain indented code, got %q", b.Content)
	}
}

func TestParse_ListWithBlockquote(t *testing.T) {
	input := "- Item one\n\n  > A quoted block\n  > inside a list item.\n\n- Item two\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block (atomic list), got %d", len(doc.Blocks))
	}

	b := doc.Blocks[0]
	if b.Type != BlockList {
		t.Errorf("expected list, got %s", b.Type)
	}
	if !strings.Contains(b.Content, "Item one") {
		t.Errorf("expected 'Item one' in content, got %q", b.Content)
	}
	if !strings.Contains(b.Content, ">") {
		t.Errorf("expected blockquote marker in list content, got %q", b.Content)
	}
	if !strings.Contains(b.Content, "quoted block") {
		t.Errorf("expected blockquote text in list content, got %q", b.Content)
	}
}

func TestParse_ListItemDefaultFallback(t *testing.T) {
	// A list item containing an HTML block exercises the default branch
	// in renderListItemContent. HTML blocks in list items are unusual
	// but possible with CommonMark's nested block rules.
	input := "- Item one\n\n  <div>\n  custom html\n  </div>\n\n- Item two\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) < 1 {
		t.Fatal("expected at least 1 block")
	}

	// The list should still parse without panicking, and contain Item one/two.
	found := false
	for _, b := range doc.Blocks {
		if b.Type == BlockList {
			found = true
			if !strings.Contains(b.Content, "Item one") {
				t.Errorf("expected 'Item one' in list content, got %q", b.Content)
			}
		}
	}
	if !found {
		t.Error("expected to find a list block")
	}
}

// ---------------------------------------------------------------------------
// Inline raw HTML and hard line breaks
// ---------------------------------------------------------------------------

func TestParse_ParagraphWithInlineRawHTML(t *testing.T) {
	input := "Some text with a <br> break and <em>emphasis</em> in it.\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	content := doc.Blocks[0].Content
	// Raw inline HTML should be preserved as-is.
	if !strings.Contains(content, "<br>") {
		t.Errorf("expected inline raw HTML <br> to be preserved, got %q", content)
	}
}

func TestParse_ParagraphWithHardLineBreak(t *testing.T) {
	// Two trailing spaces create a hard line break in CommonMark.
	input := "First line  \nsecond line\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	content := doc.Blocks[0].Content
	// Hard line break should produce a newline, not a space.
	if !strings.Contains(content, "First line\nsecond line") {
		t.Errorf("hard line break should become newline, got %q", content)
	}
}

// ---------------------------------------------------------------------------
// Code span with soft line break
// ---------------------------------------------------------------------------

func TestParse_CodeSpanAcrossLines(t *testing.T) {
	// A backtick code span that wraps across two source lines.
	// Goldmark preserves the newline in the text segment (SoftLineBreak=false)
	// rather than collapsing it to a space per CommonMark spec. Our renderer
	// preserves goldmark's output faithfully.
	input := "Here is `some code\nthat wraps` in a paragraph.\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	content := doc.Blocks[0].Content
	// The code span text should be preserved with backticks.
	if !strings.Contains(content, "`some code") {
		t.Errorf("expected code span content to be preserved, got %q", content)
	}
	if !strings.Contains(content, "that wraps`") {
		t.Errorf("expected code span closing to be preserved, got %q", content)
	}
}

// ---------------------------------------------------------------------------
// MinLevel computation
// ---------------------------------------------------------------------------

func TestParse_MinLevel_StartsAtH1(t *testing.T) {
	doc := Parse([]byte("# Title\n## Section\n"))
	if doc.MinLevel != 1 {
		t.Errorf("expected MinLevel 1, got %d", doc.MinLevel)
	}
}

func TestParse_MinLevel_StartsAtH3(t *testing.T) {
	doc := Parse([]byte("### Title\n#### Section\n"))
	if doc.MinLevel != 3 {
		t.Errorf("expected MinLevel 3, got %d", doc.MinLevel)
	}
}

func TestParse_MinLevel_NoHeadings(t *testing.T) {
	doc := Parse([]byte("Just a paragraph.\n"))
	if doc.MinLevel != 0 {
		t.Errorf("expected MinLevel 0 for no headings, got %d", doc.MinLevel)
	}
}

// ---------------------------------------------------------------------------
// Fixture file: mixed content
// ---------------------------------------------------------------------------

func TestParse_MixedContentFixture(t *testing.T) {
	data := readFixture(t, "mixed_content.md")
	doc := Parse(data)

	if len(doc.Blocks) == 0 {
		t.Fatal("expected blocks from mixed_content.md")
	}

	// Check that we have a variety of block types.
	types := make(map[BlockType]int)
	for _, b := range doc.Blocks {
		types[b.Type]++
	}

	if types[BlockHeading] == 0 {
		t.Error("expected heading blocks")
	}
	if types[BlockParagraph] == 0 {
		t.Error("expected paragraph blocks")
	}
	if types[BlockCodeBlock] == 0 {
		t.Error("expected code block(s)")
	}
	if types[BlockList] == 0 {
		t.Error("expected list block(s)")
	}
	if types[BlockTable] == 0 {
		t.Error("expected table block(s)")
	}
	if types[BlockBlockquote] == 0 {
		t.Error("expected blockquote block(s)")
	}
	if types[BlockThematicBreak] == 0 {
		t.Error("expected thematic break")
	}
}

func TestParse_CodeBlocksFixture(t *testing.T) {
	data := readFixture(t, "code_blocks.md")
	doc := Parse(data)

	// Collect code blocks.
	var codeBlocks []Block
	for _, b := range doc.Blocks {
		if b.Type == BlockCodeBlock {
			codeBlocks = append(codeBlocks, b)
		}
	}

	if len(codeBlocks) < 4 {
		t.Fatalf("expected at least 4 code blocks, got %d", len(codeBlocks))
	}

	// Check languages.
	languages := make(map[string]bool)
	for _, cb := range codeBlocks {
		if cb.Language != "" {
			languages[cb.Language] = true
		}
	}

	if !languages["go"] {
		t.Error("expected a Go code block")
	}
	if !languages["python"] {
		t.Error("expected a Python code block")
	}
	if !languages["bash"] {
		t.Error("expected a Bash code block")
	}
}

// ---------------------------------------------------------------------------
// Fixture file: no headings
// ---------------------------------------------------------------------------

func TestParse_NoHeadingsFixture(t *testing.T) {
	data := readFixture(t, "no_headings.md")
	doc := Parse(data)

	if len(doc.Blocks) == 0 {
		t.Fatal("expected blocks from no_headings.md")
	}
	if doc.MinLevel != 0 {
		t.Errorf("expected MinLevel 0 for no-headings document, got %d", doc.MinLevel)
	}

	for _, b := range doc.Blocks {
		if b.Type == BlockHeading {
			t.Error("no_headings.md should not contain heading blocks")
		}
		if len(b.Heading) != 0 {
			t.Errorf("blocks should have empty heading hierarchy, got %v", b.Heading)
		}
	}

	// Should contain inline code and link text.
	foundCode := false
	foundLink := false
	for _, b := range doc.Blocks {
		if strings.Contains(b.Content, "`inline code`") {
			foundCode = true
		}
		if strings.Contains(b.Content, "link") {
			foundLink = true
		}
	}
	if !foundCode {
		t.Error("expected inline code to be preserved")
	}
	if !foundLink {
		t.Error("expected link text to be preserved")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}
