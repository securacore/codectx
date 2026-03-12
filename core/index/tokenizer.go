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

	"github.com/kljensen/snowball"
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

// isCodeIdentifier reports whether a lowercased token looks like a code
// identifier that should not be stemmed. Criteria:
//   - Contains a dot (dotted path: http.handler, os.path.join)
//   - Contains an underscore (snake_case: user_id, get_name)
//   - Contains both letters and digits (version-like: v2, sha256, http2)
//   - Is all uppercase in the original form (constant: API, JWT, HTTP)
func isCodeIdentifier(lower, original string) bool {
	if strings.ContainsAny(lower, "._") {
		return true
	}

	// Check for mixed letters+digits (e.g. "sha256", "v2", "http2").
	hasLetter := false
	hasDigit := false
	for _, r := range lower {
		if r >= 'a' && r <= 'z' {
			hasLetter = true
		} else if r >= '0' && r <= '9' {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}

	// Pure numeric tokens (like "200") are not code identifiers per se,
	// but the stemmer handles them fine (returns them unchanged).

	// ALL_CAPS in original form (before lowering) — likely an acronym.
	// Must contain at least one letter (pure numeric like "200" is not ALL_CAPS).
	if len(original) >= 2 && hasLetter && original == strings.ToUpper(original) {
		return true
	}

	// CamelCase / PascalCase in original form — likely a code identifier.
	// Detected by finding an uppercase letter after a lowercase letter or
	// a lowercase letter after an uppercase letter (beyond the first char).
	if hasMixedCase(original) {
		return true
	}

	return false
}

// hasMixedCase reports whether s contains an uppercase ASCII letter after the
// first character. This catches camelCase (getUserByID), PascalCase
// (CreateUser), and acronym-prefix identifiers (HTTPServer) while ignoring
// simple sentence-start capitalization (Hello, Running).
func hasMixedCase(s string) bool {
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

// stemToken applies Snowball/Porter2 stemming to a single lowercased token.
// For hyphenated compounds, each component is stemmed independently and
// rejoined. Code identifiers and technical stopwords are returned unchanged.
func stemToken(lower, original string) string {
	// Never stem technical stopwords (e.g. "false" → "fals" is wrong).
	if technicalStopwords[lower] {
		return lower
	}

	// Never stem code identifiers.
	if isCodeIdentifier(lower, original) {
		return lower
	}

	// Hyphenated compounds: stem each component and rejoin.
	if strings.Contains(lower, "-") {
		parts := strings.Split(lower, "-")
		for i, part := range parts {
			if stemmed, err := snowball.Stem(part, "english", false); err == nil {
				parts[i] = stemmed
			}
		}
		return strings.Join(parts, "-")
	}

	// Regular token: stem it.
	if stemmed, err := snowball.Stem(lower, "english", false); err == nil {
		return stemmed
	}
	return lower
}

// Tokenize splits text into search tokens using the domain-aware tokenizer.
//
// Behavior:
//   - Preserves compound terms: "error-handling" stays whole (components stemmed)
//   - Preserves code identifiers: "CreateUser" → "createuser" (not stemmed)
//   - Preserves dotted paths: "http.Handler" → "http.handler" (not stemmed)
//   - Lowercases all tokens for case-insensitive matching
//   - Applies Snowball/Porter2 stemming to regular English tokens
//   - Removes standard English stopwords (40 words)
//   - Preserves technical stopwords: null, void, async, await, true, false, nil, err
//
// The same function is used at both index time and query time, ensuring
// symmetric stemming.
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

		tokens = append(tokens, stemToken(lower, word))
	}

	if len(tokens) == 0 {
		return nil
	}
	return tokens
}
