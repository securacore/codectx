package cmdx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecode_KVToTable(t *testing.T) {
	// @KV{} → markdown table with Field/Type/Description headers.
	input := "@CMDX v1\n@KV{\n  id:string~User ID\n  name:string~Display name\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "| Field | Type | Description |")
	assert.Contains(t, output, "| id | string | User ID |")
	assert.Contains(t, output, "| name | string | Display name |")
}

func TestDecode_ParamsToTable(t *testing.T) {
	// @PARAMS{} → markdown table with Name/Type/Required/Description headers.
	input := "@CMDX v1\n@PARAMS{\n  id:string:R~User ID\n  name:string:O~Display name\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "| Name | Type | Required | Description |")
	assert.Contains(t, output, "| id | string | Yes | User ID |")
	assert.Contains(t, output, "| name | string | No | Display name |")
}

func TestDecode_EndpointToHeading(t *testing.T) {
	// @ENDPOINT{GET /path} → ### GET /path.
	input := "@CMDX v1\n@ENDPOINT{GET /users/{id}}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "### GET /users/{id}")
}

func TestDecode_ReturnsToText(t *testing.T) {
	// @RETURNS{200:OK|404:Not found} → appropriate markdown.
	input := "@CMDX v1\n@RETURNS{200:OK|404:Not found}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "200: OK")
	assert.Contains(t, output, "404: Not found")
}

func TestDecode_DefToTable(t *testing.T) {
	// @DEF{} → markdown table with Term/Definition headers.
	input := "@CMDX v1\n@DEF{\n  API~Application Programming Interface\n  URL~Uniform Resource Locator\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "| Term | Definition |")
	assert.Contains(t, output, "| API | Application Programming Interface |")
	assert.Contains(t, output, "| URL | Uniform Resource Locator |")
}

func TestDecode_NoteToBlockquote(t *testing.T) {
	// @NOTE{text} → > **Note:** text.
	input := "@CMDX v1\n@NOTE{This is important.}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "> **Note:** This is important.")
}

// --- META block tests ---

func TestParse_metaBlock(t *testing.T) {
	input := "@CMDX v1\n@META{title:My Doc;author:Jane}\n@P Hello\n"
	doc, err := Parse([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, "My Doc", doc.Meta["title"])
	assert.Equal(t, "Jane", doc.Meta["author"])
	assert.Len(t, doc.Body, 1)
}

func TestParse_metaBlockEscapedSemicolon(t *testing.T) {
	input := "@CMDX v1\n@META{desc:value\\;with\\;semicolons}\n@P Hello\n"
	doc, err := Parse([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, "value;with;semicolons", doc.Meta["desc"])
}

func TestParse_metaBlockMultipleKeys(t *testing.T) {
	input := "@CMDX v1\n@META{a:1;b:2;c:3}\n@P x\n"
	doc, err := Parse([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, "1", doc.Meta["a"])
	assert.Equal(t, "2", doc.Meta["b"])
	assert.Equal(t, "3", doc.Meta["c"])
}

func TestParse_metaBlockEmptyValue(t *testing.T) {
	input := "@CMDX v1\n@META{key:}\n@P x\n"
	doc, err := Parse([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, "", doc.Meta["key"])
}

func TestSplitUnescapedSemicolon(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a;b;c", []string{"a", "b", "c"}},
		{`a\;b;c`, []string{`a\;b`, "c"}},
		{"single", []string{"single"}},
		{"", []string{""}},
		{`a\;b\;c`, []string{`a\;b\;c`}},
		{";", []string{"", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitUnescapedSemicolon(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecode_metaRoundTrip(t *testing.T) {
	// META block should be preserved through encode→decode roundtrip.
	// (META is only in CMDX; it's not in markdown, so just test parse→serialize.)
	input := "@CMDX v1\n@META{title:Test}\n@P Hello world\n"
	doc, err := Parse([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, "Test", doc.Meta["title"])

	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "Hello world")
}

// --- Decode admonitions: Tip, Warn, Important ---

func TestDecode_TipToBlockquote(t *testing.T) {
	input := "@CMDX v1\n@TIP{Use this shortcut.}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "> **Tip:** Use this shortcut.")
}

func TestDecode_WarnToBlockquote(t *testing.T) {
	input := "@CMDX v1\n@WARN{Be careful here.}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "> **Warning:** Be careful here.")
}

func TestDecode_ImportantToBlockquote(t *testing.T) {
	input := "@CMDX v1\n@IMPORTANT{Do not skip this step.}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "> **Important:** Do not skip this step.")
}

// --- escapeLeadingBlockMarker tests ---

func TestEscapeLeadingBlockMarker(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"heading hash space", "# heading", "\\# heading"},
		{"heading hash alone", "#", "\\#"},
		{"heading double hash", "## sub", "\\## sub"},
		{"hash no space", "#tag", "#tag"},
		{"plus space", "+ item", "\\+ item"},
		{"plus tab", "+\titem", "\\+\titem"},
		{"plus alone", "+", "\\+"},
		{"plus no space", "+1", "+1"},
		{"dash space", "- item", "\\- item"},
		{"dash tab", "-\titem", "\\-\titem"},
		{"dash alone", "-", "\\-"},
		{"star space", "* item", "\\* item"},
		{"star tab", "*\titem", "\\*\titem"},
		{"star alone", "*", "*"}, // bare * is emphasis, not list
		{"blockquote", "> text", "\\> text"},
		{"blockquote alone", ">", "\\>"},
		{"ordered dot", "1. item", "\\1. item"},
		{"ordered paren", "2) item", "\\2) item"},
		{"ordered dot tab", "1.\titem", "\\1.\titem"},
		{"ordered paren tab", "3)\titem", "\\3)\titem"},
		{"ordered dot alone", "1.", "\\1."},
		{"ordered paren alone", "2)", "\\2)"},
		{"normal text", "hello world", "hello world"},
		{"empty", "", ""},
		{"digit no marker", "123abc", "123abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, escapeLeadingBlockMarker(tt.input))
		})
	}
}

// --- escapeMDURL tests ---

func TestEscapeMDURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no escape needed", "https://example.com", "https://example.com"},
		{"parens", "https://en.wikipedia.org/wiki/Go_(programming)", "https://en.wikipedia.org/wiki/Go_%28programming%29"},
		{"space", "https://example.com/my path", "https://example.com/my%20path"},
		{"tab", "https://example.com/\tpath", "https://example.com/%09path"},
		{"backslash", "https://example.com/a\\b", "https://example.com/a%5Cb"},
		{"bracket-paren sequence", "https://example.com/](other", "https://example.com/]%28other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, escapeMDURL(tt.input))
		})
	}
}

// --- emphDelimiters tests ---

func TestEmphDelimiters_parentSameChar(t *testing.T) {
	var buf strings.Builder
	open, close := emphDelimiters("*", '*', &buf)
	assert.Equal(t, "_", open)
	assert.Equal(t, "_", close)
}

func TestEmphDelimiters_parentDifferentChar(t *testing.T) {
	var buf strings.Builder
	open, close := emphDelimiters("*", '_', &buf)
	assert.Equal(t, "*", open)
	assert.Equal(t, "*", close)
}

func TestEmphDelimiters_bufferEndsWithSame(t *testing.T) {
	var buf strings.Builder
	buf.WriteString("text*")
	open, close := emphDelimiters("**", 0, &buf)
	assert.Equal(t, "__", open)
	assert.Equal(t, "__", close)
}

func TestEmphDelimiters_bufferEndsWithEscaped(t *testing.T) {
	var buf strings.Builder
	buf.WriteString("text\\*")
	open, close := emphDelimiters("*", 0, &buf)
	// Escaped char — should NOT switch
	assert.Equal(t, "*", open)
	assert.Equal(t, "*", close)
}

func TestEmphDelimiters_emptyBuffer(t *testing.T) {
	var buf strings.Builder
	open, close := emphDelimiters("**", 0, &buf)
	assert.Equal(t, "**", open)
	assert.Equal(t, "**", close)
}

// --- serializeTableMD tests ---

func TestSerializeTableMD_emptyHeaders(t *testing.T) {
	var buf strings.Builder
	serializeTableMD(&buf, nil, nil, "")
	assert.Equal(t, "", buf.String())
}

func TestSerializeTableMD_shortRow(t *testing.T) {
	var buf strings.Builder
	serializeTableMD(&buf, []string{"A", "B", "C"}, [][]string{{"x"}}, "")
	output := buf.String()
	assert.Contains(t, output, "| A | B | C |")
	// Row should be padded to header length
	assert.Contains(t, output, "| x |  |  |")
}

func TestSerializeTableMD_withPrefix(t *testing.T) {
	var buf strings.Builder
	serializeTableMD(&buf, []string{"H"}, [][]string{{"v"}}, "> ")
	output := buf.String()
	assert.Contains(t, output, "> | H |")
	assert.Contains(t, output, "> | v |")
}

// --- Parse error tests ---

func TestParse_invalidHeader(t *testing.T) {
	_, err := Parse([]byte("not cmdx"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CMDX header")
}

func TestParse_empty(t *testing.T) {
	_, err := Parse([]byte(""))
	assert.Error(t, err)
}

// --- Decode with dictionary tests ---

func TestDecode_withDictionary(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=application\n}\n@P The $0 is running.\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "The application is running.")
}

func TestDecode_dictWithEscapedDollar(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=test\n}\n@P Cost is $$5 and $0.\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "Cost is $5 and test.")
}

// --- Decode Returns with status-only ---

func TestDecode_ReturnsStatusOnly(t *testing.T) {
	input := "@CMDX v1\n@RETURNS{204}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "204")
}

// --- Decode BR block ---

func TestDecode_BRBlock(t *testing.T) {
	input := "@CMDX v1\n@BR\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "  \n")
}

// --- Decode empty blockquote ---

func TestDecode_EmptyBlockquote(t *testing.T) {
	input := "@CMDX v1\n@BQ{\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), ">\n")
}

// --- Decode multi-child blockquote ---

func TestDecode_MultiChildBlockquote(t *testing.T) {
	input := "@CMDX v1\n@BQ{\n@P First paragraph\n@P Second paragraph\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "> First paragraph")
	assert.Contains(t, output, "> Second paragraph")
}

// --- Decode setext headings (H1/H2 with hard breaks) ---

func TestDecode_SetextHeadingH1(t *testing.T) {
	// A heading with a hard break should use setext style for H1.
	input := "@CMDX v1\n@H1 line one@BR{}line two\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "=")
	assert.Contains(t, output, "line one")
	assert.Contains(t, output, "line two")
}

// --- expandNodeDict tests for domain attributes ---

func TestExpandNodeDict_params(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "string"}},
	}
	node := Node{
		Tag: TagParams,
		Attrs: NodeAttrs{
			Params: []ParamItem{
				{Name: "id", Type: "$0", Required: true, Description: "The $0 ID"},
			},
		},
	}
	expandNodeDict(&node, dict)
	assert.Equal(t, "string", node.Attrs.Params[0].Type)
	assert.Equal(t, "The string ID", node.Attrs.Params[0].Description)
}

func TestExpandNodeDict_returns(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "success"}},
	}
	node := Node{
		Tag: TagReturns,
		Attrs: NodeAttrs{
			Returns: []ReturnDef{
				{Status: "200", Description: "$0"},
			},
		},
	}
	expandNodeDict(&node, dict)
	assert.Equal(t, "success", node.Attrs.Returns[0].Description)
}

func TestExpandNodeDict_endpoint(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "/users"}},
	}
	node := Node{
		Tag: TagEndpoint,
		Attrs: NodeAttrs{
			Endpoint: &EndpointDef{Method: "GET", Path: "$0/{id}"},
		},
	}
	expandNodeDict(&node, dict)
	assert.Equal(t, "/users/{id}", node.Attrs.Endpoint.Path)
}

func TestExpandNodeDict_headers(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "Name"}},
	}
	node := Node{
		Tag:   TagTable,
		Attrs: NodeAttrs{Headers: []string{"$0", "Value"}},
	}
	expandNodeDict(&node, dict)
	assert.Equal(t, "Name", node.Attrs.Headers[0])
}

// --- Decode BoldItalic inline ---

func TestDecode_BoldItalicInline(t *testing.T) {
	input := "@CMDX v1\n@P @BI{bold italic text}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "***bold italic text***")
}

// --- Decode code span with backticks ---

func TestDecode_codeSpanWithBackticks(t *testing.T) {
	input := "@CMDX v1\n@P use @C{has ` backtick}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	// Uses `` delimiters to avoid backtick conflict. Padding is added
	// only when content starts/ends with backtick.
	assert.Contains(t, output, "``has ` backtick``")
}

// --- Decode code span with leading/trailing spaces ---

func TestDecode_codeSpanSpacePadding(t *testing.T) {
	input := "@CMDX v1\n@P use @C{ padded }\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	// Content with leading+trailing space should get padded with extra space.
	assert.Contains(t, output, "`  padded  `")
}

// --- Decode image ---

func TestDecode_imageInline(t *testing.T) {
	input := "@CMDX v1\n@P see @IMG{alt text>https://img.png}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "![alt text](https://img.png)")
}

// --- Decode hard break after backslash ---

func TestDecode_hardBreakAfterBackslash(t *testing.T) {
	input := "@CMDX v1\n@P text\\\\@BR{}next\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	// After a backslash, BR should use "  \n" form
	assert.Contains(t, output, "next")
}

// --- Decode hard break with block marker following ---

func TestDecode_hardBreakFollowedByMarker(t *testing.T) {
	input := "@CMDX v1\n@P first@BR{}# heading-like\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	// The # after BR should be escaped
	assert.Contains(t, output, "\\#")
}

// --- Decode strikethrough adjacent to tilde ---

func TestDecode_strikethroughAdjacentTilde(t *testing.T) {
	input := "@CMDX v1\n@P @S{deleted}~more\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	// Should have space between ~~ and ~ to prevent ambiguity
	assert.Contains(t, output, "~~ ~more")
}

// --- Decode code fence with backtick in language ---

func TestDecode_codeFenceTildeLanguage(t *testing.T) {
	input := "@CMDX v1\n@CODE:my`lang\ncode\n@/CODE\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "~~~")
	assert.Contains(t, output, "my`lang")
}

// --- Decode tilde-preceded strikethrough ---

func TestDecode_tildePrecededStrikethrough(t *testing.T) {
	input := "@CMDX v1\n@P text~@S{deleted}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	// Should have space between ~ and ~~ to prevent ~~~
	assert.Contains(t, output, "~ ~~deleted~~")
}

func TestExpandNodeDict_cells(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "foo"}},
	}
	node := Node{
		Tag:   TagTable,
		Attrs: NodeAttrs{Cells: [][]string{{"$0", "bar"}}},
	}
	expandNodeDict(&node, dict)
	assert.Equal(t, "foo", node.Attrs.Cells[0][0])
}
