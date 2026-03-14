package query

import (
	"math"
	"sort"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
)

// windowRatio defines how many candidates the graph re-ranker considers
// relative to the output size. 2.15 = one full output list of headroom
// plus a cherry-picking buffer for graph-promoted candidates.
const windowRatio = 2.15

// GraphRerank applies graph-based score boosts to RRF results using
// relationship signals from the manifest and metadata.
//
// Three boost signals:
//   - Adjacent: chunk's previous/next neighbor also scored in the top window
//   - Spec: chunk's paired spec/object counterpart also scored
//   - CrossRef: chunk's source document cross-references another scored document
//
// The window size is derived as ceil(topN × windowRatio).
func GraphRerank(
	results []RRFResult,
	mfst *manifest.Manifest,
	metadata *manifest.Metadata,
	cfg project.GraphRerankConfig,
	topN int,
) []RRFResult {
	if !cfg.EffectiveEnabled() || len(results) == 0 || mfst == nil {
		return results
	}

	windowSize := int(math.Ceil(float64(topN) * windowRatio))
	if windowSize > len(results) {
		windowSize = len(results)
	}

	// Build a set of chunk IDs that scored in the top window.
	scoredIDs := make(map[string]bool, windowSize)
	for i := 0; i < windowSize; i++ {
		scoredIDs[results[i].ID] = true
	}

	// Build a set of source documents that appear in scored results.
	scoredDocs := make(map[string]bool)
	for id := range scoredIDs {
		if entry := mfst.LookupEntry(id); entry != nil {
			scoredDocs[entry.Source] = true
		}
	}

	// Apply boosts.
	boosted := make([]RRFResult, len(results))
	copy(boosted, results)

	for i := range boosted {
		entry := mfst.LookupEntry(boosted[i].ID)
		if entry == nil {
			continue
		}

		multiplier := 1.0

		// Adjacent chunk boost.
		if entry.Adjacent != nil {
			if entry.Adjacent.Previous != nil && scoredIDs[*entry.Adjacent.Previous] {
				multiplier += cfg.AdjacentBoost
			}
			if entry.Adjacent.Next != nil && scoredIDs[*entry.Adjacent.Next] {
				multiplier += cfg.AdjacentBoost
			}
		}

		// Spec chunk boost — if this chunk's spec also scored.
		if entry.SpecChunk != nil && scoredIDs[*entry.SpecChunk] {
			multiplier += cfg.SpecBoost
		}
		// Reverse: if this is a spec chunk and its parent object scored.
		if entry.ParentObject != nil && scoredIDs[*entry.ParentObject] {
			multiplier += cfg.SpecBoost
		}

		// Cross-reference boost — if a document this chunk's source
		// references (or is referenced by) also has scored chunks.
		if metadata != nil {
			if docEntry, ok := metadata.Documents[entry.Source]; ok {
				for _, ref := range docEntry.ReferencesTo {
					if scoredDocs[ref.Path] {
						multiplier += cfg.CrossRefBoost
						break
					}
				}
				for _, ref := range docEntry.ReferencedBy {
					if scoredDocs[ref.Path] {
						multiplier += cfg.CrossRefBoost
						break
					}
				}
			}
		}

		boosted[i].RRFScore *= multiplier
	}

	// Re-sort after boosting.
	sort.Slice(boosted, func(i, j int) bool {
		if boosted[i].RRFScore != boosted[j].RRFScore {
			return boosted[i].RRFScore > boosted[j].RRFScore
		}
		return boosted[i].ID < boosted[j].ID
	})

	return boosted
}
