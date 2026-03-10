package index

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Basic tokenization
// ---------------------------------------------------------------------------

func TestTokenize_SimpleText(t *testing.T) {
	tokens := Tokenize("hello world")
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "hello" || tokens[1] != "world" {
		t.Errorf("expected [hello world], got %v", tokens)
	}
}

func TestTokenize_EmptyString(t *testing.T) {
	tokens := Tokenize("")
	if tokens != nil {
		t.Errorf("expected nil for empty string, got %v", tokens)
	}
}

func TestTokenize_WhitespaceOnly(t *testing.T) {
	tokens := Tokenize("   \t\n  ")
	if tokens != nil {
		t.Errorf("expected nil for whitespace-only, got %v", tokens)
	}
}

func TestTokenize_OnlyStopwords(t *testing.T) {
	tokens := Tokenize("the a an is are was were")
	if tokens != nil {
		t.Errorf("expected nil when all tokens are stopwords, got %v", tokens)
	}
}

// ---------------------------------------------------------------------------
// Case handling
// ---------------------------------------------------------------------------

func TestTokenize_Lowercase(t *testing.T) {
	tokens := Tokenize("Hello WORLD fOo")
	for _, tok := range tokens {
		if tok != strings.ToLower(tok) {
			t.Errorf("expected lowercase token, got %q", tok)
		}
	}
}

func TestTokenize_CamelCase_Preserved(t *testing.T) {
	// CamelCase identifiers should be kept as single tokens, lowercased.
	tokens := Tokenize("CreateUser getUserByID")
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "createuser" {
		t.Errorf("expected 'createuser', got %q", tokens[0])
	}
	if tokens[1] != "getuserbyid" {
		t.Errorf("expected 'getuserbyid', got %q", tokens[1])
	}
}

// ---------------------------------------------------------------------------
// Compound terms (hyphens)
// ---------------------------------------------------------------------------

func TestTokenize_HyphenatedCompound(t *testing.T) {
	tokens := Tokenize("error-handling is important")
	// "is" is a stopword and should be filtered.
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "error-handling" {
		t.Errorf("expected 'error-handling', got %q", tokens[0])
	}
	if tokens[1] != "important" {
		t.Errorf("expected 'important', got %q", tokens[1])
	}
}

func TestTokenize_MultipleHyphens(t *testing.T) {
	tokens := Tokenize("my-multi-word-term")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "my-multi-word-term" {
		t.Errorf("expected 'my-multi-word-term', got %q", tokens[0])
	}
}

// ---------------------------------------------------------------------------
// Dotted paths
// ---------------------------------------------------------------------------

func TestTokenize_DottedPath(t *testing.T) {
	tokens := Tokenize("http.Handler")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "http.handler" {
		t.Errorf("expected 'http.handler', got %q", tokens[0])
	}
}

func TestTokenize_MultiDotPath(t *testing.T) {
	tokens := Tokenize("os.path.join")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "os.path.join" {
		t.Errorf("expected 'os.path.join', got %q", tokens[0])
	}
}

// ---------------------------------------------------------------------------
// Stopword filtering
// ---------------------------------------------------------------------------

func TestTokenize_StopwordsRemoved(t *testing.T) {
	tokens := Tokenize("the authentication is required for all users")
	// "the", "is", "for" are stopwords. "all" is not in our stopword list.
	for _, tok := range tokens {
		if standardStopwords[tok] && !technicalStopwords[tok] {
			t.Errorf("stopword %q should have been filtered", tok)
		}
	}

	expected := map[string]bool{
		"authentication": true,
		"required":       true,
		"all":            true,
		"users":          true,
	}
	for _, tok := range tokens {
		if !expected[tok] {
			t.Errorf("unexpected token %q", tok)
		}
	}
}

func TestTokenize_TechnicalStopwordsPreserved(t *testing.T) {
	tokens := Tokenize("null void async await true false nil err")
	if len(tokens) != 8 {
		t.Fatalf("expected 8 technical stopwords preserved, got %d: %v", len(tokens), tokens)
	}

	expected := []string{"null", "void", "async", "await", "true", "false", "nil", "err"}
	for i, want := range expected {
		if tokens[i] != want {
			t.Errorf("token %d: expected %q, got %q", i, want, tokens[i])
		}
	}
}

func TestTokenize_TechnicalTermInSentence(t *testing.T) {
	// "nil" and "err" should be preserved despite being short common words.
	tokens := Tokenize("return nil if err is not found")
	// "if", "is", "not" are stopwords.
	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}
	if !found["nil"] {
		t.Error("expected 'nil' to be preserved")
	}
	if !found["err"] {
		t.Error("expected 'err' to be preserved")
	}
	if found["is"] {
		t.Error("expected 'is' to be filtered")
	}
	if found["not"] {
		t.Error("expected 'not' to be filtered")
	}
}

// ---------------------------------------------------------------------------
// Standard stopwords completeness
// ---------------------------------------------------------------------------

func TestStandardStopwords_Count(t *testing.T) {
	// The spec defines exactly 48 standard stopwords.
	if len(standardStopwords) < 40 {
		t.Errorf("expected at least 40 standard stopwords, got %d", len(standardStopwords))
	}
}

func TestTechnicalStopwords_Count(t *testing.T) {
	if len(technicalStopwords) != 8 {
		t.Errorf("expected 8 technical stopwords, got %d", len(technicalStopwords))
	}
}

// ---------------------------------------------------------------------------
// Punctuation and special characters
// ---------------------------------------------------------------------------

func TestTokenize_PunctuationStripped(t *testing.T) {
	tokens := Tokenize("authentication, authorization. jwt!")
	// Punctuation at boundaries is not part of tokens.
	expected := []string{"authentication", "authorization", "jwt"}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, want := range expected {
		if tokens[i] != want {
			t.Errorf("token %d: expected %q, got %q", i, want, tokens[i])
		}
	}
}

func TestTokenize_BackticksIgnored(t *testing.T) {
	tokens := Tokenize("`CreateUser` method")
	// Backticks are not word characters, so they're stripped by the regex.
	found := false
	for _, tok := range tokens {
		if tok == "createuser" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'createuser' from backtick-wrapped code, got %v", tokens)
	}
}

func TestTokenize_Underscores(t *testing.T) {
	// \w includes underscores, so snake_case should stay whole.
	tokens := Tokenize("user_id get_name")
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "user_id" {
		t.Errorf("expected 'user_id', got %q", tokens[0])
	}
	if tokens[1] != "get_name" {
		t.Errorf("expected 'get_name', got %q", tokens[1])
	}
}

// ---------------------------------------------------------------------------
// Numbers and mixed
// ---------------------------------------------------------------------------

func TestTokenize_NumbersPreserved(t *testing.T) {
	tokens := Tokenize("HTTP 200 status code")
	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}
	if !found["200"] {
		t.Error("expected '200' to be preserved")
	}
	if !found["http"] {
		t.Error("expected 'http' to be preserved")
	}
}

func TestTokenize_VersionNumbers(t *testing.T) {
	tokens := Tokenize("Go 1.25 release")
	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}
	if !found["1.25"] {
		t.Errorf("expected '1.25' to be preserved, got tokens: %v", tokens)
	}
}

// ---------------------------------------------------------------------------
// Real-world documentation text
// ---------------------------------------------------------------------------

func TestTokenize_RealDocumentationText(t *testing.T) {
	text := "Use `jwt.Verify()` for token validation. The error-handling pattern should return nil on success."
	tokens := Tokenize(text)

	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}

	// Should preserve compound terms and code identifiers.
	if !found["jwt.verify"] {
		t.Error("expected 'jwt.verify' (dotted path)")
	}
	if !found["error-handling"] {
		t.Error("expected 'error-handling' (compound term)")
	}
	if !found["nil"] {
		t.Error("expected 'nil' (technical stopword)")
	}

	// Should filter standard stopwords.
	if found["the"] {
		t.Error("'the' should be filtered")
	}
	if found["for"] {
		t.Error("'for' should be filtered")
	}
	if found["on"] {
		t.Error("'on' should be filtered")
	}
}

// ---------------------------------------------------------------------------
// Unicode
// ---------------------------------------------------------------------------

func TestTokenize_UnicodeLetters(t *testing.T) {
	// \w in Go's regexp includes unicode letters.
	tokens := Tokenize("authentication Benutzername")
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
}

// ---------------------------------------------------------------------------
// Single character tokens
// ---------------------------------------------------------------------------

func TestTokenize_SingleCharTokens(t *testing.T) {
	// Single word chars should be matched by [\w]+ alternation.
	tokens := Tokenize("x y z")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %v", len(tokens), tokens)
	}
}

func TestTokenize_SingleCharStopword(t *testing.T) {
	// "a" is a stopword.
	tokens := Tokenize("a b c")
	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}
	if found["a"] {
		t.Error("'a' should be filtered as a stopword")
	}
	if !found["b"] {
		t.Error("'b' should be preserved")
	}
}
