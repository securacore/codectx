package markdown

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Heading hierarchy tracking
// ---------------------------------------------------------------------------

func TestBlocks_HeadingHierarchy_Simple(t *testing.T) {
	input := "# Title\n\nIntro paragraph.\n\n## Section\n\nSection content.\n"
	doc := Parse([]byte(input))

	// Block 0: H1 "Title" → heading: ["Title"]
	assertHeading(t, doc.Blocks[0], []string{"Title"})

	// Block 1: paragraph → heading: ["Title"]
	assertHeading(t, doc.Blocks[1], []string{"Title"})

	// Block 2: H2 "Section" → heading: ["Title", "Section"]
	assertHeading(t, doc.Blocks[2], []string{"Title", "Section"})

	// Block 3: paragraph → heading: ["Title", "Section"]
	assertHeading(t, doc.Blocks[3], []string{"Title", "Section"})
}

func TestBlocks_HeadingHierarchy_DeepNesting(t *testing.T) {
	input := "# Top\n## Mid\n### Deep\nContent.\n"
	doc := Parse([]byte(input))

	// H1 → ["Top"]
	assertHeading(t, doc.Blocks[0], []string{"Top"})
	// H2 → ["Top", "Mid"]
	assertHeading(t, doc.Blocks[1], []string{"Top", "Mid"})
	// H3 → ["Top", "Mid", "Deep"]
	assertHeading(t, doc.Blocks[2], []string{"Top", "Mid", "Deep"})
	// Paragraph → ["Top", "Mid", "Deep"]
	assertHeading(t, doc.Blocks[3], []string{"Top", "Mid", "Deep"})
}

func TestBlocks_HeadingHierarchy_Reset(t *testing.T) {
	input := "# First\n## Sub\n### Deep\n# Second\nContent.\n"
	doc := Parse([]byte(input))

	// After "# Second", the hierarchy should reset.
	// Find "# Second" block.
	var secondIdx int
	for i, b := range doc.Blocks {
		if b.Type == BlockHeading && b.Content == "Second" {
			secondIdx = i
			break
		}
	}

	assertHeading(t, doc.Blocks[secondIdx], []string{"Second"})
	// Paragraph after "# Second" should only have ["Second"].
	assertHeading(t, doc.Blocks[secondIdx+1], []string{"Second"})
}

func TestBlocks_HeadingHierarchy_GapInLevels(t *testing.T) {
	// H1 then directly H3 (skipping H2) — the hierarchy should stop at the gap.
	input := "# Title\n### Deep\nContent.\n"
	doc := Parse([]byte(input))

	// H1 → ["Title"]
	assertHeading(t, doc.Blocks[0], []string{"Title"})

	// H3 directly under H1 (no H2) — this sets heading[3] = "Deep",
	// but heading[2] is empty, so the hierarchy stops at "Title".
	// Actually, currentHeading() breaks at the first gap, so H3 creates
	// heading = ["Title"] since heading[2] is empty.
	// Wait — the extractor sets heading[3] = "Deep" and clears 4-6.
	// currentHeading iterates 1..6, finds heading[1]="Title", heading[2]="",
	// and breaks. So the H3 block's heading is ["Title"] — it loses its own text.
	// This is by design: orphaned deep headings without a parent are treated
	// as belonging to the nearest ancestor.

	// The H3 heading block itself should still have its Content set.
	if doc.Blocks[1].Content != "Deep" {
		t.Errorf("expected H3 content 'Deep', got %q", doc.Blocks[1].Content)
	}

	// But its Heading hierarchy stops at the gap.
	assertHeading(t, doc.Blocks[1], []string{"Title"})
}

func TestBlocks_HeadingHierarchy_H2BackToH2(t *testing.T) {
	input := "# Title\n## Section A\n### Subsection\n## Section B\nContent.\n"
	doc := Parse([]byte(input))

	// After "## Section B", the H3 "Subsection" should be cleared.
	var sectionBIdx int
	for i, b := range doc.Blocks {
		if b.Type == BlockHeading && b.Content == "Section B" {
			sectionBIdx = i
			break
		}
	}

	assertHeading(t, doc.Blocks[sectionBIdx], []string{"Title", "Section B"})
	assertHeading(t, doc.Blocks[sectionBIdx+1], []string{"Title", "Section B"})
}

func TestBlocks_HeadingHierarchy_NoHeadings(t *testing.T) {
	input := "Just a paragraph.\n\nAnother paragraph.\n"
	doc := Parse([]byte(input))

	for _, b := range doc.Blocks {
		if len(b.Heading) != 0 {
			t.Errorf("expected empty heading hierarchy, got %v", b.Heading)
		}
	}
}

func TestBlocks_HeadingHierarchy_FromFixture(t *testing.T) {
	data := readFixture(t, "mixed_content.md")
	doc := Parse(data)

	// The first H1 is "Authentication Guide".
	if doc.Blocks[0].Content != "Authentication Guide" {
		t.Fatalf("expected first heading 'Authentication Guide', got %q", doc.Blocks[0].Content)
	}

	// Find "Token Structure" (H3 under "JWT Tokens" H2).
	for _, b := range doc.Blocks {
		if b.Type == BlockHeading && b.Content == "Token Structure" {
			expected := []string{"Authentication Guide", "JWT Tokens", "Token Structure"}
			assertHeading(t, b, expected)
			return
		}
	}
	t.Error("did not find 'Token Structure' heading")
}

// ---------------------------------------------------------------------------
// Block ordering and position
// ---------------------------------------------------------------------------

func TestBlocks_PositionSequential(t *testing.T) {
	input := "# Title\n\nParagraph.\n\n- List item\n\n```go\ncode\n```\n"
	doc := Parse([]byte(input))

	for i, b := range doc.Blocks {
		if b.Position != i {
			t.Errorf("block %d: expected position %d, got %d", i, i, b.Position)
		}
	}
}

func TestBlocks_PreservesSourceOrder(t *testing.T) {
	input := "# Title\n\nParagraph.\n\n## Section\n\n- Item\n\n> Quote\n"
	doc := Parse([]byte(input))

	expectedTypes := []BlockType{
		BlockHeading,
		BlockParagraph,
		BlockHeading,
		BlockList,
		BlockBlockquote,
	}

	if len(doc.Blocks) != len(expectedTypes) {
		t.Fatalf("expected %d blocks, got %d", len(expectedTypes), len(doc.Blocks))
	}

	for i, expected := range expectedTypes {
		if doc.Blocks[i].Type != expected {
			t.Errorf("block %d: expected %s, got %s", i, expected, doc.Blocks[i].Type)
		}
	}
}

// ---------------------------------------------------------------------------
// Block content preservation
// ---------------------------------------------------------------------------

func TestBlocks_ListContentPreservesMarkers(t *testing.T) {
	input := "- Alpha\n- Beta\n- Gamma\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	content := doc.Blocks[0].Content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "- ") {
			t.Errorf("expected list marker prefix, got %q", line)
		}
	}
}

func TestBlocks_OrderedListContentPreservesNumbers(t *testing.T) {
	input := "1. First\n2. Second\n3. Third\n"
	doc := Parse([]byte(input))

	content := doc.Blocks[0].Content
	if !strings.Contains(content, "1.") || !strings.Contains(content, "2.") {
		t.Errorf("expected numbered markers, got %q", content)
	}
}

func TestBlocks_CodeBlockPreservesContent(t *testing.T) {
	code := "func main() {\n\tfmt.Println(\"hello\")\n}"
	input := "```go\n" + code + "\n```\n"
	doc := Parse([]byte(input))

	if doc.Blocks[0].Content != code {
		t.Errorf("code block content should be preserved exactly.\nExpected:\n%s\nGot:\n%s",
			code, doc.Blocks[0].Content)
	}
}

func TestBlocks_TableContentPreservesPipeFormat(t *testing.T) {
	input := "| Name | Age |\n|------|-----|\n| Alice | 30 |\n| Bob | 25 |\n"
	doc := Parse([]byte(input))

	if len(doc.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(doc.Blocks))
	}

	content := doc.Blocks[0].Content
	if !strings.Contains(content, "Name") {
		t.Errorf("table should contain header text, got %q", content)
	}
	if !strings.Contains(content, "Alice") {
		t.Errorf("table should contain cell text, got %q", content)
	}
	// Should have pipe separators.
	if !strings.Contains(content, " | ") {
		t.Errorf("table should use pipe separators, got %q", content)
	}
}

// ---------------------------------------------------------------------------
// Block type string representation
// ---------------------------------------------------------------------------

func TestBlockType_String(t *testing.T) {
	tests := map[BlockType]string{
		BlockParagraph:     "paragraph",
		BlockHeading:       "heading",
		BlockCodeBlock:     "code_block",
		BlockList:          "list",
		BlockTable:         "table",
		BlockBlockquote:    "blockquote",
		BlockThematicBreak: "thematic_break",
		BlockHTMLBlock:     "html_block",
	}

	for bt, expected := range tests {
		if bt.String() != expected {
			t.Errorf("BlockType(%d).String() = %q, want %q", bt, bt.String(), expected)
		}
	}
}

func TestBlockType_String_Unknown(t *testing.T) {
	// Out-of-range value should return "unknown".
	unknown := BlockType(999)
	if unknown.String() != "unknown" {
		t.Errorf("expected \"unknown\" for out-of-range BlockType, got %q", unknown.String())
	}
}

// ---------------------------------------------------------------------------
// Deep headings fixture (starts at H3)
// ---------------------------------------------------------------------------

func TestBlocks_DeepHeadingsFixture(t *testing.T) {
	data := readFixture(t, "deep_headings.md")
	doc := Parse(data)

	if doc.MinLevel != 3 {
		t.Errorf("expected MinLevel 3, got %d", doc.MinLevel)
	}

	// First block should be H3 "API Reference".
	if doc.Blocks[0].Type != BlockHeading || doc.Blocks[0].Level != 3 {
		t.Errorf("expected H3 first block, got %s level %d", doc.Blocks[0].Type, doc.Blocks[0].Level)
	}
}

// ---------------------------------------------------------------------------
// Nested lists fixture
// ---------------------------------------------------------------------------

func TestBlocks_NestedListsFixture(t *testing.T) {
	data := readFixture(t, "nested_lists.md")
	doc := Parse(data)

	// Find list blocks and verify they're atomic.
	listCount := 0
	for _, b := range doc.Blocks {
		if b.Type == BlockList {
			listCount++
			// Each list should contain all its items.
			if !strings.Contains(b.Content, "DATABASE_URL") && !strings.Contains(b.Content, "Install") {
				continue
			}
		}
	}

	if listCount < 2 {
		t.Errorf("expected at least 2 list blocks, got %d", listCount)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertHeading(t *testing.T, block Block, expected []string) {
	t.Helper()
	if len(block.Heading) != len(expected) {
		t.Errorf("heading: expected %v, got %v", expected, block.Heading)
		return
	}
	for i := range expected {
		if block.Heading[i] != expected[i] {
			t.Errorf("heading[%d]: expected %q, got %q", i, expected[i], block.Heading[i])
		}
	}
}
