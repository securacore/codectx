package index

import (
	"math"
	"sort"
)

// ScoredResult pairs a chunk ID with its BM25 relevance score.
type ScoredResult struct {
	// ChunkID is the full chunk identifier (e.g. "obj:a1b2c3d4.3").
	ChunkID string

	// Score is the BM25 relevance score. Higher is more relevant.
	Score float64
}

// BM25 implements the Okapi BM25 ranking algorithm as an inverted index
// over tokenized documents (chunks).
//
// All exported fields are serializable via gob encoding for persistence
// across CLI invocations.
//
// Scoring formula:
//
//	score(term, doc) = IDF(term) * (TF * (k1 + 1)) / (TF + k1 * (1 - b + b * |doc| / avgdl))
//	IDF(term)        = log(1 + (N - n(term) + 0.5) / (n(term) + 0.5))
//
// The IDF uses the Lucene-adjusted formula (log(1 + ...)) to ensure
// terms are never penalized for appearing in more than half the corpus.
//
// Where:
//   - TF = term frequency in document
//   - N = total document count
//   - n(term) = number of documents containing the term
//   - |doc| = document length in tokens
//   - avgdl = average document length across the corpus
type BM25 struct {
	// K1 controls term frequency saturation. Default 1.2.
	// Higher values give more weight to repeated terms.
	K1 float64

	// B controls document length normalization. Default 0.75.
	// 0 = no normalization, 1 = full normalization.
	B float64

	// AvgDocLen is the mean document length (in tokens) across the corpus.
	// Computed during Build().
	AvgDocLen float64

	// DocCount is the total number of documents in the index.
	DocCount int

	// DocLengths stores the token count for each document, indexed by docIndex.
	DocLengths []int

	// DocIDs maps docIndex to chunk ID (e.g. "obj:a1b2c3d4.3").
	DocIDs []string

	// TermDocFreq maps each term to the number of documents containing it.
	// This is n(term) in the IDF formula.
	TermDocFreq map[string]int

	// DocTermFreqs stores per-document term frequencies.
	// DocTermFreqs[docIndex][term] = count of term in that document.
	DocTermFreqs []map[string]int

	// IDFCache caches computed IDF values per term.
	// Populated during Build().
	IDFCache map[string]float64
}

// NewBM25 creates an empty BM25 index with the given parameters.
// Documents must be added via AddDocument, then Build() must be called
// before querying.
func NewBM25(k1, b float64) *BM25 {
	return &BM25{
		K1:           k1,
		B:            b,
		TermDocFreq:  make(map[string]int),
		DocTermFreqs: make([]map[string]int, 0),
		DocLengths:   make([]int, 0),
		DocIDs:       make([]string, 0),
		IDFCache:     make(map[string]float64),
	}
}

// AddDocument adds a tokenized document to the index.
// The chunkID identifies the source chunk. The tokens should be
// pre-processed by Tokenize().
//
// AddDocument must be called before Build(). Adding documents after
// Build() invalidates the cached IDF values and avgDocLen.
func (idx *BM25) AddDocument(chunkID string, tokens []string) {
	docIndex := idx.DocCount
	idx.DocCount++

	idx.DocIDs = append(idx.DocIDs, chunkID)
	idx.DocLengths = append(idx.DocLengths, len(tokens))

	// Count term frequencies within this document.
	termFreq := make(map[string]int, len(tokens))
	for _, t := range tokens {
		termFreq[t]++
	}
	idx.DocTermFreqs = append(idx.DocTermFreqs, termFreq)

	// Update corpus-level document frequency counts.
	// Each term is counted once per document, regardless of how many
	// times it appears in that document.
	_ = docIndex // used implicitly via DocCount
	for term := range termFreq {
		idx.TermDocFreq[term]++
	}
}

// Build finalizes the index by computing the average document length
// and pre-caching IDF values for all terms. Must be called after all
// documents have been added and before any queries.
func (idx *BM25) Build() {
	if idx.DocCount == 0 {
		idx.AvgDocLen = 0
		return
	}

	// Compute average document length.
	totalLen := 0
	for _, dl := range idx.DocLengths {
		totalLen += dl
	}
	idx.AvgDocLen = float64(totalLen) / float64(idx.DocCount)

	// Pre-compute IDF for all terms using the Lucene-adjusted formula:
	//   IDF(term) = log(1 + (N - n(term) + 0.5) / (n(term) + 0.5))
	//
	// The +1 inside the log ensures IDF is always non-negative. Without it,
	// terms appearing in more than half the corpus get a negative IDF, which
	// would penalize documents containing common terms. The Lucene adjustment
	// gives common terms a small positive weight instead of zero/negative.
	idx.IDFCache = make(map[string]float64, len(idx.TermDocFreq))
	n := float64(idx.DocCount)
	for term, docFreq := range idx.TermDocFreq {
		df := float64(docFreq)
		idx.IDFCache[term] = math.Log(1 + (n-df+0.5)/(df+0.5))
	}
}

// Score computes BM25 relevance scores for a tokenized query and returns
// the top N results sorted by score descending. If topN <= 0, all documents
// with non-zero scores are returned.
//
// The query tokens should be pre-processed by Tokenize().
func (idx *BM25) Score(query []string, topN int) []ScoredResult {
	if idx.DocCount == 0 || len(query) == 0 {
		return nil
	}

	scores := make([]float64, idx.DocCount)

	for _, term := range query {
		idf, ok := idx.IDFCache[term]
		if !ok {
			// Term not in corpus — skip.
			continue
		}

		for docIdx := 0; docIdx < idx.DocCount; docIdx++ {
			tf := float64(idx.DocTermFreqs[docIdx][term])
			if tf == 0 {
				continue
			}

			docLen := float64(idx.DocLengths[docIdx])
			k := idx.K1 * (1 - idx.B + idx.B*docLen/idx.AvgDocLen)
			scores[docIdx] += idf * (tf * (idx.K1 + 1)) / (tf + k)
		}
	}

	// Collect non-zero results.
	results := make([]ScoredResult, 0)
	for i, score := range scores {
		if score > 0 {
			results = append(results, ScoredResult{
				ChunkID: idx.DocIDs[i],
				Score:   score,
			})
		}
	}

	// Sort by score descending, then by chunk ID for stability.
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ChunkID < results[j].ChunkID
	})

	// Limit to topN.
	if topN > 0 && len(results) > topN {
		results = results[:topN]
	}

	return results
}
