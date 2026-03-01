package md

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode_headingLevels(t *testing.T) {
	tests := []struct {
		md       string
		expected string
	}{
		{"# H1\n", "# H1\n"},
		{"## H2\n", "## H2\n"},
		{"### H3\n", "### H3\n"},
		{"#### H4\n", "#### H4\n"},
		{"##### H5\n", "##### H5\n"},
		{"###### H6\n", "###### H6\n"},
	}
	for _, tt := range tests {
		encoded, err := Encode([]byte(tt.md))
		require.NoError(t, err)
		output := string(encoded)
		assert.Contains(t, output, tt.expected, "input: %s", tt.md)
	}
}

func TestEncode_paragraphJoins(t *testing.T) {
	// Soft-wrapped lines should be joined into a single paragraph.
	input := "This is a long\nparagraph that wraps.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	// Should be a single line (soft break becomes space).
	assert.Contains(t, output, "This is a long paragraph that wraps.")
	// Should not contain a hard line break within the paragraph.
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 1, "soft-wrapped paragraph should produce exactly 1 line")
}

func TestEncode_codeBlockPreservesContent(t *testing.T) {
	input := "```python\n  def hello():\n      print(\"Hi\")\n```\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "```python")
	assert.Contains(t, output, "  def hello():")
	assert.Contains(t, output, "      print(\"Hi\")")
	assert.Contains(t, output, "```")
}

func TestEncode_table(t *testing.T) {
	input := "| Animal | Sound |\n|--------|-------|\n| Cat | Meow |\n| Dog | Woof |\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	// Compact table format (no spaces around pipes, minimal separator).
	assert.Contains(t, output, "|Animal|Sound|")
	assert.Contains(t, output, "|-|-|")
	assert.Contains(t, output, "|Cat|Meow|")
	assert.Contains(t, output, "|Dog|Woof|")
}

func TestEncode_kvTable(t *testing.T) {
	// KV-style table should remain as a standard markdown table (no domain detection).
	input := "| Field | Type | Description |\n|-------|------|-------------|\n| id | string | User ID |\n| name | string | Display name |\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "|Field|Type|Description|")
	assert.Contains(t, output, "|id|string|User ID|")
}

func TestEncode_paramsTable(t *testing.T) {
	// Params-style table should remain as a standard markdown table.
	input := "| Name | Type | Required | Description |\n|------|------|----------|-------------|\n| id | string | Yes | User ID |\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "|Name|Type|Required|Description|")
	assert.Contains(t, output, "|id|string|Yes|User ID|")
}

func TestEncode_endpointHeading(t *testing.T) {
	// Endpoint-style headings should remain as standard headings.
	input := "### GET /users/{id}\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "### GET /users/{id}")
}

func TestEncode_blockquote(t *testing.T) {
	input := "> This is a quote.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "> This is a quote.")
}

func TestEncode_admonition_boldNote(t *testing.T) {
	// Bold-prefix blockquotes should remain as blockquotes (no domain detection).
	input := "> **Note:** This is important.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "> ")
	assert.Contains(t, output, "Note:")
}

func TestEncode_admonition_githubStyle(t *testing.T) {
	// GitHub-style callouts should remain as blockquotes.
	input := "> [!NOTE]\n> This is a note.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "> ")
}

func TestEncode_htmlBlock(t *testing.T) {
	// HTML blocks should be preserved as raw content.
	input := "<div>\nsome content\n</div>\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "<div>")
}

func TestEncode_indentedCodeBlock(t *testing.T) {
	// Indented code block (4 spaces) should be converted to fenced.
	input := "Paragraph.\n\n    indented code\n    second line\n\nMore text.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "```")
	assert.Contains(t, output, "indented code")
}

func TestEncode_autoLink(t *testing.T) {
	input := "Visit <https://example.com> now.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "https://example.com")
}

func TestEncode_rawHTML(t *testing.T) {
	input := "Some <em>html</em> text.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "<em>")
}

func TestEncode_inlineFormatting(t *testing.T) {
	input := "Some **bold**, *italic*, and `code` text.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	// Bold, italic, and code should be preserved in markdown syntax.
	assert.Contains(t, output, "**")
	assert.Contains(t, output, "`code`")
}

func TestEncode_strikethrough(t *testing.T) {
	input := "That's ~~not~~ all.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "~~not~~")
}

func TestEncode_links(t *testing.T) {
	input := "A [link](https://example.com) here.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "[link](https://example.com)")
}

func TestEncode_images(t *testing.T) {
	input := "An ![alt text](https://example.com/img.png) here.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "![alt text](https://example.com/img.png)")
}

func TestEncode_horizontalRule(t *testing.T) {
	input := "Above\n\n---\n\nBelow\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "---")
}

func TestEncode_lists(t *testing.T) {
	input := "- Item one\n- Item two\n- Item three\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "- Item one")
	assert.Contains(t, output, "- Item two")
	assert.Contains(t, output, "- Item three")
}

func TestEncode_orderedList(t *testing.T) {
	input := "1. First\n2. Second\n3. Third\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "1. First")
	assert.Contains(t, output, "2. Second")
	assert.Contains(t, output, "3. Third")
}

func TestEncode_empty(t *testing.T) {
	encoded, err := Encode([]byte(""))
	require.NoError(t, err)
	output := string(encoded)
	// Empty input should produce minimal output (just a newline).
	assert.Equal(t, "\n", output)
}

func TestEncode_outputIsValidMarkdown(t *testing.T) {
	// The encoded output should be parseable as markdown without errors.
	input := readTestdata("api_docs.md")
	encoded, err := Encode(input)
	require.NoError(t, err)

	// Re-encoding the output should be stable (idempotent).
	reencoded, err := Encode(encoded)
	require.NoError(t, err)
	assert.Equal(t, string(encoded), string(reencoded),
		"encoding should be idempotent: encoding the output again should produce the same result")
}

// --- Reference link deduplication tests ---

func TestEncode_refLinks_longURLDedup(t *testing.T) {
	// Long URL repeated 3 times should be converted to reference-style links.
	url := "https://example.com/very/long/path/to/resource/page"
	input := fmt.Sprintf("See [one](%s) and [two](%s) and [three](%s).\n", url, url, url)
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)

	// Should contain reference-style links: [text][label]
	assert.Contains(t, output, "[one][1]", "should use reference link for 'one'")
	assert.Contains(t, output, "[two][1]", "should use reference link for 'two'")
	assert.Contains(t, output, "[three][1]", "should use reference link for 'three'")
	// Should contain the reference definition at the end.
	assert.Contains(t, output, "[1]: "+url, "should have reference definition")
	// Should NOT contain inline URL in link syntax.
	assert.NotContains(t, output, "]("+url+")", "should not have inline URL")
}

func TestEncode_refLinks_shortURLNoDedup(t *testing.T) {
	// Short URL repeated 2 times — reference style might cost MORE tokens.
	// This test verifies we don't blindly convert short URLs.
	url := "https://x.co/a"
	input := fmt.Sprintf("[one](%s) and [two](%s).\n", url, url)
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)

	// For very short URLs, inline is cheaper — should stay inline.
	assert.Contains(t, output, "]("+url+")", "short URL should remain inline")
}

func TestEncode_refLinks_singleOccurrence(t *testing.T) {
	// URL appears only once — must stay inline regardless of length.
	url := "https://example.com/very/long/path/to/resource/page"
	input := fmt.Sprintf("See [link](%s) here.\n", url)
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)

	// Must stay inline since only one occurrence.
	assert.Contains(t, output, "]("+url+")", "single-occurrence URL should stay inline")
}

func TestEncode_refLinks_roundTrip(t *testing.T) {
	// Verify that reference-style link output round-trips through goldmark.
	url := "https://example.com/very/long/path/to/resource/page"
	input := fmt.Sprintf("See [one](%s) and [two](%s) and [three](%s).\n", url, url, url)
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)

	equal, diff, err := CompareASTs([]byte(input), encoded)
	require.NoError(t, err)
	assert.True(t, equal, "reference links should round-trip:\n%s", diff)
}

func TestEncode_refLinks_multipleURLGroups(t *testing.T) {
	// Two different long URLs, each repeated 2+ times.
	url1 := "https://example.com/path/one/very/long/resource/here"
	url2 := "https://example.com/path/two/another/long/resource/there"
	input := fmt.Sprintf("[a](%s) [b](%s) [c](%s) [d](%s)\n", url1, url2, url1, url2)
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)

	// Both URLs should be reference-style with different labels.
	assert.NotContains(t, output, "]("+url1+")", "url1 should be reference-style")
	assert.NotContains(t, output, "]("+url2+")", "url2 should be reference-style")
	// Should have two reference definitions.
	assert.Contains(t, output, "[1]: ", "should have ref def 1")
	assert.Contains(t, output, "[2]: ", "should have ref def 2")
}

// --- TOC table-to-list conversion tests ---

func TestEncode_tocTable_basic(t *testing.T) {
	// A 2-column table where col1 is a link and col2 is description.
	input := `| Document | Purpose |
| --- | --- |
| [routing](routing.md) | Route conventions |
| [hooks](hooks.md) | Hook patterns |
`
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)

	// Should be converted to a UL list.
	assert.Contains(t, output, "- [routing](routing.md) -- Route conventions")
	assert.Contains(t, output, "- [hooks](hooks.md) -- Hook patterns")
	// Should NOT contain table syntax.
	assert.NotContains(t, output, "|")
}

func TestEncode_tocTable_notConvertedWhenCol1NotLink(t *testing.T) {
	// A 2-column table where col1 is NOT a link — should stay as table.
	input := `| Name | Value |
| --- | --- |
| foo | bar |
| baz | qux |
`
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)

	// Should remain as a table.
	assert.Contains(t, output, "|")
}

func TestEncode_tocTable_notConvertedWhenMoreThan2Cols(t *testing.T) {
	// A 3-column table — should stay as table even if col1 has links.
	input := `| Document | Purpose | Extra |
| --- | --- | --- |
| [routing](routing.md) | Route conventions | note |
`
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)

	// Should remain as a table (3 columns).
	assert.Contains(t, output, "|")
}

func TestEncode_tocTable_roundTrip(t *testing.T) {
	// The TOC table produces a list, which should parse correctly.
	// Note: this is NOT a true round-trip (table → list is lossy for the
	// table headers), but the list output must be valid markdown that
	// goldmark parses into the expected list structure.
	input := `| Document | Purpose |
| --- | --- |
| [routing](routing.md) | Route conventions |
| [hooks](hooks.md) | Hook patterns |
`
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)

	// Parse the encoded output and verify it's a valid list.
	body, err := markdownToAST(encoded)
	require.NoError(t, err)
	require.Len(t, body, 1, "should produce exactly one node")
	assert.Equal(t, TagUL, body[0].Tag, "should be an unordered list")
	assert.Len(t, body[0].Children, 2, "should have 2 list items")
}

func TestEncode_tocTable_mixedRows(t *testing.T) {
	// If any row doesn't have col1 as a link, the whole table stays as table.
	input := `| Document | Purpose |
| --- | --- |
| [routing](routing.md) | Route conventions |
| not-a-link | Something else |
`
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)

	// Should remain as a table (row 2 col1 is not a link).
	assert.Contains(t, output, "|")
}

// --- escapeTrailingHashesInNodes tests ---

func TestEscapeTrailingHashesInNodes_simple(t *testing.T) {
	nodes := []Node{{Tag: TagRaw, Content: "heading ##"}}
	escapeTrailingHashesInNodes(nodes)
	assert.Equal(t, "heading \\##", nodes[0].Content)
}

func TestEscapeTrailingHashesInNodes_nestedChild(t *testing.T) {
	nodes := []Node{
		{Tag: TagBold, Children: []Node{{Tag: TagRaw, Content: "text #"}}},
	}
	escapeTrailingHashesInNodes(nodes)
	assert.Equal(t, "text \\#", nodes[0].Children[0].Content)
}

func TestEscapeTrailingHashesInNodes_alreadyEscaped(t *testing.T) {
	nodes := []Node{{Tag: TagRaw, Content: "text \\#"}}
	escapeTrailingHashesInNodes(nodes)
	// Should not double-escape
	assert.Equal(t, "text \\#", nodes[0].Content)
}
