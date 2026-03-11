package index

import (
	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/project"
)

// IndexType identifies which of the three BM25 indexes to query.
type IndexType string

const (
	// IndexObjects is the index over instruction chunks from compiled/objects/.
	IndexObjects IndexType = "objects"

	// IndexSpecs is the index over reasoning chunks from compiled/specs/.
	IndexSpecs IndexType = "specs"

	// IndexSystem is the index over system/compiler chunks from compiled/system/.
	IndexSystem IndexType = "system"
)

// chunkTypeToIndex maps chunk.ChunkType to the corresponding IndexType.
var chunkTypeToIndex = map[chunk.ChunkType]IndexType{
	chunk.ChunkObject: IndexObjects,
	chunk.ChunkSpec:   IndexSpecs,
	chunk.ChunkSystem: IndexSystem,
}

// indexTypeForChunk returns the IndexType corresponding to a ChunkType.
// Defaults to IndexObjects for unknown chunk types.
func indexTypeForChunk(ct chunk.ChunkType) IndexType {
	if it, ok := chunkTypeToIndex[ct]; ok {
		return it
	}
	return IndexObjects
}

// allIndexTypes returns all three index types in a stable order.
func allIndexTypes() []IndexType {
	return []IndexType{IndexObjects, IndexSpecs, IndexSystem}
}

// Index manages three BM25 indexes — one for each content type.
// Each index has its own IDF calculations scoped to its corpus, so
// different content types don't dilute each other's relevance scores.
type Index struct {
	// Indexes maps each IndexType to its BM25 index.
	Indexes map[IndexType]*BM25
}

// New creates an Index with three empty BM25 indexes using the given
// scoring parameters.
func New(k1, b float64) *Index {
	return &Index{
		Indexes: map[IndexType]*BM25{
			IndexObjects: NewBM25(k1, b),
			IndexSpecs:   NewBM25(k1, b),
			IndexSystem:  NewBM25(k1, b),
		},
	}
}

// NewFromConfig creates an Index using BM25 parameters from the
// preferences configuration.
func NewFromConfig(cfg project.BM25Config) *Index {
	return New(cfg.K1, cfg.B)
}

// BuildFromChunks populates the three indexes from a slice of chunks.
// Each chunk is routed to the appropriate index based on its Type.
// The chunk's Content field (not the meta header) is tokenized and indexed.
//
// After all chunks are added, Build() is called on each index to
// finalize IDF caches and average document lengths.
func (idx *Index) BuildFromChunks(chunks []chunk.Chunk) {
	for i := range chunks {
		c := &chunks[i]
		it := indexTypeForChunk(c.Type)
		bm25 := idx.Indexes[it]

		tokens := Tokenize(c.Content)
		bm25.AddDocument(c.ID, tokens)
	}

	// Finalize all indexes.
	for _, bm25 := range idx.Indexes {
		bm25.Build()
	}
}

// Query runs a search query against a specific index type.
// The raw query string is tokenized using the domain-aware tokenizer
// before scoring.
//
// Returns up to topN results sorted by score descending.
// If topN <= 0, all matching results are returned.
func (idx *Index) Query(indexType IndexType, query string, topN int) []ScoredResult {
	bm25, ok := idx.Indexes[indexType]
	if !ok {
		return nil
	}
	tokens := Tokenize(query)
	if len(tokens) == 0 {
		return nil
	}
	return bm25.Score(tokens, topN)
}

// QueryAll runs the query against all three indexes and returns results
// grouped by index type.
func (idx *Index) QueryAll(query string, topN int) map[IndexType][]ScoredResult {
	results := make(map[IndexType][]ScoredResult, len(idx.Indexes))
	tokens := Tokenize(query)
	if len(tokens) == 0 {
		return results
	}

	for it, bm25 := range idx.Indexes {
		r := bm25.Score(tokens, topN)
		if len(r) > 0 {
			results[it] = r
		}
	}

	return results
}

// IndexStats holds statistics about a single BM25 index, used for
// heuristics.yaml reporting.
type IndexStats struct {
	// IndexedTerms is the number of unique terms in the index.
	IndexedTerms int

	// IndexedChunks is the number of documents in the index.
	IndexedChunks int

	// AvgChunkLength is the mean document length in tokens.
	AvgChunkLength float64
}

// Stats returns statistics for a specific index type.
// Returns zero-value stats if the index type is not found.
func (idx *Index) Stats(indexType IndexType) IndexStats {
	bm25, ok := idx.Indexes[indexType]
	if !ok {
		return IndexStats{}
	}
	return IndexStats{
		IndexedTerms:   len(bm25.TermDocFreq),
		IndexedChunks:  bm25.DocCount,
		AvgChunkLength: bm25.AvgDocLen,
	}
}
