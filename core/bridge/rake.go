// Package bridge generates deterministic one-line bridge summaries for
// adjacent chunk boundaries. Bridges summarize what the previous chunk
// established that the next chunk assumes the reader already knows.
//
// Three layers compose to produce each bridge:
//   - Heading transition: detects when the heading breadcrumb path changes
//   - RAKE key phrase extraction: surfaces technical terms from the chunk tail
//   - Last sentence: extracts the concluding prose sentence when available
//
// The package has no external dependencies beyond the standard library.
package bridge

import (
	"sort"
	"strings"
)

// rakePhrase is a candidate key phrase scored by RAKE.
type rakePhrase struct {
	text  string
	score float64
}

// phraseDelimiters splits text at punctuation boundaries for RAKE candidate
// phrase extraction. These delimiters separate semantic units in prose.
var phraseDelimiters = [...]byte{'.', '!', '?', ',', ';', ':', '\n', '\t', '(', ')', '[', ']', '"', '\''}

// extractKeyPhrases implements RAKE (Rapid Automatic Keyword Extraction)
// to extract the top-N scored phrases from text.
//
// RAKE scores multi-word phrases by degree(w)/freq(w), which naturally
// surfaces compound technical terms like "dependency injection" over
// generic words like "used" or "pattern".
func extractKeyPhrases(text string, topN int) []rakePhrase {
	if topN <= 0 || text == "" {
		return nil
	}

	words := tokenizeForRAKE(text)
	if len(words) == 0 {
		return nil
	}

	// Build candidate phrases: sequences of non-stop words.
	phrases := buildCandidatePhrases(words)
	if len(phrases) == 0 {
		return nil
	}

	// Score words: degree(w) / freq(w).
	wordScore := scoreWords(phrases)

	// Score each phrase and deduplicate.
	return scorePhrases(phrases, wordScore, topN)
}

// tokenizeForRAKE splits text into lowercase words, replacing phrase
// delimiters with spaces first.
func tokenizeForRAKE(text string) []string {
	cleaned := strings.ToLower(text)
	for _, d := range phraseDelimiters {
		cleaned = strings.ReplaceAll(cleaned, string(d), " ")
	}
	return strings.Fields(cleaned)
}

// buildCandidatePhrases extracts sequences of consecutive non-stop words.
// Stop words and short tokens act as phrase boundaries.
func buildCandidatePhrases(words []string) [][]string {
	var phrases [][]string
	var current []string

	for _, w := range words {
		if rakeStopWords[w] || len(w) < 2 {
			if len(current) > 0 {
				phrases = append(phrases, current)
				current = nil
			}
		} else {
			current = append(current, w)
		}
	}
	if len(current) > 0 {
		phrases = append(phrases, current)
	}

	return phrases
}

// scoreWords computes RAKE word scores as degree(w)/freq(w).
// Degree is the sum of phrase lengths containing the word; frequency is
// the total occurrence count.
func scoreWords(phrases [][]string) map[string]float64 {
	freq := make(map[string]float64)
	degree := make(map[string]float64)

	for _, p := range phrases {
		pLen := float64(len(p))
		for _, w := range p {
			freq[w]++
			degree[w] += pLen
		}
	}

	scores := make(map[string]float64, len(freq))
	for w, f := range freq {
		scores[w] = degree[w] / f
	}
	return scores
}

// maxPhraseWords caps the length of RAKE candidate phrases. Phrases longer
// than this are typically run-on sequences where a stop word was missed.
const maxPhraseWords = 5

// scorePhrases scores candidate phrases, deduplicates, and returns the top-N.
// Phrases exceeding maxPhraseWords are skipped.
func scorePhrases(phrases [][]string, wordScore map[string]float64, topN int) []rakePhrase {
	type scored struct {
		text  string
		score float64
	}

	var results []scored
	for _, p := range phrases {
		if len(p) == 0 || len(p) > maxPhraseWords {
			continue
		}
		var s float64
		for _, w := range p {
			s += wordScore[w]
		}
		results = append(results, scored{strings.Join(p, " "), s})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	seen := make(map[string]bool)
	var out []rakePhrase
	for _, r := range results {
		if !seen[r.text] && len(out) < topN {
			seen[r.text] = true
			out = append(out, rakePhrase(r))
		}
	}
	return out
}

// rakeStopWords is a minimal set of English stop words for RAKE phrase boundary
// detection. These are function words that carry no semantic weight in technical
// documentation contexts.
var rakeStopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"but": true, "in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true, "from": true,
	"is": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "being": true, "has": true, "have": true, "had": true,
	"do": true, "does": true, "did": true, "will": true, "would": true,
	"shall": true, "should": true, "may": true, "might": true, "can": true,
	"could": true, "must": true,
	"that": true, "this": true, "these": true, "those": true,
	"which": true, "who": true, "whom": true, "whose": true,
	"what": true, "where": true, "when": true, "how": true, "why": true,
	"it": true, "its": true, "if": true, "then": true, "else": true,
	"than": true, "so": true, "as": true, "not": true, "no": true,
	"also": true, "each": true, "every": true, "all": true, "any": true,
	"both": true, "few": true, "more": true, "most": true, "other": true,
	"some": true, "such": true, "only": true, "same": true, "into": true,
	"about": true, "above": true, "after": true, "before": true, "between": true,
	"through": true, "during": true, "until": true, "against": true,
	"up": true, "down": true, "out": true, "off": true, "over": true,
	"under": true, "again": true, "further": true, "once": true,
	// Verbs commonly acting as function words in technical prose.
	"using": true, "used": true, "use": true, "uses": true,
	"ensures": true, "ensure": true, "requires": true, "require": true,
	"begins": true, "begin": true, "allows": true, "allow": true,
	"provides": true, "provide": true, "contains": true, "contain": true,
	"includes": true, "include": true, "needs": true, "need": true,
	"makes": true, "make": true, "takes": true, "take": true,
	"gets": true, "get": true, "sets": true, "set": true,
	"just": true, "like": true, "well": true, "very": true,
}
