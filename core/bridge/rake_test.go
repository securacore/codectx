package bridge

import (
	"strings"
	"testing"
)

func TestExtractKeyPhrases_BasicText(t *testing.T) {
	text := "The refresh token lifecycle begins when the client sends an expired access token. " +
		"The server validates the refresh token signature using RS256 signing requirements. " +
		"Token expiry validation ensures that stale tokens are rejected immediately."

	phrases := extractKeyPhrases(text, 4)
	if len(phrases) == 0 {
		t.Fatal("expected at least one key phrase")
	}

	// RAKE should surface multi-word technical terms with higher scores
	// than single common words.
	allText := make([]string, len(phrases))
	for i, p := range phrases {
		allText[i] = p.text
	}
	joined := strings.Join(allText, " | ")
	t.Logf("extracted phrases: %s", joined)

	// Verify we get compound terms, not just single words.
	hasMultiWord := false
	for _, p := range phrases {
		if strings.Contains(p.text, " ") {
			hasMultiWord = true
			break
		}
	}
	if !hasMultiWord {
		t.Error("expected at least one multi-word phrase from RAKE")
	}
}

func TestExtractKeyPhrases_EmptyInput(t *testing.T) {
	if got := extractKeyPhrases("", 4); got != nil {
		t.Errorf("expected nil for empty input, got %v", got)
	}
}

func TestExtractKeyPhrases_ZeroTopN(t *testing.T) {
	if got := extractKeyPhrases("some text here", 0); got != nil {
		t.Errorf("expected nil for topN=0, got %v", got)
	}
}

func TestExtractKeyPhrases_StopWordsOnly(t *testing.T) {
	phrases := extractKeyPhrases("the a an is are was were", 4)
	if len(phrases) != 0 {
		t.Errorf("expected no phrases from stop words only, got %v", phrases)
	}
}

func TestExtractKeyPhrases_Deduplication(t *testing.T) {
	// Repeat the same phrase multiple times.
	text := "token validation. token validation. token validation. other concept here."
	phrases := extractKeyPhrases(text, 10)

	seen := make(map[string]bool)
	for _, p := range phrases {
		if seen[p.text] {
			t.Errorf("duplicate phrase: %s", p.text)
		}
		seen[p.text] = true
	}
}

func TestExtractKeyPhrases_RespectsTopN(t *testing.T) {
	text := "authentication tokens, authorization rules, database connections, " +
		"repository patterns, configuration files, environment variables"
	phrases := extractKeyPhrases(text, 3)
	if len(phrases) > 3 {
		t.Errorf("expected at most 3 phrases, got %d", len(phrases))
	}
}

func TestExtractKeyPhrases_ScoresCompoundTermsHigher(t *testing.T) {
	text := "The refresh token lifecycle is important. " +
		"It handles credential rotation for all clients. " +
		"Database connection pooling is the standard approach. " +
		"Caching improves the overall speed."

	phrases := extractKeyPhrases(text, 10)
	if len(phrases) < 2 {
		t.Fatalf("expected at least 2 phrases, got %d", len(phrases))
	}

	// Multi-word phrases should score higher than single words in RAKE.
	if phrases[0].score < phrases[len(phrases)-1].score {
		t.Errorf("expected first phrase (%.2f) to score higher than last (%.2f)",
			phrases[0].score, phrases[len(phrases)-1].score)
	}
}

func TestTokenizeForRAKE_DelimiterHandling(t *testing.T) {
	text := "hello, world! how (are) you?"
	words := tokenizeForRAKE(text)
	for _, w := range words {
		for _, d := range phraseDelimiters {
			if strings.ContainsRune(w, rune(d)) {
				t.Errorf("word %q contains delimiter %q", w, string(d))
			}
		}
	}
}

func TestBuildCandidatePhrases_SplitsOnStopWords(t *testing.T) {
	words := []string{"refresh", "token", "is", "validated", "using", "rs256", "signing"}
	phrases := buildCandidatePhrases(words)

	// "is" and "using" are stop words, so we expect:
	// ["refresh", "token"], ["validated"], ["rs256", "signing"]
	if len(phrases) < 2 {
		t.Errorf("expected at least 2 phrases, got %d", len(phrases))
	}
}
