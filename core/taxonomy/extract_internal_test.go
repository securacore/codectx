package taxonomy

import (
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/markdown"
)

// ---------------------------------------------------------------------------
// sourceRank
// ---------------------------------------------------------------------------

func TestSourceRank(t *testing.T) {
	tests := []struct {
		source string
		want   int
	}{
		{SourceHeading, 0},
		{SourceCodeIdentifier, 1},
		{SourceBoldTerm, 2},
		{SourceStructuredPosition, 3},
		{SourcePOS, 4},
		{"unknown", 5},
	}

	for _, tt := range tests {
		got := sourceRank(tt.source)
		if got != tt.want {
			t.Errorf("sourceRank(%q) = %d, want %d", tt.source, got, tt.want)
		}
	}

	// Verify ordering: heading < code < bold < structured < pos.
	if sourceRank(SourceHeading) >= sourceRank(SourceCodeIdentifier) {
		t.Error("heading should rank higher (lower number) than code identifier")
	}
	if sourceRank(SourceCodeIdentifier) >= sourceRank(SourceBoldTerm) {
		t.Error("code identifier should rank higher than bold term")
	}
	if sourceRank(SourceBoldTerm) >= sourceRank(SourceStructuredPosition) {
		t.Error("bold term should rank higher than structured position")
	}
	if sourceRank(SourceStructuredPosition) >= sourceRank(SourcePOS) {
		t.Error("structured position should rank higher than POS extraction")
	}
}

// ---------------------------------------------------------------------------
// isDefinitionPosition
// ---------------------------------------------------------------------------

func TestIsDefinitionPosition(t *testing.T) {
	tests := []struct {
		content string
		offset  int
		want    bool
	}{
		// At start of content.
		{"**term** description", 0, true},
		// After newline (start of line).
		{"first line\n**term** description", 11, true},
		// After list marker.
		{"- **term** description", 2, true},
		// After whitespace only.
		{"  **term** description", 2, true},
		// In the middle of a sentence.
		{"This is a **term** in context", 10, false},
		// After numbered list marker.
		{"1. **term** description", 3, true},
	}

	for _, tt := range tests {
		got := isDefinitionPosition(tt.content, tt.offset)
		if got != tt.want {
			t.Errorf("isDefinitionPosition(%q, %d) = %v, want %v", tt.content, tt.offset, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// isListMarker
// ---------------------------------------------------------------------------

func TestIsListMarker(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"-", true},
		{"*", true},
		{"+", true},
		{"1.", true},
		{"2.", true},
		{"10.", true},
		{"1)", true},
		{"word", false},
		{"- text", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isListMarker(tt.input)
		if got != tt.want {
			t.Errorf("isListMarker(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// isGenericTableHeader
// ---------------------------------------------------------------------------

func TestIsGenericTableHeader(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Name", true},
		{"name", true},
		{"Type", true},
		{"Description", true},
		{"Default", true},
		{"Authentication", false},
		{"OAuth", false},
		{"Environment", false},
	}

	for _, tt := range tests {
		got := isGenericTableHeader(tt.input)
		if got != tt.want {
			t.Errorf("isGenericTableHeader(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// looksLikeSentence
// ---------------------------------------------------------------------------

func TestLooksLikeSentence(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// Short terms are never sentences.
		{"JWT", false},
		{"OAuth 2.0", false},
		// Long text with verbs is a sentence.
		{"This is a very long description that should be considered a sentence", true},
		// Long text without common verbs is not a sentence.
		{"Very Long Technical Architecture Documentation Reference Guide", false},
	}

	for _, tt := range tests {
		got := looksLikeSentence(tt.input)
		if got != tt.want {
			t.Errorf("looksLikeSentence(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// appendCodeIdentifiers
// ---------------------------------------------------------------------------

func TestAppendCodeIdentifiers_Go(t *testing.T) {
	block := markdown.Block{
		Type:     markdown.BlockCodeBlock,
		Content:  "func HandleRequest(ctx context.Context) error {\n\treturn nil\n}\n\ntype UserService struct {\n\tdb *sql.DB\n}",
		Language: "go",
	}

	candidates := appendCodeIdentifiers(nil, block, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[c.canonical] = true
	}

	if !found["HandleRequest"] {
		t.Error("expected HandleRequest")
	}
	if !found["UserService"] {
		t.Error("expected UserService")
	}
}

func TestAppendCodeIdentifiers_Python(t *testing.T) {
	block := markdown.Block{
		Type:     markdown.BlockCodeBlock,
		Content:  "class AuthProvider:\n    pass\n\ndef validate_token(token):\n    return True",
		Language: "python",
	}

	candidates := appendCodeIdentifiers(nil, block, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[c.canonical] = true
	}

	if !found["AuthProvider"] {
		t.Error("expected AuthProvider")
	}
	if !found["validate_token"] {
		t.Error("expected validate_token")
	}
}

func TestAppendCodeIdentifiers_ShortNamesFiltered(t *testing.T) {
	block := markdown.Block{
		Type:    markdown.BlockCodeBlock,
		Content: "func fn(x int) int {\n\treturn x\n}",
	}

	candidates := appendCodeIdentifiers(nil, block, "chunk1")

	for _, c := range candidates {
		if c.canonical == "fn" || c.canonical == "x" {
			t.Errorf("short identifier %q should be filtered", c.canonical)
		}
	}
}

func TestAppendCodeIdentifiers_EmptyContent(t *testing.T) {
	block := markdown.Block{
		Type:    markdown.BlockCodeBlock,
		Content: "",
	}

	candidates := appendCodeIdentifiers(nil, block, "chunk1")
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates for empty code, got %d", len(candidates))
	}
}

// ---------------------------------------------------------------------------
// Phase 4.3: JS/TS, Rust, Java code identifiers
// ---------------------------------------------------------------------------

func TestAppendCodeIdentifiers_JavaScript(t *testing.T) {
	block := markdown.Block{
		Type:     markdown.BlockCodeBlock,
		Content:  "function processOrder(items) {\n  return items;\n}\n\nclass OrderService {\n  constructor() {}\n}\n\nexport function validateInput(data) {\n  return true;\n}",
		Language: "javascript",
	}

	candidates := appendCodeIdentifiers(nil, block, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[c.canonical] = true
	}

	if !found["processOrder"] {
		t.Error("expected processOrder from JS function")
	}
	if !found["OrderService"] {
		t.Error("expected OrderService from JS class")
	}
	if !found["validateInput"] {
		t.Error("expected validateInput from JS export function")
	}
}

func TestAppendCodeIdentifiers_Rust(t *testing.T) {
	block := markdown.Block{
		Type:     markdown.BlockCodeBlock,
		Content:  "pub fn calculate_total(items: &[Item]) -> f64 {\n    0.0\n}\n\nstruct HttpClient {\n    base_url: String,\n}\n\nenum RequestStatus {\n    Pending,\n    Complete,\n}\n\ntrait Serializable {\n    fn serialize(&self) -> Vec<u8>;\n}\n\nimpl HttpClient {\n    fn new() -> Self { todo!() }\n}",
		Language: "rust",
	}

	candidates := appendCodeIdentifiers(nil, block, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[c.canonical] = true
	}

	if !found["calculate_total"] {
		t.Error("expected calculate_total from Rust pub fn")
	}
	if !found["HttpClient"] {
		t.Error("expected HttpClient from Rust struct")
	}
	if !found["RequestStatus"] {
		t.Error("expected RequestStatus from Rust enum")
	}
	if !found["Serializable"] {
		t.Error("expected Serializable from Rust trait")
	}
}

func TestAppendCodeIdentifiers_Java(t *testing.T) {
	block := markdown.Block{
		Type:     markdown.BlockCodeBlock,
		Content:  "public class UserRepository {\n    public void save(User user) {}\n}\n\npublic interface AuthProvider {\n    boolean authenticate(String token);\n}\n\npublic abstract class BaseService {\n    protected abstract void init();\n}",
		Language: "java",
	}

	candidates := appendCodeIdentifiers(nil, block, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[c.canonical] = true
	}

	if !found["UserRepository"] {
		t.Error("expected UserRepository from Java public class")
	}
	if !found["AuthProvider"] {
		t.Error("expected AuthProvider from Java public interface")
	}
	if !found["BaseService"] {
		t.Error("expected BaseService from Java public abstract class")
	}
}

// ---------------------------------------------------------------------------
// Phase 4.4: Go method receiver pattern
// ---------------------------------------------------------------------------

func TestAppendCodeIdentifiers_GoMethodReceiver(t *testing.T) {
	block := markdown.Block{
		Type:     markdown.BlockCodeBlock,
		Content:  "func (s *Server) HandleConnection(conn net.Conn) error {\n\treturn nil\n}",
		Language: "go",
	}

	candidates := appendCodeIdentifiers(nil, block, "chunk1")

	found := make(map[string]bool)
	for _, c := range candidates {
		found[c.canonical] = true
	}

	if !found["HandleConnection"] {
		t.Error("expected HandleConnection from Go method receiver pattern")
	}
}

// ---------------------------------------------------------------------------
// extractStructural
// ---------------------------------------------------------------------------

func TestExtractStructural_AllBlockTypes(t *testing.T) {
	chunks := []chunk.Chunk{
		{
			ID:     "obj:test.1",
			Type:   chunk.ChunkObject,
			Source: "test.md",
			Blocks: []markdown.Block{
				{Type: markdown.BlockHeading, Content: "Overview", Level: 1, Heading: []string{"Overview"}},
				{Type: markdown.BlockParagraph, Content: "**Key Concept** is important."},
				{Type: markdown.BlockCodeBlock, Content: "func ProcessData(input string) {}", Language: "go"},
				{Type: markdown.BlockList, Content: "- Error Handler: manages errors\n- Logger: handles logging"},
				{Type: markdown.BlockTable, Content: "| Feature | Enabled |\n|---|---|\n| Auth | yes |"},
			},
		},
	}

	candidates := extractStructural(chunks)

	sources := make(map[string]int)
	for _, c := range candidates {
		sources[c.source]++
	}

	if sources[SourceHeading] == 0 {
		t.Error("expected heading candidates")
	}
	if sources[SourceCodeIdentifier] == 0 {
		t.Error("expected code identifier candidates")
	}
}

// ---------------------------------------------------------------------------
// resolveLink
// ---------------------------------------------------------------------------

func TestResolveLink(t *testing.T) {
	tests := []struct {
		source     string
		linkTarget string
		want       string
	}{
		{"docs/topics/auth.md", "../middleware.md", "docs/middleware.md"},
		{"docs/topics/auth.md", "./oauth.md", "docs/topics/oauth.md"},
		{"docs/topics/auth.md", "oauth.md", "docs/topics/oauth.md"},
		{"docs/topics/auth.md", "#section", ""},
		{"docs/topics/auth.md", "https://example.com", ""},
		{"docs/topics/auth.md", "other.md#section", "docs/topics/other.md"},
		{"auth.md", "other.md", "other.md"},
	}

	for _, tt := range tests {
		got := resolveLink(tt.source, tt.linkTarget)
		if got != tt.want {
			t.Errorf("resolveLink(%q, %q) = %q, want %q", tt.source, tt.linkTarget, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// normalizePath
// ---------------------------------------------------------------------------

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"docs/topics/../middleware.md", "docs/middleware.md"},
		{"docs/topics/./oauth.md", "docs/topics/oauth.md"},
		{"a/b/c/../../d.md", "a/d.md"},
		{"simple.md", "simple.md"},
		{"../above.md", "above.md"},
	}

	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// deduplicate
// ---------------------------------------------------------------------------

func TestDeduplicate_MergesChunks(t *testing.T) {
	candidates := []candidate{
		{canonical: "Authentication", source: SourceHeading, chunkID: "obj:a.1"},
		{canonical: "Authentication", source: SourceHeading, chunkID: "obj:b.1"},
	}

	terms := deduplicate(candidates, 1)

	term := terms["authentication"]
	if term == nil {
		t.Fatal("expected 'authentication' term")
	}
	if len(term.Chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(term.Chunks))
	}
}

func TestDeduplicate_HigherConfidenceWins(t *testing.T) {
	candidates := []candidate{
		{canonical: "auth", source: SourceCodeIdentifier, chunkID: "obj:a.1"},
		{canonical: "Auth", source: SourceHeading, chunkID: "obj:b.1"},
	}

	terms := deduplicate(candidates, 1)

	term := terms["auth"]
	if term == nil {
		t.Fatal("expected 'auth' term")
	}
	if term.Source != SourceHeading {
		t.Errorf("expected source %q, got %q", SourceHeading, term.Source)
	}
	if term.Canonical != "Auth" {
		t.Errorf("expected canonical %q, got %q", "Auth", term.Canonical)
	}
}

func TestDeduplicate_FrequencyFilter(t *testing.T) {
	candidates := []candidate{
		{canonical: "Rare", source: SourceHeading, chunkID: "obj:a.1"},
		{canonical: "Common", source: SourceHeading, chunkID: "obj:a.1"},
		{canonical: "Common", source: SourceHeading, chunkID: "obj:b.1"},
	}

	terms := deduplicate(candidates, 2)

	if terms["rare"] != nil {
		t.Error("'rare' should be filtered (appears in only 1 chunk)")
	}
	if terms["common"] == nil {
		t.Error("'common' should survive (appears in 2 chunks)")
	}
}

func TestDeduplicate_EmptyKey(t *testing.T) {
	candidates := []candidate{
		{canonical: "", source: SourceHeading, chunkID: "obj:a.1"},
		{canonical: "   ", source: SourceHeading, chunkID: "obj:a.1"},
	}

	terms := deduplicate(candidates, 1)

	if len(terms) != 0 {
		t.Errorf("expected 0 terms for empty keys, got %d", len(terms))
	}
}

// ---------------------------------------------------------------------------
// buildChunkTermsMap
// ---------------------------------------------------------------------------

func TestBuildChunkTermsMap(t *testing.T) {
	terms := map[string]*Term{
		"auth": {
			Chunks: []string{"obj:a.1", "obj:b.1"},
		},
		"oauth": {
			Chunks: []string{"obj:a.1"},
		},
	}

	result := buildChunkTermsMap(terms)

	if len(result["obj:a.1"]) != 2 {
		t.Errorf("expected 2 terms for obj:a.1, got %d", len(result["obj:a.1"]))
	}
	if len(result["obj:b.1"]) != 1 {
		t.Errorf("expected 1 term for obj:b.1, got %d", len(result["obj:b.1"]))
	}

	// Should be sorted.
	terms1 := result["obj:a.1"]
	if terms1[0] != "auth" || terms1[1] != "oauth" {
		t.Errorf("expected sorted [auth, oauth], got %v", terms1)
	}
}
