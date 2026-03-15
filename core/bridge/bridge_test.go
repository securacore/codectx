package bridge

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
)

func TestHeadingBridge_DifferentSections(t *testing.T) {
	got := headingBridge(
		"Authentication > JWT Tokens > Refresh Flow",
		"Authentication > JWT Tokens > Validation",
	)

	if !strings.Contains(got, "Completed: Refresh Flow") {
		t.Errorf("expected 'Completed: Refresh Flow', got: %s", got)
	}
	if !strings.Contains(got, "Entering: Validation") {
		t.Errorf("expected 'Entering: Validation', got: %s", got)
	}
}

func TestHeadingBridge_SameHeading(t *testing.T) {
	got := headingBridge(
		"Authentication > JWT Tokens",
		"Authentication > JWT Tokens",
	)
	if got != "" {
		t.Errorf("expected empty for same heading, got: %s", got)
	}
}

func TestHeadingBridge_CompletedOnly(t *testing.T) {
	got := headingBridge(
		"Authentication > JWT Tokens > Refresh Flow",
		"Authentication > JWT Tokens",
	)
	if !strings.Contains(got, "Completed: Refresh Flow") {
		t.Errorf("expected 'Completed: Refresh Flow', got: %s", got)
	}
	if strings.Contains(got, "Entering") {
		t.Errorf("expected no 'Entering' clause, got: %s", got)
	}
}

func TestHeadingBridge_EnteringOnly(t *testing.T) {
	got := headingBridge(
		"Authentication > JWT Tokens",
		"Authentication > JWT Tokens > Validation Rules",
	)
	if !strings.Contains(got, "Entering: Validation Rules") {
		t.Errorf("expected 'Entering: Validation Rules', got: %s", got)
	}
	if strings.Contains(got, "Completed") {
		t.Errorf("expected no 'Completed' clause, got: %s", got)
	}
}

func TestHeadingBridge_CompletelyDifferent(t *testing.T) {
	got := headingBridge("Authentication", "Database Setup")
	if !strings.Contains(got, "Completed: Authentication") {
		t.Errorf("expected 'Completed: Authentication', got: %s", got)
	}
	if !strings.Contains(got, "Entering: Database Setup") {
		t.Errorf("expected 'Entering: Database Setup', got: %s", got)
	}
}

func TestHeadingBridge_EmptyHeadings(t *testing.T) {
	if got := headingBridge("", ""); got != "" {
		t.Errorf("expected empty for both empty headings, got: %s", got)
	}
}

func TestHeadingBridge_PrevEmpty(t *testing.T) {
	got := headingBridge("", "Authentication")
	if !strings.Contains(got, "Entering: Authentication") {
		t.Errorf("expected 'Entering: Authentication', got: %s", got)
	}
}

func TestHeadingBridge_NextEmpty(t *testing.T) {
	got := headingBridge("Authentication", "")
	if !strings.Contains(got, "Completed: Authentication") {
		t.Errorf("expected 'Completed: Authentication', got: %s", got)
	}
}

func TestTailWindow_ShortContent(t *testing.T) {
	text := "Short content here."
	if got := tailWindow(text, 600); got != text {
		t.Errorf("expected full text for short content, got: %s", got)
	}
}

func TestTailWindow_LongContent(t *testing.T) {
	text := strings.Repeat("word ", 200) // 1000 chars
	got := tailWindow(text, 100)
	if len(got) > 110 { // some slack for word boundary
		t.Errorf("expected ~100 chars, got %d", len(got))
	}
	// Should not start mid-word.
	if strings.HasPrefix(got, " ") {
		t.Error("tail should not start with space")
	}
}

func TestLastSentence_ProseEnding(t *testing.T) {
	text := "JWT tokens use RS256 signing. The refresh token must be rotated on every use to prevent replay attacks."
	got := lastSentence(text)
	if got != "The refresh token must be rotated on every use to prevent replay attacks." {
		t.Errorf("unexpected last sentence: %s", got)
	}
}

func TestLastSentence_CodeBlockEnding(t *testing.T) {
	text := "Here is an example:\n```go\nfunc main() {}\n```"
	got := lastSentence(text)
	if got != "" {
		t.Errorf("expected empty for code block ending, got: %s", got)
	}
}

func TestLastSentence_UnclosedCodeFence(t *testing.T) {
	text := "Some text.\n```\nunclosed code"
	got := lastSentence(text)
	if got != "" {
		t.Errorf("expected empty for unclosed code fence, got: %s", got)
	}
}

func TestLastSentence_TooShort(t *testing.T) {
	text := "Short text. OK."
	got := lastSentence(text)
	if got != "" {
		t.Errorf("expected empty for short sentence, got: %s", got)
	}
}

func TestLastSentence_HeadingLine(t *testing.T) {
	text := "Some introduction paragraph here with enough length to pass.\n# Section Header."
	got := lastSentence(text)
	// Should skip the heading and return the paragraph.
	if strings.HasPrefix(got, "#") {
		t.Errorf("should not extract heading as last sentence: %s", got)
	}
}

func TestLastSentence_EmptyInput(t *testing.T) {
	if got := lastSentence(""); got != "" {
		t.Errorf("expected empty for empty input, got: %s", got)
	}
}

func TestLastSentence_ListEnding(t *testing.T) {
	text := "Configuration options are available.\n- Option A enables caching.\n- Option B disables logging."
	got := lastSentence(text)
	if strings.HasPrefix(got, "- ") {
		t.Errorf("should not extract list item as last sentence: %s", got)
	}
}

func TestGenerate_AllLayers(t *testing.T) {
	input := BridgeInput{
		PrevHeading: "Authentication > JWT Tokens > Refresh Flow",
		NextHeading: "Authentication > JWT Tokens > Validation",
		PrevContent: "The refresh token lifecycle begins when the client presents an expired access token. " +
			"The server validates the refresh token signature using RS256 signing requirements. " +
			"Token expiry validation ensures that compromised tokens cannot be reused after their rotation window closes.",
	}

	got := generate(input)
	t.Logf("bridge: %s", got)

	if got == "" {
		t.Fatal("expected non-empty bridge")
	}

	// Should contain heading transition.
	if !strings.Contains(got, "Completed: Refresh Flow") {
		t.Error("expected heading transition in bridge")
	}

	// Should contain "Established:" from RAKE.
	if !strings.Contains(got, "Established:") {
		t.Error("expected RAKE phrases in bridge")
	}
}

func TestGenerate_SameHeading_NoContent(t *testing.T) {
	input := BridgeInput{
		PrevHeading: "Auth",
		NextHeading: "Auth",
		PrevContent: "",
	}

	got := generate(input)
	if got != "" {
		t.Errorf("expected empty bridge for same heading and no content, got: %s", got)
	}
}

func TestGenerate_OnlyHeadingTransition(t *testing.T) {
	input := BridgeInput{
		PrevHeading: "Setup",
		NextHeading: "Configuration",
		PrevContent: "ok.",
	}

	got := generate(input)
	if !strings.Contains(got, "Completed: Setup") {
		t.Errorf("expected heading transition, got: %s", got)
	}
}

// --- GenerateAll ---

func TestGenerateAll_BasicPairs(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:aaa.01", Type: chunk.ChunkObject, Source: "docs/auth.md", Heading: "Auth", Sequence: 1, Content: "Authentication overview with detailed explanation of the token lifecycle."},
		{ID: "obj:aaa.02", Type: chunk.ChunkObject, Source: "docs/auth.md", Heading: "Auth > JWT", Sequence: 2, Content: "JWT token signing uses RS256 algorithm for production security."},
		{ID: "obj:aaa.03", Type: chunk.ChunkObject, Source: "docs/auth.md", Heading: "Auth > JWT > Refresh", Sequence: 3, Content: "Refresh tokens are rotated on every use."},
		{ID: "obj:bbb.01", Type: chunk.ChunkObject, Source: "docs/database.md", Heading: "Database", Sequence: 1, Content: "Database connection pooling and query optimization strategies."},
		{ID: "obj:bbb.02", Type: chunk.ChunkObject, Source: "docs/database.md", Heading: "Database > Migrations", Sequence: 2, Content: "Schema migrations run in a transaction with automatic rollback."},
	}

	mfst := &manifest.Manifest{
		Objects: make(map[string]*manifest.ManifestEntry),
		Specs:   make(map[string]*manifest.ManifestEntry),
		System:  make(map[string]*manifest.ManifestEntry),
	}

	bridges := GenerateAll(chunks, mfst)

	// Should generate bridges for adjacent pairs within the same file:
	// auth.md: chunk 1->2, chunk 2->3 (2 bridges)
	// database.md: chunk 1->2 (1 bridge)
	if len(bridges) != 3 {
		t.Fatalf("expected 3 bridges, got %d", len(bridges))
	}

	// Verify bridges exist for the correct chunk IDs (the "prev" chunk gets the bridge).
	for _, id := range []string{"obj:aaa.01", "obj:aaa.02", "obj:bbb.01"} {
		if _, ok := bridges[id]; !ok {
			t.Errorf("expected bridge for %q", id)
		}
	}

	// Verify no bridge for last chunks in each file.
	for _, id := range []string{"obj:aaa.03", "obj:bbb.02"} {
		if _, ok := bridges[id]; ok {
			t.Errorf("unexpected bridge for last chunk %q", id)
		}
	}

	// No cross-file bridges: bbb.01 bridge should not reference auth headings.
	if b, ok := bridges["obj:bbb.01"]; ok {
		if strings.Contains(b, "Auth") {
			t.Errorf("bridge for bbb.01 should not reference auth headings, got: %s", b)
		}
	}
}

func TestGenerateAll_SkipsSpecChunks(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:aaa.01", Type: chunk.ChunkObject, Source: "docs/auth.md", Heading: "Auth", Sequence: 1, Content: "Authentication overview."},
		{ID: "spec:ccc.01", Type: chunk.ChunkSpec, Source: "docs/auth.md", Heading: "Auth", Sequence: 2, Content: "Spec reasoning for auth design decisions."},
		{ID: "obj:aaa.02", Type: chunk.ChunkObject, Source: "docs/auth.md", Heading: "Auth > JWT", Sequence: 3, Content: "JWT token signing uses RS256 algorithm."},
	}

	mfst := &manifest.Manifest{
		Objects: make(map[string]*manifest.ManifestEntry),
		Specs:   make(map[string]*manifest.ManifestEntry),
		System:  make(map[string]*manifest.ManifestEntry),
	}

	bridges := GenerateAll(chunks, mfst)

	// The spec chunk should be excluded from bridge generation.
	// Only obj:aaa.01 -> obj:aaa.02 should produce a bridge.
	if _, ok := bridges["spec:ccc.01"]; ok {
		t.Error("spec chunk should not receive a bridge")
	}

	if len(bridges) != 1 {
		t.Fatalf("expected 1 bridge (obj:aaa.01 -> obj:aaa.02), got %d", len(bridges))
	}
	if _, ok := bridges["obj:aaa.01"]; !ok {
		t.Error("expected bridge for obj:aaa.01")
	}
}

func TestGenerateAll_SkipsExistingBridges(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:aaa.01", Type: chunk.ChunkObject, Source: "docs/auth.md", Heading: "Auth", Sequence: 1, Content: "Authentication overview with detailed explanation."},
		{ID: "obj:aaa.02", Type: chunk.ChunkObject, Source: "docs/auth.md", Heading: "Auth > JWT", Sequence: 2, Content: "JWT token signing uses RS256 algorithm for security."},
		{ID: "obj:aaa.03", Type: chunk.ChunkObject, Source: "docs/auth.md", Heading: "Auth > JWT > Refresh", Sequence: 3, Content: "Refresh tokens are rotated on every use."},
	}

	existingBridge := "Existing bridge summary from cache."
	mfst := &manifest.Manifest{
		Objects: map[string]*manifest.ManifestEntry{
			"obj:aaa.01": {
				Type:         "object",
				Source:       "docs/auth.md",
				Heading:      "Auth",
				Sequence:     1,
				BridgeToNext: &existingBridge,
			},
		},
		Specs:  make(map[string]*manifest.ManifestEntry),
		System: make(map[string]*manifest.ManifestEntry),
	}

	bridges := GenerateAll(chunks, mfst)

	// obj:aaa.01 already has a bridge in the manifest — should be skipped.
	if _, ok := bridges["obj:aaa.01"]; ok {
		t.Error("should skip chunk pair where manifest already has a bridge")
	}

	// obj:aaa.02 -> obj:aaa.03 has no existing bridge — should be generated.
	if _, ok := bridges["obj:aaa.02"]; !ok {
		t.Error("expected bridge for obj:aaa.02 (no existing bridge in manifest)")
	}

	if len(bridges) != 1 {
		t.Fatalf("expected 1 bridge (skipping existing), got %d", len(bridges))
	}
}
