package cmdx

import (
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
		{"# H1\n", "@H1 H1\n"},
		{"## H2\n", "@H2 H2\n"},
		{"### H3\n", "@H3 H3\n"},
		{"#### H4\n", "@H4 H4\n"},
		{"##### H5\n", "@H5 H5\n"},
		{"###### H6\n", "@H6 H6\n"},
	}
	for _, tt := range tests {
		encoded, err := Encode([]byte(tt.md))
		require.NoError(t, err)
		output := string(encoded)
		assert.Contains(t, output, tt.expected, "input: %s", tt.md)
	}
}

func TestEncode_paragraphJoins(t *testing.T) {
	// Soft-wrapped lines should be joined into a single @P.
	input := "This is a long\nparagraph that wraps.\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	// Should be a single @P line (not two).
	pLines := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "@P ") {
			pLines++
		}
	}
	assert.Equal(t, 1, pLines, "soft-wrapped paragraph should produce exactly 1 @P line")
	assert.Contains(t, output, "This is a long paragraph that wraps.")
}

func TestEncode_codeBlockPreservesContent(t *testing.T) {
	input := "```python\n  def hello():\n      print(\"Hi\")\n```\n"
	encoded, err := Encode([]byte(input))
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@CODE:python")
	assert.Contains(t, output, "  def hello():")
	assert.Contains(t, output, "      print(\"Hi\")")
	assert.Contains(t, output, "@/CODE")
}

func TestDecode_invalidHeader(t *testing.T) {
	_, err := Decode([]byte("not a cmdx file"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CMDX header")
}

func TestDecode_unknownTag(t *testing.T) {
	input := "@CMDX v1\n@FOO bar"
	_, err := Decode([]byte(input))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tag")
}

// --- Phase 3: Domain detection tests ---

func TestDetect_KVTable(t *testing.T) {
	// Table with Field/Type/Description headers → @KV{}.
	input := []byte("| Field | Type | Description |\n|-------|------|-------------|\n| id | string | User ID |\n| name | string | Display name |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@KV{", "should detect KV table")
	assert.NotContains(t, output, "@TABLE{", "should not remain as TABLE")
}

func TestDetect_KVTable_keyColumn(t *testing.T) {
	// Table with Key/Type/Description should remain TABLE because KV decodes as
	// "Field/Type/Description" — using "Key" would be lossy. Only "Field" triggers KV.
	input := []byte("| Key | Type | Description |\n|-----|------|-------------|\n| host | string | Hostname |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@TABLE{", "Key/Type/Description should remain TABLE for lossless round-trip")
}

func TestDetect_KVTable_nameColumn(t *testing.T) {
	// Table with Name/Type/Description (no Required) should remain TABLE because
	// KV decodes as "Field/Type/Description" — using "Name" would be lossy.
	input := []byte("| Name | Type | Description |\n|------|------|-------------|\n| host | string | Hostname |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@TABLE{", "Name/Type/Description without Required should remain TABLE (KV decodes as Field, not Name)")
}

func TestDetect_KVTable_caseInsensitive(t *testing.T) {
	// Headers with non-canonical case should NOT be detected (to preserve lossless round-trip).
	// The decoder outputs "Field/Type/Description", so only exact canonical case triggers detection.
	input := []byte("| field | TYPE | Description |\n|-------|------|-------------|\n| x | int | A number |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@TABLE{", "non-canonical case should remain as TABLE for lossless round-trip")
	assert.NotContains(t, output, "@KV{", "should not be detected as KV with non-canonical headers")
}

func TestDetect_ParamsTable(t *testing.T) {
	// Table with Name/Type/Required/Description → @PARAMS{}.
	input := []byte("| Name | Type | Required | Description |\n|------|------|----------|-------------|\n| id | string | Yes | User ID |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@PARAMS{", "should detect PARAMS table")
	assert.NotContains(t, output, "@TABLE{", "should not remain as TABLE")
}

func TestDetect_ParamsTable_requiredMapping(t *testing.T) {
	// Only "Yes"/"No" (case-insensitive) are accepted for lossless round-trip.
	// Yes→R, No→O.
	input := []byte("| Name | Type | Required | Description |\n|------|------|----------|-------------|\n| a | string | Yes | Desc A |\n| b | int | No | Desc B |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, ":R~", "Yes should map to R")
	assert.Contains(t, output, ":O~", "No should map to O")

	// "True"/"False" are NOT lossless (decoder outputs "Yes"/"No"), so table stays as @TABLE{}.
	input2 := []byte("| Name | Type | Required | Description |\n|------|------|----------|-------------|\n| c | bool | True | Desc C |\n| d | float | False | Desc D |\n")
	encoded2, err := Encode(input2)
	require.NoError(t, err)
	output2 := string(encoded2)
	assert.Contains(t, output2, "@TABLE{", "True/False should not convert to PARAMS")
	assert.NotContains(t, output2, "@PARAMS{", "True/False is not lossless for PARAMS")
}

func TestDetect_ParamsTable_notKV(t *testing.T) {
	// PARAMS table (has Required column) should NOT be misidentified as KV.
	input := []byte("| Name | Type | Required | Description |\n|------|------|----------|-------------|\n| x | string | Yes | Test |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@PARAMS{", "should be PARAMS, not KV")
	assert.NotContains(t, output, "@KV{", "should not be misidentified as KV")
}

func TestDetect_Endpoint(t *testing.T) {
	// H3 with HTTP method + path → @ENDPOINT{}.
	input := []byte("### GET /users/{id}\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@ENDPOINT{GET /users/{id}}", "should detect endpoint")
}

func TestDetect_Endpoint_allMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		input := []byte("### " + method + " /resource\n")
		encoded, err := Encode(input)
		require.NoError(t, err)
		output := string(encoded)
		assert.Contains(t, output, "@ENDPOINT{"+method+" /resource}", "method %s should be detected", method)
	}
}

func TestDetect_Endpoint_notH2(t *testing.T) {
	// H2 with endpoint text → NOT detected (only H3 triggers).
	input := []byte("## GET /users\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.NotContains(t, output, "@ENDPOINT{", "H2 should not trigger endpoint detection")
	assert.Contains(t, output, "@H2", "should remain as H2")
}

func TestDetect_Endpoint_noMatch(t *testing.T) {
	// H3 without HTTP method prefix → NOT detected.
	input := []byte("### User Details\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.NotContains(t, output, "@ENDPOINT{", "non-method H3 should not trigger endpoint detection")
	assert.Contains(t, output, "@H3", "should remain as H3")
}

func TestDetect_StandardTable(t *testing.T) {
	// Table whose headers don't match → remains @TABLE{}.
	input := []byte("| Animal | Sound |\n|--------|-------|\n| Cat | Meow |\n| Dog | Woof |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@TABLE{", "non-matching table should remain as TABLE")
	assert.NotContains(t, output, "@KV{", "should not be detected as KV")
	assert.NotContains(t, output, "@PARAMS{", "should not be detected as PARAMS")
}

func TestDetect_Admonition_boldNote(t *testing.T) {
	// Blockquote starting with **Note:** → @NOTE{} or @WARN{}.
	input := []byte("> **Note:** This is important.\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	// Could be @NOTE{} or @WARN{} based on prefix match.
	hasAdmonition := strings.Contains(output, "@NOTE{") || strings.Contains(output, "@WARN{")
	assert.True(t, hasAdmonition, "bold-prefix blockquote should be detected as admonition: %s", output)
	assert.NotContains(t, output, "@BQ", "should not remain as blockquote")
}

func TestDetect_Admonition_githubStyle(t *testing.T) {
	// GitHub-style callout → @NOTE{}.
	input := []byte("> [!NOTE]\n> This is a note.\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@NOTE{", "GitHub-style callout should be detected as NOTE: %s", output)
}

// --- serializeCMDX META block test ---

func TestSerializeCMDX_withMeta(t *testing.T) {
	doc := &Document{
		Version: "1",
		Meta:    map[string]string{"title": "Test"},
		Body: []Node{
			{Tag: TagP, Children: []Node{{Tag: TagRaw, Content: "Hello"}}},
		},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "@CMDX v1")
	assert.Contains(t, output, "@META{")
	assert.Contains(t, output, "title:Test")
	assert.Contains(t, output, "@P Hello")
}

func TestSerializeCMDX_metaEscaping(t *testing.T) {
	doc := &Document{
		Version: "1",
		Meta:    map[string]string{"desc": "a;b"},
		Body:    []Node{},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "desc:a\\;b")
}

// --- serializeNode coverage tests ---

func TestSerializeNode_TagBR(t *testing.T) {
	// BR as a block-level node
	doc := &Document{
		Version: "1",
		Body:    []Node{{Tag: TagBR}},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "@BR\n")
}

func TestSerializeNode_TagTip(t *testing.T) {
	doc := &Document{
		Version: "1",
		Body:    []Node{{Tag: TagTip, Attrs: NodeAttrs{Callout: "Pro tip"}}},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "@TIP{Pro tip}")
}

func TestSerializeNode_TagImportant(t *testing.T) {
	doc := &Document{
		Version: "1",
		Body:    []Node{{Tag: TagImportant, Attrs: NodeAttrs{Callout: "Critical info"}}},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "@IMPORTANT{Critical info}")
}

func TestSerializeNode_TagReturns(t *testing.T) {
	doc := &Document{
		Version: "1",
		Body: []Node{{Tag: TagReturns, Attrs: NodeAttrs{
			Returns: []ReturnDef{{Status: "200", Description: "OK"}, {Status: "404", Description: "Not Found"}},
		}}},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "@RETURNS{200:OK|404:Not Found}")
}

func TestSerializeNode_TagRaw(t *testing.T) {
	doc := &Document{
		Version: "1",
		Body:    []Node{{Tag: TagRaw, Content: "raw html content"}},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "raw html content")
}

func TestSerializeNode_TagEndpoint(t *testing.T) {
	doc := &Document{
		Version: "1",
		Body: []Node{{Tag: TagEndpoint, Attrs: NodeAttrs{
			Endpoint: &EndpointDef{Method: "POST", Path: "/api/users"},
		}}},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "@ENDPOINT{POST /api/users}")
}

func TestSerializeNode_TagDef(t *testing.T) {
	doc := &Document{
		Version: "1",
		Body: []Node{{Tag: TagDef, Attrs: NodeAttrs{
			Items: []KVItem{{Key: "API", Description: "Application Programming Interface"}},
		}}},
	}
	output := string(serializeCMDX(doc))
	assert.Contains(t, output, "@DEF{")
	assert.Contains(t, output, "API~Application Programming Interface")
}

// --- serializeInlineCMDX coverage tests ---

func TestSerializeInlineCMDX_boldItalic(t *testing.T) {
	nodes := []Node{
		{Tag: TagBoldItalic, Children: []Node{{Tag: TagRaw, Content: "emphasized"}}},
	}
	output := serializeInlineCMDX(nodes)
	assert.Equal(t, "@BI{emphasized}", output)
}

func TestSerializeInlineCMDX_strikethrough(t *testing.T) {
	nodes := []Node{
		{Tag: TagStrikethrough, Children: []Node{{Tag: TagRaw, Content: "deleted"}}},
	}
	output := serializeInlineCMDX(nodes)
	assert.Equal(t, "@S{deleted}", output)
}

func TestSerializeInlineCMDX_image(t *testing.T) {
	nodes := []Node{
		{Tag: TagImage, Attrs: NodeAttrs{URL: "https://img.png", Display: ""}, Children: []Node{{Tag: TagRaw, Content: "alt"}}},
	}
	output := serializeInlineCMDX(nodes)
	assert.Contains(t, output, "@IMG{alt>https://img.png}")
}

func TestSerializeInlineCMDX_braces(t *testing.T) {
	// Braces in raw text should be escaped
	nodes := []Node{
		{Tag: TagRaw, Content: "a{b}c"},
	}
	output := serializeInlineCMDX(nodes)
	assert.Equal(t, "a\\{b\\}c", output)
}

func TestSerializeInlineCMDX_codeWithBraces(t *testing.T) {
	// Braces in code content should be escaped
	nodes := []Node{
		{Tag: TagCode, Content: "map[string]{"},
	}
	output := serializeInlineCMDX(nodes)
	assert.Contains(t, output, "@C{map[string]\\{}")
}

// --- extractAdmonitionContent tests ---

func TestExtractAdmonitionContent_multiParagraph(t *testing.T) {
	inline := []Node{{Tag: TagRaw, Content: " First part."}}
	blocks := []Node{
		{Tag: TagP, Children: []Node{{Tag: TagRaw, Content: "Second part."}}},
	}
	result := extractAdmonitionContent(inline, blocks)
	assert.Equal(t, "First part. Second part.", result)
}

func TestExtractAdmonitionContent_emptyInline(t *testing.T) {
	result := extractAdmonitionContent(nil, nil)
	assert.Equal(t, "", result)
}

func TestExtractAdmonitionContent_onlyBlocks(t *testing.T) {
	blocks := []Node{
		{Tag: TagP, Children: []Node{{Tag: TagRaw, Content: "Block text."}}},
	}
	result := extractAdmonitionContent(nil, blocks)
	assert.Equal(t, "Block text.", result)
}

// --- isAdmonitionContentPlainText tests ---

func TestIsAdmonitionContentPlainText_allRaw(t *testing.T) {
	inline := []Node{{Tag: TagRaw, Content: "text"}}
	blocks := []Node{{Tag: TagP, Children: []Node{{Tag: TagRaw, Content: "more"}}}}
	assert.True(t, isAdmonitionContentPlainText(inline, blocks))
}

func TestIsAdmonitionContentPlainText_inlineFormatting(t *testing.T) {
	inline := []Node{{Tag: TagBold, Children: []Node{{Tag: TagRaw, Content: "bold"}}}}
	assert.False(t, isAdmonitionContentPlainText(inline, nil))
}

func TestIsAdmonitionContentPlainText_blockFormatting(t *testing.T) {
	inline := []Node{{Tag: TagRaw, Content: "text"}}
	blocks := []Node{
		{Tag: TagP, Children: []Node{
			{Tag: TagCode, Content: "code"},
		}},
	}
	assert.False(t, isAdmonitionContentPlainText(inline, blocks))
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

// --- Detect extra columns ---

func TestDetect_KVTable_extraColumns(t *testing.T) {
	// KV requires exactly 3 columns.
	input := []byte("| Field | Type | Description | Extra |\n|-------|------|-------------|-------|\n| a | b | c | d |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@TABLE{", "4-column table with Field header should remain TABLE")
	assert.NotContains(t, output, "@KV{")
}

func TestDetect_ParamsTable_extraColumns(t *testing.T) {
	// PARAMS requires exactly 4 columns.
	input := []byte("| Name | Type | Required | Description | Notes |\n|------|------|----------|-------------|-------|\n| x | s | Yes | D | N |\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@TABLE{", "5-column table should remain TABLE")
	assert.NotContains(t, output, "@PARAMS{")
}

// --- Detect endpoint with inline formatting ---

func TestDetect_Endpoint_withFormatting(t *testing.T) {
	// Endpoint detection requires purely-text content.
	input := []byte("### GET `/users/{id}`\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.NotContains(t, output, "@ENDPOINT{", "H3 with inline formatting should not be detected as endpoint")
	assert.Contains(t, output, "@H3")
}

// --- Detect admonition case sensitivity ---

// --- convertBlock: HTMLBlock and indented code block ---

func TestEncode_htmlBlock(t *testing.T) {
	// HTML blocks should be encoded as TagRaw.
	input := []byte("<div>\nsome content\n</div>\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	// HTML block should appear in output (as raw content)
	assert.Contains(t, output, "<div>")
}

func TestEncode_indentedCodeBlock(t *testing.T) {
	// Indented code block (4 spaces).
	input := []byte("Paragraph.\n\n    indented code\n    second line\n\nMore text.\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@CODE")
	assert.Contains(t, output, "indented code")
}

// --- convertInline: AutoLink and RawHTML ---

func TestEncode_autoLink(t *testing.T) {
	input := []byte("Visit <https://example.com> now.\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "https://example.com")
}

func TestEncode_rawHTML(t *testing.T) {
	input := []byte("Some <em>html</em> text.\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "<em>")
}

func TestDetect_Admonition_wrongCase(t *testing.T) {
	// Bold prefix must be exactly title-case.
	input := []byte("> **note:** lowercase note.\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.NotContains(t, output, "@NOTE{", "lowercase 'note:' should not trigger admonition")
	assert.Contains(t, output, "@BQ")
}
