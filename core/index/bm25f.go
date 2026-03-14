package index

import (
	"math"
	"sort"

	"github.com/securacore/codectx/core/project"
)

// WeightedTerm pairs a search term with a weight multiplier from
// taxonomy expansion. Original terms have weight 1.0; aliases, narrower,
// and related terms have decreasing weights.
type WeightedTerm struct {
	// Text is the stemmed/tokenized term.
	Text string

	// Weight is the relevance multiplier (1.0 for original, <1.0 for expansions).
	Weight float64

	// Tier identifies the expansion source: "original", "alias", "narrower", "related".
	Tier string
}

// BM25F implements BM25F (BM25 with Fields) scoring, where each document
// is split into multiple fields (heading, body, code, terms) that are
// scored independently with configurable weights and length normalization.
//
// All exported fields are serializable via gob encoding.
//
// Scoring formula:
//
//	For each query term t with expansion weight w_t:
//	  weightedTF = Σ_field ( fieldWeight_f * tf_f / (1 - b_f + b_f * dl_f / avgdl_f) )
//	  score(t) = w_t * IDF(t) * (weightedTF * (K1 + 1)) / (weightedTF + K1)
//	  IDF(t) = log(1 + (N - n(t) + 0.5) / (n(t) + 0.5))
type BM25F struct {
	// K1 controls term frequency saturation.
	K1 float64

	// FieldNames is the ordered list of field names (for deterministic iteration).
	FieldNames []string

	// FieldWeights maps field name to its importance multiplier.
	FieldWeights map[string]float64

	// FieldB maps field name to its per-field length normalization parameter.
	FieldB map[string]float64

	// AvgFieldLen maps field name to the average field length across the corpus.
	AvgFieldLen map[string]float64

	// DocCount is the total number of documents in the index.
	DocCount int

	// DocIDs maps docIndex to chunk ID.
	DocIDs []string

	// FieldTermFreq stores per-field, per-document term frequencies.
	// FieldTermFreq[fieldName][docIndex][term] = count.
	FieldTermFreq map[string][]map[string]int

	// FieldLengths stores per-field document lengths (token counts).
	// FieldLengths[fieldName][docIndex] = length.
	FieldLengths map[string][]int

	// DocFreq maps each term to the number of documents containing it
	// (across all fields combined). Used for IDF computation.
	DocFreq map[string]int

	// IDFCache caches computed IDF values per term.
	IDFCache map[string]float64
}

// NewBM25F creates an empty BM25F index from the given configuration.
func NewBM25F(cfg project.BM25FConfig) *BM25F {
	fieldNames := make([]string, 0, len(cfg.Fields))
	weights := make(map[string]float64, len(cfg.Fields))
	bs := make(map[string]float64, len(cfg.Fields))

	// Sort field names for deterministic ordering.
	sorted := make([]string, 0, len(cfg.Fields))
	for name := range cfg.Fields {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		fc := cfg.Fields[name]
		fieldNames = append(fieldNames, name)
		weights[name] = fc.Weight
		bs[name] = fc.B
	}

	fieldTermFreq := make(map[string][]map[string]int, len(fieldNames))
	fieldLengths := make(map[string][]int, len(fieldNames))
	avgFieldLen := make(map[string]float64, len(fieldNames))
	for _, name := range fieldNames {
		fieldTermFreq[name] = make([]map[string]int, 0)
		fieldLengths[name] = make([]int, 0)
	}

	return &BM25F{
		K1:            cfg.K1,
		FieldNames:    fieldNames,
		FieldWeights:  weights,
		FieldB:        bs,
		AvgFieldLen:   avgFieldLen,
		DocCount:      0,
		DocIDs:        make([]string, 0),
		FieldTermFreq: fieldTermFreq,
		FieldLengths:  fieldLengths,
		DocFreq:       make(map[string]int),
		IDFCache:      make(map[string]float64),
	}
}

// AddDocument adds a document with per-field tokenized content.
// The fields map should contain fieldName → tokens for each configured field.
// Fields not present in the map are treated as empty.
func (idx *BM25F) AddDocument(chunkID string, fields map[string][]string) {
	idx.DocIDs = append(idx.DocIDs, chunkID)
	docIdx := idx.DocCount
	idx.DocCount++

	// Track which terms appear in this document (across all fields)
	// for corpus-level document frequency.
	seenTerms := make(map[string]bool)

	for _, fieldName := range idx.FieldNames {
		tokens := fields[fieldName]

		// Build term frequency for this field.
		tf := make(map[string]int, len(tokens))
		for _, t := range tokens {
			tf[t]++
			seenTerms[t] = true
		}

		// Ensure slice is long enough for this docIndex.
		for len(idx.FieldTermFreq[fieldName]) <= docIdx {
			idx.FieldTermFreq[fieldName] = append(idx.FieldTermFreq[fieldName], nil)
		}
		idx.FieldTermFreq[fieldName][docIdx] = tf

		for len(idx.FieldLengths[fieldName]) <= docIdx {
			idx.FieldLengths[fieldName] = append(idx.FieldLengths[fieldName], 0)
		}
		idx.FieldLengths[fieldName][docIdx] = len(tokens)
	}

	// Update corpus-level document frequency.
	for term := range seenTerms {
		idx.DocFreq[term]++
	}
}

// Build finalizes the index by computing average field lengths and
// pre-caching IDF values. Must be called after all documents are added.
func (idx *BM25F) Build() {
	if idx.DocCount == 0 {
		return
	}

	// Compute average field lengths.
	for _, fieldName := range idx.FieldNames {
		total := 0
		for _, l := range idx.FieldLengths[fieldName] {
			total += l
		}
		idx.AvgFieldLen[fieldName] = float64(total) / float64(idx.DocCount)
	}

	// Pre-compute IDF using the Lucene-adjusted formula.
	idx.IDFCache = make(map[string]float64, len(idx.DocFreq))
	n := float64(idx.DocCount)
	for term, docFreq := range idx.DocFreq {
		df := float64(docFreq)
		idx.IDFCache[term] = math.Log(1 + (n-df+0.5)/(df+0.5))
	}
}

// Score computes BM25F relevance scores for weighted query terms and returns
// the top N results sorted by score descending.
func (idx *BM25F) Score(query []WeightedTerm, topN int) []ScoredResult {
	if idx.DocCount == 0 || len(query) == 0 {
		return nil
	}

	scores := make([]float64, idx.DocCount)

	for _, term := range query {
		idf, ok := idx.IDFCache[term.Text]
		if !ok {
			continue
		}

		for docIdx := 0; docIdx < idx.DocCount; docIdx++ {
			// Compute the weighted pseudo-TF across all fields.
			var weightedTF float64
			for _, fieldName := range idx.FieldNames {
				tfMap := idx.FieldTermFreq[fieldName][docIdx]
				if tfMap == nil {
					continue
				}
				tf := float64(tfMap[term.Text])
				if tf == 0 {
					continue
				}

				b := idx.FieldB[fieldName]
				avgLen := idx.AvgFieldLen[fieldName]
				docLen := float64(idx.FieldLengths[fieldName][docIdx])

				// Per-field normalized TF.
				var normTF float64
				if avgLen == 0 {
					normTF = tf
				} else {
					normTF = tf / (1 - b + b*docLen/avgLen)
				}
				weightedTF += idx.FieldWeights[fieldName] * normTF
			}

			if weightedTF == 0 {
				continue
			}

			// BM25 saturation applied to the combined weighted TF.
			bm25Score := idf * (weightedTF * (idx.K1 + 1)) / (weightedTF + idx.K1)
			scores[docIdx] += bm25Score * term.Weight
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

	if topN > 0 && len(results) > topN {
		results = results[:topN]
	}

	return results
}
