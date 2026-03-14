package index

import (
	"math"
	"testing"

	"github.com/securacore/codectx/core/project"
)

func defaultBM25FCfg() project.BM25FConfig {
	return project.DefaultBM25FConfig()
}

func buildTestBM25F() *BM25F {
	idx := NewBM25F(defaultBM25FCfg())

	idx.AddDocument("obj:abc.1", map[string][]string{
		"heading": Tokenize("JWT Authentication Refresh Flow"),
		"body":    Tokenize("The JWT refresh token is validated using RS256 signing. Tokens expire after 24 hours."),
		"code":    Tokenize("func ValidateJWT(token string) error"),
		"terms":   Tokenize("authentication jwt"),
	})

	idx.AddDocument("obj:def.1", map[string][]string{
		"heading": Tokenize("Error Handling Middleware"),
		"body":    Tokenize("The error handling middleware catches panics and returns structured error responses."),
		"code":    Tokenize("func ErrorMiddleware(next http.Handler) http.Handler"),
		"terms":   Tokenize("error-handling middleware"),
	})

	idx.AddDocument("obj:ghi.1", map[string][]string{
		"heading": Tokenize("Database Connection Pool"),
		"body":    Tokenize("The database connection pool manages connections with configurable limits and health checks."),
		"code":    Tokenize("func NewPool(dsn string, maxConns int) *Pool"),
		"terms":   Tokenize("database connection"),
	})

	idx.Build()
	return idx
}

func TestNewBM25F_Defaults(t *testing.T) {
	cfg := defaultBM25FCfg()
	idx := NewBM25F(cfg)

	if idx.K1 != 1.2 {
		t.Errorf("K1 = %f, want 1.2", idx.K1)
	}
	if idx.DocCount != 0 {
		t.Errorf("DocCount = %d, want 0", idx.DocCount)
	}
	if len(idx.FieldNames) != 4 {
		t.Errorf("FieldNames count = %d, want 4", len(idx.FieldNames))
	}
	// Field names should be sorted.
	expected := []string{"body", "code", "heading", "terms"}
	for i, name := range idx.FieldNames {
		if name != expected[i] {
			t.Errorf("FieldNames[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestBM25F_AddDocument(t *testing.T) {
	idx := NewBM25F(defaultBM25FCfg())
	idx.AddDocument("doc1", map[string][]string{
		"heading": {"jwt", "auth"},
		"body":    {"the", "jwt", "token"},
		"code":    {},
		"terms":   {"authentication"},
	})

	if idx.DocCount != 1 {
		t.Errorf("DocCount = %d, want 1", idx.DocCount)
	}
	if idx.DocIDs[0] != "doc1" {
		t.Errorf("DocIDs[0] = %q, want %q", idx.DocIDs[0], "doc1")
	}

	// Check heading field.
	if idx.FieldLengths["heading"][0] != 2 {
		t.Errorf("heading length = %d, want 2", idx.FieldLengths["heading"][0])
	}
	if idx.FieldTermFreq["heading"][0]["jwt"] != 1 {
		t.Errorf("heading TF for 'jwt' = %d, want 1", idx.FieldTermFreq["heading"][0]["jwt"])
	}

	// Check body field.
	if idx.FieldLengths["body"][0] != 3 {
		t.Errorf("body length = %d, want 3", idx.FieldLengths["body"][0])
	}

	// DocFreq: jwt appears in heading + body, but counted once per doc.
	if idx.DocFreq["jwt"] != 1 {
		t.Errorf("DocFreq[jwt] = %d, want 1", idx.DocFreq["jwt"])
	}
}

func TestBM25F_Build_AvgFieldLen(t *testing.T) {
	idx := NewBM25F(defaultBM25FCfg())

	idx.AddDocument("doc1", map[string][]string{
		"heading": {"a", "b"},      // len 2
		"body":    {"x", "y", "z"}, // len 3
		"code":    {},              // len 0
		"terms":   {"t1"},          // len 1
	})
	idx.AddDocument("doc2", map[string][]string{
		"heading": {"c", "d", "e", "f"}, // len 4
		"body":    {"p"},                // len 1
		"code":    {"code1", "code2"},   // len 2
		"terms":   {"t2", "t3", "t4"},   // len 3
	})

	idx.Build()

	if idx.AvgFieldLen["heading"] != 3.0 { // (2+4)/2
		t.Errorf("AvgFieldLen[heading] = %f, want 3.0", idx.AvgFieldLen["heading"])
	}
	if idx.AvgFieldLen["body"] != 2.0 { // (3+1)/2
		t.Errorf("AvgFieldLen[body] = %f, want 2.0", idx.AvgFieldLen["body"])
	}
	if idx.AvgFieldLen["code"] != 1.0 { // (0+2)/2
		t.Errorf("AvgFieldLen[code] = %f, want 1.0", idx.AvgFieldLen["code"])
	}
	if idx.AvgFieldLen["terms"] != 2.0 { // (1+3)/2
		t.Errorf("AvgFieldLen[terms] = %f, want 2.0", idx.AvgFieldLen["terms"])
	}
}

func TestBM25F_Build_IDF(t *testing.T) {
	idx := NewBM25F(defaultBM25FCfg())

	idx.AddDocument("doc1", map[string][]string{
		"body": {"common", "rare"},
	})
	idx.AddDocument("doc2", map[string][]string{
		"body": {"common"},
	})

	idx.Build()

	// "rare" appears in 1/2 docs — should have higher IDF than "common" (2/2 docs).
	idfRare := idx.IDFCache["rare"]
	idfCommon := idx.IDFCache["common"]
	if idfRare <= idfCommon {
		t.Errorf("IDF(rare) = %f should be > IDF(common) = %f", idfRare, idfCommon)
	}
	if idfCommon <= 0 {
		t.Errorf("IDF(common) = %f should be positive (Lucene-adjusted)", idfCommon)
	}
}

func TestBM25F_Score_HeadingWeightHigher(t *testing.T) {
	idx := NewBM25F(defaultBM25FCfg())

	// Doc1: "jwt" in heading only (weight 3.0)
	idx.AddDocument("doc1", map[string][]string{
		"heading": {"jwt"},
		"body":    {"unrelated", "content"},
		"code":    {},
		"terms":   {},
	})

	// Doc2: "jwt" in body only (weight 1.0)
	idx.AddDocument("doc2", map[string][]string{
		"heading": {"unrelated"},
		"body":    {"jwt"},
		"code":    {},
		"terms":   {},
	})

	idx.Build()

	query := []WeightedTerm{{Text: "jwt", Weight: 1.0, Tier: "original"}}
	results := idx.Score(query, 0)

	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ChunkID != "doc1" {
		t.Errorf("heading match should rank first, got %q", results[0].ChunkID)
	}
	if results[0].Score <= results[1].Score {
		t.Errorf("heading score (%f) should be > body score (%f)",
			results[0].Score, results[1].Score)
	}
}

func TestBM25F_Score_TermsWeightHigher(t *testing.T) {
	idx := NewBM25F(defaultBM25FCfg())

	// Doc1: "auth" in terms field (weight 2.0)
	idx.AddDocument("doc1", map[string][]string{
		"heading": {},
		"body":    {"unrelated"},
		"code":    {},
		"terms":   {"auth"},
	})

	// Doc2: "auth" in body (weight 1.0)
	idx.AddDocument("doc2", map[string][]string{
		"heading": {},
		"body":    {"auth"},
		"code":    {},
		"terms":   {},
	})

	idx.Build()

	query := []WeightedTerm{{Text: "auth", Weight: 1.0, Tier: "original"}}
	results := idx.Score(query, 0)

	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ChunkID != "doc1" {
		t.Errorf("terms field match should rank first, got %q", results[0].ChunkID)
	}
}

func TestBM25F_Score_WeightedTerms(t *testing.T) {
	idx := buildTestBM25F()

	// "jwt" at full weight should score higher than "jwt" at reduced weight.
	fullWeight := []WeightedTerm{{Text: "jwt", Weight: 1.0, Tier: "original"}}
	halfWeight := []WeightedTerm{{Text: "jwt", Weight: 0.5, Tier: "narrower"}}

	fullResults := idx.Score(fullWeight, 1)
	halfResults := idx.Score(halfWeight, 1)

	if len(fullResults) == 0 || len(halfResults) == 0 {
		t.Fatal("expected results for both queries")
	}

	if fullResults[0].Score <= halfResults[0].Score {
		t.Errorf("full weight score (%f) should be > half weight score (%f)",
			fullResults[0].Score, halfResults[0].Score)
	}

	// Score should scale proportionally with weight.
	ratio := fullResults[0].Score / halfResults[0].Score
	if math.Abs(ratio-2.0) > 0.01 {
		t.Errorf("score ratio = %f, want 2.0", ratio)
	}
}

func TestBM25F_Score_MultiTerm(t *testing.T) {
	idx := buildTestBM25F()

	// Query matching multiple terms in the same doc should boost it.
	query := []WeightedTerm{
		{Text: "jwt", Weight: 1.0, Tier: "original"},
		{Text: "authent", Weight: 1.0, Tier: "original"}, // stemmed "authentication"
	}
	results := idx.Score(query, 0)

	if len(results) == 0 {
		t.Fatal("expected results")
	}
	// The JWT doc should rank first since it matches both terms.
	if results[0].ChunkID != "obj:abc.1" {
		t.Errorf("multi-term match should rank first, got %q", results[0].ChunkID)
	}
}

func TestBM25F_Score_EmptyQuery(t *testing.T) {
	idx := buildTestBM25F()
	results := idx.Score(nil, 0)
	if results != nil {
		t.Errorf("expected nil for empty query, got %d results", len(results))
	}
}

func TestBM25F_Score_EmptyCorpus(t *testing.T) {
	idx := NewBM25F(defaultBM25FCfg())
	idx.Build()

	query := []WeightedTerm{{Text: "test", Weight: 1.0, Tier: "original"}}
	results := idx.Score(query, 0)
	if results != nil {
		t.Errorf("expected nil for empty corpus, got %d results", len(results))
	}
}

func TestBM25F_Score_TopN(t *testing.T) {
	idx := buildTestBM25F()

	// Use a term that appears in multiple docs' body (tokenizer stems "connection" → "connect").
	term := WeightedTerm{Text: "token", Weight: 1.0, Tier: "original"}
	all := idx.Score([]WeightedTerm{term}, 0)

	if len(all) < 1 {
		// Fall back to a broader query.
		term = WeightedTerm{Text: "jwt", Weight: 1.0, Tier: "original"}
		all = idx.Score([]WeightedTerm{term}, 0)
	}

	if len(all) < 1 {
		t.Fatalf("expected at least 1 result, got %d", len(all))
	}

	limited := idx.Score([]WeightedTerm{term}, 1)
	if len(limited) != 1 {
		t.Fatalf("expected 1 result with topN=1, got %d", len(limited))
	}
	if limited[0].ChunkID != all[0].ChunkID {
		t.Errorf("topN should keep the highest scorer")
	}
}

func TestBM25F_Score_UnknownTerm(t *testing.T) {
	idx := buildTestBM25F()

	query := []WeightedTerm{{Text: "nonexistent_xyz", Weight: 1.0, Tier: "original"}}
	results := idx.Score(query, 0)
	if len(results) != 0 {
		t.Errorf("expected 0 results for unknown term, got %d", len(results))
	}
}

func TestBM25F_MissingFields(t *testing.T) {
	idx := NewBM25F(defaultBM25FCfg())

	// Only provide body field — others should default to empty.
	idx.AddDocument("doc1", map[string][]string{
		"body": {"jwt", "token"},
	})
	idx.Build()

	query := []WeightedTerm{{Text: "jwt", Weight: 1.0, Tier: "original"}}
	results := idx.Score(query, 0)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestBM25F_FieldB_Zero_NoLengthNormalization(t *testing.T) {
	// Create a config where all fields have b=0 (no length normalization).
	cfg := project.BM25FConfig{
		K1: 1.2,
		Fields: map[string]project.BM25FFieldConfig{
			"body": {Weight: 1.0, B: 0.0},
		},
	}
	idx := NewBM25F(cfg)

	// Short doc and long doc both contain "jwt" once.
	idx.AddDocument("short", map[string][]string{
		"body": {"jwt"},
	})
	idx.AddDocument("long", map[string][]string{
		"body": {"jwt", "padding", "padding2", "padding3", "padding4", "padding5"},
	})
	idx.Build()

	query := []WeightedTerm{{Text: "jwt", Weight: 1.0, Tier: "original"}}
	results := idx.Score(query, 0)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// With b=0, doc length should not affect scoring — scores should be equal.
	if results[0].Score != results[1].Score {
		t.Errorf("with b=0, scores should be equal: %f vs %f",
			results[0].Score, results[1].Score)
	}
}
