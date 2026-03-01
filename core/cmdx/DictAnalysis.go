package cmdx

import (
	"fmt"
	"sort"
	"strings"

	"github.com/securacore/codectx/core/tokenizer"
)

// GlobalDictAnalysis holds the results of analyzing cross-file dictionary
// opportunity across a compiled documentation corpus. It quantifies how
// much additional compression a global dictionary would provide beyond
// existing per-file dictionaries.
type GlobalDictAnalysis struct {
	// Corpus statistics.
	Files          int `yaml:"files"`
	TotalSegments  int `yaml:"total_segments"`
	TotalTextBytes int `yaml:"total_text_bytes"`

	// Hypothetical global dictionary properties.
	DictEntries   int `yaml:"dict_entries"`
	DictFileBytes int `yaml:"dict_file_bytes"`
	DictTokens    int `yaml:"dict_tokens"`
	CoverageFiles int `yaml:"coverage_files"` // files using >= 1 global entry

	// Savings breakdown (all in bytes).
	CrossFileSaved       int     `yaml:"cross_file_bytes_saved"`
	DisambiguationCost   int     `yaml:"disambiguation_overhead"`
	PerFileDictReduction int     `yaml:"per_file_dict_reduction"`
	DictOverhead         int     `yaml:"dict_overhead"`
	NetBytesSaved        int     `yaml:"net_bytes_saved"`
	ImprovementPercent   float64 `yaml:"improvement_percent"`

	// Most valuable cross-file entries (up to 10).
	TopEntries []GlobalDictEntry `yaml:"top_entries,omitempty"`
}

// GlobalDictEntry describes a single high-value cross-file dictionary candidate.
type GlobalDictEntry struct {
	Value       string `yaml:"value"`
	Files       int    `yaml:"files"`
	Occurrences int    `yaml:"occurrences"`
	BytesSaved  int    `yaml:"bytes_saved"`
}

// crossCandidate extends the per-file candidate with cross-file tracking.
type crossCandidate struct {
	value    string
	freq     int          // total occurrences across all files
	fileSet  map[int]bool // set of file indices containing this candidate
	firstPos int          // position of first occurrence in concatenated text
	score    int          // byte savings score
}

// analysisFileSegments holds text segments extracted from a single file.
type analysisFileSegments struct {
	segments []string
}

// AnalyzeGlobalDictOpportunity examines raw markdown files and reports how
// much additional compression a cross-file global dictionary would provide.
// The files map is keyed by a stable identifier (e.g., relative path) with
// raw markdown content as value. The currentTotalBytes parameter is the
// total byte size of the currently-stored compressed objects, used to
// calculate the improvement percentage.
func AnalyzeGlobalDictOpportunity(files map[string][]byte, currentTotalBytes int) *GlobalDictAnalysis {
	if len(files) < 2 {
		return &GlobalDictAnalysis{Files: len(files)}
	}

	opts := DefaultEncoderOptions()

	// Parse each file through the encoder's first passes to get escaped
	// text segments, grouped by file index.
	var allFileSegments []analysisFileSegments
	totalSegments := 0
	totalTextBytes := 0

	// Sort keys for deterministic ordering.
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		content := files[key]
		body, err := markdownToAST(content)
		if err != nil {
			continue
		}

		doc := &Document{Version: "1", Body: body}
		if opts.EnableDomainBlocks {
			detectDomainPatterns(doc)
		}
		escapeAllText(doc.Body)

		segs := collectTextSegments(doc.Body)
		allFileSegments = append(allFileSegments, analysisFileSegments{segments: segs})
		totalSegments += len(segs)
		for _, s := range segs {
			totalTextBytes += len(s)
		}
	}

	if len(allFileSegments) < 2 {
		return &GlobalDictAnalysis{
			Files:          len(files),
			TotalSegments:  totalSegments,
			TotalTextBytes: totalTextBytes,
		}
	}

	// Extract candidates with cross-file tracking.
	candidates := extractCrossCandidates(allFileSegments, opts.MinStringLength)

	// Filter by minimum frequency and require appearance in >= 2 files.
	var filtered []crossCandidate
	for _, c := range candidates {
		if c.freq >= opts.MinFrequency && len(c.value) >= opts.MinStringLength && len(c.fileSet) >= 2 {
			filtered = append(filtered, c)
		}
	}

	if len(filtered) == 0 {
		return &GlobalDictAnalysis{
			Files:          len(files),
			TotalSegments:  totalSegments,
			TotalTextBytes: totalTextBytes,
		}
	}

	// Score each candidate with cross-file weighting.
	totalFiles := len(allFileSegments)
	for i := range filtered {
		filtered[i].score = calcCrossScore(filtered[i], i, totalFiles)
	}

	// Sort by score descending, tie-break by first occurrence.
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].score != filtered[j].score {
			return filtered[i].score > filtered[j].score
		}
		return filtered[i].firstPos < filtered[j].firstPos
	})

	// Greedy selection.
	allSegments := flattenFileSegments(allFileSegments)
	selected, claimed := greedyCrossSelect(filtered, allSegments, allFileSegments, opts.MaxDictEntries)

	if len(selected) == 0 {
		return &GlobalDictAnalysis{
			Files:          len(files),
			TotalSegments:  totalSegments,
			TotalTextBytes: totalTextBytes,
		}
	}

	// Sort selected by first occurrence for stable index assignment.
	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].firstPos < selected[j].firstPos
	})

	// Calculate savings.
	return buildAnalysisResult(selected, claimed, allFileSegments, totalSegments, totalTextBytes, currentTotalBytes, opts)
}

// extractCrossCandidates finds repeated substrings across all file segments,
// tracking which files each candidate appears in.
func extractCrossCandidates(fileSegments []analysisFileSegments, minLen int) []crossCandidate {
	freqMap := make(map[string]*crossCandidate)
	globalPos := 0

	for fileIdx, fs := range fileSegments {
		for _, seg := range fs.segments {
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
						freqMap[substr] = &crossCandidate{
							value:    substr,
							firstPos: globalPos + start,
							fileSet:  make(map[int]bool),
						}
					}
					freqMap[substr].fileSet[fileIdx] = true
				}
			}
			globalPos += len(seg) + 1
		}
	}

	// Count actual frequency across all segments.
	allSegments := flattenFileSegments(fileSegments)
	for _, c := range freqMap {
		freq := 0
		for _, seg := range allSegments {
			freq += strings.Count(seg, c.value)
		}
		c.freq = freq
	}

	result := make([]crossCandidate, 0, len(freqMap))
	for _, c := range freqMap {
		result = append(result, *c)
	}
	// Sort by firstPos for deterministic output.
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].firstPos < result[j].firstPos
	})
	return result
}

// calcCrossScore computes the net byte savings for a cross-file candidate.
// It uses the same base formula as calcScore but adds weight for cross-file
// spread: candidates appearing in more files are prioritized.
func calcCrossScore(c crossCandidate, dictIdx int, totalFiles int) int {
	ref := fmt.Sprintf("$%d", dictIdx)
	// Base overhead: dictionary entry line + all reference occurrences.
	overhead := len(ref) + 1 + len(c.value) + 1 // "$N=value\n"
	overhead += c.freq * len(ref)               // all references

	// Disambiguation cost: 1 extra byte per reference for a prefix character.
	overhead += c.freq

	savings := (c.freq - 1) * len(c.value)
	baseScore := savings - overhead

	// Cross-file weighting: boost score proportional to file coverage.
	// A candidate in 10/20 files gets a 50% boost.
	if totalFiles > 0 && baseScore > 0 {
		crossRatio := float64(len(c.fileSet)) / float64(totalFiles)
		return int(float64(baseScore) * (1.0 + crossRatio))
	}
	return baseScore
}

// greedyCrossSelect picks candidates greedily with overlap handling,
// similar to greedySelect but using cross-file candidates. It returns
// the selected candidates and the claimed-position map so that
// downstream accounting can use countUnclaimed for accurate counts.
func greedyCrossSelect(sorted []crossCandidate, allSegments []string, fileSegments []analysisFileSegments, maxEntries int) ([]crossCandidate, map[int]map[int]bool) {
	var selected []crossCandidate
	claimed := make(map[int]map[int]bool) // segment index -> character positions

	for len(selected) < maxEntries && len(sorted) > 0 {
		bestIdx := -1
		for i, c := range sorted {
			if c.score > 0 {
				bestIdx = i
				break
			}
		}
		if bestIdx < 0 {
			break
		}

		best := sorted[bestIdx]
		selected = append(selected, best)

		// Mark occurrences as claimed.
		for si, seg := range allSegments {
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

		// Remove selected and recalculate.
		sorted = append(sorted[:bestIdx], sorted[bestIdx+1:]...)
		for i := range sorted {
			newFreq := countUnclaimed(sorted[i].value, allSegments, claimed)
			sorted[i].freq = newFreq
			// Recount which files still contain unclaimed occurrences.
			sorted[i].fileSet = recountFileSet(sorted[i].value, fileSegments, claimed)
			sorted[i].score = calcCrossScoreWithRef(sorted[i], len(selected), len(fileSegments))
		}

		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i].score != sorted[j].score {
				return sorted[i].score > sorted[j].score
			}
			return sorted[i].firstPos < sorted[j].firstPos
		})
	}

	return selected, claimed
}

// calcCrossScoreWithRef recalculates score using the next available dictionary index.
func calcCrossScoreWithRef(c crossCandidate, nextIdx int, totalFiles int) int {
	ref := fmt.Sprintf("$%d", nextIdx)
	overhead := len(ref) + 1 + len(c.value) + 1
	overhead += c.freq * len(ref)
	overhead += c.freq // disambiguation cost
	savings := (c.freq - 1) * len(c.value)
	baseScore := savings - overhead

	if totalFiles > 0 && baseScore > 0 {
		crossRatio := float64(len(c.fileSet)) / float64(totalFiles)
		return int(float64(baseScore) * (1.0 + crossRatio))
	}
	return baseScore
}

// buildAnalysisResult computes the full analysis from selected candidates.
// The claimed map from greedy selection is used for accurate occurrence
// counting that respects overlap (bug fix: prevents double-counting).
func buildAnalysisResult(
	selected []crossCandidate,
	claimed map[int]map[int]bool,
	fileSegments []analysisFileSegments,
	totalSegments, totalTextBytes, currentTotalBytes int,
	opts EncoderOptions,
) *GlobalDictAnalysis {
	result := &GlobalDictAnalysis{
		Files:          len(fileSegments),
		TotalSegments:  totalSegments,
		TotalTextBytes: totalTextBytes,
		DictEntries:    len(selected),
	}

	// Calculate dictionary file size: header + entries + footer.
	// Format: "@DICT{\n" + "  $N=value\n" per entry + "}\n"
	var dictBlock strings.Builder
	dictBlock.WriteString("@DICT{\n")
	for i, c := range selected {
		ref := fmt.Sprintf("$%d", i)
		dictBlock.WriteString("  " + ref + "=" + c.value + "\n")
	}
	dictBlock.WriteString("}\n")
	dictText := dictBlock.String()
	dictSize := len(dictText)
	result.DictFileBytes = dictSize
	result.DictTokens = tokenizer.CountTokens(dictText)

	// Calculate cross-file savings: for each selected entry, use
	// countUnclaimed to get accurate non-overlapping occurrence counts.
	allSegments := flattenFileSegments(fileSegments)
	totalCrossFileSaved := 0
	totalDisambiguationCost := 0

	coverageSet := make(map[int]bool)

	for i, c := range selected {
		ref := fmt.Sprintf("$%d", i)
		totalOccurrences := countUnclaimed(c.value, allSegments, claimed)

		// Bytes saved: each occurrence replaces len(value) with len(ref).
		saved := totalOccurrences * (len(c.value) - len(ref))
		totalCrossFileSaved += saved

		// Disambiguation: 1 extra byte per reference.
		disambig := totalOccurrences
		totalDisambiguationCost += disambig

		// Track file coverage.
		for fileIdx := range c.fileSet {
			coverageSet[fileIdx] = true
		}

		// Build top entry.
		if len(result.TopEntries) < 10 {
			netSaved := saved - disambig
			result.TopEntries = append(result.TopEntries, GlobalDictEntry{
				Value:       c.value,
				Files:       len(c.fileSet),
				Occurrences: totalOccurrences,
				BytesSaved:  netSaved,
			})
		}
	}

	result.CoverageFiles = len(coverageSet)
	result.CrossFileSaved = totalCrossFileSaved
	result.DisambiguationCost = totalDisambiguationCost
	result.DictOverhead = dictSize

	// Estimate per-file dictionary reduction: for each file, determine how
	// many current per-file dict entries would be eliminated by global entries.
	totalPerFileReduction := 0
	for _, fs := range fileSegments {
		if len(fs.segments) == 0 {
			continue
		}

		// Build per-file dictionary from the file's segments.
		currentDict := BuildDictionary(fs.segments, opts)
		if currentDict == nil {
			continue
		}

		// Replace global entries with a placeholder to preserve word
		// boundaries without inflating byte counts (bug fix: previously
		// deleted text entirely, fusing adjacent words).
		strippedSegments := make([]string, len(fs.segments))
		copy(strippedSegments, fs.segments)
		for _, gc := range selected {
			for si, seg := range strippedSegments {
				strippedSegments[si] = strings.ReplaceAll(seg, gc.value, "$_")
			}
		}
		reducedDict := BuildDictionary(strippedSegments, opts)

		currentSize := dictBlockSize(currentDict)
		reducedSize := 0
		if reducedDict != nil {
			reducedSize = dictBlockSize(reducedDict)
		}
		totalPerFileReduction += currentSize - reducedSize
	}

	result.PerFileDictReduction = totalPerFileReduction

	// Net savings.
	result.NetBytesSaved = result.CrossFileSaved - result.DisambiguationCost -
		result.DictOverhead + result.PerFileDictReduction

	if currentTotalBytes > 0 {
		result.ImprovementPercent = float64(result.NetBytesSaved) / float64(currentTotalBytes) * 100
	}

	return result
}

// recountFileSet rebuilds the fileSet for a candidate by checking which
// files still contain unclaimed occurrences of the value. This prevents
// stale fileSet data from inflating the cross-file weighting boost after
// earlier candidates have claimed positions.
func recountFileSet(value string, fileSegments []analysisFileSegments, claimed map[int]map[int]bool) map[int]bool {
	result := make(map[int]bool)
	segIdx := 0
	for fileIdx, fs := range fileSegments {
		for _, seg := range fs.segments {
			if countUnclaimedSingle(value, seg, claimed[segIdx]) > 0 {
				result[fileIdx] = true
			}
			segIdx++
		}
	}
	return result
}

// countUnclaimedSingle counts non-overlapping occurrences of value in a
// single segment that don't overlap with claimed positions.
func countUnclaimedSingle(value, segment string, segClaimed map[int]bool) int {
	count := 0
	idx := 0
	for {
		pos := strings.Index(segment[idx:], value)
		if pos < 0 {
			break
		}
		absPos := idx + pos
		overlaps := false
		for k := absPos; k < absPos+len(value); k++ {
			if segClaimed[k] {
				overlaps = true
				break
			}
		}
		if !overlaps {
			count++
		}
		idx = absPos + len(value)
	}
	return count
}

// dictBlockSize estimates the byte size of a @DICT block.
func dictBlockSize(dict *Dictionary) int {
	if dict == nil || len(dict.Entries) == 0 {
		return 0
	}
	size := len("@DICT{\n") + len("}\n")
	for _, e := range dict.Entries {
		ref := fmt.Sprintf("$%d", e.Index)
		size += len("  ") + len(ref) + len("=") + len(e.Value) + len("\n")
	}
	return size
}

// flattenFileSegments concatenates all segments from all files into a single slice.
func flattenFileSegments(fileSegments []analysisFileSegments) []string {
	var all []string
	for _, fs := range fileSegments {
		all = append(all, fs.segments...)
	}
	return all
}
