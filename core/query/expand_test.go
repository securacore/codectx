package query

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
)

func testTaxonomy() *taxonomy.Taxonomy {
	return &taxonomy.Taxonomy{
		Terms: map[string]*taxonomy.Term{
			"authentication": {
				Canonical: "Authentication",
				Aliases:   []string{"auth", "authn", "identity verification"},
				Narrower:  []string{"oauth", "jwt"},
				Related:   []string{"authorization"},
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
			"authorization": {
				Canonical: "Authorization",
				Aliases:   []string{"authz", "permissions"},
				Related:   []string{"authentication"},
			},
			"error-handling": {
				Canonical: "Error Handling",
				Aliases:   []string{"exception handling", "try-catch"},
			},
		},
	}
}

func defaultExpansionConfig() project.ExpansionConfig {
	return project.DefaultQueryConfig().Expansion
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

// --- ExpandQueryWeighted tests ---

func TestExpandQueryWeighted_OriginalTermsFullWeight(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)
	cfg := defaultExpansionConfig()

	eq := ExpandQueryWeighted("jwt", tax, aliasIdx, cfg)

	if len(eq.Terms) == 0 {
		t.Fatal("expected terms")
	}

	// First term should be the original at weight 1.0.
	if eq.Terms[0].Text != "jwt" {
		t.Errorf("first term = %q, want 'jwt'", eq.Terms[0].Text)
	}
	if eq.Terms[0].Weight != 1.0 {
		t.Errorf("original weight = %f, want 1.0", eq.Terms[0].Weight)
	}
	if eq.Terms[0].Tier != "original" {
		t.Errorf("original tier = %q, want 'original'", eq.Terms[0].Tier)
	}
}

func TestExpandQueryWeighted_TierWeights(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)
	cfg := defaultExpansionConfig()

	eq := ExpandQueryWeighted("authentication", tax, aliasIdx, cfg)

	// Build a map of term text → weighted term for inspection.
	termMap := make(map[string]float64)
	tierMap := make(map[string]string)
	for _, wt := range eq.Terms {
		termMap[wt.Text] = wt.Weight
		tierMap[wt.Text] = wt.Tier
	}

	// Original should be at 1.0.
	if w, ok := termMap["authent"]; !ok || w != 1.0 {
		t.Errorf("original 'authent' weight = %f, want 1.0", w)
	}

	// Aliases should be at AliasWeight (1.0).
	if _, ok := termMap["auth"]; !ok {
		t.Error("expected alias 'auth' in expanded terms")
	}

	// Narrower terms (jwt, oauth) should be at NarrowerWeight (0.7).
	if w, ok := termMap["jwt"]; !ok {
		t.Error("expected narrower 'jwt' in expanded terms")
	} else if w != cfg.NarrowerWeight {
		t.Errorf("narrower 'jwt' weight = %f, want %f", w, cfg.NarrowerWeight)
	}

	// Related terms (authorization) should be at RelatedWeight (0.4).
	if w, ok := termMap["author"]; !ok {
		// "Authorization" stems to "author"
		t.Error("expected related 'author' (stemmed authorization) in expanded terms")
	} else if w != cfg.RelatedWeight {
		t.Errorf("related 'author' weight = %f, want %f", w, cfg.RelatedWeight)
	}
}

func TestExpandQueryWeighted_MaxExpansionTerms(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)
	cfg := defaultExpansionConfig()
	cfg.MaxExpansionTerms = 3

	eq := ExpandQueryWeighted("authentication", tax, aliasIdx, cfg)

	if len(eq.Terms) > 3 {
		t.Errorf("expected at most 3 terms, got %d", len(eq.Terms))
	}
}

func TestExpandQueryWeighted_ExpansionDisabled(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)
	cfg := defaultExpansionConfig()
	cfg.Enabled = project.BoolPtr(false)

	eq := ExpandQueryWeighted("authentication", tax, aliasIdx, cfg)

	// Should only have the original stemmed token.
	if len(eq.Terms) != 1 {
		t.Errorf("expected 1 term with expansion disabled, got %d", len(eq.Terms))
	}
	if eq.Terms[0].Text != "authent" {
		t.Errorf("term = %q, want 'authent'", eq.Terms[0].Text)
	}
}

func TestExpandQueryWeighted_NilTaxonomy(t *testing.T) {
	cfg := defaultExpansionConfig()
	eq := ExpandQueryWeighted("hello world", nil, nil, cfg)

	if len(eq.Terms) != 2 {
		t.Errorf("expected 2 terms, got %d", len(eq.Terms))
	}
	for _, wt := range eq.Terms {
		if wt.Tier != "original" {
			t.Errorf("without taxonomy, all terms should be 'original', got %q", wt.Tier)
		}
	}
}

func TestExpandQueryWeighted_FlatTokensMatch(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)
	cfg := defaultExpansionConfig()

	eq := ExpandQueryWeighted("jwt", tax, aliasIdx, cfg)

	if len(eq.FlatTokens) != len(eq.Terms) {
		t.Errorf("FlatTokens length (%d) != Terms length (%d)",
			len(eq.FlatTokens), len(eq.Terms))
	}
	for i, ft := range eq.FlatTokens {
		if ft != eq.Terms[i].Text {
			t.Errorf("FlatTokens[%d] = %q, Terms[%d].Text = %q",
				i, ft, i, eq.Terms[i].Text)
		}
	}
}

func TestExpandQueryWeighted_EmptyQuery(t *testing.T) {
	cfg := defaultExpansionConfig()
	eq := ExpandQueryWeighted("", nil, nil, cfg)

	if len(eq.Terms) != 0 {
		t.Errorf("expected 0 terms for empty query, got %d", len(eq.Terms))
	}
}

func TestExpandQueryWeighted_RelatedTermExpansion(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)
	cfg := defaultExpansionConfig()

	eq := ExpandQueryWeighted("authentication", tax, aliasIdx, cfg)

	// Check that the related term "authorization" was included.
	found := false
	for _, wt := range eq.Terms {
		if wt.Tier == "related" {
			found = true
			if wt.Weight != cfg.RelatedWeight {
				t.Errorf("related term weight = %f, want %f", wt.Weight, cfg.RelatedWeight)
			}
		}
	}
	if !found {
		t.Error("expected at least one related term in expansion")
	}
}

func TestExpandQueryWeighted_Display(t *testing.T) {
	tax := testTaxonomy()
	aliasIdx := taxonomy.BuildAliasIndex(tax)
	cfg := defaultExpansionConfig()

	eq := ExpandQueryWeighted("authentication", tax, aliasIdx, cfg)

	if eq.Display == "" {
		t.Error("expected non-empty Display string")
	}
	if !strings.Contains(eq.Display, "authent") {
		t.Errorf("Display should contain stemmed original, got %q", eq.Display)
	}
}
