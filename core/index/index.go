package index

import (
	"regexp"
	"strings"
	"sync"

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

// query runs a search query against a specific index type.
// The raw query string is tokenized using the domain-aware tokenizer
// before scoring.
//
// Returns up to topN results sorted by score descending.
// If topN <= 0, all matching results are returned.
func (idx *Index) query(indexType IndexType, q string, topN int) []ScoredResult {
	bm25, ok := idx.Indexes[indexType]
	if !ok {
		return nil
	}
	tokens := Tokenize(q)
	if len(tokens) == 0 {
		return nil
	}
	return bm25.Score(tokens, topN)
}

// QueryAll runs the query against all three indexes in parallel and returns
// results grouped by index type.
func (idx *Index) QueryAll(query string, topN int) map[IndexType][]ScoredResult {
	tokens := Tokenize(query)
	if len(tokens) == 0 {
		return make(map[IndexType][]ScoredResult)
	}
	return idx.scoreAll(tokens, topN)
}

// QueryAllWithTokens runs pre-tokenized/expanded tokens against all three
// indexes in parallel. Unlike QueryAll, this skips the Tokenize step,
// allowing the caller to supply tokens that have already been expanded
// (e.g. via taxonomy-based query expansion).
func (idx *Index) QueryAllWithTokens(tokens []string, topN int) map[IndexType][]ScoredResult {
	if len(tokens) == 0 {
		return make(map[IndexType][]ScoredResult)
	}
	return idx.scoreAll(tokens, topN)
}

// scoreAll fans out BM25 scoring across all indexes in parallel and
// collects the results. Each Score call is read-only on an independent
// BM25 struct, so no synchronization is needed beyond the WaitGroup.
func (idx *Index) scoreAll(tokens []string, topN int) map[IndexType][]ScoredResult {
	types := allIndexTypes()
	scored := make([][]ScoredResult, len(types))

	var wg sync.WaitGroup
	for i, it := range types {
		bm25, ok := idx.Indexes[it]
		if !ok {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			scored[i] = bm25.Score(tokens, topN)
		}()
	}
	wg.Wait()

	results := make(map[IndexType][]ScoredResult, len(types))
	for i, it := range types {
		if len(scored[i]) > 0 {
			results[it] = scored[i]
		}
	}

	return results
}

// FieldIndex manages three BM25F field-weighted indexes — one for each
// content type. Unlike Index, documents are split into fields (heading,
// body, code, terms) before indexing.
type FieldIndex struct {
	// Indexes maps each IndexType to its BM25F field index.
	Indexes map[IndexType]*BM25F
}

// NewFieldIndex creates a FieldIndex with three empty BM25F indexes.
func NewFieldIndex(cfg project.BM25FConfig) *FieldIndex {
	return &FieldIndex{
		Indexes: map[IndexType]*BM25F{
			IndexObjects: NewBM25F(cfg),
			IndexSpecs:   NewBM25F(cfg),
			IndexSystem:  NewBM25F(cfg),
		},
	}
}

// codeBlockPattern matches fenced code blocks in markdown.
var codeBlockPattern = regexp.MustCompile("(?s)```[^\n]*\n(.*?)```")

// ChunkFields holds per-field tokenized content for a chunk.
type ChunkFields struct {
	Heading []string
	Body    []string
	Code    []string
	Terms   []string
}

// ParseChunkFields splits a chunk's content into field tokens.
// The heading comes from the chunk's Heading field.
// Terms come from the manifest entry's Terms field.
// Code is extracted from fenced code blocks in the content.
// Body is the remaining prose after removing code blocks.
func ParseChunkFields(content, heading string, terms []string) ChunkFields {
	// Extract code blocks.
	codeMatches := codeBlockPattern.FindAllStringSubmatch(content, -1)
	var codeParts []string
	for _, m := range codeMatches {
		if len(m) > 1 {
			codeParts = append(codeParts, m[1])
		}
	}
	codeText := strings.Join(codeParts, " ")

	// Remove code blocks from body.
	bodyText := codeBlockPattern.ReplaceAllString(content, "")

	// Strip the context header comment block if present.
	if idx := strings.Index(bodyText, "-->"); idx >= 0 {
		bodyText = bodyText[idx+3:]
	}

	return ChunkFields{
		Heading: Tokenize(heading),
		Body:    Tokenize(strings.TrimSpace(bodyText)),
		Code:    Tokenize(codeText),
		Terms:   Tokenize(strings.Join(terms, " ")),
	}
}

// BuildFieldIndexFromChunks populates the three BM25F indexes from chunks.
// The termsByChunkID map provides taxonomy terms for each chunk (from the manifest).
// If termsByChunkID is nil, the terms field is left empty.
func (idx *FieldIndex) BuildFieldIndexFromChunks(chunks []chunk.Chunk, termsByChunkID map[string][]string) {
	for i := range chunks {
		c := &chunks[i]
		it := indexTypeForChunk(c.Type)
		bm25f := idx.Indexes[it]

		var terms []string
		if termsByChunkID != nil {
			terms = termsByChunkID[c.ID]
		}

		fields := ParseChunkFields(c.Content, c.Heading, terms)
		bm25f.AddDocument(c.ID, map[string][]string{
			"heading": fields.Heading,
			"body":    fields.Body,
			"code":    fields.Code,
			"terms":   fields.Terms,
		})
	}

	// Finalize all indexes.
	for _, bm25f := range idx.Indexes {
		bm25f.Build()
	}
}

// QueryAllWeighted runs weighted query terms against all three indexes
// in parallel and returns results grouped by index type.
func (idx *FieldIndex) QueryAllWeighted(terms []WeightedTerm, topN int) map[IndexType][]ScoredResult {
	if len(terms) == 0 {
		return make(map[IndexType][]ScoredResult)
	}
	return idx.scoreAllWeighted(terms, topN)
}

// scoreAllWeighted fans out BM25F scoring across all indexes in parallel.
func (idx *FieldIndex) scoreAllWeighted(terms []WeightedTerm, topN int) map[IndexType][]ScoredResult {
	types := allIndexTypes()
	scored := make([][]ScoredResult, len(types))

	var wg sync.WaitGroup
	for i, it := range types {
		bm25f, ok := idx.Indexes[it]
		if !ok {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			scored[i] = bm25f.Score(terms, topN)
		}()
	}
	wg.Wait()

	results := make(map[IndexType][]ScoredResult, len(types))
	for i, it := range types {
		if len(scored[i]) > 0 {
			results[it] = scored[i]
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
