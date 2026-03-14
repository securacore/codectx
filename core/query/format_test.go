package query_test

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/query"
)

// ---------------------------------------------------------------------------
// FormatQueryResults
// ---------------------------------------------------------------------------

func TestFormatQueryResults_AllSections(t *testing.T) {
	r := &query.QueryResult{
		RawQuery: "jwt refresh",
		Instructions: []query.ResultEntry{
			{ChunkID: "obj:abc123.1", Score: 8.42, Heading: "Auth > JWT", Source: "topics/auth.md", Sequence: 1, TotalInFile: 3, Tokens: 400},
		},
		Reasoning: []query.ResultEntry{
			{ChunkID: "spec:def456.1", Score: 5.10, Heading: "Auth > JWT", Source: "topics/auth.spec.md", Sequence: 1, TotalInFile: 1, Tokens: 300},
		},
		System: []query.ResultEntry{
			{ChunkID: "sys:ghi789.1", Score: 2.50, Heading: "Compiler", Source: "system/topics/compile.md", Sequence: 1, TotalInFile: 1, Tokens: 200},
		},
		Related: []query.RelatedEntry{
			{ChunkID: "obj:abc123.2", Heading: "Auth > JWT > Tokens", Tokens: 350},
		},
	}

	got := query.FormatQueryResults(r)

	// Check query header.
	if !strings.Contains(got, `"jwt refresh"`) {
		t.Errorf("missing query header, got:\n%s", got)
	}

	// Check sections.
	if !strings.Contains(got, "Instructions:") {
		t.Error("missing Instructions section")
	}
	if !strings.Contains(got, "Reasoning:") {
		t.Error("missing Reasoning section")
	}
	if !strings.Contains(got, "System:") {
		t.Error("missing System section")
	}
	if !strings.Contains(got, "Related chunks") {
		t.Error("missing Related section")
	}

	// Check entry format.
	if !strings.Contains(got, "score: 8.42") {
		t.Error("missing score in entry")
	}
	if !strings.Contains(got, "obj:abc123.1") {
		t.Error("missing chunk ID in entry")
	}
	if !strings.Contains(got, "chunk 1/3") {
		t.Error("missing chunk position in entry")
	}
	if !strings.Contains(got, "400 tokens") {
		t.Error("missing token count in entry")
	}

	// Check related entry.
	if !strings.Contains(got, "obj:abc123.2") {
		t.Error("missing related chunk ID")
	}
}

func TestFormatQueryResults_NoResults(t *testing.T) {
	r := &query.QueryResult{
		RawQuery: "nonexistent topic",
	}

	got := query.FormatQueryResults(r)

	if !strings.Contains(got, "No results found") {
		t.Errorf("expected 'No results found', got:\n%s", got)
	}
}

func TestFormatQueryResults_InstructionsOnly(t *testing.T) {
	r := &query.QueryResult{
		RawQuery: "auth",
		Instructions: []query.ResultEntry{
			{ChunkID: "obj:abc123.1", Score: 6.00, Heading: "Auth", Source: "topics/auth.md", Sequence: 1, TotalInFile: 1, Tokens: 400},
		},
	}

	got := query.FormatQueryResults(r)

	if !strings.Contains(got, "Instructions:") {
		t.Error("missing Instructions section")
	}
	if strings.Contains(got, "Reasoning:") {
		t.Error("unexpected Reasoning section")
	}
	if strings.Contains(got, "System:") {
		t.Error("unexpected System section")
	}
	if strings.Contains(got, "No results found") {
		t.Error("unexpected 'No results found'")
	}
}

func TestFormatQueryResults_WithExpandedQuery(t *testing.T) {
	r := &query.QueryResult{
		RawQuery:      "login failures",
		ExpandedQuery: "login failures authentication error-handling",
		Instructions: []query.ResultEntry{
			{ChunkID: "obj:abc123.1", Score: 6.00, Heading: "Auth", Source: "topics/auth.md", Sequence: 1, TotalInFile: 1, Tokens: 400},
		},
	}

	got := query.FormatQueryResults(r)

	if !strings.Contains(got, "Expanded:") {
		t.Error("missing Expanded label")
	}
	if !strings.Contains(got, "authentication error-handling") {
		t.Errorf("missing expanded query content, got:\n%s", got)
	}
}

func TestFormatQueryResults_NoExpandedWhenSameAsRaw(t *testing.T) {
	r := &query.QueryResult{
		RawQuery:      "jwt auth",
		ExpandedQuery: "jwt auth",
		Instructions: []query.ResultEntry{
			{ChunkID: "obj:abc123.1", Score: 6.00, Heading: "Auth", Source: "topics/auth.md", Sequence: 1, TotalInFile: 1, Tokens: 400},
		},
	}

	got := query.FormatQueryResults(r)

	if strings.Contains(got, "Expanded:") {
		t.Error("should not show Expanded when same as raw query")
	}
}

func TestFormatQueryResults_MultipleEntries(t *testing.T) {
	r := &query.QueryResult{
		RawQuery: "auth",
		Instructions: []query.ResultEntry{
			{ChunkID: "obj:abc123.1", Score: 8.00, Heading: "Auth > Login", Source: "topics/auth.md", Sequence: 1, TotalInFile: 3, Tokens: 400},
			{ChunkID: "obj:abc123.2", Score: 6.50, Heading: "Auth > Tokens", Source: "topics/auth.md", Sequence: 2, TotalInFile: 3, Tokens: 350},
		},
	}

	got := query.FormatQueryResults(r)

	// Entries should be numbered with scores and chunk IDs.
	if !strings.Contains(got, "1.") || !strings.Contains(got, "score: 8.00") {
		t.Error("missing first entry numbering or score")
	}
	if !strings.Contains(got, "2.") || !strings.Contains(got, "score: 6.50") {
		t.Error("missing second entry numbering or score")
	}
}

// ---------------------------------------------------------------------------
// FormatGenerateSummary
// ---------------------------------------------------------------------------

func TestFormatGenerateSummary_Basic(t *testing.T) {
	r := &query.GenerateResult{
		TotalTokens: 1772,
		ContentHash: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6abcd",
		ChunkIDs:    []string{"obj:abc123.03", "spec:def456.02"},
		Sources:     []string{"topics/auth.md", "topics/auth.spec.md"},
	}

	histPath := "docs/.codectx/history/docs/a1b2c3d4e5f6.1700000000000000000.md"
	got := query.FormatGenerateSummary(r, histPath, "", false)

	if !strings.Contains(got, histPath) {
		t.Error("missing history path")
	}
	if !strings.Contains(got, "1,772 tokens") {
		t.Error("missing token count")
	}
	if !strings.Contains(got, "a1b2c3d4e5f6") {
		t.Error("missing short hash in header")
	}
	if !strings.Contains(got, "obj:abc123.03") || !strings.Contains(got, "spec:def456.02") {
		t.Error("missing chunk IDs")
	}
}

func TestFormatGenerateSummary_WithRelated(t *testing.T) {
	r := &query.GenerateResult{
		TotalTokens: 500,
		ContentHash: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		ChunkIDs:    []string{"obj:abc123.01"},
		Sources:     []string{"topics/test.md"},
		Related: []query.RelatedEntry{
			{ChunkID: "obj:abc123.02", Heading: "Test > Next", Tokens: 300},
		},
	}

	got := query.FormatGenerateSummary(r, "docs/.codectx/history/docs/test.md", "", false)

	if !strings.Contains(got, "Related chunks not included") {
		t.Error("missing related section")
	}
	if !strings.Contains(got, "obj:abc123.02") {
		t.Error("missing related chunk ID")
	}
	if !strings.Contains(got, "300 tokens") {
		t.Error("missing related token count")
	}
}

func TestFormatGenerateSummary_NoRelated(t *testing.T) {
	r := &query.GenerateResult{
		TotalTokens: 500,
		ContentHash: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		ChunkIDs:    []string{"obj:abc123.01"},
		Sources:     []string{"topics/test.md"},
	}

	got := query.FormatGenerateSummary(r, "", "", false)

	if strings.Contains(got, "Related") {
		t.Error("unexpected related section when no related chunks")
	}
}

func TestFormatGenerateSummary_WithFilePath(t *testing.T) {
	r := &query.GenerateResult{
		TotalTokens: 500,
		ContentHash: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		ChunkIDs:    []string{"obj:abc123.01"},
		Sources:     []string{"topics/test.md"},
	}

	got := query.FormatGenerateSummary(r, "", "/path/to/output.md", false)

	if !strings.Contains(got, "/path/to/output.md") {
		t.Error("missing file path in output")
	}
	if !strings.Contains(got, "Written to") {
		t.Error("missing 'Written to' label")
	}
}

func TestFormatGenerateSummary_HistoryAndFilePath(t *testing.T) {
	r := &query.GenerateResult{
		TotalTokens: 500,
		ContentHash: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		ChunkIDs:    []string{"obj:abc123.01"},
		Sources:     []string{"topics/test.md"},
	}

	histPath := "docs/.codectx/history/docs/a1b2c3.md"
	filePath := "/tmp/output.md"
	got := query.FormatGenerateSummary(r, histPath, filePath, false)

	if !strings.Contains(got, histPath) {
		t.Error("missing history path")
	}
	if !strings.Contains(got, filePath) {
		t.Error("missing file path")
	}
}

// ---------------------------------------------------------------------------
// FormatQueryResults — Unified (BM25F) mode
// ---------------------------------------------------------------------------

func TestFormatQueryResults_Unified(t *testing.T) {
	r := &query.QueryResult{
		RawQuery:      "jwt authentication",
		ExpandedQuery: "jwt authent auth oauth",
		Unified: []query.ResultEntry{
			{
				ChunkID:      "obj:abc123.1",
				Score:        0.0312,
				Heading:      "Auth > JWT > Refresh",
				Source:       "topics/auth/jwt.md",
				Sequence:     3,
				TotalInFile:  7,
				Tokens:       462,
				IndexSources: map[string]int{"objects": 1, "specs": 5},
			},
			{
				ChunkID:      "spec:def456.1",
				Score:        0.0198,
				Heading:      "Auth > JWT > Reasoning",
				Source:       "topics/auth/jwt.spec.md",
				Sequence:     1,
				TotalInFile:  2,
				Tokens:       300,
				IndexSources: map[string]int{"specs": 1},
			},
		},
		Related: []query.RelatedEntry{
			{ChunkID: "obj:abc123.2", Heading: "Auth > JWT > Tokens", Tokens: 350},
		},
	}

	got := query.FormatQueryResults(r)

	// Header with result count and pipeline info.
	if !strings.Contains(got, "bm25f + rrf") {
		t.Error("missing bm25f + rrf indicator")
	}
	if !strings.Contains(got, "(2,") {
		t.Error("missing result count in header")
	}

	// Results section (not "Instructions:" or "Reasoning:").
	if !strings.Contains(got, "Results") {
		t.Error("missing Results section header")
	}
	if strings.Contains(got, "Instructions:") {
		t.Error("unified mode should not show Instructions section")
	}

	// Score precision should be 4 decimal places.
	if !strings.Contains(got, "0.0312") {
		t.Error("missing 4-decimal score for first result")
	}

	// Index source annotation.
	if !strings.Contains(got, "objects:#1") {
		t.Error("missing objects source annotation")
	}
	if !strings.Contains(got, "specs:#5") {
		t.Error("missing specs source annotation")
	}

	// Total token summary.
	if !strings.Contains(got, "762 tokens") {
		t.Errorf("missing total tokens summary (expected 762), got:\n%s", got)
	}
	if !strings.Contains(got, "2 results") {
		t.Error("missing result count in summary")
	}

	// Expanded query.
	if !strings.Contains(got, "Expanded:") {
		t.Error("missing expanded query display")
	}

	// Related.
	if !strings.Contains(got, "Related") {
		t.Error("missing related chunks")
	}
}

func TestFormatQueryResults_UnifiedNoResults(t *testing.T) {
	r := &query.QueryResult{
		RawQuery: "nonexistent",
		Unified:  []query.ResultEntry{},
	}

	got := query.FormatQueryResults(r)
	if !strings.Contains(got, "No results found") {
		t.Error("should show no results message for empty unified list")
	}
}

func TestFormatQueryResults_UnifiedSingleSource(t *testing.T) {
	r := &query.QueryResult{
		RawQuery: "test",
		Unified: []query.ResultEntry{
			{
				ChunkID:      "obj:abc.1",
				Score:        0.05,
				Heading:      "Test",
				Source:       "test.md",
				Sequence:     1,
				TotalInFile:  1,
				Tokens:       100,
				IndexSources: map[string]int{"objects": 1},
			},
		},
	}

	got := query.FormatQueryResults(r)

	// Only objects source, no specs or system.
	if !strings.Contains(got, "objects:#1") {
		t.Error("missing objects source")
	}
	if strings.Contains(got, "specs:") {
		t.Error("should not show specs source when not present")
	}
}

func TestFormatGenerateSummary_CacheHit(t *testing.T) {
	r := &query.GenerateResult{
		TotalTokens: 500,
		ContentHash: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		ChunkIDs:    []string{"obj:abc123.01"},
		Sources:     []string{"topics/test.md"},
	}

	got := query.FormatGenerateSummary(r, "", "", true)

	if !strings.Contains(got, "[from cache]") {
		t.Error("missing [from cache] annotation")
	}
}
