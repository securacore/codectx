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
	got := query.FormatGenerateSummary(r, histPath, "")

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

	got := query.FormatGenerateSummary(r, "docs/.codectx/history/docs/test.md", "")

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

	got := query.FormatGenerateSummary(r, "", "")

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

	got := query.FormatGenerateSummary(r, "", "/path/to/output.md")

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
	got := query.FormatGenerateSummary(r, histPath, filePath)

	if !strings.Contains(got, histPath) {
		t.Error("missing history path")
	}
	if !strings.Contains(got, filePath) {
		t.Error("missing file path")
	}
}
