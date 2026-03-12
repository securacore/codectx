package query

import (
	"regexp"
	"strings"

	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/taxonomy"
)

// rawWordPattern extracts word tokens from text without stemming, used for
// alias index lookups where we need the original word forms.
var rawWordPattern = regexp.MustCompile(`[\w][\w\-\.]*[\w]|[\w]+`)

// ExpandQuery takes raw query text and expands it using the taxonomy.
//
// The expansion algorithm:
//  1. Extract raw words from the query (no stemming) for alias lookups
//  2. Tokenize the query with stemming for BM25 matching
//  3. For each raw word, look up in the alias index -> resolve to term
//  4. If a term is found, add: stemmed canonical + stemmed aliases +
//     narrower term keys (one level down)
//  5. Also check the dictionary for tokens not found in the taxonomy
//  6. Deduplicate and return the expanded token set
//
// Returns the expanded tokens (stemmed, ready for BM25) and a
// human-readable expansion string.
// If no taxonomy is available, returns the original stemmed tokens unchanged.
func ExpandQuery(rawQuery string, tax *taxonomy.Taxonomy, aliasIdx *taxonomy.AliasIndex) (tokens []string, expanded string) {
	stemmed := index.Tokenize(rawQuery)
	if len(stemmed) == 0 {
		return nil, ""
	}

	if tax == nil || aliasIdx == nil {
		return stemmed, strings.Join(stemmed, " ")
	}

	// Extract raw (unstemmed, lowercased) words for alias index lookups.
	rawWords := rawWordPattern.FindAllString(rawQuery, -1)
	rawWordsLower := make([]string, 0, len(rawWords))
	for _, w := range rawWords {
		rawWordsLower = append(rawWordsLower, strings.ToLower(w))
	}

	seen := make(map[string]bool, len(stemmed)*3)
	var result []string

	// Add all stemmed tokens first (these are the base query).
	for _, tok := range stemmed {
		if !seen[tok] {
			seen[tok] = true
			result = append(result, tok)
		}
	}

	// For each raw word, attempt taxonomy expansion.
	for _, raw := range rawWordsLower {
		termKey := aliasIdx.LookupByAlias(raw)
		if termKey == "" {
			// Fall back to dictionary for tokens not in the taxonomy.
			dictSyns := taxonomy.DictionaryLookup(raw)
			for _, syn := range dictSyns {
				addTokenized(&result, seen, syn)
			}
			continue
		}

		term := tax.Terms[termKey]
		if term == nil {
			continue
		}

		// Add the canonical form — tokenize it through BM25 tokenizer
		// so it matches indexed content (stemmed).
		addTokenized(&result, seen, term.Canonical)

		// Add all aliases — tokenize them through the BM25 tokenizer
		// so they match indexed forms (stemmed).
		for _, alias := range term.Aliases {
			addTokenized(&result, seen, alias)
		}

		// Add direct narrower terms (one level down) — tokenize their
		// canonical forms for BM25 matching.
		for _, childKey := range term.Narrower {
			if childTerm := tax.Terms[childKey]; childTerm != nil {
				addTokenized(&result, seen, childTerm.Canonical)
			}
		}
	}

	return result, strings.Join(result, " ")
}

// addTokenized runs text through the BM25 tokenizer (which applies stemming)
// and adds any new tokens to result, using seen for deduplication.
func addTokenized(result *[]string, seen map[string]bool, text string) {
	toks := index.Tokenize(text)
	for _, tok := range toks {
		if !seen[tok] {
			seen[tok] = true
			*result = append(*result, tok)
		}
	}
}
