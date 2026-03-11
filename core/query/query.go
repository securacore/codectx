// Package query implements the search and assembly interface for codectx.
// It loads compiled BM25 indexes and manifests, runs queries against all
// three index types (objects, specs, system), enriches results with manifest
// metadata, and assembles selected chunks into coherent reading documents.
//
// This package is consumed by the cmds/query and cmds/generate CLI commands.
// It does not handle taxonomy-based query expansion (deferred to Step 9).
package query

import (
	"fmt"
	"path/filepath"

	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
)

// QueryResult holds the complete result of a search query across all indexes.
type QueryResult struct {
	// RawQuery is the original search query string.
	RawQuery string

	// Instructions contains scored object (instruction) chunk results.
	Instructions []ResultEntry

	// Reasoning contains scored spec (reasoning) chunk results.
	Reasoning []ResultEntry

	// System contains scored system chunk results.
	System []ResultEntry

	// Related contains adjacent chunks to top results that were not
	// themselves scored in the results. Useful for exploration.
	Related []RelatedEntry
}

// ResultEntry is a single scored query result enriched with manifest metadata.
type ResultEntry struct {
	// ChunkID is the full chunk identifier (e.g. "obj:a1b2c3.03").
	ChunkID string

	// Score is the BM25 relevance score.
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
// It loads the BM25 indexes from disk, runs the query against all three
// index types, enriches results with manifest metadata, and collects
// related adjacent chunks.
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

	// Query all three indexes.
	allResults := idx.QueryAll(query, topN)

	result := &QueryResult{
		RawQuery: query,
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
