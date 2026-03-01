package cmdx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInline_plainText(t *testing.T) {
	nodes := parseInline("hello world")
	require.Len(t, nodes, 1)
	assert.Equal(t, TagRaw, nodes[0].Tag)
	assert.Equal(t, "hello world", nodes[0].Content)
}

func TestParseInline_bold(t *testing.T) {
	nodes := parseInline("before @B{bold} after")
	require.Len(t, nodes, 3)
	assert.Equal(t, TagRaw, nodes[0].Tag)
	assert.Equal(t, "before ", nodes[0].Content)
	assert.Equal(t, TagBold, nodes[1].Tag)
	require.Len(t, nodes[1].Children, 1)
	assert.Equal(t, "bold", nodes[1].Children[0].Content)
	assert.Equal(t, TagRaw, nodes[2].Tag)
	assert.Equal(t, " after", nodes[2].Content)
}

func TestParseInline_nested(t *testing.T) {
	nodes := parseInline("@B{some @I{italic} text}")
	require.Len(t, nodes, 1)
	bold := nodes[0]
	assert.Equal(t, TagBold, bold.Tag)
	require.Len(t, bold.Children, 3)
	assert.Equal(t, "some ", bold.Children[0].Content)
	assert.Equal(t, TagItalic, bold.Children[1].Tag)
	assert.Equal(t, " text", bold.Children[2].Content)
}

func TestParseInline_code(t *testing.T) {
	nodes := parseInline("use @C{fmt.Println} here")
	require.Len(t, nodes, 3)
	assert.Equal(t, TagCode, nodes[1].Tag)
	assert.Equal(t, "fmt.Println", nodes[1].Content)
}

func TestParseInline_link(t *testing.T) {
	nodes := parseInline("see @LINK{Example>https://example.com} now")
	require.Len(t, nodes, 3)
	link := nodes[1]
	assert.Equal(t, TagLink, link.Tag)
	assert.Equal(t, "https://example.com", link.Attrs.URL)
	require.Len(t, link.Children, 1)
	assert.Equal(t, "Example", link.Children[0].Content)
}

func TestParseInline_image(t *testing.T) {
	nodes := parseInline("@IMG{alt text>https://img.png}")
	require.Len(t, nodes, 1)
	img := nodes[0]
	assert.Equal(t, TagImage, img.Tag)
	assert.Equal(t, "https://img.png", img.Attrs.URL)
	assert.Equal(t, "alt text", img.Attrs.Display)
}

func TestParseInline_escapedAt(t *testing.T) {
	nodes := parseInline("hello @@ world")
	require.Len(t, nodes, 1)
	assert.Equal(t, "hello @@ world", nodes[0].Content)
}

func TestParseInline_escapedDollar(t *testing.T) {
	nodes := parseInline("cost $$5")
	require.Len(t, nodes, 1)
	assert.Equal(t, "cost $$5", nodes[0].Content)
}

func TestParseInline_strikethrough(t *testing.T) {
	nodes := parseInline("@S{deleted}")
	require.Len(t, nodes, 1)
	assert.Equal(t, TagStrikethrough, nodes[0].Tag)
}

func TestTagParser_heading(t *testing.T) {
	p := newTagParser("@H1 Hello World")
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagH1, nodes[0].Tag)
	require.Len(t, nodes[0].Children, 1)
	assert.Equal(t, "Hello World", nodes[0].Children[0].Content)
}

func TestTagParser_paragraph(t *testing.T) {
	p := newTagParser("@P Some text here")
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagP, nodes[0].Tag)
}

func TestTagParser_codeBlock(t *testing.T) {
	input := "@CODE:go\nfunc main() {}\n@/CODE"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagCodeBlock, nodes[0].Tag)
	assert.Equal(t, "go", nodes[0].Attrs.Language)
	assert.Equal(t, "func main() {}", nodes[0].Content)
}

func TestTagParser_codeBlockEscapedAt(t *testing.T) {
	input := "@CODE:go\n\\@SomeAnnotation\nfunc main() {}\n@/CODE"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "@SomeAnnotation\nfunc main() {}", nodes[0].Content)
}

func TestTagParser_unorderedList(t *testing.T) {
	input := "@UL{\n- Item one\n- Item two\n- Item three\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagUL, nodes[0].Tag)
	assert.Len(t, nodes[0].Children, 3)
}

func TestTagParser_orderedList(t *testing.T) {
	input := "@OL{\n1. First\n2. Second\n3. Third\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagOL, nodes[0].Tag)
	assert.Len(t, nodes[0].Children, 3)
}

func TestTagParser_table(t *testing.T) {
	input := "@TABLE{\n@THEAD{Name|Value}\nfoo|bar\nbaz|qux\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagTable, nodes[0].Tag)
	assert.Equal(t, []string{"Name", "Value"}, nodes[0].Attrs.Headers)
	assert.Len(t, nodes[0].Attrs.Cells, 2)
}

func TestTagParser_blockquoteSingleLine(t *testing.T) {
	input := "@BQ Some quoted text"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagBQ, nodes[0].Tag)
	require.Len(t, nodes[0].Children, 1)
	assert.Equal(t, TagP, nodes[0].Children[0].Tag)
}

func TestTagParser_blockquoteBlock(t *testing.T) {
	input := "@BQ{\n@P Line one\n@P Line two\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagBQ, nodes[0].Tag)
	assert.Len(t, nodes[0].Children, 2)
}

func TestTagParser_horizontalRule(t *testing.T) {
	p := newTagParser("@HR")
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagHR, nodes[0].Tag)
}

func TestTagParser_unknownTag(t *testing.T) {
	p := newTagParser("@FOO bar")
	_, err := p.ParseBody()
	assert.Error(t, err)
}

func TestSplitLinkContent(t *testing.T) {
	display, url := splitLinkContent("text>https://example.com")
	assert.Equal(t, "text", display)
	assert.Equal(t, "https://example.com", url)
}

func TestSplitLinkContent_escapedArrow(t *testing.T) {
	display, url := splitLinkContent(`a \> b>https://example.com`)
	assert.Equal(t, "a > b", display)
	assert.Equal(t, "https://example.com", url)
}

func TestSplitLinkContent_nestedBraces(t *testing.T) {
	display, url := splitLinkContent("@B{bold}>https://example.com")
	assert.Equal(t, "@B{bold}", display)
	assert.Equal(t, "https://example.com", url)
}

// --- readBracedBlock edge cases ---

func TestReadBracedBlock_singleLineClose(t *testing.T) {
	// Opening and closing brace on same line: @TAG{content}
	p := newTagParser("@NOTE{hello world}")
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagNote, nodes[0].Tag)
	assert.Equal(t, "hello world", nodes[0].Attrs.Callout)
}

func TestReadBracedBlock_escapedBraces(t *testing.T) {
	// Escaped braces should not count for nesting.
	input := "@UL{\n- item with \\{ braces \\}\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagUL, nodes[0].Tag)
}

func TestReadBracedBlock_nestedBraces(t *testing.T) {
	// Nested braces should be handled correctly.
	input := "@BQ{\n@UL{\n- inner item\n}\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagBQ, nodes[0].Tag)
}

func TestReadBracedBlock_unclosed(t *testing.T) {
	// Unclosed brace should return error.
	input := "@UL{\n- item one\n- item two"
	p := newTagParser(input)
	_, err := p.ParseBody()
	assert.Error(t, err)
}

func TestReadBracedBlock_contentOnCloseLine(t *testing.T) {
	// Content before closing brace on the same line.
	input := "@UL{\n- item one\n- last item}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagUL, nodes[0].Tag)
}

func TestReadBracedBlock_noBrace(t *testing.T) {
	// Line with no opening brace should error.
	p := &TagParser{lines: []string{"@UL no brace"}, pos: 0}
	_, err := p.readBracedBlock()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected {")
}

func TestReadBracedBlock_emptyContent(t *testing.T) {
	// Empty braced block: @BQ{}
	input := "@BQ{\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagBQ, nodes[0].Tag)
}

func TestReadBracedBlock_singleLineWithContent(t *testing.T) {
	// Single-line braced block: @TABLE{...} on one line (multi-line body normally)
	// Exercise same-line close with content.
	input := "@TABLE{\n@THEAD{A|B}\nx|y}\n"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagTable, nodes[0].Tag)
}

func TestReadBracedBlock_multiLineEscapedBraces(t *testing.T) {
	// Escaped braces in multi-line content should not affect depth counting.
	input := "@TABLE{\n@THEAD{A|B}\nfoo \\{ \\}|bar\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagTable, nodes[0].Tag)
}

func TestReadBracedBlock_nestedBracesMultiLine(t *testing.T) {
	// Nested braces in multi-line content.
	input := "@BQ{\n@UL{\n- item\n}\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagBQ, nodes[0].Tag)
	// Should have a nested UL child
	assert.Len(t, nodes[0].Children, 1)
}

func TestReadBracedBlock_sameLineEscaped(t *testing.T) {
	// Escaped brace on the opening line.
	input := "@TABLE{\n@THEAD{A\\{|B}\nx|y\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagTable, nodes[0].Tag)
}

func TestReadBracedBlock_contentAfterOpenBrace(t *testing.T) {
	// Content on the same line as the opening brace.
	input := "@UL{- item one\n- item two\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagUL, nodes[0].Tag)
}

// --- collectNestedBraced edge cases ---

func TestCollectNestedBraced_noBrace(t *testing.T) {
	lines := []string{"no brace here"}
	inner, end := collectNestedBraced(lines, 0)
	assert.Nil(t, inner)
	assert.Equal(t, 1, end)
}

func TestCollectNestedBraced_immediateClosure(t *testing.T) {
	lines := []string{"tag{}"}
	inner, end := collectNestedBraced(lines, 0)
	assert.Empty(t, inner)
	assert.Equal(t, 1, end)
}

func TestCollectNestedBraced_escapedBraces(t *testing.T) {
	lines := []string{"tag{", "a \\{ b \\}", "}"}
	inner, end := collectNestedBraced(lines, 0)
	assert.Equal(t, []string{"a \\{ b \\}"}, inner)
	assert.Equal(t, 3, end)
}

func TestCollectNestedBraced_contentOnCloseLine(t *testing.T) {
	lines := []string{"tag{", "content}"}
	inner, end := collectNestedBraced(lines, 0)
	assert.Equal(t, []string{"content"}, inner)
	assert.Equal(t, 2, end)
}

// --- parseDEFBlock edge cases ---

func TestTagParser_defBlockEmpty(t *testing.T) {
	input := "@DEF{\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagDef, nodes[0].Tag)
	assert.Empty(t, nodes[0].Attrs.Items)
}

func TestTagParser_defBlockEmptyLines(t *testing.T) {
	input := "@DEF{\n\n  API~Application\n\n  URL~Locator\n\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Items, 2)
}

func TestTagParser_defBlockKeyOnly(t *testing.T) {
	// Key without ~description: parsed as KVItem with empty description.
	input := "@DEF{\n  API\n  URL~Locator\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	// Both items should be parsed — API gets empty description.
	assert.Len(t, nodes[0].Attrs.Items, 2)
	assert.Equal(t, "API", nodes[0].Attrs.Items[0].Key)
	assert.Equal(t, "", nodes[0].Attrs.Items[0].Description)
	assert.Equal(t, "URL", nodes[0].Attrs.Items[1].Key)
	assert.Equal(t, "Locator", nodes[0].Attrs.Items[1].Description)
}

// --- parseKVBlock edge cases ---

func TestTagParser_kvBlockEmpty(t *testing.T) {
	input := "@KV{\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagKV, nodes[0].Tag)
	assert.Empty(t, nodes[0].Attrs.Items)
}

func TestTagParser_kvBlockEmptyLines(t *testing.T) {
	input := "@KV{\n\n  id:string~ID\n\n  name:string~Name\n\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Items, 2)
}

func TestTagParser_kvBlockTypeOnly(t *testing.T) {
	// key:type without ~description
	input := "@KV{\n  id:string\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Items, 1)
	assert.Equal(t, "id", nodes[0].Attrs.Items[0].Key)
	assert.Equal(t, "string", nodes[0].Attrs.Items[0].Type)
	assert.Equal(t, "", nodes[0].Attrs.Items[0].Description)
}

func TestTagParser_kvBlockNoColon(t *testing.T) {
	// Lines without colon should be skipped
	input := "@KV{\n  invalid line\n  id:string~ID\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Items, 1)
}

// --- parseParamsBlock edge cases ---

func TestTagParser_paramsBlockEmpty(t *testing.T) {
	input := "@PARAMS{\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagParams, nodes[0].Tag)
	assert.Empty(t, nodes[0].Attrs.Params)
}

func TestTagParser_paramsBlockEmptyLines(t *testing.T) {
	input := "@PARAMS{\n\n  id:string:R~ID\n\n  name:string:O~Name\n\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Params, 2)
}

func TestTagParser_paramsBlockNoTilde(t *testing.T) {
	// Params without ~ description
	input := "@PARAMS{\n  id:string:R\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Params, 1)
	assert.Equal(t, "id", nodes[0].Attrs.Params[0].Name)
	assert.Equal(t, true, nodes[0].Attrs.Params[0].Required)
	assert.Equal(t, "", nodes[0].Attrs.Params[0].Description)
}

func TestTagParser_paramsBlockFewerThan3Colons(t *testing.T) {
	// Lines with fewer than 3 colon-separated parts should be skipped
	input := "@PARAMS{\n  onlytwo:parts\n  id:string:R~ID\n}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Params, 1)
	assert.Equal(t, "id", nodes[0].Attrs.Params[0].Name)
}

// --- parseEndpointTag edge cases ---

func TestTagParser_endpointMethodOnly(t *testing.T) {
	input := "@ENDPOINT{GET}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, TagEndpoint, nodes[0].Tag)
	assert.Equal(t, "GET", nodes[0].Attrs.Endpoint.Method)
	assert.Equal(t, "", nodes[0].Attrs.Endpoint.Path)
}

// --- parseReturnsTag edge cases ---

func TestTagParser_returnsMultiple(t *testing.T) {
	input := "@RETURNS{200:OK|404:Not Found|500:Error}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Returns, 3)
	assert.Equal(t, "200", nodes[0].Attrs.Returns[0].Status)
	assert.Equal(t, "OK", nodes[0].Attrs.Returns[0].Description)
}

func TestTagParser_returnsStatusOnly(t *testing.T) {
	input := "@RETURNS{204}"
	p := newTagParser(input)
	nodes, err := p.ParseBody()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Len(t, nodes[0].Attrs.Returns, 1)
	assert.Equal(t, "204", nodes[0].Attrs.Returns[0].Status)
	assert.Equal(t, "", nodes[0].Attrs.Returns[0].Description)
}

// --- extractBracedContent tests ---

func TestExtractBracedContent_noPrefix(t *testing.T) {
	result := extractBracedContent("@OTHER{content}", "@NOTE{")
	assert.Equal(t, "", result)
}

func TestExtractBracedContent_noClosingBrace(t *testing.T) {
	result := extractBracedContent("@NOTE{content", "@NOTE{")
	assert.Equal(t, "content", result)
}

// --- parseBlockquoteBlock edge cases ---

func TestTagParser_blockquoteBlockError(t *testing.T) {
	// Blockquote block with unclosed brace should error.
	input := "@BQ{\n@P unclosed"
	p := newTagParser(input)
	_, err := p.ParseBody()
	assert.Error(t, err)
}

func TestTagParser_blockquoteBlockInvalidChild(t *testing.T) {
	// Blockquote block with invalid child tag should error.
	input := "@BQ{\n@INVALID tag\n}"
	p := newTagParser(input)
	_, err := p.ParseBody()
	assert.Error(t, err)
}

// --- findUnescaped edge cases ---

func TestTagParser_findUnescaped(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		ch       byte
		expected int
	}{
		{"simple", "a:b", ':', 1},
		{"escaped", "a\\:b:c", ':', 4},
		{"not found", "abc", ':', -1},
		{"start", ":abc", ':', 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, findUnescaped(tt.input, tt.ch))
		})
	}
}
