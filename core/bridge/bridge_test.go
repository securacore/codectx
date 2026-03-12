package bridge

import (
	"strings"
	"testing"
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

	got := Generate(input)
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

	got := Generate(input)
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

	got := Generate(input)
	if !strings.Contains(got, "Completed: Setup") {
		t.Errorf("expected heading transition, got: %s", got)
	}
}
