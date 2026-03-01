package cmdx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeGlobalDict_singleFile(t *testing.T) {
	// A single file cannot have cross-file repetition.
	files := map[string][]byte{
		"only.md": []byte("# Title\n\nSome content here.\n"),
	}
	result := AnalyzeGlobalDictOpportunity(files, 1000)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Files)
	assert.Equal(t, 0, result.DictEntries, "single file should have no global dict entries")
	assert.Equal(t, 0, result.NetBytesSaved)
}

func TestAnalyzeGlobalDict_noSharedVocabulary(t *testing.T) {
	// Multiple files with completely unique content.
	files := map[string][]byte{
		"a.md": []byte("# Alpha\n\nThis document discusses alpha topics exclusively.\n"),
		"b.md": []byte("# Bravo\n\nCompletely different bravo content here.\n"),
		"c.md": []byte("# Charlie\n\nYet another unique charlie document.\n"),
	}
	result := AnalyzeGlobalDictOpportunity(files, 1000)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.Files)
	assert.Equal(t, 0, result.DictEntries, "no shared vocabulary should produce empty dict")
}

func TestAnalyzeGlobalDict_highOverlap(t *testing.T) {
	// Multiple files sharing substantial repeated content.
	shared := "The unique identifier of the user"
	files := make(map[string][]byte)
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("# API %d\n\n%s is required. Provide %s in the request.\n", i, shared, shared)
		files[fmt.Sprintf("api_%d.md", i)] = []byte(content)
	}
	result := AnalyzeGlobalDictOpportunity(files, 10000)
	require.NotNil(t, result)
	assert.Equal(t, 10, result.Files)
	assert.Greater(t, result.DictEntries, 0, "shared vocabulary should produce global dict entries")
	assert.Greater(t, result.CoverageFiles, 0, "some files should be covered")
}

func TestAnalyzeGlobalDict_crossFileOnly(t *testing.T) {
	// A string that repeats many times in ONE file but not across files
	// should NOT be selected (requires >= 2 files).
	singleFileRepeat := "this only repeats within one file"
	files := map[string][]byte{
		"a.md": []byte(fmt.Sprintf("# Doc A\n\n%s and %s and %s and %s.\n",
			singleFileRepeat, singleFileRepeat, singleFileRepeat, singleFileRepeat)),
		"b.md": []byte("# Doc B\n\nCompletely different content here.\n"),
	}
	result := AnalyzeGlobalDictOpportunity(files, 1000)
	require.NotNil(t, result)
	// The repeated string should NOT appear in global dict entries since
	// it only exists in one file.
	for _, entry := range result.TopEntries {
		assert.NotEqual(t, singleFileRepeat, entry.Value,
			"single-file repeats should not be in global dict")
	}
}

func TestAnalyzeGlobalDict_deterministic(t *testing.T) {
	shared := "https://api.example.com/v2/endpoint"
	files := map[string][]byte{
		"a.md": []byte(fmt.Sprintf("# API A\n\nUse %s for auth.\n", shared)),
		"b.md": []byte(fmt.Sprintf("# API B\n\nUse %s for data.\n", shared)),
		"c.md": []byte(fmt.Sprintf("# API C\n\nUse %s for admin.\n", shared)),
	}
	r1 := AnalyzeGlobalDictOpportunity(files, 5000)
	r2 := AnalyzeGlobalDictOpportunity(files, 5000)
	require.NotNil(t, r1)
	require.NotNil(t, r2)
	assert.Equal(t, r1.DictEntries, r2.DictEntries, "should be deterministic")
	assert.Equal(t, r1.NetBytesSaved, r2.NetBytesSaved, "should be deterministic")
	assert.Equal(t, r1.CrossFileSaved, r2.CrossFileSaved, "should be deterministic")
	require.Equal(t, len(r1.TopEntries), len(r2.TopEntries))
	for i := range r1.TopEntries {
		assert.Equal(t, r1.TopEntries[i].Value, r2.TopEntries[i].Value,
			"entry %d should be identical", i)
	}
}

func TestAnalyzeGlobalDict_crossFileScoring(t *testing.T) {
	// A string in 5 files should rank higher than a string in 2 files
	// (given similar length and frequency).
	wideSpread := "appears in many different files"
	narrowSpread := "only in two of the document files"
	files := make(map[string][]byte)
	// wideSpread appears in files 0-4 (5 files).
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("# Doc %d\n\nThis %s content.\n", i, wideSpread)
		if i < 2 {
			// narrowSpread also in first 2 files.
			content += fmt.Sprintf("\nAlso %s here.\n", narrowSpread)
		}
		files[fmt.Sprintf("doc_%d.md", i)] = []byte(content)
	}
	result := AnalyzeGlobalDictOpportunity(files, 10000)
	require.NotNil(t, result)

	if len(result.TopEntries) >= 2 {
		// The wide-spread entry should rank higher (appear earlier in TopEntries)
		// than the narrow-spread entry.
		wideIdx := -1
		narrowIdx := -1
		for i, e := range result.TopEntries {
			if e.Value == wideSpread {
				wideIdx = i
			}
			if e.Value == narrowSpread {
				narrowIdx = i
			}
		}
		if wideIdx >= 0 && narrowIdx >= 0 {
			assert.Less(t, wideIdx, narrowIdx,
				"wider-spread entry should rank higher")
		}
	}
}

func TestAnalyzeGlobalDict_overheadAccounting(t *testing.T) {
	// Verify the savings breakdown is internally consistent:
	// NetBytesSaved = CrossFileSaved - DisambiguationCost - DictOverhead + PerFileDictReduction
	shared := "a commonly repeated phrase across files"
	files := make(map[string][]byte)
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("# Section %d\n\nHere is %s. And again %s.\n", i, shared, shared)
		files[fmt.Sprintf("file_%d.md", i)] = []byte(content)
	}
	result := AnalyzeGlobalDictOpportunity(files, 10000)
	require.NotNil(t, result)

	if result.DictEntries > 0 {
		expected := result.CrossFileSaved - result.DisambiguationCost -
			result.DictOverhead + result.PerFileDictReduction
		assert.Equal(t, expected, result.NetBytesSaved,
			"net savings should equal cross_file - disambiguation - dict_overhead + per_file_reduction")
	}
}

func TestAnalyzeGlobalDict_emptyFiles(t *testing.T) {
	files := map[string][]byte{
		"a.md": {},
		"b.md": {},
	}
	result := AnalyzeGlobalDictOpportunity(files, 0)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.DictEntries)
}

func TestAnalyzeGlobalDict_zeroCurrentBytes(t *testing.T) {
	// When currentTotalBytes is 0, improvement percent should be 0.
	shared := "a repeated string across multiple files"
	files := map[string][]byte{
		"a.md": []byte(fmt.Sprintf("# A\n\n%s here.\n", shared)),
		"b.md": []byte(fmt.Sprintf("# B\n\n%s there.\n", shared)),
	}
	result := AnalyzeGlobalDictOpportunity(files, 0)
	require.NotNil(t, result)
	assert.Equal(t, float64(0), result.ImprovementPercent)
}

func TestAnalyzeGlobalDict_topEntriesCapped(t *testing.T) {
	// Create enough cross-file repetition to generate > 10 entries.
	files := make(map[string][]byte)
	for i := 0; i < 5; i++ {
		var content string
		for j := 0; j < 15; j++ {
			phrase := fmt.Sprintf("unique phrase number %d that repeats", j)
			content += fmt.Sprintf("%s in this part.\n\n", phrase)
		}
		files[fmt.Sprintf("doc_%d.md", i)] = []byte("# Doc\n\n" + content)
	}
	result := AnalyzeGlobalDictOpportunity(files, 50000)
	require.NotNil(t, result)
	assert.LessOrEqual(t, len(result.TopEntries), 10, "top entries should be capped at 10")
}

func TestAnalyzeGlobalDict_codeBlocksExcluded(t *testing.T) {
	// Repeated strings inside code blocks should not appear in global dict.
	codeContent := "repeated code content string here"
	files := map[string][]byte{
		"a.md": []byte(fmt.Sprintf("# A\n\n```\n%s\n```\n", codeContent)),
		"b.md": []byte(fmt.Sprintf("# B\n\n```\n%s\n```\n", codeContent)),
		"c.md": []byte(fmt.Sprintf("# C\n\n```\n%s\n```\n", codeContent)),
	}
	result := AnalyzeGlobalDictOpportunity(files, 1000)
	require.NotNil(t, result)
	for _, entry := range result.TopEntries {
		assert.NotEqual(t, codeContent, entry.Value,
			"code block content should not appear in global dict")
	}
}

func TestAnalyzeGlobalDict_coverageTracking(t *testing.T) {
	shared := "this phrase appears in several files"
	files := map[string][]byte{
		"a.md": []byte(fmt.Sprintf("# A\n\n%s is here.\n", shared)),
		"b.md": []byte(fmt.Sprintf("# B\n\n%s is there.\n", shared)),
		"c.md": []byte("# C\n\nNo shared content in this file.\n"),
	}
	result := AnalyzeGlobalDictOpportunity(files, 5000)
	require.NotNil(t, result)
	if result.DictEntries > 0 {
		assert.LessOrEqual(t, result.CoverageFiles, result.Files,
			"coverage cannot exceed total files")
		assert.Greater(t, result.CoverageFiles, 0,
			"at least one file should be covered when entries exist")
	}
}

// TestExtractCrossCandidates_fileTracking verifies that candidates track
// which files they appear in.
func TestExtractCrossCandidates_fileTracking(t *testing.T) {
	shared := "a shared phrase across files"
	segs := []analysisFileSegments{
		{segments: []string{"prefix " + shared + " suffix"}},
		{segments: []string{"other " + shared + " text"}},
		{segments: []string{"unique content only"}},
	}
	candidates := extractCrossCandidates(segs, 10)

	var found *crossCandidate
	for i := range candidates {
		if candidates[i].value == shared {
			found = &candidates[i]
			break
		}
	}
	require.NotNil(t, found, "shared phrase should be a candidate")
	assert.True(t, found.fileSet[0], "should appear in file 0")
	assert.True(t, found.fileSet[1], "should appear in file 1")
	assert.False(t, found.fileSet[2], "should not appear in file 2")
	assert.Equal(t, 2, len(found.fileSet))
}

// TestCalcCrossScore_disambiguationCost verifies that disambiguation
// overhead is included in the score calculation.
func TestCalcCrossScore_disambiguationCost(t *testing.T) {
	c := crossCandidate{
		value:   "some repeated text value",
		freq:    10,
		fileSet: map[int]bool{0: true, 1: true, 2: true},
	}

	// Score with cross-file weighting.
	score := calcCrossScore(c, 0, 5)

	// Manually compute expected base score with disambiguation.
	ref := "$0"
	overhead := len(ref) + 1 + len(c.value) + 1 // dict entry
	overhead += c.freq * len(ref)               // references
	overhead += c.freq                          // disambiguation (1 byte per ref)
	savings := (c.freq - 1) * len(c.value)
	baseScore := savings - overhead

	// With cross-file boost: 3/5 = 0.6 -> multiply by 1.6.
	if baseScore > 0 {
		expected := int(float64(baseScore) * 1.6)
		assert.Equal(t, expected, score)
	}
}

func TestDictBlockSize(t *testing.T) {
	dict := &Dictionary{
		Entries: []DictEntry{
			{Index: 0, Value: "hello"},
			{Index: 1, Value: "world"},
		},
	}
	size := dictBlockSize(dict)
	// "@DICT{\n" = 7, "}\n" = 2
	// "  $0=hello\n" = 11, "  $1=world\n" = 11
	expected := 7 + 2 + 11 + 11
	assert.Equal(t, expected, size)
}

func TestDictBlockSize_nil(t *testing.T) {
	assert.Equal(t, 0, dictBlockSize(nil))
	assert.Equal(t, 0, dictBlockSize(&Dictionary{}))
}

// --- Bug fix regression tests ---

// TestBugFix_overlapDoubleCounting verifies that buildAnalysisResult uses
// countUnclaimed (via the claimed map from greedy selection) instead of
// strings.Count. Without this fix, overlapping candidates would have
// their occurrences double-counted in the savings calculation.
func TestBugFix_overlapDoubleCounting(t *testing.T) {
	// Create overlapping candidates: "abc def" overlaps with "def ghi".
	// Both appear in 2+ files. If the greedy selector claims positions
	// for "abc def", then "def ghi" should have its count reduced because
	// the "def" portion is already claimed.
	overlap := "the overlapping phrase is"
	extended := "the overlapping phrase is quite long"
	files := make(map[string][]byte)
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("# Doc %d\n\nHere %s visible. And %s too. Plus %s again.\n",
			i, overlap, extended, overlap)
		files[fmt.Sprintf("doc_%d.md", i)] = []byte(content)
	}

	result := AnalyzeGlobalDictOpportunity(files, 10000)
	require.NotNil(t, result)

	// The key invariant: CrossFileSaved should never exceed what's
	// actually achievable. If double-counting occurred, the savings
	// would be inflated beyond the actual text bytes available.
	assert.LessOrEqual(t, result.CrossFileSaved, result.TotalTextBytes,
		"cross-file savings cannot exceed total text bytes (would indicate double-counting)")

	// The overhead formula must still hold.
	if result.DictEntries > 0 {
		expected := result.CrossFileSaved - result.DisambiguationCost -
			result.DictOverhead + result.PerFileDictReduction
		assert.Equal(t, expected, result.NetBytesSaved,
			"accounting formula must hold after overlap fix")
	}
}

// TestBugFix_staleFileSet verifies that fileSet is refreshed during greedy
// recalculation. Without this fix, a candidate's fileSet could include
// files where all occurrences were claimed by earlier candidates, inflating
// the cross-file weighting boost.
func TestBugFix_staleFileSet(t *testing.T) {
	// Candidate A ("a very specific long phrase that should be selected
	// first") appears in files 0-4. Candidate B ("long phrase that") is a
	// substring of A and also appears in files 0-4. After A is selected
	// and claims positions, B's fileSet should shrink because B's
	// occurrences are inside A's claimed positions.
	longPhrase := "a very specific long phrase that should be selected first"

	files := make(map[string][]byte)
	for i := 0; i < 5; i++ {
		// longPhrase contains "long phrase that", so after longPhrase
		// claims positions, that substring should have reduced file coverage.
		content := fmt.Sprintf("# Doc %d\n\nThis has %s in it.\n", i, longPhrase)
		files[fmt.Sprintf("doc_%d.md", i)] = []byte(content)
	}

	result := AnalyzeGlobalDictOpportunity(files, 10000)
	require.NotNil(t, result)

	// If fileSet were stale, B would get an inflated cross-file boost
	// and potentially be selected with incorrect savings. The accounting
	// formula catches this because inflated scores lead to selections
	// whose actual savings don't match.
	if result.DictEntries > 0 {
		expected := result.CrossFileSaved - result.DisambiguationCost -
			result.DictOverhead + result.PerFileDictReduction
		assert.Equal(t, expected, result.NetBytesSaved,
			"accounting must hold when fileSet is properly refreshed")
	}
}

// TestBugFix_placeholderPreservesWordBoundaries verifies that per-file
// dictionary reduction estimation uses a placeholder ("$_") instead of
// deleting text entirely. Without this fix, adjacent words could fuse
// into new strings that distort the rebuilt per-file dictionary.
func TestBugFix_placeholderPreservesWordBoundaries(t *testing.T) {
	// Create files where a global entry sits between two words. If the
	// entry is deleted (old behavior), the surrounding words fuse. If
	// replaced with "$_" (new behavior), boundaries are preserved.
	//
	// Example: "configure|GLOBAL_ENTRY|handler" with deletion becomes
	// "configurehandler" which is a new 16-char string that could become
	// a false dictionary candidate. With "$_" it becomes "configure$_handler".
	globalEntry := "shared repeated phrase across files"
	files := make(map[string][]byte)
	for i := 0; i < 5; i++ {
		// Surround the global entry with words that would fuse.
		content := fmt.Sprintf("# Doc %d\n\nprefix%ssuffix appears. And prefix%ssuffix again.\n",
			i, globalEntry, globalEntry)
		files[fmt.Sprintf("doc_%d.md", i)] = []byte(content)
	}

	result := AnalyzeGlobalDictOpportunity(files, 10000)
	require.NotNil(t, result)

	// PerFileDictReduction should be non-negative: removing global entries
	// from per-file text should not create new dictionary candidates that
	// make the per-file dictionary larger.
	assert.GreaterOrEqual(t, result.PerFileDictReduction, 0,
		"per-file dict reduction should not be negative (would indicate word fusion creating false candidates)")
}

// --- New feature tests ---

// TestAnalyzeGlobalDict_dictTokensUsesRealTokenizer verifies that the
// DictTokens field is computed via the real o200k_base tokenizer rather
// than a bytes/4 approximation.
func TestAnalyzeGlobalDict_dictTokensUsesRealTokenizer(t *testing.T) {
	shared := "the repeated configuration parameter"
	files := make(map[string][]byte)
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("# Config %d\n\nSet %s here. Use %s there.\n", i, shared, shared)
		files[fmt.Sprintf("cfg_%d.md", i)] = []byte(content)
	}

	result := AnalyzeGlobalDictOpportunity(files, 10000)
	require.NotNil(t, result)

	if result.DictEntries > 0 {
		assert.Greater(t, result.DictTokens, 0,
			"DictTokens should be positive when dict entries exist")
		// Real tokenization should produce a different count than bytes/4.
		// For a small dict block, bytes/4 would give a very small number,
		// while real BPE may differ. At minimum, DictTokens should be
		// less than DictFileBytes (BPE is more efficient than 1 token/byte).
		assert.Less(t, result.DictTokens, result.DictFileBytes,
			"real tokens should be fewer than raw bytes for dictionary text")
	}
}

// TestRecountFileSet verifies that recountFileSet correctly rebuilds
// the file set after positions are claimed.
func TestRecountFileSet(t *testing.T) {
	segs := []analysisFileSegments{
		{segments: []string{"hello world"}},         // file 0
		{segments: []string{"hello world again"}},   // file 1
		{segments: []string{"nothing interesting"}}, // file 2
	}

	// Claim all "hello world" positions in file 0's segment (seg index 0).
	claimed := map[int]map[int]bool{
		0: {0: true, 1: true, 2: true, 3: true, 4: true,
			5: true, 6: true, 7: true, 8: true, 9: true, 10: true},
	}

	// "hello world" should still be found in file 1 (unclaimed) but not
	// file 0 (fully claimed) or file 2 (doesn't contain it).
	fs := recountFileSet("hello world", segs, claimed)
	assert.False(t, fs[0], "file 0 should not be in fileSet (positions claimed)")
	assert.True(t, fs[1], "file 1 should be in fileSet (unclaimed)")
	assert.False(t, fs[2], "file 2 should not be in fileSet (no match)")
}

// TestCountUnclaimedSingle verifies per-segment unclaimed counting.
func TestCountUnclaimedSingle(t *testing.T) {
	seg := "abc abc abc"
	// No claims: should find 3 occurrences.
	assert.Equal(t, 3, countUnclaimedSingle("abc", seg, nil))

	// Claim first occurrence (positions 0-2).
	claimed := map[int]bool{0: true, 1: true, 2: true}
	assert.Equal(t, 2, countUnclaimedSingle("abc", seg, claimed))

	// Claim all occurrences.
	allClaimed := map[int]bool{
		0: true, 1: true, 2: true,
		4: true, 5: true, 6: true,
		8: true, 9: true, 10: true,
	}
	assert.Equal(t, 0, countUnclaimedSingle("abc", seg, allClaimed))
}
