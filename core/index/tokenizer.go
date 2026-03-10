// Package index implements BM25 Okapi search indexing for the codectx
// compilation pipeline. It builds three separate inverted indexes over
// compiled chunks (objects, specs, system) and provides query-time scoring.
//
// The package includes a domain-aware tokenizer that preserves compound
// terms, code identifiers, and dotted paths while filtering standard
// English stopwords. Technical stopwords (null, void, async, etc.) are
// preserved because they carry semantic weight in code documentation.
//
// The BM25 implementation follows the Okapi BM25 scoring formula with
// configurable k1 (term frequency saturation) and b (document length
// normalization) parameters. Indexes are serialized to disk using gob
// encoding for persistence across CLI invocations.
package index

import (
	"regexp"
	"strings"
)

// tokenPattern matches word tokens while preserving compound terms.
//
// Two alternations:
//   - [\w][\w\-\.]*[\w] — matches hyphenated compounds (error-handling),
//     dotted paths (http.Handler), and multi-character identifiers
//   - [\w]+ — matches single-character and plain tokens
var tokenPattern = regexp.MustCompile(`[\w][\w\-\.]*[\w]|[\w]+`)

// standardStopwords contains common English function words that add noise
// to search scoring. These are filtered during tokenization unless they
// also appear in technicalStopwords.
var standardStopwords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "could": true, "should": true,
	"may": true, "might": true, "shall": true, "can": true,
	"to": true, "of": true, "in": true, "for": true, "on": true,
	"with": true, "at": true, "by": true, "from": true, "as": true,
	"into": true, "through": true, "during": true, "before": true, "after": true,
	"this": true, "that": true, "these": true, "those": true,
	"it": true, "its": true, "and": true, "or": true, "but": true, "not": true,
}

// technicalStopwords are terms that would normally be filtered as stopwords
// but carry semantic weight in code documentation. When a token is in both
// standardStopwords and technicalStopwords, the technical list wins and
// the token is preserved.
var technicalStopwords = map[string]bool{
	"null": true, "void": true, "async": true, "await": true,
	"true": true, "false": true, "nil": true, "err": true,
}

// Tokenize splits text into search tokens using the domain-aware tokenizer.
//
// Behavior:
//   - Preserves compound terms: "error-handling" stays whole
//   - Preserves code identifiers: "CreateUser" → "createuser"
//   - Preserves dotted paths: "http.Handler" → "http.handler"
//   - Lowercases all tokens for case-insensitive matching
//   - Removes standard English stopwords (40 words)
//   - Preserves technical stopwords: null, void, async, await, true, false, nil, err
//
// Returns nil for empty input.
func Tokenize(text string) []string {
	if text == "" {
		return nil
	}

	words := tokenPattern.FindAllString(text, -1)
	if len(words) == 0 {
		return nil
	}

	tokens := make([]string, 0, len(words))
	for _, word := range words {
		lower := strings.ToLower(word)

		// Skip standard stopwords unless they're technical terms.
		if standardStopwords[lower] && !technicalStopwords[lower] {
			continue
		}

		tokens = append(tokens, lower)
	}

	if len(tokens) == 0 {
		return nil
	}
	return tokens
}
