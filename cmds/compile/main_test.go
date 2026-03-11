package compile

import (
	"strings"
	"testing"

	corecompile "github.com/securacore/codectx/core/compile"
)

// ---------------------------------------------------------------------------
// stageTitle
// ---------------------------------------------------------------------------

func TestStageTitle_KnownStages(t *testing.T) {
	tests := []struct {
		stage    string
		detail   string
		contains string
	}{
		{corecompile.StagePrepare, "", "Preparing output directories..."},
		{corecompile.StageDiscover, "", "Discovering source files..."},
		{corecompile.StageParse, "", "Parsing and validating..."},
		{corecompile.StageChunk, "", "Chunking documents..."},
		{corecompile.StageWrite, "", "Writing chunk files..."},
		{corecompile.StageIndex, "", "Building search index..."},
		{corecompile.StageManifest, "", "Generating manifests..."},
		{corecompile.StageContext, "", "Assembling session context..."},
		{corecompile.StageLink, "", "Updating entry points..."},
		{corecompile.StageHeuristic, "", "Computing heuristics..."},
	}

	for _, tt := range tests {
		got := stageTitle(tt.stage, tt.detail)
		if got != tt.contains {
			t.Errorf("stageTitle(%q, %q) = %q, want %q", tt.stage, tt.detail, got, tt.contains)
		}
	}
}

func TestStageTitle_WithDetail(t *testing.T) {
	got := stageTitle(corecompile.StageParse, "12 files")
	if !strings.Contains(got, "Parsing and validating...") {
		t.Errorf("expected title text, got %q", got)
	}
	if !strings.Contains(got, "(12 files)") {
		t.Errorf("expected detail in parentheses, got %q", got)
	}
}

func TestStageTitle_UnknownStage(t *testing.T) {
	got := stageTitle("unknown-stage", "")
	if got != "unknown-stage" {
		t.Errorf("expected raw stage name for unknown stage, got %q", got)
	}
}

func TestStageTitle_UnknownStageWithDetail(t *testing.T) {
	got := stageTitle("custom", "detail")
	want := "custom (detail)"
	if got != want {
		t.Errorf("stageTitle(%q, %q) = %q, want %q", "custom", "detail", got, want)
	}
}

// ---------------------------------------------------------------------------
// countNonZero
// ---------------------------------------------------------------------------

func TestCountNonZero_AllZero(t *testing.T) {
	if got := countNonZero(0, 0, 0); got != 0 {
		t.Errorf("countNonZero(0,0,0) = %d, want 0", got)
	}
}

func TestCountNonZero_AllNonZero(t *testing.T) {
	if got := countNonZero(1, 2, 3); got != 3 {
		t.Errorf("countNonZero(1,2,3) = %d, want 3", got)
	}
}

func TestCountNonZero_Mixed(t *testing.T) {
	if got := countNonZero(0, 5, 0); got != 1 {
		t.Errorf("countNonZero(0,5,0) = %d, want 1", got)
	}
}

func TestCountNonZero_SingleArg(t *testing.T) {
	if got := countNonZero(42); got != 1 {
		t.Errorf("countNonZero(42) = %d, want 1", got)
	}
}

func TestCountNonZero_NoArgs(t *testing.T) {
	if got := countNonZero(); got != 0 {
		t.Errorf("countNonZero() = %d, want 0", got)
	}
}

func TestCountNonZero_NegativeValues(t *testing.T) {
	// Negative values are treated as zero (only positive counts matter).
	if got := countNonZero(-1, 0, 1); got != 1 {
		t.Errorf("countNonZero(-1,0,1) = %d, want 1", got)
	}
}

// ---------------------------------------------------------------------------
// renderSummary
// ---------------------------------------------------------------------------

func TestRenderSummary_ContainsAllFields(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:   10,
		SpecFiles:    2,
		TotalChunks:  25,
		ObjectChunks: 15,
		SpecChunks:   5,
		SystemChunks: 5,
		TotalTokens:  12500,
		AvgTokens:    500,
		MinTokens:    200,
		MaxTokens:    800,
		TotalSeconds: 1.5,
	}

	got := renderSummary(result, "my-project", "claude-sonnet-4", "/tmp/compiled", "/tmp")

	checks := []string{
		"Compilation complete",
		"10 files",
		"25 chunks",
		"12,500 tokens",
		"objects: 15",
		"specs: 5",
		"system: 5",
		"3 indexes",
		"avg 500",
		"min 200",
		"max 800",
		"claude-sonnet-4",
		"compiled",
	}

	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("renderSummary missing %q in output:\n%s", want, got)
		}
	}
}

func TestRenderSummary_NoChunks(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:   0,
		TotalChunks:  0,
		TotalTokens:  0,
		TotalSeconds: 0.05,
	}

	got := renderSummary(result, "empty-project", "claude-sonnet-4", "/tmp/compiled", "/tmp")

	if !strings.Contains(got, "Compilation complete") {
		t.Error("expected success header even with no chunks")
	}
	if strings.Contains(got, "avg") {
		t.Error("should not show token avg/min/max when no chunks")
	}
	if strings.Contains(got, "indexes") {
		t.Error("should not show index line when no chunks of any type")
	}
}

func TestRenderSummary_OversizedChunks(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:   5,
		TotalChunks:  10,
		ObjectChunks: 10,
		TotalTokens:  5000,
		AvgTokens:    500,
		MinTokens:    200,
		MaxTokens:    1200,
		Oversized:    2,
		TotalSeconds: 0.5,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if !strings.Contains(got, "Oversized") {
		t.Error("expected oversized warning in summary")
	}
	if !strings.Contains(got, "2 chunks exceed max_tokens") {
		t.Error("expected oversized count in summary")
	}
}

func TestRenderSummary_NoOversized(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:   5,
		TotalChunks:  10,
		ObjectChunks: 10,
		TotalTokens:  5000,
		AvgTokens:    500,
		MinTokens:    200,
		MaxTokens:    800,
		Oversized:    0,
		TotalSeconds: 0.3,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if strings.Contains(got, "Oversized") {
		t.Error("should not show oversized line when count is 0")
	}
}

func TestRenderSummary_RelativePath(t *testing.T) {
	got := renderSummary(
		&corecompile.Result{TotalSeconds: 0.1},
		"test", "model",
		"/projects/myapp/docs/.codectx/compiled",
		"/projects/myapp",
	)

	if !strings.Contains(got, "docs/.codectx/compiled") {
		t.Errorf("expected relative output path, got:\n%s", got)
	}
}

func TestRenderSummary_OnlyObjectChunks(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:   3,
		TotalChunks:  8,
		ObjectChunks: 8,
		TotalTokens:  4000,
		AvgTokens:    500,
		MinTokens:    200,
		MaxTokens:    700,
		TotalSeconds: 0.2,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if !strings.Contains(got, "1 indexes") {
		t.Errorf("expected 1 index, got:\n%s", got)
	}
	if !strings.Contains(got, "objects: 8") {
		t.Errorf("expected objects count, got:\n%s", got)
	}
	if strings.Contains(got, "specs:") {
		t.Error("should not show specs when count is 0")
	}
	if strings.Contains(got, "system:") {
		t.Error("should not show system when count is 0")
	}
}

func TestRenderSummary_WithSessionData(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:    10,
		TotalChunks:   25,
		ObjectChunks:  25,
		TotalTokens:   12500,
		AvgTokens:     500,
		MinTokens:     200,
		MaxTokens:     800,
		SessionTokens: 28450,
		SessionBudget: 30000,
		TotalSeconds:  1.0,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if !strings.Contains(got, "Session") {
		t.Error("expected Session line in summary")
	}
	if !strings.Contains(got, "28,450") {
		t.Error("expected session token count")
	}
	if !strings.Contains(got, "30,000") {
		t.Error("expected session budget")
	}
	if !strings.Contains(got, "94.8%") {
		t.Error("expected session utilization percentage")
	}
}

func TestRenderSummary_NoSessionData(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:   5,
		TotalChunks:  10,
		ObjectChunks: 10,
		TotalTokens:  5000,
		AvgTokens:    500,
		MinTokens:    200,
		MaxTokens:    800,
		TotalSeconds: 0.5,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if strings.Contains(got, "Session") {
		t.Error("should not show Session line when no session data")
	}
}

// ---------------------------------------------------------------------------
// renderWarnings
// ---------------------------------------------------------------------------

func TestRenderWarnings_NoWarnings(t *testing.T) {
	got := renderWarnings(nil)
	if got != "" {
		t.Errorf("expected empty string for nil warnings, got %q", got)
	}

	got = renderWarnings([]string{})
	if got != "" {
		t.Errorf("expected empty string for empty warnings, got %q", got)
	}
}

func TestRenderWarnings_SingleWarning(t *testing.T) {
	got := renderWarnings([]string{"missing README in topics/auth"})

	if !strings.Contains(got, "1 validation warning") {
		t.Errorf("expected warning count, got:\n%s", got)
	}
	if !strings.Contains(got, "missing README in topics/auth") {
		t.Errorf("expected warning text, got:\n%s", got)
	}
}

func TestRenderWarnings_MultipleWarnings(t *testing.T) {
	warnings := []string{
		"missing README in topics/auth",
		"no headings in topics/guide.md",
		"file exceeds max_file_tokens: topics/large.md",
	}

	got := renderWarnings(warnings)

	if !strings.Contains(got, "3 validation warning") {
		t.Errorf("expected warning count, got:\n%s", got)
	}
	for _, w := range warnings {
		if !strings.Contains(got, w) {
			t.Errorf("expected warning %q in output:\n%s", w, got)
		}
	}
}

// ---------------------------------------------------------------------------
// renderCompileError
// ---------------------------------------------------------------------------

func TestRenderCompileError_ReturnsWrappedError(t *testing.T) {
	err := renderCompileError(errTest("something went wrong"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "compilation failed") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("expected original error in chain, got: %v", err)
	}
}

// errTest is a simple error type for testing.
type errTest string

func (e errTest) Error() string { return string(e) }
