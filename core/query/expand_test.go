package query

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/taxonomy"
)

func testTaxonomy() *taxonomy.Taxonomy {
	return &taxonomy.Taxonomy{
		Terms: map[string]*taxonomy.Term{
			"authentication": {
				Canonical: "Authentication",
				Aliases:   []string{"auth", "authn", "identity verification"},
				Narrower:  []string{"oauth", "jwt"},
			},
			"jwt": {
				Canonical: "JWT",
				Aliases:   []string{"json web token"},
				Broader:   "authentication",
			},
			"oauth": {
				Canonical: "OAuth",
				Aliases:   []string{"open authorization"},
				Broader:   "authentication",
			},
			"error-handling": {
				Canonical: "Error Handling",
				Aliases:   []string{"exception handling", "try-catch"},
			},
		},
	}
}

func TestExpandQuery_WithTaxonomy(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)

	tokens, expanded := ExpandQuery("authentication", tax, aliasIdx)

	// Should include original token + canonical + aliases + narrower terms.
	tokenSet := make(map[string]bool)
	for _, tok := range tokens {
		tokenSet[tok] = true
	}

	// The original token (stemmed: "authent") should be present.
	if !tokenSet["authent"] {
		t.Errorf("expected stemmed 'authent' in tokens, got: %v", tokens)
	}

	// Alias "auth" should be expanded.
	if !tokenSet["auth"] {
		t.Errorf("expected 'auth' alias in tokens, got: %v", tokens)
	}

	// Narrower terms "oauth" and "jwt" should be included.
	if !tokenSet["oauth"] {
		t.Errorf("expected 'oauth' narrower term in tokens, got: %v", tokens)
	}
	if !tokenSet["jwt"] {
		t.Errorf("expected 'jwt' narrower term in tokens, got: %v", tokens)
	}

	if expanded == "" {
		t.Error("expected non-empty expanded string")
	}
}

func TestExpandQuery_AliasLookup(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)

	// Querying by alias "auth" should expand to the full authentication term.
	tokens, _ := ExpandQuery("auth", tax, aliasIdx)

	tokenSet := make(map[string]bool)
	for _, tok := range tokens {
		tokenSet[tok] = true
	}

	if !tokenSet["auth"] {
		t.Errorf("expected original 'auth' in tokens, got: %v", tokens)
	}

	// Should resolve to "authentication" term and include its stemmed canonical.
	// "Authentication" stems to "authent".
	if !tokenSet["authent"] {
		t.Errorf("expected stemmed 'authent' (from Authentication) in tokens, got: %v", tokens)
	}
}

func TestExpandQuery_NilTaxonomy(t *testing.T) {
	tokens, expanded := ExpandQuery("hello world", nil, nil)

	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens without taxonomy, got %d: %v", len(tokens), tokens)
	}
	if expanded != "hello world" {
		t.Errorf("expected 'hello world', got %q", expanded)
	}
}

func TestExpandQuery_EmptyQuery(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)

	tokens, expanded := ExpandQuery("", tax, aliasIdx)
	if tokens != nil {
		t.Errorf("expected nil tokens for empty query, got %v", tokens)
	}
	if expanded != "" {
		t.Errorf("expected empty expanded for empty query, got %q", expanded)
	}
}

func TestExpandQuery_StopwordsOnly(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)

	tokens, _ := ExpandQuery("the a an is", tax, aliasIdx)
	if tokens != nil {
		t.Errorf("expected nil tokens for stopwords-only query, got %v", tokens)
	}
}

func TestExpandQuery_NoTaxonomyMatch(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)

	tokens, expanded := ExpandQuery("foobar", tax, aliasIdx)

	// Should still include the original token.
	if len(tokens) < 1 {
		t.Fatalf("expected at least 1 token, got %d", len(tokens))
	}
	if tokens[0] != "foobar" {
		t.Errorf("expected 'foobar' as first token, got %q", tokens[0])
	}
	if !strings.Contains(expanded, "foobar") {
		t.Errorf("expected expanded to contain 'foobar', got %q", expanded)
	}
}

func TestExpandQuery_DictionaryFallback(t *testing.T) {
	// Term not in taxonomy but IS in dictionary.
	tax := &taxonomy.Taxonomy{
		Terms: map[string]*taxonomy.Term{},
	}
	aliasIdx := taxonomy.BuildAliasIndex(tax)

	tokens, _ := ExpandQuery("jwt", tax, aliasIdx)

	// The dictionary has jwt -> "json web token".
	tokenSet := make(map[string]bool)
	for _, tok := range tokens {
		tokenSet[tok] = true
	}

	if !tokenSet["jwt"] {
		t.Errorf("expected original 'jwt' in tokens, got: %v", tokens)
	}

	// Dictionary lookup should add tokenized "json web token" -> ["json", "web", "token"].
	if !tokenSet["json"] {
		t.Errorf("expected 'json' from dictionary expansion, got: %v", tokens)
	}
	if !tokenSet["web"] {
		t.Errorf("expected 'web' from dictionary expansion, got: %v", tokens)
	}
	if !tokenSet["token"] {
		t.Errorf("expected 'token' from dictionary expansion, got: %v", tokens)
	}
}

func TestExpandQuery_MultiWordAlias(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)

	// "jwt" in the taxonomy has alias "json web token".
	tokens, _ := ExpandQuery("jwt", tax, aliasIdx)

	tokenSet := make(map[string]bool)
	for _, tok := range tokens {
		tokenSet[tok] = true
	}

	// Multi-word alias "json web token" should be tokenized into individual tokens.
	if !tokenSet["json"] {
		t.Errorf("expected 'json' from multi-word alias, got: %v", tokens)
	}
	if !tokenSet["web"] {
		t.Errorf("expected 'web' from multi-word alias, got: %v", tokens)
	}
	if !tokenSet["token"] {
		t.Errorf("expected 'token' from multi-word alias, got: %v", tokens)
	}
}

func TestExpandQuery_Deduplication(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)

	tokens, _ := ExpandQuery("jwt jwt jwt", tax, aliasIdx)

	// Should not have duplicate tokens.
	seen := make(map[string]int)
	for _, tok := range tokens {
		seen[tok]++
	}
	for tok, count := range seen {
		if count > 1 {
			t.Errorf("duplicate token %q appears %d times", tok, count)
		}
	}
}
