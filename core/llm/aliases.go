package llm

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// defaultAliasBatchSize is the number of terms per API call.
const defaultAliasBatchSize = 50

// AliasRequest represents a single taxonomy term to generate aliases for.
type AliasRequest struct {
	// Key is the normalized term key (e.g. "authentication").
	Key string

	// Canonical is the preferred display label (e.g. "Authentication").
	Canonical string

	// Source is the discovery method (heading, code_identifier, etc.).
	Source string

	// Broader is the parent term key in the hierarchy, or empty.
	Broader string

	// Narrower lists child term keys.
	Narrower []string

	// Related lists lateral related term keys.
	Related []string
}

// AliasResult holds the output of batched alias generation.
type AliasResult struct {
	// Aliases maps normalized term keys to their generated aliases.
	Aliases map[string][]string

	// TotalAliases is the sum of all generated aliases across all terms.
	TotalAliases int

	// Errors is the number of batch requests that failed (non-fatal).
	Errors int
}

// GenerateAliases sends batched alias generation requests to the LLM.
//
// Terms are first grouped by taxonomy branch (terms sharing a broader
// parent appear together in the same batch) for better LLM context.
// Each batch contains up to batchSize terms (default 50 if <= 0).
//
// The instructions parameter is the content of the taxonomy-generation
// README.md file, used as the system prompt.
//
// Returns partial results if some batches fail (graceful degradation).
func GenerateAliases(ctx context.Context, sender Sender, terms []*AliasRequest, instructions string, maxAliasCount, batchSize int) *AliasResult {
	if batchSize <= 0 {
		batchSize = defaultAliasBatchSize
	}
	if maxAliasCount <= 0 {
		maxAliasCount = 10
	}

	result := &AliasResult{
		Aliases: make(map[string][]string),
	}

	if len(terms) == 0 {
		return result
	}

	// Group terms by taxonomy branch for better context.
	grouped := groupByBranch(terms)

	// Split into batches.
	for i := 0; i < len(grouped); i += batchSize {
		end := i + batchSize
		if end > len(grouped) {
			end = len(grouped)
		}
		batch := grouped[i:end]

		prompt := buildAliasBatchPrompt(batch, maxAliasCount)

		resp, err := sender.SendAliases(ctx, instructions, prompt)
		if err != nil {
			result.Errors++
			continue
		}

		// Apply max alias limit and merge into result.
		applyMaxAliases(resp, maxAliasCount)

		// Build a set of valid keys for this batch.
		validKeys := make(map[string]bool, len(batch))
		for _, req := range batch {
			validKeys[req.Key] = true
		}

		for _, term := range resp.Terms {
			if !validKeys[term.Key] {
				continue // Ignore unexpected keys from the LLM.
			}
			if len(term.Aliases) > 0 {
				result.Aliases[term.Key] = term.Aliases
				result.TotalAliases += len(term.Aliases)
			}
		}
	}

	return result
}

// buildAliasBatchPrompt constructs the user message for a batch of terms.
func buildAliasBatchPrompt(batch []*AliasRequest, maxAliasCount int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Generate aliases for each term below. For each term, provide common "+
		"abbreviations, synonyms, casual shorthand, and related acronyms that a developer "+
		"might search for.\n\n")
	fmt.Fprintf(&b, "Maximum %d aliases per term.\n\n", maxAliasCount)
	b.WriteString("Terms:\n")

	for _, req := range batch {
		b.WriteString("\n---\n")
		fmt.Fprintf(&b, "Key: %s\n", req.Key)
		fmt.Fprintf(&b, "Canonical: %s\n", req.Canonical)
		fmt.Fprintf(&b, "Source: %s\n", req.Source)

		if req.Broader != "" {
			fmt.Fprintf(&b, "Broader: %s\n", req.Broader)
		} else {
			b.WriteString("Broader: (none)\n")
		}

		if len(req.Narrower) > 0 {
			fmt.Fprintf(&b, "Narrower: %s\n", strings.Join(req.Narrower, ", "))
		}

		if len(req.Related) > 0 {
			fmt.Fprintf(&b, "Related: %s\n", strings.Join(req.Related, ", "))
		}
	}

	return b.String()
}

// groupByBranch sorts terms so that terms sharing the same broader parent
// are adjacent in the list. This gives the LLM better context for generating
// aliases within a semantic cluster.
//
// Within each group, terms are sorted alphabetically by key for determinism.
// Top-level terms (no broader parent) form their own group, sorted last.
func groupByBranch(terms []*AliasRequest) []*AliasRequest {
	// Group by broader key.
	groups := make(map[string][]*AliasRequest)
	for _, t := range terms {
		key := t.Broader
		if key == "" {
			key = "\xff" // Sort top-level terms last.
		}
		groups[key] = append(groups[key], t)
	}

	// Sort group keys.
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Sort terms within each group by key.
	result := make([]*AliasRequest, 0, len(terms))
	for _, k := range keys {
		group := groups[k]
		sort.Slice(group, func(i, j int) bool {
			return group[i].Key < group[j].Key
		})
		result = append(result, group...)
	}

	return result
}

// applyMaxAliases truncates each term's aliases to maxAliasCount.
func applyMaxAliases(resp *AliasResponse, maxAliasCount int) {
	for i := range resp.Terms {
		if len(resp.Terms[i].Aliases) > maxAliasCount {
			resp.Terms[i].Aliases = resp.Terms[i].Aliases[:maxAliasCount]
		}
	}
}
