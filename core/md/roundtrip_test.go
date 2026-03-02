package md

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// roundTrip encodes markdown -> compact markdown and compares the ASTs
// of the original and encoded output. Since the encoder normalizes markdown
// without changing semantics, the ASTs should be identical.
func roundTrip(t *testing.T, input []byte) {
	t.Helper()
	encoded, err := Encode(input)
	require.NoError(t, err, "encode failed")

	equal, diff, err := CompareASTs(input, encoded)
	require.NoError(t, err, "compare failed")
	require.True(t, equal, "round-trip mismatch:\noriginal:\n%s\nencoded:\n%s\ndiff:\n%s",
		string(input), string(encoded), diff)
}

func readTestdata(name string) []byte {
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		panic("missing testdata/" + name + ": " + err.Error())
	}
	return data
}

func TestRoundTrip_simple(t *testing.T) {
	roundTrip(t, readTestdata("simple.md"))
}

func TestRoundTrip_headingsOnly(t *testing.T) {
	input := []byte("# H1\n\n## H2\n\n### H3\n\n#### H4\n\n##### H5\n\n###### H6\n")
	roundTrip(t, input)
}

func TestRoundTrip_paragraph(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		roundTrip(t, []byte("Hello world.\n"))
	})
	t.Run("multi", func(t *testing.T) {
		roundTrip(t, []byte("First paragraph.\n\nSecond paragraph.\n"))
	})
	t.Run("soft-wrapped", func(t *testing.T) {
		roundTrip(t, []byte("This is a long\nparagraph that wraps.\n"))
	})
}

func TestRoundTrip_inlineFormatting(t *testing.T) {
	roundTrip(t, []byte("Some **bold**, *italic*, and `code` text.\n"))
}

func TestRoundTrip_nestedInline(t *testing.T) {
	roundTrip(t, []byte("Some **bold with *italic* inside** text.\n"))
}

func TestRoundTrip_links(t *testing.T) {
	roundTrip(t, []byte("A [link](https://example.com) here.\n"))
}

func TestRoundTrip_images(t *testing.T) {
	roundTrip(t, []byte("An ![alt text](https://example.com/img.png) here.\n"))
}

func TestRoundTrip_codeBlock(t *testing.T) {
	t.Run("with-language", func(t *testing.T) {
		roundTrip(t, []byte("```go\nfunc main() {}\n```\n"))
	})
	t.Run("without-language", func(t *testing.T) {
		roundTrip(t, []byte("```\nhello world\n```\n"))
	})
}

func TestRoundTrip_codeBlockWithAtSign(t *testing.T) {
	input := []byte("```go\n@SomeAnnotation\nfunc main() {}\n```\n")
	roundTrip(t, input)
}

func TestRoundTrip_codeBlockWithSpecialContent(t *testing.T) {
	input := []byte("```\nspecial content\nstill inside code block\n```\n")
	roundTrip(t, input)
}

func TestRoundTrip_unorderedList(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		roundTrip(t, []byte("- Item one\n- Item two\n- Item three\n"))
	})
	t.Run("nested", func(t *testing.T) {
		roundTrip(t, []byte("- Parent\n  - Child one\n  - Child two\n- Another\n"))
	})
}

func TestRoundTrip_orderedList(t *testing.T) {
	roundTrip(t, []byte("1. First\n2. Second\n3. Third\n"))
}

func TestRoundTrip_mixedLists(t *testing.T) {
	roundTrip(t, []byte("- Unordered\n  1. Ordered child\n  2. Another\n- More\n"))
}

func TestRoundTrip_blockquote(t *testing.T) {
	t.Run("single-line", func(t *testing.T) {
		roundTrip(t, []byte("> Quoted text.\n"))
	})
	t.Run("multi-line", func(t *testing.T) {
		roundTrip(t, []byte("> Line one.\n> Line two.\n"))
	})
}

func TestRoundTrip_table(t *testing.T) {
	input := []byte("| Name | Value |\n|------|-------|\n| foo | bar |\n| baz | qux |\n")
	roundTrip(t, input)
}

func TestRoundTrip_horizontalRule(t *testing.T) {
	roundTrip(t, []byte("Above\n\n---\n\nBelow\n"))
}

func TestRoundTrip_empty(t *testing.T) {
	encoded, err := Encode([]byte(""))
	require.NoError(t, err)

	// Empty input should produce empty (or minimal) output.
	equal, _, err := CompareASTs([]byte(""), encoded)
	require.NoError(t, err)
	require.True(t, equal)
}

func TestRoundTrip_atSignInText(t *testing.T) {
	roundTrip(t, []byte("Email me at user@example.com today.\n"))
}

func TestRoundTrip_dollarSignInText(t *testing.T) {
	roundTrip(t, []byte("The cost is $5 and $PATH is set.\n"))
}

func TestRoundTrip_strikethrough(t *testing.T) {
	roundTrip(t, []byte("That's ~~not~~ all.\n"))
}

func TestRoundTrip_apiDocsFullPipeline(t *testing.T) {
	roundTrip(t, readTestdata("api_docs.md"))
}

func TestRoundTrip_mixedContent(t *testing.T) {
	input := []byte(`# Mixed Content

## API Fields

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| name | string | Display name |

## Other Data

| Animal | Sound |
|--------|-------|
| Cat | Meow |
| Dog | Woof |

## Notes

Some regular paragraph text.

` + "```go\nfunc main() {}\n```\n")
	roundTrip(t, input)
}

func TestRoundTrip_indentedCodeBlock(t *testing.T) {
	input := []byte("Some text.\n\n    line one\n    line two\n\nMore text.\n")
	roundTrip(t, input)
}

func TestRoundTrip_boldItalic(t *testing.T) {
	roundTrip(t, []byte("This is ***bold and italic*** text.\n"))
}

func TestRoundTrip_hardBreak(t *testing.T) {
	roundTrip(t, []byte("Line one\\\nLine two\n"))
}

func TestRoundTrip_hardBreakTrailingSpaces(t *testing.T) {
	roundTrip(t, []byte("Line one  \nLine two\n"))
}

func TestRoundTrip_nestedBlockquote(t *testing.T) {
	roundTrip(t, []byte("> First paragraph.\n>\n> Second paragraph.\n"))
}

func TestRoundTrip_warningAdmonition(t *testing.T) {
	roundTrip(t, []byte("> **Warning:** Be careful.\n"))
}

func TestRoundTrip_tipAdmonition(t *testing.T) {
	roundTrip(t, []byte("> **Tip:** Try this shortcut.\n"))
}

func TestRoundTrip_importantAdmonition(t *testing.T) {
	roundTrip(t, []byte("> **Important:** Do not skip.\n"))
}

func TestRoundTrip_endpointH3(t *testing.T) {
	roundTrip(t, []byte("### GET /users/{id}\n"))
}

func TestRoundTrip_defTable(t *testing.T) {
	input := []byte("| Term | Definition |\n|------|------------|\n| API | Application Programming Interface |\n| URL | Uniform Resource Locator |\n")
	roundTrip(t, input)
}

func TestRoundTrip_paramsTable(t *testing.T) {
	input := []byte("| Name | Type | Required | Description |\n|------|------|----------|-------------|\n| id | string | Yes | User ID |\n| name | string | No | Display name |\n")
	roundTrip(t, input)
}

func TestRoundTrip_consecutiveStrikethrough(t *testing.T) {
	roundTrip(t, []byte("~~first~~ and ~~second~~\n"))
}

func TestRoundTrip_linkWithFormattedText(t *testing.T) {
	roundTrip(t, []byte("[**bold link**](https://example.com)\n"))
}

func TestRoundTrip_imageInParagraph(t *testing.T) {
	roundTrip(t, []byte("Before ![alt](https://img.png) after.\n"))
}

func TestRoundTrip_codeBlockWithBackticks(t *testing.T) {
	roundTrip(t, []byte("````\n```\nnested fences\n```\n````\n"))
}

func TestRoundTrip_escapedCharacters(t *testing.T) {
	roundTrip(t, []byte("Escaped \\*asterisks\\* and \\[brackets\\]\n"))
}
