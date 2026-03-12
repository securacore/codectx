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
		FilePath:    "/tmp/codectx/auth-jwt.1700000000.md",
		TotalTokens: 1772,
		ChunkIDs:    []string{"obj:abc123.03", "spec:def456.02"},
		Sources:     []string{"topics/auth.md", "topics/auth.spec.md"},
	}

	got := query.FormatGenerateSummary(r)

	if !strings.Contains(got, "/tmp/codectx/auth-jwt.1700000000.md") {
		t.Error("missing file path")
	}
	if !strings.Contains(got, "1,772 tokens") {
		t.Error("missing token count")
	}
	if !strings.Contains(got, "obj:abc123.03") || !strings.Contains(got, "spec:def456.02") {
		t.Error("missing chunk IDs")
	}
}

func TestFormatGenerateSummary_WithRelated(t *testing.T) {
	r := &query.GenerateResult{
		FilePath:    "/tmp/codectx/test.123.md",
		TotalTokens: 500,
		ChunkIDs:    []string{"obj:abc123.01"},
		Sources:     []string{"topics/test.md"},
		Related: []query.RelatedEntry{
			{ChunkID: "obj:abc123.02", Heading: "Test > Next", Tokens: 300},
		},
	}

	got := query.FormatGenerateSummary(r)

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
		FilePath:    "/tmp/codectx/test.123.md",
		TotalTokens: 500,
		ChunkIDs:    []string{"obj:abc123.01"},
		Sources:     []string{"topics/test.md"},
	}

	got := query.FormatGenerateSummary(r)

	if strings.Contains(got, "Related") {
		t.Error("unexpected related section when no related chunks")
	}
}
