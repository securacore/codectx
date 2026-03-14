// Package query implements the search and assembly interface for codectx.
// It loads compiled BM25 indexes and manifests, runs queries against all
// three index types (objects, specs, system), enriches results with manifest
// metadata, and assembles selected chunks into coherent reading documents.
//
// Query expansion via the compiled taxonomy is supported: tokens are
// expanded with aliases, synonyms, and narrower terms before BM25 scoring.
//
// This package is consumed by the cmds/query and cmds/generate CLI commands.
package query

import (
	"fmt"
	"path/filepath"

	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
)

// QueryResult holds the complete result of a search query across all indexes.
type QueryResult struct {
	// RawQuery is the original search query string.
	RawQuery string

	// ExpandedQuery is the post-expansion query string, showing all tokens
	// that were actually searched. Empty if no expansion occurred.
	ExpandedQuery string

	// Instructions contains scored object (instruction) chunk results.
	// Populated in BM25 mode. Empty in BM25F mode (use Unified instead).
	Instructions []ResultEntry

	// Reasoning contains scored spec (reasoning) chunk results.
	// Populated in BM25 mode. Empty in BM25F mode (use Unified instead).
	Reasoning []ResultEntry

	// System contains scored system chunk results.
	// Populated in BM25 mode. Empty in BM25F mode (use Unified instead).
	System []ResultEntry

	// Unified contains the fused ranked list from RRF + graph re-ranking.
	// Populated in BM25F mode. Empty in BM25 mode (use per-type lists instead).
	Unified []ResultEntry

	// Related contains adjacent chunks to top results that were not
	// themselves scored in the results. Useful for exploration.
	Related []RelatedEntry
}

// ResultEntry is a single scored query result enriched with manifest metadata.
type ResultEntry struct {
	// ChunkID is the full chunk identifier (e.g. "obj:a1b2c3.03").
	ChunkID string

	// Score is the BM25 relevance score (or RRF score in unified mode).
	Score float64

	// Heading is the heading breadcrumb (e.g. "Authentication > JWT Tokens > Refresh Flow").
	Heading string

	// Source is the relative path to the source markdown file.
	Source string

	// Sequence is the 1-based chunk index within the source file.
	Sequence int

	// TotalInFile is the total chunks from this source file.
	TotalInFile int

	// Tokens is the token count of this chunk.
	Tokens int

	// IndexSources maps index name to rank (1-based) in that index's result list.
	// Only populated in unified (BM25F) mode from RRF fusion.
	IndexSources map[string]int
}

// RelatedEntry is an adjacent chunk referenced by a top result but not
// itself scored in the query results.
type RelatedEntry struct {
	// ChunkID is the chunk identifier.
	ChunkID string

	// Heading is the heading breadcrumb.
	Heading string

	// Tokens is the token count.
	Tokens int
}

// CompiledDir returns the absolute path to the compiled output directory
// for a project.
func CompiledDir(projectDir string, cfg *project.Config) string {
	rootDir := project.RootDir(projectDir, cfg)
	return filepath.Join(rootDir, project.CodectxDir, project.CompiledDir)
}

// RunQuery executes a search query against the compiled documentation.
//
// It loads the BM25 indexes from disk, attempts to load the taxonomy for
// query expansion, runs the (possibly expanded) query against all three
// index types, enriches results with manifest metadata, and collects
// related adjacent chunks.
//
// If the taxonomy cannot be loaded (e.g. first compile hasn't run yet),
// the query proceeds without expansion — this is not an error.
func RunQuery(compiledDir, query string, topN int) (*QueryResult, error) {
	// Load BM25 indexes.
	idx, err := index.Load(compiledDir)
	if err != nil {
		return nil, fmt.Errorf("loading BM25 indexes: %w", err)
	}

	// Load manifest for metadata enrichment.
	mfst, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	// Attempt to load taxonomy for query expansion.
	// Failure is non-fatal — we just skip expansion.
	var tax *taxonomy.Taxonomy
	var aliasIdx *taxonomy.AliasIndex
	taxPath := taxonomy.TaxonomyPath(compiledDir)
	if loaded, loadErr := taxonomy.Load(taxPath); loadErr == nil {
		tax = loaded
		aliasIdx = taxonomy.BuildAliasIndex(tax)
	}

	// Expand query tokens using taxonomy aliases and dictionary.
	expandedTokens, expandedStr := ExpandQuery(query, tax, aliasIdx)

	// Query all three indexes with expanded tokens.
	var allResults map[index.IndexType][]index.ScoredResult
	if len(expandedTokens) > 0 {
		allResults = idx.QueryAllWithTokens(expandedTokens, topN)
	} else {
		allResults = make(map[index.IndexType][]index.ScoredResult)
	}

	result := &QueryResult{
		RawQuery:      query,
		ExpandedQuery: expandedStr,
	}

	// Track all result chunk IDs to exclude from related.
	seen := make(map[string]bool)

	// Enrich results for each index type.
	result.Instructions = enrichResults(allResults[index.IndexObjects], mfst, seen)
	result.Reasoning = enrichResults(allResults[index.IndexSpecs], mfst, seen)
	result.System = enrichResults(allResults[index.IndexSystem], mfst, seen)

	// Collect related chunks from adjacency of top instruction results.
	instrIDs := make([]string, len(result.Instructions))
	for i, e := range result.Instructions {
		instrIDs[i] = e.ChunkID
	}
	result.Related = CollectRelated(instrIDs, mfst, seen)

	return result, nil
}

// RunQueryUnified executes a search query using the BM25F pipeline with
// RRF fusion and graph re-ranking, producing a single unified result list.
//
// Pipeline: Weighted Expansion → BM25F Scoring → RRF Fusion → Graph Re-ranking.
func RunQueryUnified(compiledDir, query string, topN int, queryCfg project.QueryConfig) (*QueryResult, error) {
	// Load BM25F field indexes.
	fieldIdx, err := index.LoadFieldIndex(compiledDir)
	if err != nil {
		return nil, fmt.Errorf("loading BM25F field indexes: %w", err)
	}

	// Load manifest for metadata enrichment and graph re-ranking.
	mfst, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	// Attempt to load taxonomy for query expansion.
	var tax *taxonomy.Taxonomy
	var aliasIdx *taxonomy.AliasIndex
	taxPath := taxonomy.TaxonomyPath(compiledDir)
	if loaded, loadErr := taxonomy.Load(taxPath); loadErr == nil {
		tax = loaded
		aliasIdx = taxonomy.BuildAliasIndex(tax)
	}

	// Layer 1: Weighted taxonomy expansion.
	expanded := ExpandQueryWeighted(query, tax, aliasIdx, queryCfg.Expansion)

	if len(expanded.Terms) == 0 {
		return &QueryResult{
			RawQuery:      query,
			ExpandedQuery: expanded.Display,
		}, nil
	}

	// Layer 2: Parallel BM25F scoring across all three indexes.
	// Use a larger candidate set for RRF — topN*3 per index gives RRF room.
	candidateN := topN * 3
	allResults := fieldIdx.QueryAllWeighted(expanded.Terms, candidateN)

	// Layer 3: Reciprocal Rank Fusion.
	fused := WeightedRRF(allResults, queryCfg.RRF)

	// Layer 4: Graph-based re-ranking.
	// Load metadata for cross-reference boost (nil is handled gracefully).
	var meta *manifest.Metadata
	metaPath := manifest.MetadataPath(compiledDir)
	if loaded, loadErr := manifest.LoadMetadata(metaPath); loadErr == nil {
		meta = loaded
	}

	reranked := GraphRerank(fused, mfst, meta, queryCfg.GraphRerank, topN)

	// Truncate to topN.
	if len(reranked) > topN {
		reranked = reranked[:topN]
	}

	// Enrich with manifest metadata.
	result := &QueryResult{
		RawQuery:      query,
		ExpandedQuery: expanded.Display,
	}

	seen := make(map[string]bool)
	for _, rr := range reranked {
		if entry := mfst.LookupEntry(rr.ID); entry != nil {
			result.Unified = append(result.Unified, ResultEntry{
				ChunkID:      rr.ID,
				Score:        rr.RRFScore,
				Heading:      entry.Heading,
				Source:       entry.Source,
				Sequence:     entry.Sequence,
				TotalInFile:  entry.TotalInFile,
				Tokens:       entry.Tokens,
				IndexSources: rr.Sources,
			})
		}
		seen[rr.ID] = true
	}

	// Collect related chunks from the unified list.
	unifiedIDs := make([]string, len(result.Unified))
	for i, e := range result.Unified {
		unifiedIDs[i] = e.ChunkID
	}
	// Limit related to the top results (not the full unified list).
	maxRelatedSource := min(len(unifiedIDs), topN)
	result.Related = CollectRelated(unifiedIDs[:maxRelatedSource], mfst, seen)

	return result, nil
}

// enrichResults converts scored BM25 results into ResultEntry values enriched
// with manifest metadata. Each chunk ID is added to seen regardless of whether
// the manifest entry exists (to exclude from related).
func enrichResults(scored []index.ScoredResult, mfst *manifest.Manifest, seen map[string]bool) []ResultEntry {
	var entries []ResultEntry
	for _, sr := range scored {
		if entry := mfst.LookupEntry(sr.ChunkID); entry != nil {
			entries = append(entries, ResultEntry{
				ChunkID:     sr.ChunkID,
				Score:       sr.Score,
				Heading:     entry.Heading,
				Source:      entry.Source,
				Sequence:    entry.Sequence,
				TotalInFile: entry.TotalInFile,
				Tokens:      entry.Tokens,
			})
		}
		seen[sr.ChunkID] = true
	}
	return entries
}

// CollectRelated finds adjacent chunks to the given chunk IDs that are not
// already in the seen set. Returns at most 5 related entries.
// Used by both RunQuery and RunGenerate to find related chunks.
func CollectRelated(chunkIDs []string, mfst *manifest.Manifest, seen map[string]bool) []RelatedEntry {
	const maxRelated = 5

	var related []RelatedEntry

	for _, id := range chunkIDs {
		me := mfst.LookupEntry(id)
		if me == nil || me.Adjacent == nil {
			continue
		}

		for _, adjID := range []*string{me.Adjacent.Previous, me.Adjacent.Next} {
			if adjID == nil || seen[*adjID] {
				continue
			}
			seen[*adjID] = true

			if adjEntry := mfst.LookupEntry(*adjID); adjEntry != nil {
				related = append(related, RelatedEntry{
					ChunkID: *adjID,
					Heading: adjEntry.Heading,
					Tokens:  adjEntry.Tokens,
				})
			}

			if len(related) >= maxRelated {
				return related
			}
		}
	}

	return related
}
