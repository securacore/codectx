package tokens_test

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/tokens"
)

// ---------------------------------------------------------------------------
// New — constructor
// ---------------------------------------------------------------------------

func TestNew_Cl100kBase(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Encoding() != "cl100k_base" {
		t.Errorf("expected encoding %q, got %q", "cl100k_base", c.Encoding())
	}
}

func TestNew_O200kBase(t *testing.T) {
	c, err := tokens.New(tokens.O200kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Encoding() != "o200k_base" {
		t.Errorf("expected encoding %q, got %q", "o200k_base", c.Encoding())
	}
}

func TestNew_P50kBase(t *testing.T) {
	c, err := tokens.New(tokens.P50kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Encoding() != "p50k_base" {
		t.Errorf("expected encoding %q, got %q", "p50k_base", c.Encoding())
	}
}

func TestNew_R50kBase(t *testing.T) {
	c, err := tokens.New(tokens.R50kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Encoding() != "r50k_base" {
		t.Errorf("expected encoding %q, got %q", "r50k_base", c.Encoding())
	}
}

func TestNew_UnsupportedEncoding(t *testing.T) {
	_, err := tokens.New("unknown_encoding")
	if err == nil {
		t.Fatal("expected error for unsupported encoding")
	}
	if !strings.Contains(err.Error(), "unsupported encoding") {
		t.Errorf("error should mention unsupported encoding, got: %v", err)
	}
}

func TestNew_EmptyEncoding(t *testing.T) {
	_, err := tokens.New("")
	if err == nil {
		t.Fatal("expected error for empty encoding")
	}
	if !strings.Contains(err.Error(), "unsupported encoding") {
		t.Errorf("error should mention unsupported encoding, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Count — token counting
// ---------------------------------------------------------------------------

func TestCount_SimpleText(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n, err := c.Count("The quick brown fox jumps over the lazy dog.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n <= 0 {
		t.Errorf("expected positive token count for English prose, got %d", n)
	}
}

func TestCount_EmptyString(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n, err := c.Count("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 tokens for empty string, got %d", n)
	}
}

func TestCount_CodeBlock(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prose := "Authentication tokens are cryptographic credentials that prove identity."
	code := `func validateToken(token string) error {
	if token == "" {
		return fmt.Errorf("empty token")
	}
	claims, err := jwt.Parse(token, keyFunc)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	return claims.Valid()
}`

	proseTokens, err := c.Count(prose)
	if err != nil {
		t.Fatalf("counting prose: %v", err)
	}
	codeTokens, err := c.Count(code)
	if err != nil {
		t.Fatalf("counting code: %v", err)
	}

	// Code typically produces more tokens than prose of similar length
	// because symbols are individual tokens. The code string is ~2.5x longer
	// than the prose, but should produce disproportionately more tokens.
	if codeTokens <= proseTokens {
		t.Errorf("expected code (%d tokens) to produce more tokens than prose (%d tokens)",
			codeTokens, proseTokens)
	}
}

func TestCount_Deterministic(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := "Token counting must be deterministic for chunk ID stability."
	first, err := c.Count(text)
	if err != nil {
		t.Fatalf("first count: %v", err)
	}

	// Count the same text multiple times.
	for i := 0; i < 10; i++ {
		n, err := c.Count(text)
		if err != nil {
			t.Fatalf("count iteration %d: %v", i, err)
		}
		if n != first {
			t.Fatalf("count %d: got %d, expected %d (determinism broken)", i, n, first)
		}
	}
}

func TestCount_DifferentEncodings(t *testing.T) {
	cl, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("cl100k: %v", err)
	}
	o2, err := tokens.New(tokens.O200kBase)
	if err != nil {
		t.Fatalf("o200k: %v", err)
	}

	// Use a reasonably long text where encoding differences are measurable.
	text := strings.Repeat("The authentication system uses JWT tokens with RSA-256 signatures for secure session management. ", 10)

	clCount, err := cl.Count(text)
	if err != nil {
		t.Fatalf("cl100k count: %v", err)
	}
	o2Count, err := o2.Count(text)
	if err != nil {
		t.Fatalf("o200k count: %v", err)
	}

	// Both should produce positive counts.
	if clCount <= 0 || o2Count <= 0 {
		t.Fatalf("expected positive counts, got cl100k=%d, o200k=%d", clCount, o2Count)
	}

	// The counts should differ (different vocabularies). o200k_base has a larger
	// vocabulary than cl100k_base so it generally produces fewer tokens.
	if clCount == o2Count {
		t.Logf("warning: cl100k and o200k produced identical counts (%d) for this text", clCount)
		// This isn't necessarily wrong for all inputs, but for repeated
		// English prose it's very unlikely. Log it but don't fail.
	}
}

func TestCount_KnownValue(t *testing.T) {
	// "Hello, world!" with cl100k_base should produce a known token count.
	// tiktoken reference: "Hello, world!" → [9906, 11, 1917, 0] → 4 tokens
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n, err := c.Count("Hello, world!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 tokens for 'Hello, world!' with cl100k_base, got %d", n)
	}
}

func TestCount_LongText(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ~1000 repetitions of a sentence = substantial text.
	text := strings.Repeat("Documentation compilers transform raw markdown into indexed, searchable content. ", 1000)
	n, err := c.Count(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n < 1000 {
		t.Errorf("expected at least 1000 tokens for large text, got %d", n)
	}
}

func TestCount_UnicodeText(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unicode text — CJK characters, emoji, etc.
	n, err := c.Count("This has unicode: cafe\u0301 and Chinese: \u4f60\u597d\u4e16\u754c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n <= 0 {
		t.Errorf("expected positive token count for unicode text, got %d", n)
	}
}

func TestCount_WhitespaceOnly(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n, err := c.Count("   \n\t\n   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Whitespace still produces tokens (spaces/tabs/newlines are tokenized).
	if n <= 0 {
		t.Errorf("expected positive token count for whitespace, got %d", n)
	}
}

// ---------------------------------------------------------------------------
// Encoding — accessor
// ---------------------------------------------------------------------------

func TestEncoding_ReturnsConstructorValue(t *testing.T) {
	encodings := []string{
		tokens.Cl100kBase,
		tokens.O200kBase,
		tokens.P50kBase,
		tokens.R50kBase,
	}
	for _, enc := range encodings {
		c, err := tokens.New(enc)
		if err != nil {
			t.Fatalf("New(%q): %v", enc, err)
		}
		if c.Encoding() != enc {
			t.Errorf("expected Encoding()=%q, got %q", enc, c.Encoding())
		}
	}
}

// ---------------------------------------------------------------------------
// CountBlocks — document annotation
// ---------------------------------------------------------------------------

func TestCountBlocks_PopulatesTokens(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := markdown.Parse([]byte("# Authentication\n\nJWT tokens provide stateless authentication.\n\n```go\nfunc validate() error {\n\treturn nil\n}\n```\n"))

	if err := tokens.CountBlocks(doc, c); err != nil {
		t.Fatalf("CountBlocks: %v", err)
	}

	for i, b := range doc.Blocks {
		if b.Content != "" && b.Tokens <= 0 {
			t.Errorf("block %d (%s): expected positive token count, got %d", i, b.Type, b.Tokens)
		}
	}
}

func TestCountBlocks_TotalTokens(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := markdown.Parse([]byte("# Title\n\nFirst paragraph with some content.\n\nSecond paragraph with more content.\n"))

	if err := tokens.CountBlocks(doc, c); err != nil {
		t.Fatalf("CountBlocks: %v", err)
	}

	// Verify TotalTokens equals the sum of individual block tokens.
	sum := 0
	for _, b := range doc.Blocks {
		sum += b.Tokens
	}

	if doc.TotalTokens != sum {
		t.Errorf("TotalTokens (%d) != sum of block tokens (%d)", doc.TotalTokens, sum)
	}
	if doc.TotalTokens <= 0 {
		t.Errorf("expected positive TotalTokens, got %d", doc.TotalTokens)
	}
}

func TestCountBlocks_EmptyDocument(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := markdown.Parse([]byte(""))

	if err := tokens.CountBlocks(doc, c); err != nil {
		t.Fatalf("CountBlocks: %v", err)
	}

	if doc.TotalTokens != 0 {
		t.Errorf("expected TotalTokens=0 for empty doc, got %d", doc.TotalTokens)
	}
}

func TestCountBlocks_PreservesOtherFields(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := markdown.Parse([]byte("# Title\n\nSome content here.\n\n```go\nfmt.Println(\"hello\")\n```\n"))

	// Snapshot field values before counting.
	type snapshot struct {
		Type     markdown.BlockType
		Content  string
		Level    int
		Heading  []string
		Position int
		Language string
	}
	before := make([]snapshot, len(doc.Blocks))
	for i, b := range doc.Blocks {
		heading := make([]string, len(b.Heading))
		copy(heading, b.Heading)
		before[i] = snapshot{
			Type:     b.Type,
			Content:  b.Content,
			Level:    b.Level,
			Heading:  heading,
			Position: b.Position,
			Language: b.Language,
		}
	}

	if err := tokens.CountBlocks(doc, c); err != nil {
		t.Fatalf("CountBlocks: %v", err)
	}

	// Verify all non-Tokens fields are unchanged.
	for i, b := range doc.Blocks {
		s := before[i]
		if b.Type != s.Type {
			t.Errorf("block %d: Type changed from %v to %v", i, s.Type, b.Type)
		}
		if b.Content != s.Content {
			t.Errorf("block %d: Content changed", i)
		}
		if b.Level != s.Level {
			t.Errorf("block %d: Level changed from %d to %d", i, s.Level, b.Level)
		}
		if b.Position != s.Position {
			t.Errorf("block %d: Position changed from %d to %d", i, s.Position, b.Position)
		}
		if b.Language != s.Language {
			t.Errorf("block %d: Language changed from %q to %q", i, s.Language, b.Language)
		}
		if len(b.Heading) != len(s.Heading) {
			t.Errorf("block %d: Heading length changed from %d to %d", i, len(s.Heading), len(b.Heading))
		} else {
			for j := range b.Heading {
				if b.Heading[j] != s.Heading[j] {
					t.Errorf("block %d: Heading[%d] changed from %q to %q", i, j, s.Heading[j], b.Heading[j])
				}
			}
		}
	}
}

func TestCountBlocks_OverwritesPreviousCounts(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := markdown.Parse([]byte("# Title\n\nContent paragraph.\n"))

	// Count once.
	if err := tokens.CountBlocks(doc, c); err != nil {
		t.Fatalf("first CountBlocks: %v", err)
	}
	firstTotal := doc.TotalTokens

	// Count again — should produce identical results.
	if err := tokens.CountBlocks(doc, c); err != nil {
		t.Fatalf("second CountBlocks: %v", err)
	}

	if doc.TotalTokens != firstTotal {
		t.Errorf("second CountBlocks produced different TotalTokens: %d vs %d",
			doc.TotalTokens, firstTotal)
	}
}

func TestCountBlocks_AllBlockTypes(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Document with every block type.
	input := `# Heading

A paragraph of text.

- List item one
- List item two

> A blockquote with content.

` + "```go\nfunc main() {}\n```" + `

| Col A | Col B |
|-------|-------|
| val1  | val2  |
`

	doc := markdown.Parse([]byte(input))
	if err := tokens.CountBlocks(doc, c); err != nil {
		t.Fatalf("CountBlocks: %v", err)
	}

	// Every block with content should have a positive token count.
	for i, b := range doc.Blocks {
		if b.Content != "" && b.Tokens <= 0 {
			t.Errorf("block %d (%s): expected positive tokens, got %d (content: %q)",
				i, b.Type, b.Tokens, b.Content)
		}
	}

	if doc.TotalTokens <= 0 {
		t.Errorf("expected positive TotalTokens, got %d", doc.TotalTokens)
	}
}

func TestCountBlocks_StrippedDocument(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse, strip, then count — the normal pipeline order.
	doc := markdown.Parse([]byte("### Deep Heading\n\n---\n\nSome content.\n\n<!-- comment -->\n"))
	stripped := markdown.Strip(doc)

	if err := tokens.CountBlocks(stripped, c); err != nil {
		t.Fatalf("CountBlocks on stripped doc: %v", err)
	}

	// Stripped doc should have fewer blocks (thematic break and comment removed).
	if len(stripped.Blocks) >= len(doc.Blocks) {
		t.Errorf("strip should remove blocks: before=%d, after=%d",
			len(doc.Blocks), len(stripped.Blocks))
	}

	// But the remaining blocks should all have token counts.
	for i, b := range stripped.Blocks {
		if b.Content != "" && b.Tokens <= 0 {
			t.Errorf("stripped block %d (%s): expected positive tokens, got %d",
				i, b.Type, b.Tokens)
		}
	}

	if stripped.TotalTokens <= 0 {
		t.Errorf("expected positive TotalTokens on stripped doc, got %d", stripped.TotalTokens)
	}
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func TestEncodingConstants(t *testing.T) {
	if tokens.Cl100kBase != "cl100k_base" {
		t.Errorf("Cl100kBase = %q, want %q", tokens.Cl100kBase, "cl100k_base")
	}
	if tokens.O200kBase != "o200k_base" {
		t.Errorf("O200kBase = %q, want %q", tokens.O200kBase, "o200k_base")
	}
	if tokens.P50kBase != "p50k_base" {
		t.Errorf("P50kBase = %q, want %q", tokens.P50kBase, "p50k_base")
	}
	if tokens.R50kBase != "r50k_base" {
		t.Errorf("R50kBase = %q, want %q", tokens.R50kBase, "r50k_base")
	}
}

func TestCountBlocks_NilDocument(t *testing.T) {
	c, err := tokens.New(tokens.Cl100kBase)
	if err != nil {
		t.Fatalf("creating counter: %v", err)
	}

	err = tokens.CountBlocks(nil, c)
	if err == nil {
		t.Fatal("expected error for nil document")
	}
}
