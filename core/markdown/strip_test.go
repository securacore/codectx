package markdown

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Heading level normalization
// ---------------------------------------------------------------------------

func TestStrip_HeadingNormalization_StartsAtH3(t *testing.T) {
	data := readFixture(t, "deep_headings.md")
	doc := Parse(data)

	if doc.MinLevel != 3 {
		t.Fatalf("fixture should start at H3, got MinLevel %d", doc.MinLevel)
	}

	stripped := Strip(doc)

	// After stripping, MinLevel should be 1.
	if stripped.MinLevel != 1 {
		t.Errorf("expected MinLevel 1 after normalization, got %d", stripped.MinLevel)
	}

	// All headings should be shifted: H3→H1, H4→H2, H5→H3.
	for _, b := range stripped.Blocks {
		if b.Type == BlockHeading {
			if b.Level < 1 || b.Level > 3 {
				t.Errorf("heading %q: expected level 1-3 after normalization, got %d",
					b.Content, b.Level)
			}
		}
	}

	// First heading should now be H1.
	if stripped.Blocks[0].Level != 1 {
		t.Errorf("first heading should be H1, got H%d", stripped.Blocks[0].Level)
	}
}

func TestStrip_HeadingNormalization_AlreadyH1(t *testing.T) {
	input := "# Title\n## Section\n### Sub\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)

	// Should be unchanged when already starting at H1.
	for i, b := range stripped.Blocks {
		if b.Type == BlockHeading && b.Level != doc.Blocks[i].Level {
			t.Errorf("heading %q: level changed from %d to %d when starting at H1",
				b.Content, doc.Blocks[i].Level, b.Level)
		}
	}
}

func TestStrip_HeadingNormalization_SingleH4(t *testing.T) {
	input := "#### Only Heading\n\nSome text.\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)

	if stripped.Blocks[0].Level != 1 {
		t.Errorf("single H4 should normalize to H1, got H%d", stripped.Blocks[0].Level)
	}
}

func TestStrip_HeadingNormalization_PreservesRelativeDepth(t *testing.T) {
	input := "## Chapter\n### Section\n#### Subsection\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)

	// H2→H1, H3→H2, H4→H3.
	expectedLevels := []int{1, 2, 3}
	for i, b := range stripped.Blocks {
		if b.Level != expectedLevels[i] {
			t.Errorf("block %d: expected level %d, got %d", i, expectedLevels[i], b.Level)
		}
	}
}

func TestStrip_HeadingNormalization_ClampsToMinimumOne(t *testing.T) {
	// Construct a synthetic document with a heading at level 1 and
	// MinLevel > 1 to exercise the b.Level < 1 safety clamp.
	// This can't happen through normal Parse but is defensive code.
	doc := &Document{
		Source: []byte("synthetic"),
		Blocks: []Block{
			{Type: BlockHeading, Content: "Broken Heading", Level: 1, Heading: []string{"Broken Heading"}, Position: 0},
			{Type: BlockParagraph, Content: "Content.", Heading: []string{"Broken Heading"}, Position: 1},
		},
		MinLevel: 3, // Forces delta of 2, so level 1 - 2 = -1, clamped to 1.
	}

	stripped := Strip(doc)

	if stripped.Blocks[0].Level != 1 {
		t.Errorf("heading level should clamp to 1, got %d", stripped.Blocks[0].Level)
	}
}

// ---------------------------------------------------------------------------
// Heading hierarchy recomputation after normalization
// ---------------------------------------------------------------------------

func TestStrip_HeadingHierarchy_RecomputedAfterNormalization(t *testing.T) {
	input := "### API\n#### Endpoints\n##### GET /users\n\nContent here.\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)

	// After normalization: H3→H1, H4→H2, H5→H3.
	// Content paragraph should have heading ["API", "Endpoints", "GET /users"].
	contentBlock := stripped.Blocks[len(stripped.Blocks)-1]
	if contentBlock.Type != BlockParagraph {
		t.Fatalf("expected last block to be paragraph, got %s", contentBlock.Type)
	}

	expected := []string{"API", "Endpoints", "GET /users"}
	assertHeading(t, contentBlock, expected)
}

// ---------------------------------------------------------------------------
// Thematic break removal
// ---------------------------------------------------------------------------

func TestStrip_RemovesThematicBreaks(t *testing.T) {
	input := "Before.\n\n---\n\nAfter.\n"
	doc := Parse([]byte(input))

	// Verify thematic break exists before stripping.
	hasBreak := false
	for _, b := range doc.Blocks {
		if b.Type == BlockThematicBreak {
			hasBreak = true
		}
	}
	if !hasBreak {
		t.Fatal("fixture should have a thematic break")
	}

	stripped := Strip(doc)

	for _, b := range stripped.Blocks {
		if b.Type == BlockThematicBreak {
			t.Error("thematic breaks should be removed by Strip")
		}
	}

	// Should have 2 blocks remaining.
	if len(stripped.Blocks) != 2 {
		t.Errorf("expected 2 blocks after removing thematic break, got %d", len(stripped.Blocks))
	}
}

func TestStrip_RemovesMultipleThematicBreaks(t *testing.T) {
	input := "A.\n\n---\n\nB.\n\n***\n\nC.\n\n___\n\nD.\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)

	for _, b := range stripped.Blocks {
		if b.Type == BlockThematicBreak {
			t.Error("all thematic breaks should be removed")
		}
	}

	// Should have 4 paragraph blocks.
	if len(stripped.Blocks) != 4 {
		t.Errorf("expected 4 blocks, got %d", len(stripped.Blocks))
	}
}

// ---------------------------------------------------------------------------
// HTML comment removal
// ---------------------------------------------------------------------------

func TestStrip_RemovesHTMLComments(t *testing.T) {
	data := readFixture(t, "html_comments.md")
	doc := Parse(data)

	// Count HTML blocks before stripping.
	htmlCount := 0
	for _, b := range doc.Blocks {
		if b.Type == BlockHTMLBlock {
			htmlCount++
		}
	}
	if htmlCount == 0 {
		t.Fatal("fixture should have HTML comment blocks")
	}

	stripped := Strip(doc)

	for _, b := range stripped.Blocks {
		if b.Type == BlockHTMLBlock && isHTMLCommentContent(b.Content) {
			t.Error("HTML comments should be removed by Strip")
		}
	}
}

func TestStrip_PreservesNonCommentHTML(t *testing.T) {
	input := "<div class=\"note\">\n\nImportant content.\n\n</div>\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)

	// Non-comment HTML should survive stripping.
	hasHTML := false
	for _, b := range stripped.Blocks {
		if b.Type == BlockHTMLBlock {
			hasHTML = true
		}
	}
	if !hasHTML {
		// The "Important content" paragraph should at least exist.
		hasParagraph := false
		for _, b := range stripped.Blocks {
			if b.Type == BlockParagraph && strings.Contains(b.Content, "Important") {
				hasParagraph = true
			}
		}
		if !hasParagraph {
			t.Error("expected non-comment HTML content to be preserved")
		}
	}
}

// ---------------------------------------------------------------------------
// Position renumbering after removals
// ---------------------------------------------------------------------------

func TestStrip_PositionsRenumbered(t *testing.T) {
	input := "# Title\n\n---\n\nParagraph.\n\n---\n\nEnd.\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)

	for i, b := range stripped.Blocks {
		if b.Position != i {
			t.Errorf("block %d: expected position %d, got %d", i, i, b.Position)
		}
	}
}

// ---------------------------------------------------------------------------
// Code blocks untouched
// ---------------------------------------------------------------------------

func TestStrip_CodeBlocksUntouched(t *testing.T) {
	input := "# Title\n\n```go\nfunc main() {\n\t// --- this is not a thematic break ---\n}\n```\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)

	for _, b := range stripped.Blocks {
		if b.Type == BlockCodeBlock {
			if !strings.Contains(b.Content, "// --- this is not a thematic break ---") {
				t.Errorf("code block content should be untouched, got %q", b.Content)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Source bytes preserved
// ---------------------------------------------------------------------------

func TestStrip_PreservesSourceBytes(t *testing.T) {
	input := []byte("# Title\n\n---\n\nText.\n")
	doc := Parse(input)
	stripped := Strip(doc)

	if string(stripped.Source) != string(input) {
		t.Error("Strip should preserve original source bytes")
	}
}

// ---------------------------------------------------------------------------
// Empty and edge cases
// ---------------------------------------------------------------------------

func TestStrip_EmptyDocument(t *testing.T) {
	doc := Parse([]byte(""))
	stripped := Strip(doc)
	if len(stripped.Blocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(stripped.Blocks))
	}
}

func TestStrip_OnlyThematicBreaks(t *testing.T) {
	input := "---\n\n***\n\n___\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)
	if len(stripped.Blocks) != 0 {
		t.Errorf("expected 0 blocks after removing all thematic breaks, got %d", len(stripped.Blocks))
	}
}

func TestStrip_OnlyHTMLComments(t *testing.T) {
	input := "<!-- Comment 1 -->\n\n<!-- Comment 2 -->\n"
	doc := Parse([]byte(input))
	stripped := Strip(doc)
	if len(stripped.Blocks) != 0 {
		t.Errorf("expected 0 blocks after removing all HTML comments, got %d", len(stripped.Blocks))
	}
}

// ---------------------------------------------------------------------------
// Mixed content fixture: full pipeline
// ---------------------------------------------------------------------------

func TestStrip_MixedContentFixture(t *testing.T) {
	data := readFixture(t, "mixed_content.md")
	doc := Parse(data)
	stripped := Strip(doc)

	// The fixture has one "---" thematic break. It should be removed.
	for _, b := range stripped.Blocks {
		if b.Type == BlockThematicBreak {
			t.Error("thematic break should be removed from mixed_content.md")
		}
	}

	// All other block types should be preserved.
	types := make(map[BlockType]bool)
	for _, b := range stripped.Blocks {
		types[b.Type] = true
	}

	if !types[BlockHeading] {
		t.Error("headings should be preserved")
	}
	if !types[BlockParagraph] {
		t.Error("paragraphs should be preserved")
	}
	if !types[BlockCodeBlock] {
		t.Error("code blocks should be preserved")
	}
	if !types[BlockList] {
		t.Error("lists should be preserved")
	}
	if !types[BlockTable] {
		t.Error("tables should be preserved")
	}
	if !types[BlockBlockquote] {
		t.Error("blockquotes should be preserved")
	}

	// Stripped doc should have fewer blocks than original (at least the thematic break removed).
	if len(stripped.Blocks) >= len(doc.Blocks) {
		t.Errorf("expected fewer blocks after stripping (%d), got %d",
			len(doc.Blocks), len(stripped.Blocks))
	}
}

// ---------------------------------------------------------------------------
// Emphasis in headings stripped
// ---------------------------------------------------------------------------

func TestStrip_EmphasisInHeadingsStripped(t *testing.T) {
	data := readFixture(t, "emphasis_heavy.md")
	doc := Parse(data)
	stripped := Strip(doc)

	// The fixture has "# **Important Notice**" and "## __Configuration__".
	// After Parse, emphasis is already stripped from inline text.
	// After Strip, heading content should be clean text.
	for _, b := range stripped.Blocks {
		if b.Type == BlockHeading {
			if strings.Contains(b.Content, "**") || strings.Contains(b.Content, "__") {
				t.Errorf("heading should not contain emphasis markers: %q", b.Content)
			}
		}
	}
}

func TestStrip_EmphasisInProsePreserved(t *testing.T) {
	data := readFixture(t, "emphasis_heavy.md")
	doc := Parse(data)
	stripped := Strip(doc)

	// Paragraphs should still contain the text that was bold/italic
	// (the markers are stripped by the parser, but the text is preserved).
	for _, b := range stripped.Blocks {
		if b.Type == BlockParagraph && strings.Contains(b.Content, "bold text") {
			return // Found the content — test passes.
		}
	}
	t.Error("expected prose with emphasis text to be preserved")
}
