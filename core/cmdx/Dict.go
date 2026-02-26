package cmdx

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// BuildDictionary analyzes text segments and returns optimal dictionary entries.
// Algorithm per plan spec: sliding window candidates, frequency analysis,
// greedy selection with overlap handling, deterministic output.
func BuildDictionary(segments []string, opts EncoderOptions) *Dictionary {
	if len(segments) == 0 {
		return nil
	}

	// Step 1: Extract candidate substrings at word boundaries.
	candidates := extractCandidates(segments, opts.MinStringLength)

	// Step 2: Filter by minimum frequency.
	var filtered []candidate
	for _, c := range candidates {
		if c.freq >= opts.MinFrequency && len(c.value) >= opts.MinStringLength {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Step 3: Score each candidate.
	for i := range filtered {
		filtered[i].score = calcScore(filtered[i], i)
	}

	// Step 4: Sort by score descending, tie-break by first occurrence.
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].score != filtered[j].score {
			return filtered[i].score > filtered[j].score
		}
		return filtered[i].firstPos < filtered[j].firstPos
	})

	// Step 5: Greedy selection with overlap handling.
	selected := greedySelect(filtered, segments, opts.MaxDictEntries)
	if len(selected) == 0 {
		return nil
	}

	// Step 6: Assign indices by first occurrence position.
	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].firstPos < selected[j].firstPos
	})

	dict := &Dictionary{
		Entries: make([]DictEntry, len(selected)),
		index:   make(map[string]string),
	}
	for i, c := range selected {
		dict.Entries[i] = DictEntry{Index: i, Value: c.value}
		dict.index[c.value] = fmt.Sprintf("$%d", i)
	}
	return dict
}

type candidate struct {
	value    string
	freq     int
	firstPos int // position of first occurrence in concatenated text
	score    int
}

// extractCandidates finds repeated substrings at word boundaries across all segments.
// Generates candidates from each segment's word boundaries, then counts actual
// frequency across ALL segments (handles URLs and cross-segment matches).
func extractCandidates(segments []string, minLen int) []candidate {
	freqMap := make(map[string]*candidate)
	globalPos := 0

	for _, seg := range segments {
		boundaries := findWordBoundaries(seg)
		for i := 0; i < len(boundaries); i++ {
			for j := i + 1; j < len(boundaries); j++ {
				start := boundaries[i]
				end := boundaries[j]
				substr := strings.TrimSpace(seg[start:end])
				if len(substr) < minLen {
					continue
				}
				if len(substr) > 100 {
					break
				}
				if _, exists := freqMap[substr]; !exists {
					freqMap[substr] = &candidate{
						value:    substr,
						firstPos: globalPos + start,
					}
				}
			}
		}
		globalPos += len(seg) + 1
	}

	// Count actual frequency across all segments using substring search.
	for _, c := range freqMap {
		freq := 0
		for _, seg := range segments {
			freq += strings.Count(seg, c.value)
		}
		c.freq = freq
	}

	result := make([]candidate, 0, len(freqMap))
	for _, c := range freqMap {
		result = append(result, *c)
	}
	// Sort by firstPos for deterministic output (map iteration is random).
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].firstPos < result[j].firstPos
	})
	return result
}

// findWordBoundaries returns positions where word/space transitions occur.
func findWordBoundaries(s string) []int {
	var boundaries []int
	boundaries = append(boundaries, 0)
	prevWasSpace := false
	for i, r := range s {
		isSpace := unicode.IsSpace(r)
		if isSpace && !prevWasSpace {
			boundaries = append(boundaries, i)
		} else if !isSpace && prevWasSpace {
			boundaries = append(boundaries, i)
		}
		prevWasSpace = isSpace
	}
	boundaries = append(boundaries, len(s))
	return boundaries
}

// calcScore computes the net byte savings for a candidate.
// score = (freq - 1) * len(value) - overhead
// where overhead = len("$N=value\n") + freq * len("$N")
func calcScore(c candidate, dictIdx int) int {
	ref := fmt.Sprintf("$%d", dictIdx)
	overhead := len(ref) + 1 + len(c.value) + 1 // "$N=value\n"
	overhead += c.freq * len(ref)               // all references
	savings := (c.freq - 1) * len(c.value)
	return savings - overhead
}

// greedySelect picks candidates greedily, recalculating scores after each selection.
func greedySelect(sorted []candidate, segments []string, maxEntries int) []candidate {
	var selected []candidate
	// Track which text positions are claimed.
	claimed := make(map[int]map[int]bool) // segment index -> set of character positions

	for len(selected) < maxEntries && len(sorted) > 0 {
		// Find the best remaining candidate.
		bestIdx := -1
		for i, c := range sorted {
			if c.score > 0 {
				bestIdx = i
				break // sorted by score, first positive is best
			}
		}
		if bestIdx < 0 {
			break
		}

		best := sorted[bestIdx]
		selected = append(selected, best)

		// Mark all occurrences of the selected candidate as claimed.
		for si, seg := range segments {
			if claimed[si] == nil {
				claimed[si] = make(map[int]bool)
			}
			idx := 0
			for {
				pos := strings.Index(seg[idx:], best.value)
				if pos < 0 {
					break
				}
				absPos := idx + pos
				for k := absPos; k < absPos+len(best.value); k++ {
					claimed[si][k] = true
				}
				idx = absPos + len(best.value)
			}
		}

		// Remove the selected candidate and recalculate overlapping scores.
		sorted = append(sorted[:bestIdx], sorted[bestIdx+1:]...)
		for i := range sorted {
			// Recount effective frequency (excluding claimed positions).
			newFreq := countUnclaimed(sorted[i].value, segments, claimed)
			sorted[i].freq = newFreq
			sorted[i].score = calcScoreWithRef(sorted[i], len(selected))
		}

		// Re-sort.
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i].score != sorted[j].score {
				return sorted[i].score > sorted[j].score
			}
			return sorted[i].firstPos < sorted[j].firstPos
		})
	}

	return selected
}

// countUnclaimed counts occurrences of value in segments that don't overlap with claimed positions.
func countUnclaimed(value string, segments []string, claimed map[int]map[int]bool) int {
	count := 0
	for si, seg := range segments {
		idx := 0
		for {
			pos := strings.Index(seg[idx:], value)
			if pos < 0 {
				break
			}
			absPos := idx + pos
			overlaps := false
			for k := absPos; k < absPos+len(value); k++ {
				if claimed[si] != nil && claimed[si][k] {
					overlaps = true
					break
				}
			}
			if !overlaps {
				count++
			}
			idx = absPos + 1
		}
	}
	return count
}

// calcScoreWithRef computes score using the next available dictionary index.
func calcScoreWithRef(c candidate, nextIdx int) int {
	ref := fmt.Sprintf("$%d", nextIdx)
	overhead := len(ref) + 1 + len(c.value) + 1 // "$N=value\n"
	overhead += c.freq * len(ref)               // all references
	savings := (c.freq - 1) * len(c.value)
	return savings - overhead
}

// --- Integration with encoder ---

// collectTextSegments extracts all text from the AST for dictionary analysis.
// Skips code block content per plan spec.
func collectTextSegments(nodes []Node) []string {
	var segments []string
	for i := range nodes {
		collectNodeText(&nodes[i], &segments)
	}
	return segments
}

func collectNodeText(node *Node, segments *[]string) {
	// Skip code blocks entirely.
	if node.Tag == TagCodeBlock {
		return
	}

	if node.Content != "" {
		*segments = append(*segments, node.Content)
	}
	if node.Attrs.URL != "" {
		*segments = append(*segments, node.Attrs.URL)
	}
	if node.Attrs.Display != "" {
		*segments = append(*segments, node.Attrs.Display)
	}
	if node.Attrs.Callout != "" {
		*segments = append(*segments, node.Attrs.Callout)
	}
	*segments = append(*segments, node.Attrs.Headers...)
	for _, row := range node.Attrs.Cells {
		*segments = append(*segments, row...)
	}
	for _, item := range node.Attrs.Items {
		if item.Key != "" {
			*segments = append(*segments, item.Key)
		}
		if item.Type != "" {
			*segments = append(*segments, item.Type)
		}
		if item.Description != "" {
			*segments = append(*segments, item.Description)
		}
	}
	for _, p := range node.Attrs.Params {
		if p.Name != "" {
			*segments = append(*segments, p.Name)
		}
		if p.Type != "" {
			*segments = append(*segments, p.Type)
		}
		if p.Description != "" {
			*segments = append(*segments, p.Description)
		}
	}
	for _, r := range node.Attrs.Returns {
		if r.Status != "" {
			*segments = append(*segments, r.Status)
		}
		if r.Description != "" {
			*segments = append(*segments, r.Description)
		}
	}
	if node.Attrs.Endpoint != nil {
		if node.Attrs.Endpoint.Path != "" {
			*segments = append(*segments, node.Attrs.Endpoint.Path)
		}
	}

	for i := range node.Children {
		collectNodeText(&node.Children[i], segments)
	}
}

// applyDictionary replaces dictionary value occurrences with $N references
// in all string fields of the AST (D7).
func applyDictionary(nodes []Node, dict *Dictionary) {
	for i := range nodes {
		applyNodeDict(&nodes[i], dict)
	}
}

func applyNodeDict(node *Node, dict *Dictionary) {
	if node.Tag == TagCodeBlock {
		return
	}

	node.Content = replaceDictValues(node.Content, dict)
	node.Attrs.URL = replaceDictValues(node.Attrs.URL, dict)
	node.Attrs.Display = replaceDictValues(node.Attrs.Display, dict)
	node.Attrs.Callout = replaceDictValues(node.Attrs.Callout, dict)

	for i := range node.Attrs.Headers {
		node.Attrs.Headers[i] = replaceDictValues(node.Attrs.Headers[i], dict)
	}
	for i := range node.Attrs.Cells {
		for j := range node.Attrs.Cells[i] {
			node.Attrs.Cells[i][j] = replaceDictValues(node.Attrs.Cells[i][j], dict)
		}
	}
	for i := range node.Attrs.Items {
		node.Attrs.Items[i].Key = replaceDictValues(node.Attrs.Items[i].Key, dict)
		node.Attrs.Items[i].Type = replaceDictValues(node.Attrs.Items[i].Type, dict)
		node.Attrs.Items[i].Description = replaceDictValues(node.Attrs.Items[i].Description, dict)
	}
	for i := range node.Attrs.Params {
		node.Attrs.Params[i].Name = replaceDictValues(node.Attrs.Params[i].Name, dict)
		node.Attrs.Params[i].Type = replaceDictValues(node.Attrs.Params[i].Type, dict)
		node.Attrs.Params[i].Description = replaceDictValues(node.Attrs.Params[i].Description, dict)
	}
	for i := range node.Attrs.Returns {
		node.Attrs.Returns[i].Status = replaceDictValues(node.Attrs.Returns[i].Status, dict)
		node.Attrs.Returns[i].Description = replaceDictValues(node.Attrs.Returns[i].Description, dict)
	}

	for i := range node.Children {
		applyNodeDict(&node.Children[i], dict)
	}
}

// replaceDictValues replaces occurrences of dictionary values with $N references.
// Processes longest matches first to avoid partial replacements.
func replaceDictValues(s string, dict *Dictionary) string {
	if s == "" || dict == nil {
		return s
	}
	// Sort entries by value length descending for longest-match-first.
	type entry struct {
		value string
		ref   string
	}
	entries := make([]entry, len(dict.Entries))
	for i, e := range dict.Entries {
		entries[i] = entry{value: e.Value, ref: fmt.Sprintf("$%d", e.Index)}
	}
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].value) > len(entries[j].value)
	})

	for _, e := range entries {
		s = replaceNonAmbiguous(s, e.value, e.ref)
	}
	return s
}

// replaceNonAmbiguous replaces occurrences of old with new, but skips
// replacements where the ref ($N) would be immediately followed by a digit
// in the resulting string (which would create an ambiguous reference).
func replaceNonAmbiguous(s, old, ref string) string {
	var result strings.Builder
	for {
		idx := strings.Index(s, old)
		if idx < 0 {
			result.WriteString(s)
			break
		}
		// Check if the character after the match is a digit.
		afterIdx := idx + len(old)
		if afterIdx < len(s) && s[afterIdx] >= '0' && s[afterIdx] <= '9' {
			// Skip this occurrence — would create ambiguous $N reference.
			result.WriteString(s[:idx+len(old)])
			s = s[idx+len(old):]
			continue
		}
		result.WriteString(s[:idx])
		result.WriteString(ref)
		s = s[afterIdx:]
	}
	return result.String()
}

// escapeAllText walks the AST and applies EscapeBody to all text content.
// Per D6: escape @→@@ and $→$$ BEFORE dictionary replacement.
func escapeAllText(nodes []Node) {
	for i := range nodes {
		escapeNodeText(&nodes[i])
	}
}

func escapeNodeText(node *Node) {
	if node.Tag == TagCodeBlock {
		return
	}

	node.Content = EscapeBody(node.Content)
	node.Attrs.URL = EscapeBody(node.Attrs.URL)
	node.Attrs.Display = EscapeBody(node.Attrs.Display)
	node.Attrs.Callout = EscapeBody(node.Attrs.Callout)

	for i := range node.Attrs.Headers {
		node.Attrs.Headers[i] = EscapeBody(node.Attrs.Headers[i])
	}
	for i := range node.Attrs.Cells {
		for j := range node.Attrs.Cells[i] {
			node.Attrs.Cells[i][j] = EscapeBody(node.Attrs.Cells[i][j])
		}
	}
	for i := range node.Attrs.Items {
		node.Attrs.Items[i].Key = EscapeBody(node.Attrs.Items[i].Key)
		node.Attrs.Items[i].Type = EscapeBody(node.Attrs.Items[i].Type)
		node.Attrs.Items[i].Description = EscapeBody(node.Attrs.Items[i].Description)
	}
	for i := range node.Attrs.Params {
		node.Attrs.Params[i].Name = EscapeBody(node.Attrs.Params[i].Name)
		node.Attrs.Params[i].Type = EscapeBody(node.Attrs.Params[i].Type)
		node.Attrs.Params[i].Description = EscapeBody(node.Attrs.Params[i].Description)
	}
	for i := range node.Attrs.Returns {
		node.Attrs.Returns[i].Status = EscapeBody(node.Attrs.Returns[i].Status)
		node.Attrs.Returns[i].Description = EscapeBody(node.Attrs.Returns[i].Description)
	}

	for i := range node.Children {
		escapeNodeText(&node.Children[i])
	}
}
