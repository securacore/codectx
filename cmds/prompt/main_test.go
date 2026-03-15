package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	corequery "github.com/securacore/codectx/core/query"
)

// ---------------------------------------------------------------------------
// collectAllResults
// ---------------------------------------------------------------------------

func TestCollectAllResults_UnifiedPassthrough(t *testing.T) {
	r := &corequery.QueryResult{
		Unified: []corequery.ResultEntry{
			{ChunkID: "obj:aaa.1", Score: 0.9},
			{ChunkID: "obj:bbb.2", Score: 0.5},
		},
	}

	got := collectAllResults(r)

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].ChunkID != "obj:aaa.1" || got[1].ChunkID != "obj:bbb.2" {
		t.Errorf("unexpected order: %v", got)
	}
}

func TestCollectAllResults_BM25MergeAndSort(t *testing.T) {
	r := &corequery.QueryResult{
		Instructions: []corequery.ResultEntry{
			{ChunkID: "obj:a.1", Score: 5.0},
			{ChunkID: "obj:a.2", Score: 1.0},
		},
		Reasoning: []corequery.ResultEntry{
			{ChunkID: "spec:b.1", Score: 8.0},
		},
		System: []corequery.ResultEntry{
			{ChunkID: "sys:c.1", Score: 3.0},
		},
	}

	got := collectAllResults(r)

	if len(got) != 4 {
		t.Fatalf("expected 4 results, got %d", len(got))
	}

	// Should be sorted by score descending: 8.0, 5.0, 3.0, 1.0
	expectedOrder := []string{"spec:b.1", "obj:a.1", "sys:c.1", "obj:a.2"}
	for i, want := range expectedOrder {
		if got[i].ChunkID != want {
			t.Errorf("position %d: got %s, want %s", i, got[i].ChunkID, want)
		}
	}
}

func TestCollectAllResults_Empty(t *testing.T) {
	r := &corequery.QueryResult{}

	got := collectAllResults(r)

	if len(got) != 0 {
		t.Errorf("expected 0 results, got %d", len(got))
	}
}

func TestCollectAllResults_SingleType(t *testing.T) {
	r := &corequery.QueryResult{
		System: []corequery.ResultEntry{
			{ChunkID: "sys:x.1", Score: 2.0},
			{ChunkID: "sys:x.2", Score: 1.0},
		},
	}

	got := collectAllResults(r)

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].ChunkID != "sys:x.1" {
		t.Errorf("expected sys:x.1 first, got %s", got[0].ChunkID)
	}
}

func TestCollectAllResults_StableSortOnTies(t *testing.T) {
	r := &corequery.QueryResult{
		Instructions: []corequery.ResultEntry{
			{ChunkID: "obj:first.1", Score: 5.0},
		},
		System: []corequery.ResultEntry{
			{ChunkID: "sys:second.1", Score: 5.0},
		},
	}

	got := collectAllResults(r)

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	// Insertion sort is stable — original order preserved on ties.
	// Instructions come before System in the all slice.
	if got[0].ChunkID != "obj:first.1" {
		t.Errorf("expected obj:first.1 first on tie, got %s", got[0].ChunkID)
	}
}

// ---------------------------------------------------------------------------
// selectChunks
// ---------------------------------------------------------------------------

func TestSelectChunks_Empty(t *testing.T) {
	ids, total := selectChunks(nil, 1000)
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs, got %d", len(ids))
	}
	if total != 0 {
		t.Errorf("expected 0 tokens, got %d", total)
	}
}

func TestSelectChunks_SingleOversized(t *testing.T) {
	results := []corequery.ResultEntry{
		{ChunkID: "obj:big.1", Tokens: 5000},
	}

	ids, total := selectChunks(results, 100)

	// First result is always included even if it exceeds budget.
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID, got %d", len(ids))
	}
	if ids[0] != "obj:big.1" {
		t.Errorf("expected obj:big.1, got %s", ids[0])
	}
	if total != 5000 {
		t.Errorf("expected 5000 tokens, got %d", total)
	}
}

func TestSelectChunks_BudgetExceeded(t *testing.T) {
	results := []corequery.ResultEntry{
		{ChunkID: "obj:a.1", Tokens: 400},
		{ChunkID: "obj:a.2", Tokens: 400},
		{ChunkID: "obj:a.3", Tokens: 400},
	}

	ids, total := selectChunks(results, 700)

	// First two fit (800 > 700 on third), so only first two.
	// Actually: after first (400), total=400. Second: 400+400=800 > 700, break.
	// Wait: len(selected) > 0 is true, total+r.Tokens = 400+400 = 800 > 700, break.
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID (budget 700, first=400, second would be 800), got %d", len(ids))
	}
	if total != 400 {
		t.Errorf("expected 400 tokens, got %d", total)
	}
}

func TestSelectChunks_ExactBudgetMatch(t *testing.T) {
	results := []corequery.ResultEntry{
		{ChunkID: "obj:a.1", Tokens: 500},
		{ChunkID: "obj:a.2", Tokens: 500},
	}

	ids, total := selectChunks(results, 1000)

	// 500 + 500 = 1000, not > 1000, so both fit.
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
	if total != 1000 {
		t.Errorf("expected 1000 tokens, got %d", total)
	}
}

func TestSelectChunks_AllFitWithinBudget(t *testing.T) {
	results := []corequery.ResultEntry{
		{ChunkID: "obj:a.1", Tokens: 100},
		{ChunkID: "obj:a.2", Tokens: 200},
		{ChunkID: "obj:a.3", Tokens: 300},
	}

	ids, total := selectChunks(results, 1000)

	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}
	if total != 600 {
		t.Errorf("expected 600 tokens, got %d", total)
	}
}

func TestSelectChunks_ZeroBudget(t *testing.T) {
	results := []corequery.ResultEntry{
		{ChunkID: "obj:a.1", Tokens: 100},
	}

	ids, total := selectChunks(results, 0)

	// First result always included, even with 0 budget.
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID (first always included), got %d", len(ids))
	}
	if total != 100 {
		t.Errorf("expected 100 tokens, got %d", total)
	}
}

// ---------------------------------------------------------------------------
// outputDocument
// ---------------------------------------------------------------------------

func TestOutputDocument_FileMode(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.md")

	content := []byte("# Test Document")
	header := "header text"
	footer := "footer text"

	err := outputDocument(content, header, footer, filePath)
	if err != nil {
		t.Fatalf("outputDocument: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if string(data) != "# Test Document" {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

func TestOutputDocument_FileMode_InvalidPath(t *testing.T) {
	err := outputDocument([]byte("content"), "h", "f", "/nonexistent/deep/path/file.md")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestOutputDocument_StdoutMode(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = wOut
	os.Stderr = wErr
	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	})

	outErr := outputDocument([]byte("doc content"), "header text", "footer text", "")

	_ = wOut.Close()
	_ = wErr.Close()

	if outErr != nil {
		t.Fatalf("outputDocument: %v", outErr)
	}

	buf := make([]byte, 4096)
	n, _ := rOut.Read(buf)
	stdout := string(buf[:n])
	if !strings.Contains(stdout, "doc content") {
		t.Errorf("stdout should contain document, got %q", stdout)
	}

	n, _ = rErr.Read(buf)
	stderr := string(buf[:n])
	if !strings.Contains(stderr, "header text") {
		t.Errorf("stderr should contain header, got %q", stderr)
	}
	if !strings.Contains(stderr, "footer text") {
		t.Errorf("stderr should contain footer, got %q", stderr)
	}
}
