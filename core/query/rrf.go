package query

import (
	"sort"

	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/project"
)

// RRFResult represents a chunk after Reciprocal Rank Fusion scoring.
type RRFResult struct {
	// ID is the chunk identifier.
	ID string

	// RRFScore is the fused score across all contributing indexes.
	RRFScore float64

	// Sources maps index name to the rank (1-based) in that index's result list.
	Sources map[string]int
}

// WeightedRRF merges multiple ranked lists into a single ranked list using
// Reciprocal Rank Fusion with per-index weight multipliers.
//
// For each chunk appearing in any list:
//
//	RRF(chunk) = Σ weight_i / (k + rank_in_list_i)
//
// A chunk present in multiple indexes accumulates score from each.
func WeightedRRF(lists map[index.IndexType][]index.ScoredResult, cfg project.RRFConfig) []RRFResult {
	scores := make(map[string]*RRFResult)
	k := cfg.K

	for indexName, ranked := range lists {
		w := 1.0
		if cfg.IndexWeights != nil {
			if iw, ok := cfg.IndexWeights[string(indexName)]; ok {
				w = iw
			}
		}

		for rank, chunk := range ranked {
			r, ok := scores[chunk.ChunkID]
			if !ok {
				r = &RRFResult{
					ID:      chunk.ChunkID,
					Sources: make(map[string]int),
				}
				scores[chunk.ChunkID] = r
			}
			r.RRFScore += w * (1.0 / (k + float64(rank+1)))
			r.Sources[string(indexName)] = rank + 1
		}
	}

	results := make([]RRFResult, 0, len(scores))
	for _, r := range scores {
		results = append(results, *r)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].RRFScore != results[j].RRFScore {
			return results[i].RRFScore > results[j].RRFScore
		}
		// Stable tie-breaking by ID.
		return results[i].ID < results[j].ID
	})

	return results
}
