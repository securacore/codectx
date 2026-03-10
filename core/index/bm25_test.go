package index

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildTestIndex() *BM25 {
	idx := NewBM25(1.2, 0.75)

	// Doc 0: about JWT authentication
	idx.AddDocument("obj:aaa.1", Tokenize("JWT authentication token validation refresh flow"))
	// Doc 1: about error handling
	idx.AddDocument("obj:bbb.1", Tokenize("error-handling patterns return nil on failure"))
	// Doc 2: about JWT refresh specifically
	idx.AddDocument("obj:ccc.1", Tokenize("JWT refresh token rotation automatic JWT renewal"))
	// Doc 3: general topic
	idx.AddDocument("obj:ddd.1", Tokenize("database connection pooling configuration settings"))

	idx.Build()
	return idx
}

// ---------------------------------------------------------------------------
// NewBM25
// ---------------------------------------------------------------------------

func TestNewBM25_Defaults(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	if idx.K1 != 1.2 {
		t.Errorf("expected k1=1.2, got %f", idx.K1)
	}
	if idx.B != 0.75 {
		t.Errorf("expected b=0.75, got %f", idx.B)
	}
	if idx.DocCount != 0 {
		t.Errorf("expected 0 docs, got %d", idx.DocCount)
	}
}

// ---------------------------------------------------------------------------
// AddDocument
// ---------------------------------------------------------------------------

func TestAddDocument_IncrementsCount(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.AddDocument("a.1", []string{"hello", "world"})
	idx.AddDocument("b.1", []string{"world"})
	if idx.DocCount != 2 {
		t.Errorf("expected 2 docs, got %d", idx.DocCount)
	}
}

func TestAddDocument_TracksDocLengths(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.AddDocument("a.1", []string{"hello", "world", "foo"})
	idx.AddDocument("b.1", []string{"world"})
	if idx.DocLengths[0] != 3 {
		t.Errorf("doc 0 length: expected 3, got %d", idx.DocLengths[0])
	}
	if idx.DocLengths[1] != 1 {
		t.Errorf("doc 1 length: expected 1, got %d", idx.DocLengths[1])
	}
}

func TestAddDocument_TracksTermDocFreq(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	// "world" appears in both documents.
	idx.AddDocument("a.1", []string{"hello", "world"})
	idx.AddDocument("b.1", []string{"world", "world"}) // repeated in same doc = still 1 doc freq
	if idx.TermDocFreq["world"] != 2 {
		t.Errorf("expected doc freq 2 for 'world', got %d", idx.TermDocFreq["world"])
	}
	if idx.TermDocFreq["hello"] != 1 {
		t.Errorf("expected doc freq 1 for 'hello', got %d", idx.TermDocFreq["hello"])
	}
}

func TestAddDocument_TracksDocTermFreqs(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.AddDocument("a.1", []string{"hello", "hello", "world"})
	if idx.DocTermFreqs[0]["hello"] != 2 {
		t.Errorf("expected TF 2 for 'hello' in doc 0, got %d", idx.DocTermFreqs[0]["hello"])
	}
	if idx.DocTermFreqs[0]["world"] != 1 {
		t.Errorf("expected TF 1 for 'world' in doc 0, got %d", idx.DocTermFreqs[0]["world"])
	}
}

func TestAddDocument_EmptyTokens(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.AddDocument("a.1", []string{})
	if idx.DocCount != 1 {
		t.Errorf("expected 1 doc, got %d", idx.DocCount)
	}
	if idx.DocLengths[0] != 0 {
		t.Errorf("expected doc length 0, got %d", idx.DocLengths[0])
	}
}

func TestAddDocument_NilTokens(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.AddDocument("a.1", nil)
	if idx.DocCount != 1 {
		t.Errorf("expected 1 doc, got %d", idx.DocCount)
	}
}

// ---------------------------------------------------------------------------
// Build
// ---------------------------------------------------------------------------

func TestBuild_AvgDocLen(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.AddDocument("a.1", []string{"a", "b", "c"}) // len 3
	idx.AddDocument("b.1", []string{"d"})           // len 1
	idx.Build()

	expected := 2.0 // (3+1)/2
	if math.Abs(idx.AvgDocLen-expected) > 0.001 {
		t.Errorf("expected avgDocLen %f, got %f", expected, idx.AvgDocLen)
	}
}

func TestBuild_EmptyCorpus(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.Build()
	if idx.AvgDocLen != 0 {
		t.Errorf("expected avgDocLen 0 for empty corpus, got %f", idx.AvgDocLen)
	}
}

func TestBuild_IDFValues(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	// Term "rare" in 1/4 docs, "common" in 3/4 docs.
	idx.AddDocument("a.1", []string{"rare", "common"})
	idx.AddDocument("b.1", []string{"common"})
	idx.AddDocument("c.1", []string{"common"})
	idx.AddDocument("d.1", []string{"unique"})
	idx.Build()

	// Lucene-adjusted IDF: log(1 + (N - n + 0.5) / (n + 0.5))
	// IDF(rare) = log(1 + (4 - 1 + 0.5) / (1 + 0.5)) = log(1 + 3.5/1.5) ≈ 1.099
	rareIDF := idx.IDFCache["rare"]
	expectedRare := math.Log(1 + 3.5/1.5)
	if math.Abs(rareIDF-expectedRare) > 0.001 {
		t.Errorf("IDF(rare): expected %f, got %f", expectedRare, rareIDF)
	}

	// IDF(common) = log(1 + (4 - 3 + 0.5) / (3 + 0.5)) = log(1 + 1.5/3.5) ≈ 0.357
	// Note: with Lucene adjustment, common terms still get a positive (but small) IDF.
	commonIDF := idx.IDFCache["common"]
	expectedCommon := math.Log(1 + 1.5/3.5)
	if math.Abs(commonIDF-expectedCommon) > 0.001 {
		t.Errorf("IDF(common): expected %f, got %f", expectedCommon, commonIDF)
	}

	// Rare terms should have higher IDF than common terms.
	if rareIDF <= commonIDF {
		t.Errorf("expected rare IDF > common IDF: %f vs %f", rareIDF, commonIDF)
	}
}

// ---------------------------------------------------------------------------
// Score
// ---------------------------------------------------------------------------

func TestScore_EmptyQuery(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Score(nil, 10)
	if results != nil {
		t.Errorf("expected nil for empty query, got %v", results)
	}
}

func TestScore_EmptyCorpus(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.Build()
	results := idx.Score([]string{"hello"}, 10)
	if results != nil {
		t.Errorf("expected nil for empty corpus, got %v", results)
	}
}

func TestScore_UnknownTerm(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Score([]string{"xyznonexistent"}, 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results for unknown term, got %d", len(results))
	}
}

func TestScore_SingleTerm(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Score(Tokenize("JWT"), 10)

	if len(results) == 0 {
		t.Fatal("expected results for 'jwt' query")
	}

	// Documents containing "jwt" should score higher than those without.
	// Doc 0 and Doc 2 contain "jwt", Doc 1 and Doc 3 do not.
	for _, r := range results {
		if r.ChunkID != "obj:aaa.1" && r.ChunkID != "obj:ccc.1" {
			t.Errorf("unexpected chunk in results: %q", r.ChunkID)
		}
	}
}

func TestScore_MultiTermBoostsRelevant(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Score(Tokenize("JWT refresh token"), 10)

	if len(results) == 0 {
		t.Fatal("expected results for multi-term query")
	}

	// Doc 2 (obj:ccc.1) has "jwt", "refresh", "token" — should score highest.
	if results[0].ChunkID != "obj:ccc.1" {
		t.Errorf("expected obj:ccc.1 as top result, got %q (score: %f)",
			results[0].ChunkID, results[0].Score)
	}
}

func TestScore_TopNLimits(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Score(Tokenize("JWT refresh"), 1)

	if len(results) != 1 {
		t.Errorf("expected 1 result with topN=1, got %d", len(results))
	}
}

func TestScore_TopNZeroReturnsAll(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Score(Tokenize("JWT"), 0)

	// Should return all documents containing "jwt" (2 docs).
	if len(results) < 2 {
		t.Errorf("expected at least 2 results with topN=0, got %d", len(results))
	}
}

func TestScore_ResultsSortedByScoreDescending(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Score(Tokenize("JWT refresh token"), 0)

	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted: score[%d]=%f > score[%d]=%f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestScore_PositiveScoresOnly(t *testing.T) {
	idx := buildTestIndex()
	results := idx.Score(Tokenize("JWT"), 0)

	for _, r := range results {
		if r.Score <= 0 {
			t.Errorf("expected positive scores only, got %f for %q", r.Score, r.ChunkID)
		}
	}
}

// ---------------------------------------------------------------------------
// TF saturation
// ---------------------------------------------------------------------------

func TestScore_TFSaturation(t *testing.T) {
	// A term repeated many times should have diminishing returns.
	idx := NewBM25(1.2, 0.75)
	idx.AddDocument("once.1", []string{"jwt"})
	idx.AddDocument("many.1", []string{"jwt", "jwt", "jwt", "jwt", "jwt"})
	idx.Build()

	results := idx.Score([]string{"jwt"}, 0)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// "many" doc should score higher but NOT 5x higher (saturation).
	var onceScore, manyScore float64
	for _, r := range results {
		if r.ChunkID == "once.1" {
			onceScore = r.Score
		}
		if r.ChunkID == "many.1" {
			manyScore = r.Score
		}
	}

	if manyScore <= onceScore {
		t.Errorf("expected many-occurrence doc to score higher: once=%f, many=%f",
			onceScore, manyScore)
	}

	ratio := manyScore / onceScore
	if ratio > 3.0 {
		t.Errorf("TF saturation not working: ratio=%f (expected < 3.0)", ratio)
	}
}

// ---------------------------------------------------------------------------
// Document length normalization
// ---------------------------------------------------------------------------

func TestScore_DocLengthNormalization(t *testing.T) {
	// A short focused document should score higher than a long one
	// when both contain the query term once (with b=0.75).
	idx := NewBM25(1.2, 0.75)
	shortTokens := []string{"jwt", "validation"}
	longTokens := make([]string, 100)
	longTokens[0] = "jwt"
	for i := 1; i < 100; i++ {
		longTokens[i] = "filler"
	}
	idx.AddDocument("short.1", shortTokens)
	idx.AddDocument("long.1", longTokens)
	idx.Build()

	results := idx.Score([]string{"jwt"}, 0)

	var shortScore, longScore float64
	for _, r := range results {
		if r.ChunkID == "short.1" {
			shortScore = r.Score
		}
		if r.ChunkID == "long.1" {
			longScore = r.Score
		}
	}

	if shortScore <= longScore {
		t.Errorf("expected short doc to score higher: short=%f, long=%f",
			shortScore, longScore)
	}
}

// ---------------------------------------------------------------------------
// B=0 disables length normalization
// ---------------------------------------------------------------------------

func TestScore_BZero_NoLengthNormalization(t *testing.T) {
	idx := NewBM25(1.2, 0.0)
	idx.AddDocument("short.1", []string{"jwt"})
	idx.AddDocument("long.1", append([]string{"jwt"}, make([]string, 99)...))
	idx.Build()

	results := idx.Score([]string{"jwt"}, 0)

	// With b=0, document length should not affect scoring.
	// Both docs have the same TF for "jwt" (1), so scores should be equal.
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if math.Abs(results[0].Score-results[1].Score) > 0.001 {
		t.Errorf("with b=0, scores should be equal: %f vs %f",
			results[0].Score, results[1].Score)
	}
}

// ---------------------------------------------------------------------------
// Single document corpus
// ---------------------------------------------------------------------------

func TestScore_SingleDocCorpus(t *testing.T) {
	idx := NewBM25(1.2, 0.75)
	idx.AddDocument("only.1", []string{"hello", "world"})
	idx.Build()

	results := idx.Score([]string{"hello"}, 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ChunkID != "only.1" {
		t.Errorf("expected 'only.1', got %q", results[0].ChunkID)
	}
}
