package query

import (
	"regexp"
	"strings"

	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/project"
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

// ExpandedQuery holds the result of weighted query expansion.
type ExpandedQuery struct {
	// Original is the raw query string.
	Original string

	// Terms are the weighted terms for BM25F scoring.
	Terms []index.WeightedTerm

	// FlatTokens are the plain stemmed tokens for flat BM25 scoring.
	FlatTokens []string

	// Display is a human-readable string showing the expanded query.
	Display string
}

// ExpandQueryWeighted performs taxonomy-based query expansion with
// weighted terms for BM25F scoring. Each expansion tier gets a different
// weight multiplier:
//   - Original terms: weight 1.0
//   - Aliases: cfg.AliasWeight (default 1.0)
//   - Narrower terms: cfg.NarrowerWeight (default 0.7)
//   - Related terms: cfg.RelatedWeight (default 0.4)
//
// Returns an ExpandedQuery with both weighted terms (for BM25F) and
// flat tokens (for BM25 compatibility).
func ExpandQueryWeighted(rawQuery string, tax *taxonomy.Taxonomy, aliasIdx *taxonomy.AliasIndex, cfg project.ExpansionConfig) ExpandedQuery {
	stemmed := index.Tokenize(rawQuery)
	if len(stemmed) == 0 {
		return ExpandedQuery{Original: rawQuery}
	}

	result := ExpandedQuery{Original: rawQuery}
	seen := make(map[string]bool, len(stemmed)*3)

	// Add all stemmed tokens at full weight as originals.
	for _, tok := range stemmed {
		if !seen[tok] {
			seen[tok] = true
			result.Terms = append(result.Terms, index.WeightedTerm{
				Text:   tok,
				Weight: 1.0,
				Tier:   "original",
			})
		}
	}

	if !cfg.EffectiveEnabled() || tax == nil || aliasIdx == nil {
		result.FlatTokens = flatTokens(result.Terms)
		result.Display = strings.Join(result.FlatTokens, " ")
		return result
	}

	// Extract raw (unstemmed, lowercased) words for alias index lookups.
	rawWords := rawWordPattern.FindAllString(rawQuery, -1)
	rawWordsLower := make([]string, 0, len(rawWords))
	for _, w := range rawWords {
		rawWordsLower = append(rawWordsLower, strings.ToLower(w))
	}

	maxTerms := cfg.MaxExpansionTerms
	if maxTerms <= 0 {
		maxTerms = 20
	}

	for _, raw := range rawWordsLower {
		if len(result.Terms) >= maxTerms {
			break
		}

		termKey := aliasIdx.LookupByAlias(raw)
		if termKey == "" {
			// Fall back to dictionary for tokens not in the taxonomy.
			dictSyns := taxonomy.DictionaryLookup(raw)
			for _, syn := range dictSyns {
				addWeightedTokens(&result.Terms, seen, syn, cfg.AliasWeight, "alias", maxTerms)
			}
			continue
		}

		term := tax.Terms[termKey]
		if term == nil {
			continue
		}

		// Tier 1: Add canonical form at alias weight.
		addWeightedTokens(&result.Terms, seen, term.Canonical, cfg.AliasWeight, "alias", maxTerms)

		// Tier 1: Add all aliases at alias weight.
		for _, alias := range term.Aliases {
			addWeightedTokens(&result.Terms, seen, alias, cfg.AliasWeight, "alias", maxTerms)
		}

		// Tier 2: Add narrower terms at narrower weight.
		for _, childKey := range term.Narrower {
			if childTerm := tax.Terms[childKey]; childTerm != nil {
				addWeightedTokens(&result.Terms, seen, childTerm.Canonical, cfg.NarrowerWeight, "narrower", maxTerms)
			}
		}

		// Tier 3: Add related terms at related weight.
		for _, relKey := range term.Related {
			if relTerm := tax.Terms[relKey]; relTerm != nil {
				addWeightedTokens(&result.Terms, seen, relTerm.Canonical, cfg.RelatedWeight, "related", maxTerms)
			}
		}
	}

	result.FlatTokens = flatTokens(result.Terms)
	result.Display = strings.Join(result.FlatTokens, " ")
	return result
}

// addWeightedTokens tokenizes text through BM25 stemming and adds new terms
// with the given weight and tier, respecting the max cap.
func addWeightedTokens(terms *[]index.WeightedTerm, seen map[string]bool, text string, weight float64, tier string, maxTerms int) {
	toks := index.Tokenize(text)
	for _, tok := range toks {
		if len(*terms) >= maxTerms {
			return
		}
		if !seen[tok] {
			seen[tok] = true
			*terms = append(*terms, index.WeightedTerm{
				Text:   tok,
				Weight: weight,
				Tier:   tier,
			})
		}
	}
}

// flatTokens extracts just the text values from weighted terms.
func flatTokens(terms []index.WeightedTerm) []string {
	result := make([]string, len(terms))
	for i, t := range terms {
		result[i] = t.Text
	}
	return result
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
