package compile

import (
	"strings"
	"testing"

	corecompile "github.com/securacore/codectx/core/compile"
)

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

// ---------------------------------------------------------------------------
// renderConfigError
// ---------------------------------------------------------------------------

func TestRenderConfigError_ContainsTitle(t *testing.T) {
	got := renderConfigError("AI configuration", "ai.yml", errTest("file not found"))

	if !strings.Contains(got, "Failed to load AI configuration") {
		t.Errorf("expected title in output:\n%s", got)
	}
	if !strings.Contains(got, "ai.yml") {
		t.Errorf("expected file name in output:\n%s", got)
	}
	if !strings.Contains(got, "file not found") {
		t.Errorf("expected error message in output:\n%s", got)
	}
	if !strings.Contains(got, "codectx init") {
		t.Errorf("expected reinitialize suggestion in output:\n%s", got)
	}
}

func TestRenderConfigError_Preferences(t *testing.T) {
	got := renderConfigError("preferences", "preferences.yml", errTest("parse error"))

	if !strings.Contains(got, "Failed to load preferences") {
		t.Errorf("expected preferences title in output:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// renderSummary — LLM lines
// ---------------------------------------------------------------------------

func TestRenderSummary_LLMSkipped(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:    5,
		TotalChunks:   10,
		ObjectChunks:  10,
		TotalTokens:   5000,
		AvgTokens:     500,
		MinTokens:     200,
		MaxTokens:     800,
		LLMSkipped:    true,
		LLMSkipReason: "no provider available",
		TotalSeconds:  0.5,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if !strings.Contains(got, "LLM") {
		t.Error("expected LLM line in summary")
	}
	if !strings.Contains(got, "skipped") {
		t.Error("expected 'skipped' in LLM line")
	}
	if !strings.Contains(got, "no provider available") {
		t.Error("expected skip reason in LLM line")
	}
}

func TestRenderSummary_LLMWithResults(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:     5,
		TotalChunks:    10,
		ObjectChunks:   10,
		TotalTokens:    5000,
		AvgTokens:      500,
		MinTokens:      200,
		MaxTokens:      800,
		LLMAliasCount:  42,
		LLMBridgeCount: 8,
		LLMSeconds:     2.5,
		TotalSeconds:   3.0,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if !strings.Contains(got, "42 aliases") {
		t.Error("expected alias count in LLM line")
	}
	if !strings.Contains(got, "8 bridges") {
		t.Error("expected bridge count in LLM line")
	}
}

func TestRenderSummary_DetBridges(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:     5,
		TotalChunks:    10,
		ObjectChunks:   10,
		TotalTokens:    5000,
		AvgTokens:      500,
		MinTokens:      200,
		MaxTokens:      800,
		DetBridgeCount: 7,
		TotalSeconds:   0.5,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if !strings.Contains(got, "Bridges") {
		t.Error("expected Bridges line in summary")
	}
	if !strings.Contains(got, "deterministic") {
		t.Error("expected 'deterministic' in bridges line")
	}
}

func TestRenderSummary_IncrementalMode(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:      10,
		TotalChunks:     25,
		ObjectChunks:    25,
		TotalTokens:     12500,
		AvgTokens:       500,
		MinTokens:       200,
		MaxTokens:       800,
		IncrementalMode: true,
		NewFiles:        2,
		ModifiedFiles:   3,
		UnchangedFiles:  5,
		TotalSeconds:    1.0,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if !strings.Contains(got, "Changes") {
		t.Error("expected Changes line in incremental summary")
	}
	if !strings.Contains(got, "2 new") {
		t.Error("expected new file count")
	}
	if !strings.Contains(got, "3 modified") {
		t.Error("expected modified file count")
	}
	if !strings.Contains(got, "5 unchanged") {
		t.Error("expected unchanged file count")
	}
}

func TestRenderSummary_NonIncrementalNoChangesLine(t *testing.T) {
	result := &corecompile.Result{
		TotalFiles:      10,
		TotalChunks:     25,
		ObjectChunks:    25,
		TotalTokens:     12500,
		AvgTokens:       500,
		MinTokens:       200,
		MaxTokens:       800,
		IncrementalMode: false,
		TotalSeconds:    1.0,
	}

	got := renderSummary(result, "test", "model", "/tmp/compiled", "/tmp")

	if strings.Contains(got, "Changes") {
		t.Error("should not show Changes line in non-incremental mode")
	}
}

func TestRenderSummary_LLMNoResults(t *testing.T) {
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

	// When LLM is neither skipped nor produced results, no LLM line.
	if strings.Contains(got, "LLM") {
		t.Error("should not show LLM line when no results and not skipped")
	}
}
