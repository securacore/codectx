package cmdx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultDictOpts() EncoderOptions {
	return DefaultEncoderOptions()
}

func TestBuildDictionary_noRepeats(t *testing.T) {
	segments := []string{"hello world", "foo bar baz", "unique string here"}
	dict := buildDictionary(segments, defaultDictOpts())
	assert.Nil(t, dict, "no repeated strings should produce nil dictionary")
}

func TestBuildDictionary_singleRepeat(t *testing.T) {
	repeated := "this string repeats exactly"
	segments := []string{
		"start " + repeated + " middle",
		"other " + repeated + " end",
		"also " + repeated + " here",
	}
	dict := buildDictionary(segments, defaultDictOpts())
	require.NotNil(t, dict)
	require.GreaterOrEqual(t, len(dict.Entries), 1)
	// The repeated string should be in the dictionary.
	found := false
	for _, e := range dict.Entries {
		if e.Value == repeated {
			found = true
			break
		}
	}
	assert.True(t, found, "repeated string should be in dictionary")
}

func TestBuildDictionary_multipleRepeats(t *testing.T) {
	a := "first repeated string"
	b := "second repeated string"
	segments := []string{
		a + " and " + b,
		a + " plus " + b,
		a + " with " + b,
	}
	dict := buildDictionary(segments, defaultDictOpts())
	require.NotNil(t, dict)
	assert.GreaterOrEqual(t, len(dict.Entries), 1, "should have at least one entry")
}

func TestBuildDictionary_minFrequency(t *testing.T) {
	// String appearing only once should be excluded.
	segments := []string{"this appears only once in the segment list and is long enough"}
	dict := buildDictionary(segments, defaultDictOpts())
	assert.Nil(t, dict)
}

func TestBuildDictionary_minLength(t *testing.T) {
	// Short repeated string (< 10 chars) should be excluded.
	opts := defaultDictOpts()
	opts.MinStringLength = 10
	segments := []string{"short", "short", "short"}
	dict := buildDictionary(segments, opts)
	assert.Nil(t, dict)
}

func TestBuildDictionary_scoringPositive(t *testing.T) {
	// A string with positive savings should be included.
	repeated := "this is a long enough string for dictionary"
	segments := []string{repeated, repeated, repeated, repeated, repeated}
	dict := buildDictionary(segments, defaultDictOpts())
	require.NotNil(t, dict)
	assert.GreaterOrEqual(t, len(dict.Entries), 1)
}

func TestBuildDictionary_scoringNegative(t *testing.T) {
	// A string where overhead exceeds savings should be excluded.
	// Short string appearing only twice: savings ~= len - overhead.
	opts := defaultDictOpts()
	opts.MinStringLength = 10
	opts.MinFrequency = 2
	repeated := "exactly 10" // exactly 10 chars
	segments := []string{repeated, repeated}
	dict := buildDictionary(segments, opts)
	// With only 2 occurrences of a 10-char string, savings = 10 - (10+4+4) = -8. Negative.
	if dict != nil {
		for _, e := range dict.Entries {
			assert.NotEqual(t, repeated, e.Value, "low-value string should not be selected")
		}
	}
}

func TestBuildDictionary_maxEntries(t *testing.T) {
	opts := defaultDictOpts()
	opts.MaxDictEntries = 3
	// Create many distinct repeated strings.
	var segments []string
	for i := 0; i < 10; i++ {
		segments = append(segments, "repeated string number one hundred")
		segments = append(segments, "repeated string number two hundred")
		segments = append(segments, "repeated string number three thousand")
		segments = append(segments, "repeated string number four thousand")
		segments = append(segments, "repeated string number five thousand")
	}
	dict := buildDictionary(segments, opts)
	require.NotNil(t, dict)
	assert.LessOrEqual(t, len(dict.Entries), 3, "should cap at MaxDictEntries")
}

func TestBuildDictionary_deterministic(t *testing.T) {
	segments := []string{
		"https://api.example.com/v2 is the base URL",
		"Use https://api.example.com/v2 for requests",
		"Documentation at https://api.example.com/v2/docs",
	}
	opts := defaultDictOpts()
	dict1 := buildDictionary(segments, opts)
	dict2 := buildDictionary(segments, opts)

	if dict1 == nil && dict2 == nil {
		return
	}
	require.NotNil(t, dict1)
	require.NotNil(t, dict2)
	require.Equal(t, len(dict1.Entries), len(dict2.Entries), "same input should produce same dict size")
	for i := range dict1.Entries {
		assert.Equal(t, dict1.Entries[i].Value, dict2.Entries[i].Value,
			"entry %d should be identical", i)
	}
}

func TestBuildDictionary_tieBreaking(t *testing.T) {
	// Two candidates with equal scores: first occurrence wins.
	a := "alpha candidate string"
	b := "bravo candidate string"
	segments := []string{
		a + " then " + b,
		a + " plus " + b,
		a + " with " + b,
	}
	dict := buildDictionary(segments, defaultDictOpts())
	require.NotNil(t, dict)
	if len(dict.Entries) >= 2 {
		// Entry 0 should be the one appearing first in the text.
		assert.Equal(t, a, dict.Entries[0].Value,
			"first-occurring candidate should get index 0")
	}
}

func TestBuildDictionary_codeBlockExcluded(t *testing.T) {
	// Verify collectTextSegments skips code blocks.
	nodes := []Node{
		{Tag: TagP, Children: []Node{{Tag: TagRaw, Content: "some normal text content"}}},
		{Tag: TagCodeBlock, Content: "some normal text content"},
		{Tag: TagP, Children: []Node{{Tag: TagRaw, Content: "some normal text content"}}},
	}
	segments := collectTextSegments(nodes)
	// Should have 2 segments (from the paragraphs), not 3.
	count := 0
	for _, s := range segments {
		if s == "some normal text content" {
			count++
		}
	}
	assert.Equal(t, 2, count, "code block content should be excluded from segments")
}

func TestRoundTrip_withDictionary(t *testing.T) {
	// Document with repeated strings should round-trip with dictionary.
	input := []byte(`# API Reference

The base URL is https://api.example.com/v2 for all requests.

See https://api.example.com/v2/docs for documentation.

The base URL is https://api.example.com/v2 and it requires auth.
`)
	roundTrip(t, input)
}

func TestRoundTrip_apiDocs(t *testing.T) {
	roundTrip(t, readTestdata("api_docs.md"))
}

func TestEncode_dictOutput(t *testing.T) {
	repeated := "https://api.example.com/v2"
	input := []byte("Visit [API](" + repeated + ") or [Docs](" + repeated + "/docs) today.\n\nAlso see [Auth](" + repeated + "/auth) page.\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	assert.Contains(t, output, "@DICT{", "should have dictionary block")
	assert.Contains(t, output, "$0=", "should have at least one entry")
}

func TestEncode_dictReferences(t *testing.T) {
	repeated := "https://api.example.com/v2"
	input := []byte("Visit [API](" + repeated + ") or see [Docs](" + repeated + "/docs).\n\nMore at [Auth](" + repeated + "/auth).\n")
	encoded, err := Encode(input)
	require.NoError(t, err)
	output := string(encoded)
	// The URL should be replaced with a $N reference in the body.
	assert.Contains(t, output, "$0", "body should contain $N references")
}

func TestDecode_dictExpansion(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=expanded value here\n}\n@P The $0 appears.\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "expanded value here")
}

func TestDecode_dictExpansionInURL(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=https://api.example.com\n}\n@P See @LINK{docs>$0/docs}.\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "https://api.example.com/docs")
}

func TestDecode_dollarEscaping(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=value\n}\n@P Cost is $$5 dollars.\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "$5")
}

func TestBuildDictionary_overlapHandling(t *testing.T) {
	// Two candidates overlap in text positions. Higher-scoring candidate
	// should win and the lower one should have its frequency recalculated.
	// "the quick brown fox" appears 5 times — very high score.
	// "the quick" is a substring that also appears 5 times, but its positions
	// overlap with the larger candidate. After the larger one is selected,
	// "the quick" has 0 unclaimed occurrences and should be excluded.
	long := "the quick brown fox"
	short := "the quick"
	segments := []string{
		long + " jumps over",
		long + " runs fast",
		long + " sleeps well",
		long + " eats lots",
		long + " drinks water",
	}
	opts := defaultDictOpts()
	opts.MinStringLength = 8
	dict := buildDictionary(segments, opts)
	require.NotNil(t, dict)

	// The longer candidate should be selected.
	foundLong := false
	foundShort := false
	for _, e := range dict.Entries {
		if e.Value == long {
			foundLong = true
		}
		if e.Value == short {
			foundShort = true
		}
	}
	assert.True(t, foundLong, "longer overlapping candidate should be selected")
	assert.False(t, foundShort, "shorter overlapping candidate should be excluded (positions claimed)")
}

func TestDecode_dictExpansionInKVFields(t *testing.T) {
	// $N references in KV item fields should be expanded (D7, Phase 3 prep).
	input := "@CMDX v1\n@DICT{\n  $0=string\n  $1=The description text\n}\n@KV{\n  name:$0~$1\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "string", "KV type field should have expanded $0")
	assert.Contains(t, output, "The description text", "KV description should have expanded $1")
}

func TestDecode_invalidDictRef(t *testing.T) {
	// $99 when only 1 entry — should pass through as literal.
	input := "@CMDX v1\n@DICT{\n  $0=value\n}\n@P Reference $99 here.\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "$99")
}

// --- Dict.go domain attribute coverage tests ---

func TestCollectTextSegments_endpoint(t *testing.T) {
	nodes := []Node{
		{Tag: TagEndpoint, Attrs: NodeAttrs{
			Endpoint: &EndpointDef{Method: "GET", Path: "/api/users"},
		}},
	}
	segments := collectTextSegments(nodes)
	assert.Contains(t, segments, "/api/users")
}

func TestCollectTextSegments_returns(t *testing.T) {
	nodes := []Node{
		{Tag: TagReturns, Attrs: NodeAttrs{
			Returns: []ReturnDef{
				{Status: "200", Description: "Success response"},
			},
		}},
	}
	segments := collectTextSegments(nodes)
	assert.Contains(t, segments, "200")
	assert.Contains(t, segments, "Success response")
}

func TestCollectTextSegments_params(t *testing.T) {
	nodes := []Node{
		{Tag: TagParams, Attrs: NodeAttrs{
			Params: []ParamItem{
				{Name: "userId", Type: "string", Description: "The user identifier"},
			},
		}},
	}
	segments := collectTextSegments(nodes)
	assert.Contains(t, segments, "userId")
	assert.Contains(t, segments, "string")
	assert.Contains(t, segments, "The user identifier")
}

func TestCollectTextSegments_headers(t *testing.T) {
	nodes := []Node{
		{Tag: TagTable, Attrs: NodeAttrs{
			Headers: []string{"Name", "Value"},
			Cells:   [][]string{{"foo", "bar"}},
		}},
	}
	segments := collectTextSegments(nodes)
	assert.Contains(t, segments, "Name")
	assert.Contains(t, segments, "Value")
	assert.Contains(t, segments, "foo")
	assert.Contains(t, segments, "bar")
}

func TestCollectTextSegments_callout(t *testing.T) {
	nodes := []Node{
		{Tag: TagNote, Attrs: NodeAttrs{Callout: "important note text"}},
	}
	segments := collectTextSegments(nodes)
	assert.Contains(t, segments, "important note text")
}

func TestApplyNodeDict_returns(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "Success"}},
	}
	node := Node{
		Tag: TagReturns,
		Attrs: NodeAttrs{
			Returns: []ReturnDef{{Status: "200", Description: "Success"}},
		},
	}
	applyNodeDict(&node, dict)
	assert.Equal(t, "$0", node.Attrs.Returns[0].Description)
}

func TestApplyNodeDict_params(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "string"}},
	}
	node := Node{
		Tag: TagParams,
		Attrs: NodeAttrs{
			Params: []ParamItem{{Name: "id", Type: "string", Description: "A string value"}},
		},
	}
	applyNodeDict(&node, dict)
	assert.Equal(t, "$0", node.Attrs.Params[0].Type)
	assert.Contains(t, node.Attrs.Params[0].Description, "$0")
}

func TestApplyNodeDict_headers(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "Description"}},
	}
	node := Node{
		Tag:   TagTable,
		Attrs: NodeAttrs{Headers: []string{"Name", "Description"}},
	}
	applyNodeDict(&node, dict)
	assert.Equal(t, "$0", node.Attrs.Headers[1])
}

func TestApplyNodeDict_cells(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{{Index: 0, Value: "common"}},
	}
	node := Node{
		Tag:   TagTable,
		Attrs: NodeAttrs{Cells: [][]string{{"common", "unique"}}},
	}
	applyNodeDict(&node, dict)
	assert.Equal(t, "$0", node.Attrs.Cells[0][0])
}

func TestEscapeNodeText_returns(t *testing.T) {
	node := Node{
		Tag: TagReturns,
		Attrs: NodeAttrs{
			Returns: []ReturnDef{{Status: "$200", Description: "@ sign"}},
		},
	}
	escapeNodeText(&node)
	assert.Equal(t, "$$200", node.Attrs.Returns[0].Status)
	assert.Equal(t, "@@ sign", node.Attrs.Returns[0].Description)
}

func TestEscapeNodeText_params(t *testing.T) {
	node := Node{
		Tag: TagParams,
		Attrs: NodeAttrs{
			Params: []ParamItem{{Name: "$id", Type: "@type", Description: "test"}},
		},
	}
	escapeNodeText(&node)
	assert.Equal(t, "$$id", node.Attrs.Params[0].Name)
	assert.Equal(t, "@@type", node.Attrs.Params[0].Type)
}

func TestEscapeNodeText_codeBlockSkipped(t *testing.T) {
	node := Node{
		Tag:     TagCodeBlock,
		Content: "$ and @ should not be escaped",
	}
	escapeNodeText(&node)
	assert.Equal(t, "$ and @ should not be escaped", node.Content, "code blocks should not be escaped")
}

func TestDecode_dictExpansionInParams(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=string\n}\n@PARAMS{\n  id:$0:R~User $0 ID\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "string")
	assert.Contains(t, output, "User string ID")
}

func TestDecode_dictExpansionInReturns(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=success\n}\n@RETURNS{200:$0|404:Not found}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "200: success")
}

func TestDecode_dictExpansionInEndpoint(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=/api/v2\n}\n@ENDPOINT{GET $0/users}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "### GET /api/v2/users")
}

func TestDecode_dictExpansionInHeaders(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=Description\n}\n@TABLE{\n@THEAD{Name|$0}\nfoo|bar\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "| Name | Description |")
}

func TestDecode_dictExpansionInCells(t *testing.T) {
	input := "@CMDX v1\n@DICT{\n  $0=common value\n}\n@TABLE{\n@THEAD{A|B}\n$0|unique\n}\n"
	decoded, err := Decode([]byte(input))
	require.NoError(t, err)
	output := string(decoded)
	assert.Contains(t, output, "| common value | unique |")
}
