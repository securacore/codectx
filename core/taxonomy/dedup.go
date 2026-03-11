package taxonomy

import (
	"sort"
)

// deduplicate performs the structural portion of Pass 4: merging duplicate
// terms, applying frequency scoring, and filtering by min_term_frequency.
//
// Merge rules:
//   - When the same normalized key appears from multiple sources, the
//     highest-confidence source wins (heading > code > bold > structured).
//   - When a heading-derived form exists, it becomes the canonical label.
//   - Chunk lists are merged across all candidates with the same key.
//
// Filtering:
//   - Terms appearing in fewer than minTermFrequency distinct chunks are
//     removed from the taxonomy.
func deduplicate(candidates []candidate, minTermFrequency int) map[string]*Term {
	// Accumulate candidates by normalized key.
	type accumulated struct {
		canonical string
		source    string
		chunks    map[string]bool
	}

	acc := make(map[string]*accumulated)

	for _, c := range candidates {
		key := NormalizeKey(c.canonical)
		if key == "" {
			continue
		}

		existing, ok := acc[key]
		if !ok {
			acc[key] = &accumulated{
				canonical: c.canonical,
				source:    c.source,
				chunks:    map[string]bool{c.chunkID: true},
			}
			continue
		}

		// Merge: higher-confidence source wins both source and canonical form.
		if sourceRank(c.source) < sourceRank(existing.source) {
			existing.canonical = c.canonical
			existing.source = c.source
		}
		// When sources have equal rank, the first-seen canonical form is kept.

		existing.chunks[c.chunkID] = true
	}

	// Apply frequency filter and build Term map.
	terms := make(map[string]*Term)

	for key, a := range acc {
		if len(a.chunks) < minTermFrequency {
			continue
		}

		chunkList := make([]string, 0, len(a.chunks))
		for chunkID := range a.chunks {
			chunkList = append(chunkList, chunkID)
		}
		sort.Strings(chunkList)

		terms[key] = &Term{
			Canonical: a.canonical,
			Source:    a.source,
			Chunks:    chunkList,
		}
	}

	return terms
}

// buildChunkTermsMap builds the reverse mapping: chunk ID -> list of term keys.
// This is used to populate ManifestEntry.Terms.
func buildChunkTermsMap(terms map[string]*Term) map[string][]string {
	result := make(map[string][]string)

	for key, term := range terms {
		for _, chunkID := range term.Chunks {
			result[chunkID] = append(result[chunkID], key)
		}
	}

	// Sort term lists for deterministic output.
	for chunkID := range result {
		sort.Strings(result[chunkID])
	}

	return result
}
