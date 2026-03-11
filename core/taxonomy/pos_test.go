package taxonomy

import (
	"strings"
	"testing"

	prose "github.com/zuvaai/prose/v3"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/markdown"
)

// ---------------------------------------------------------------------------
// stripMarkdown
// ---------------------------------------------------------------------------

func TestStripMarkdown_Bold(t *testing.T) {
	input := "**Connection Pool** manages database connections."
	got := stripMarkdown(input)
	if strings.Contains(got, "**") {
		t.Errorf("expected bold markers removed, got %q", got)
	}
	if !strings.Contains(got, "Connection Pool") {
		t.Errorf("expected bold text preserved, got %q", got)
	}
}

func TestStripMarkdown_UnderscoreBold(t *testing.T) {
	input := "__Service Mesh__ provides network communication."
	got := stripMarkdown(input)
	if strings.Contains(got, "__") {
		t.Errorf("expected underscore bold markers removed, got %q", got)
	}
	if !strings.Contains(got, "Service Mesh") {
		t.Errorf("expected bold text preserved, got %q", got)
	}
}

func TestStripMarkdown_Link(t *testing.T) {
	input := "See [authentication docs](auth.md) for details."
	got := stripMarkdown(input)
	if strings.Contains(got, "[") || strings.Contains(got, "]") {
		t.Errorf("expected link brackets removed, got %q", got)
	}
	if !strings.Contains(got, "authentication docs") {
		t.Errorf("expected link text preserved, got %q", got)
	}
	if strings.Contains(got, "auth.md") {
		t.Errorf("expected link URL removed, got %q", got)
	}
}

func TestStripMarkdown_InlineCode(t *testing.T) {
	input := "Use `fmt.Println` to print output."
	got := stripMarkdown(input)
	if strings.Contains(got, "`") {
		t.Errorf("expected inline code markers removed, got %q", got)
	}
}

func TestStripMarkdown_CodeFence(t *testing.T) {
	input := "Before.\n\n```go\nfunc main() {}\n```\n\nAfter."
	got := stripMarkdown(input)
	if strings.Contains(got, "func main") {
		t.Errorf("expected code fence content removed, got %q", got)
	}
	if !strings.Contains(got, "Before") || !strings.Contains(got, "After") {
		t.Errorf("expected surrounding text preserved, got %q", got)
	}
}

func TestStripMarkdown_HeadingMarkers(t *testing.T) {
	input := "## Authentication\n\nContent here."
	got := stripMarkdown(input)
	if strings.Contains(got, "##") {
		t.Errorf("expected heading markers removed, got %q", got)
	}
	if !strings.Contains(got, "Authentication") {
		t.Errorf("expected heading text preserved, got %q", got)
	}
}

func TestStripMarkdown_HTMLTags(t *testing.T) {
	input := "<div>Some <b>text</b></div>"
	got := stripMarkdown(input)
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Errorf("expected HTML tags removed, got %q", got)
	}
	if !strings.Contains(got, "Some") || !strings.Contains(got, "text") {
		t.Errorf("expected text preserved, got %q", got)
	}
}

func TestStripMarkdown_Empty(t *testing.T) {
	if got := stripMarkdown(""); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestStripMarkdown_ListMarkers(t *testing.T) {
	input := "- Item one\n- Item two\n1. Numbered item"
	got := stripMarkdown(input)
	if !strings.Contains(got, "Item one") {
		t.Errorf("expected list item text preserved, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// isNoun / isProperNoun / isAdjective
// ---------------------------------------------------------------------------

func TestIsNoun(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"NN", true},
		{"NNS", true},
		{"NNP", true},
		{"NNPS", true},
		{"VB", false},
		{"JJ", false},
		{"DT", false},
	}
	for _, tt := range tests {
		if got := isNoun(tt.tag); got != tt.want {
			t.Errorf("isNoun(%q) = %v, want %v", tt.tag, got, tt.want)
		}
	}
}

func TestIsProperNoun(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"NNP", true},
		{"NNPS", true},
		{"NN", false},
		{"NNS", false},
		{"JJ", false},
	}
	for _, tt := range tests {
		if got := isProperNoun(tt.tag); got != tt.want {
			t.Errorf("isProperNoun(%q) = %v, want %v", tt.tag, got, tt.want)
		}
	}
}

func TestIsAdjective(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"JJ", true},
		{"JJR", true},
		{"JJS", true},
		{"NN", false},
		{"VB", false},
	}
	for _, tt := range tests {
		if got := isAdjective(tt.tag); got != tt.want {
			t.Errorf("isAdjective(%q) = %v, want %v", tt.tag, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// isValidPOSTerm
// ---------------------------------------------------------------------------

func TestIsValidPOSTerm_TooShort(t *testing.T) {
	tokens := []prose.Token{{Text: "ab", Tag: "NNP"}}
	if isValidPOSTerm("ab", tokens) {
		t.Error("expected short term to be invalid")
	}
}

func TestIsValidPOSTerm_TooLong(t *testing.T) {
	long := strings.Repeat("word ", 20)
	tokens := []prose.Token{
		{Text: "very", Tag: "JJ"},
		{Text: long, Tag: "NN"},
	}
	if isValidPOSTerm("very "+long, tokens) {
		t.Error("expected long term to be invalid")
	}
}

func TestIsValidPOSTerm_SingleCommonNoun(t *testing.T) {
	tokens := []prose.Token{{Text: "server", Tag: "NN"}}
	if isValidPOSTerm("server", tokens) {
		t.Error("expected single common noun to be invalid")
	}
}

func TestIsValidPOSTerm_SingleProperNoun(t *testing.T) {
	tokens := []prose.Token{{Text: "Kubernetes", Tag: "NNP"}}
	if !isValidPOSTerm("Kubernetes", tokens) {
		t.Error("expected single proper noun to be valid")
	}
}

func TestIsValidPOSTerm_SingleProperNounStopWord(t *testing.T) {
	tokens := []prose.Token{{Text: "Example", Tag: "NNP"}}
	if isValidPOSTerm("Example", tokens) {
		t.Error("expected stop-word proper noun to be invalid")
	}
}

func TestIsValidPOSTerm_CompoundNounPhrase(t *testing.T) {
	tokens := []prose.Token{
		{Text: "middleware", Tag: "NN"},
		{Text: "chain", Tag: "NN"},
	}
	if !isValidPOSTerm("middleware chain", tokens) {
		t.Error("expected compound noun phrase to be valid")
	}
}

func TestIsValidPOSTerm_AdjectiveNoun(t *testing.T) {
	tokens := []prose.Token{
		{Text: "distributed", Tag: "JJ"},
		{Text: "system", Tag: "NN"},
	}
	if !isValidPOSTerm("distributed system", tokens) {
		t.Error("expected adjective+noun phrase to be valid")
	}
}

// ---------------------------------------------------------------------------
// appendNounPhrases
// ---------------------------------------------------------------------------

func TestAppendNounPhrases_CompoundTerm(t *testing.T) {
	tokens := []prose.Token{
		{Text: "The", Tag: "DT"},
		{Text: "distributed", Tag: "JJ"},
		{Text: "message", Tag: "NN"},
		{Text: "queue", Tag: "NN"},
		{Text: "handles", Tag: "VBZ"},
		{Text: "events", Tag: "NNS"},
	}

	candidates := appendNounPhrases(nil, tokens, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[strings.ToLower(c.canonical)] = true
	}

	if !found["distributed message queue"] {
		t.Errorf("expected 'distributed message queue', got candidates: %v", candidateNames(candidates))
	}
}

func TestAppendNounPhrases_ProperNoun(t *testing.T) {
	tokens := []prose.Token{
		{Text: "Use", Tag: "VB"},
		{Text: "PostgreSQL", Tag: "NNP"},
		{Text: "for", Tag: "IN"},
		{Text: "storage", Tag: "NN"},
	}

	candidates := appendNounPhrases(nil, tokens, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[strings.ToLower(c.canonical)] = true
	}

	if !found["postgresql"] {
		t.Errorf("expected 'PostgreSQL', got candidates: %v", candidateNames(candidates))
	}
}

func TestAppendNounPhrases_SkipsSingleCommonNouns(t *testing.T) {
	tokens := []prose.Token{
		{Text: "The", Tag: "DT"},
		{Text: "server", Tag: "NN"},
		{Text: "runs", Tag: "VBZ"},
	}

	candidates := appendNounPhrases(nil, tokens, "chunk1")

	for _, c := range candidates {
		if strings.ToLower(c.canonical) == "server" {
			t.Error("expected single common noun 'server' to be filtered")
		}
	}
}

func TestAppendNounPhrases_Deduplicates(t *testing.T) {
	tokens := []prose.Token{
		{Text: "error", Tag: "NN"},
		{Text: "handling", Tag: "NN"},
		{Text: "and", Tag: "CC"},
		{Text: "error", Tag: "NN"},
		{Text: "handling", Tag: "NN"},
	}

	candidates := appendNounPhrases(nil, tokens, "chunk1")

	count := 0
	for _, c := range candidates {
		if strings.ToLower(c.canonical) == "error handling" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("expected 1 'error handling', got %d", count)
	}
}

func TestAppendNounPhrases_EmptyTokens(t *testing.T) {
	candidates := appendNounPhrases(nil, nil, "chunk1")
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for nil tokens, got %d", len(candidates))
	}
}

// ---------------------------------------------------------------------------
// appendNamedEntities
// ---------------------------------------------------------------------------

func TestAppendNamedEntities_Basic(t *testing.T) {
	entities := []prose.Entity{
		{Text: "Google Cloud", Label: "GPE"},
		{Text: "Amazon Web Services", Label: "GPE"},
	}

	candidates := appendNamedEntities(nil, entities, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[strings.ToLower(c.canonical)] = true
	}

	if !found["google cloud"] {
		t.Error("expected 'Google Cloud' entity")
	}
	if !found["amazon web services"] {
		t.Error("expected 'Amazon Web Services' entity")
	}
}

func TestAppendNamedEntities_SkipsShort(t *testing.T) {
	entities := []prose.Entity{
		{Text: "US", Label: "GPE"},
	}

	candidates := appendNamedEntities(nil, entities, "chunk1")
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for short entity, got %d", len(candidates))
	}
}

func TestAppendNamedEntities_SkipsStopWords(t *testing.T) {
	entities := []prose.Entity{
		{Text: "example", Label: "GPE"},
	}

	candidates := appendNamedEntities(nil, entities, "chunk1")
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for stop-word entity, got %d", len(candidates))
	}
}

func TestAppendNamedEntities_Deduplicates(t *testing.T) {
	entities := []prose.Entity{
		{Text: "Kubernetes", Label: "GPE"},
		{Text: "Kubernetes", Label: "GPE"},
	}

	candidates := appendNamedEntities(nil, entities, "chunk1")
	if len(candidates) != 1 {
		t.Errorf("expected 1 deduplicated entity, got %d", len(candidates))
	}
}

// ---------------------------------------------------------------------------
// extractPOS (integration with prose library)
// ---------------------------------------------------------------------------

func TestExtractPOS_BasicIntegration(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID:      "obj:pos.1",
			Type:    chunk.ChunkObject,
			Source:  "test.md",
			Content: "The dependency injection framework provides automatic service resolution.",
		},
	}

	candidates := extractPOS(chunks)

	// We expect at least some noun phrases to be extracted.
	if len(candidates) == 0 {
		t.Error("expected at least one POS candidate from prose text")
	}

	// All candidates should have SourcePOS.
	for _, c := range candidates {
		if c.source != SourcePOS {
			t.Errorf("expected source %q, got %q for %q", SourcePOS, c.source, c.canonical)
		}
	}
}

func TestExtractPOS_EmptyContent(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID:      "obj:pos.2",
			Type:    chunk.ChunkObject,
			Source:  "test.md",
			Content: "",
		},
	}

	candidates := extractPOS(chunks)
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for empty content, got %d", len(candidates))
	}
}

func TestExtractPOS_NilChunks(t *testing.T) {
	candidates := extractPOS(nil)
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for nil chunks, got %d", len(candidates))
	}
}

func TestExtractPOS_StripsMarkdownBeforeTagging(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID:      "obj:pos.3",
			Type:    chunk.ChunkObject,
			Source:  "test.md",
			Content: "The **connection pool** manages database connections.\n\n```go\nfunc main() {}\n```",
			Blocks: []markdown.Block{
				{Type: markdown.BlockParagraph, Content: "The **connection pool** manages database connections."},
				{Type: markdown.BlockCodeBlock, Content: "func main() {}", Language: "go"},
			},
		},
	}

	candidates := extractPOS(chunks)

	// Should not have any candidates containing markdown artifacts.
	for _, c := range candidates {
		if strings.Contains(c.canonical, "**") {
			t.Errorf("candidate contains markdown bold markers: %q", c.canonical)
		}
		if strings.Contains(c.canonical, "```") {
			t.Errorf("candidate contains code fence markers: %q", c.canonical)
		}
	}
}

// ---------------------------------------------------------------------------
// buildPhrase
// ---------------------------------------------------------------------------

func TestBuildPhrase(t *testing.T) {
	tokens := []prose.Token{
		{Text: "distributed"},
		{Text: "message"},
		{Text: "queue"},
	}
	got := buildPhrase(tokens)
	if got != "distributed message queue" {
		t.Errorf("expected %q, got %q", "distributed message queue", got)
	}
}

func TestBuildPhrase_Single(t *testing.T) {
	tokens := []prose.Token{{Text: "Kubernetes"}}
	got := buildPhrase(tokens)
	if got != "Kubernetes" {
		t.Errorf("expected %q, got %q", "Kubernetes", got)
	}
}

// ---------------------------------------------------------------------------
// extractPOS — large input (many sentences)
// ---------------------------------------------------------------------------

func TestExtractPOS_LargeInput(t *testing.T) {
	// Build a chunk with many sentences to verify no truncation occurs.
	var sb strings.Builder
	for range 50 {
		sb.WriteString("The distributed message queue handles asynchronous event processing. ")
		sb.WriteString("The authentication middleware validates incoming requests. ")
	}

	chunks := []chunk.Chunk{
		{
			ID:      "obj:large.1",
			Type:    chunk.ChunkObject,
			Source:  "test.md",
			Content: sb.String(),
		},
	}

	candidates := extractPOS(chunks)

	// With 100 sentences of prose, we should get at least some candidates.
	if len(candidates) == 0 {
		t.Error("expected at least one POS candidate from large input")
	}

	// All candidates should have correct source.
	for _, c := range candidates {
		if c.source != SourcePOS {
			t.Errorf("expected source %q, got %q for %q", SourcePOS, c.source, c.canonical)
		}
	}
}

// ---------------------------------------------------------------------------
// extractPOS — code-only chunk (no prose text)
// ---------------------------------------------------------------------------

func TestExtractPOS_CodeOnlyChunk(t *testing.T) {
	// Content that is entirely code blocks — after stripping, should be
	// empty or have very little prose for POS tagging.
	chunks := []chunk.Chunk{
		{
			ID:      "obj:code.1",
			Type:    chunk.ChunkObject,
			Source:  "test.md",
			Content: "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n\n```python\ndef foo():\n    return 42\n```",
		},
	}

	candidates := extractPOS(chunks)

	// Code-only content may produce zero or very few candidates after
	// markdown stripping removes the code fences. This is graceful handling.
	for _, c := range candidates {
		if c.source != SourcePOS {
			t.Errorf("expected source %q, got %q for %q", SourcePOS, c.source, c.canonical)
		}
	}
}

// ---------------------------------------------------------------------------
// isValidPOSTerm — all stop words phrase
// ---------------------------------------------------------------------------

func TestIsValidPOSTerm_AllStopWordsPhrase(t *testing.T) {
	tokens := []prose.Token{
		{Text: "example", Tag: "NN"},
		{Text: "case", Tag: "NN"},
	}
	if isValidPOSTerm("example case", tokens) {
		t.Error("expected all-stop-words phrase to be invalid")
	}
}

// ---------------------------------------------------------------------------
// helper
// ---------------------------------------------------------------------------

func candidateNames(cs []candidate) []string {
	names := make([]string, len(cs))
	for i, c := range cs {
		names[i] = c.canonical
	}
	return names
}
