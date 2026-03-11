package taxonomy

import (
	"time"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
)

// Result holds the output of the taxonomy extraction pipeline.
type Result struct {
	// Taxonomy is the compiled taxonomy ready for serialization.
	Taxonomy *Taxonomy

	// ChunkTerms maps chunk ID -> sorted list of term keys.
	// Used to populate ManifestEntry.Terms.
	ChunkTerms map[string][]string

	// Stats holds extraction statistics for heuristics reporting.
	Stats Stats

	// Seconds is the wall-clock duration of the extraction.
	Seconds float64
}

// Stats holds per-source extraction counts.
type Stats struct {
	CanonicalTerms      int
	TermsFromHeadings   int
	TermsFromCodeIdents int
	TermsFromBoldTerms  int
	TermsFromStructured int
}

// Extract runs the full structural taxonomy extraction pipeline:
//
//  1. Structural term extraction (headings, code identifiers, bold terms,
//     structured positions)
//  2. Deduplication and frequency filtering
//  3. Relationship inference (heading hierarchy, cross-references)
//
// The encoding and instructionsHash parameters are stored in the taxonomy
// metadata for cache invalidation on incremental builds.
func Extract(chunks []chunk.Chunk, cfg project.TaxonomyConfig, encoding, instructionsHash string) *Result {
	start := time.Now()

	// Pass 1: Structural term extraction.
	candidates := extractStructural(chunks)

	// Pass 4 (partial): Deduplication and frequency filtering.
	// Runs before relationship inference so that only surviving terms
	// get relationships assigned.
	minFreq := cfg.MinTermFrequency
	if minFreq <= 0 {
		minFreq = 2
	}
	terms := deduplicate(candidates, minFreq)

	// Build the taxonomy structure.
	tax := &Taxonomy{
		Encoding:         encoding,
		CompiledAt:       manifest.CompiledAtNow(),
		InstructionsHash: instructionsHash,
		TermCount:        len(terms),
		Terms:            terms,
	}

	// Pass 2: Relationship inference.
	inferRelationships(tax, chunks)

	// Build chunk -> terms reverse map.
	chunkTerms := buildChunkTermsMap(terms)

	// Compute stats.
	stats := computeStats(terms)

	return &Result{
		Taxonomy:   tax,
		ChunkTerms: chunkTerms,
		Stats:      stats,
		Seconds:    time.Since(start).Seconds(),
	}
}

// computeStats counts terms by source type.
func computeStats(terms map[string]*Term) Stats {
	var s Stats
	s.CanonicalTerms = len(terms)

	for _, term := range terms {
		switch term.Source {
		case SourceHeading:
			s.TermsFromHeadings++
		case SourceCodeIdentifier:
			s.TermsFromCodeIdents++
		case SourceBoldTerm:
			s.TermsFromBoldTerms++
		case SourceStructuredPosition:
			s.TermsFromStructured++
		}
	}

	return s
}
