package plan

import (
	"strings"
	"testing"
)

func TestFormatStatus(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Authentication System Migration",
		Status:      StatusInProgress,
		CurrentStep: 3,
		Steps: []Step{
			{ID: 1, Status: StepCompleted},
			{ID: 2, Status: StepCompleted},
			{
				ID:        3,
				Title:     "Implement token service refactor",
				Status:    StepInProgress,
				StartedAt: "2025-03-07T09:00:00Z",
				Notes:     "User service and payment service updated. Order service remaining.",
				Queries: []string{
					"token service refactor implementation",
					"order service authentication",
				},
			},
			{ID: 4, Title: "Migration testing", Status: StepPending, BlockedBy: []int{3}},
			{ID: 5, Title: "Production rollout", Status: StepPending, BlockedBy: []int{4}},
		},
	}

	check := &CheckResult{
		AllMatch:     false,
		ChangedCount: 1,
		Statuses: []DependencyStatus{
			{Dependency: Dependency{Path: "foundation/architecture-principles"}, Changed: false},
			{Dependency: Dependency{Path: "topics/authentication/jwt-tokens"}, Changed: true},
			{Dependency: Dependency{Path: "topics/authentication/oauth"}, Changed: false},
		},
	}

	output := FormatStatus(p, check)

	// Check key content from the spec's example output.
	assertions := []string{
		"Plan: Authentication System Migration",
		"step 3 of 5",
		"2 steps completed",
		"1 in progress",
		"2 pending",
		"Current step: Implement token service refactor",
		"Started: 2025-03-07T09:00:00Z",
		"User service and payment service updated",
		"token service refactor implementation",
		"order service authentication",
		"foundation/architecture-principles",
		"unchanged",
		"topics/authentication/jwt-tokens",
		"content changed",
		"Step 4 (Migration testing)",
		"blocked by step 3",
		"Step 5 (Production rollout)",
		"blocked by step 4",
	}

	for _, want := range assertions {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}

func TestFormatStatusNoDependencies(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Simple Plan",
		Status:      StatusDraft,
		CurrentStep: 0,
		Steps:       []Step{},
	}

	output := FormatStatus(p, nil)
	if !strings.Contains(output, "Plan: Simple Plan") {
		t.Errorf("missing plan name in output: %s", output)
	}
	if strings.Contains(output, "Dependencies") {
		t.Errorf("should not show Dependencies section when check is nil: %s", output)
	}
}

func TestFormatResumeMatch(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Auth Migration",
		Status:      StatusInProgress,
		CurrentStep: 3,
		Steps: []Step{
			{ID: 1, Status: StepCompleted},
			{ID: 2, Status: StepCompleted},
			{
				ID:     3,
				Title:  "Token refactor",
				Status: StepInProgress,
				Notes:  "In progress notes.",
			},
			{ID: 4, Status: StepPending},
			{ID: 5, Status: StepPending},
		},
	}

	generateOutputs := []string{
		"Generated: /tmp/codectx/auth.123.md (500 tokens)\nContains: obj:abc.01\n",
	}

	output := FormatResumeMatch(p, generateOutputs)

	assertions := []string{
		"Plan: Auth Migration",
		"step 3 of 5",
		"all unchanged",
		"Replaying context for step 3",
		"/tmp/codectx/auth.123.md",
		"Current step: Token refactor",
		"Notes: In progress notes.",
	}

	for _, want := range assertions {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}

func TestFormatResumeDrift(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Auth Migration",
		Status:      StatusInProgress,
		CurrentStep: 3,
		Steps: []Step{
			{ID: 1, Status: StepCompleted},
			{ID: 2, Status: StepCompleted},
			{
				ID:     3,
				Title:  "Token refactor",
				Status: StepInProgress,
				Queries: []string{
					"token service refactor",
					"order service auth",
				},
			},
			{ID: 4, Status: StepPending},
			{ID: 5, Status: StepPending},
		},
	}

	check := &CheckResult{
		AllMatch:     false,
		ChangedCount: 1,
		Statuses: []DependencyStatus{
			{Dependency: Dependency{Path: "topics/jwt"}, Changed: true},
			{Dependency: Dependency{Path: "foundation/arch"}, Changed: false},
		},
	}

	output := FormatResumeDrift(p, check)

	assertions := []string{
		"Plan: Auth Migration",
		"Documentation changes since last update",
		"topics/jwt",
		"content changed",
		"foundation/arch",
		"unchanged",
		"Stored chunks may be stale",
		"token service refactor",
		"order service auth",
		"Recommendation",
	}

	for _, want := range assertions {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}

func TestFormatProgressAllTypes(t *testing.T) {
	t.Parallel()

	result := formatProgress(2, 1, 3)
	if !strings.Contains(result, "2 steps completed") {
		t.Errorf("missing completed count: %s", result)
	}
	if !strings.Contains(result, "1 in progress") {
		t.Errorf("missing in progress count: %s", result)
	}
	if !strings.Contains(result, "3 pending") {
		t.Errorf("missing pending count: %s", result)
	}
}

func TestFormatProgressSingleStep(t *testing.T) {
	t.Parallel()

	result := formatProgress(1, 0, 0)
	if !strings.Contains(result, "1 step completed") {
		t.Errorf("should use singular 'step': %s", result)
	}
}

func TestFormatProgressEmpty(t *testing.T) {
	t.Parallel()

	result := formatProgress(0, 0, 0)
	if result != "no steps" {
		t.Errorf("result = %q, want %q", result, "no steps")
	}
}

func TestPluralize(t *testing.T) {
	t.Parallel()

	if pluralize("step", 1) != "step" {
		t.Error("singular should not be pluralized")
	}
	if pluralize("step", 0) != "steps" {
		t.Error("zero should be pluralized")
	}
	if pluralize("step", 2) != "steps" {
		t.Error("two should be pluralized")
	}
}

// Tests for ParseChunkIDs are in core/query/generate_internal_test.go
// following the extraction of the function to the shared core/query package.

// ---------------------------------------------------------------------------
// formatBlockers
// ---------------------------------------------------------------------------

func TestFormatBlockersKnownSteps(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Steps: []Step{
			{ID: 1, Title: "First"},
			{ID: 2, Title: "Second"},
			{ID: 3, Title: "Third"},
		},
	}

	result := formatBlockers(p, []int{1, 2})
	if result != "step 1, step 2" {
		t.Errorf("result = %q, want %q", result, "step 1, step 2")
	}
}

func TestFormatBlockersUnknownStep(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Steps: []Step{
			{ID: 1, Title: "First"},
		},
	}

	result := formatBlockers(p, []int{1, 99})
	if !strings.Contains(result, "step 1") {
		t.Errorf("missing known step: %s", result)
	}
	if !strings.Contains(result, "step 99 (unknown)") {
		t.Errorf("missing unknown step marker: %s", result)
	}
}

func TestFormatBlockersEmpty(t *testing.T) {
	t.Parallel()

	p := &Plan{}
	result := formatBlockers(p, nil)
	if result != "" {
		t.Errorf("result = %q, want empty", result)
	}
}

// ---------------------------------------------------------------------------
// FormatStatus edge cases
// ---------------------------------------------------------------------------

func TestFormatStatusMissingDependency(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Plan",
		Status:      StatusInProgress,
		CurrentStep: 1,
		Steps:       []Step{{ID: 1, Title: "Step", Status: StepInProgress}},
	}

	check := &CheckResult{
		Statuses: []DependencyStatus{
			{Dependency: Dependency{Path: "missing/dep"}, Changed: true, Missing: true},
		},
	}

	output := FormatStatus(p, check)
	if !strings.Contains(output, "not found in compiled output") {
		t.Errorf("missing 'not found' suffix for missing dependency: %s", output)
	}
}

func TestFormatStatusEmptyCheck(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:   "Plan",
		Status: StatusDraft,
		Steps:  []Step{},
	}

	check := &CheckResult{
		Statuses: []DependencyStatus{},
	}

	output := FormatStatus(p, check)
	// Empty statuses should not show Dependencies section.
	if strings.Contains(output, "Dependencies:") {
		t.Errorf("should not show Dependencies section with empty statuses: %s", output)
	}
}

func TestFormatStatusNoCurrentStep(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Plan",
		Status:      StatusDraft,
		CurrentStep: 0,
		Steps: []Step{
			{ID: 1, Title: "First", Status: StepPending},
		},
	}

	output := FormatStatus(p, nil)
	// Should not show "step X of Y" in status line.
	if strings.Contains(output, "step 0") {
		t.Errorf("should not show step 0: %s", output)
	}
	// Should not show "Current step:" section.
	if strings.Contains(output, "Current step:") {
		t.Errorf("should not show Current step section: %s", output)
	}
}

// ---------------------------------------------------------------------------
// FormatResumeMatch edge cases
// ---------------------------------------------------------------------------

func TestFormatResumeMatchNoOutputs(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Plan",
		Status:      StatusInProgress,
		CurrentStep: 1,
		Steps:       []Step{{ID: 1, Title: "Step", Status: StepInProgress}},
	}

	output := FormatResumeMatch(p, nil)
	if strings.Contains(output, "Replaying") {
		t.Errorf("should not show Replaying when no outputs: %s", output)
	}
	if !strings.Contains(output, "all unchanged") {
		t.Errorf("missing unchanged message: %s", output)
	}
}

func TestFormatResumeMatchNoNotes(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Plan",
		Status:      StatusInProgress,
		CurrentStep: 1,
		Steps:       []Step{{ID: 1, Title: "Step", Status: StepInProgress}},
	}

	output := FormatResumeMatch(p, []string{"test output\n"})
	if strings.Contains(output, "Notes:") {
		t.Errorf("should not show Notes when empty: %s", output)
	}
}

// ---------------------------------------------------------------------------
// FormatResumeDrift edge cases
// ---------------------------------------------------------------------------

func TestFormatResumeDriftMissingDep(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Plan",
		Status:      StatusInProgress,
		CurrentStep: 1,
		Steps: []Step{{
			ID: 1, Title: "Step", Status: StepInProgress,
			Queries: []string{"search"},
		}},
	}

	check := &CheckResult{
		Statuses: []DependencyStatus{
			{Dependency: Dependency{Path: "missing/path"}, Changed: true, Missing: true},
		},
	}

	output := FormatResumeDrift(p, check)
	if !strings.Contains(output, "not found in compiled output") {
		t.Errorf("missing 'not found' for missing dep: %s", output)
	}
}

func TestFormatResumeDriftNoQueries(t *testing.T) {
	t.Parallel()

	p := &Plan{
		Name:        "Plan",
		Status:      StatusInProgress,
		CurrentStep: 1,
		Steps:       []Step{{ID: 1, Title: "Step", Status: StepInProgress}},
	}

	check := &CheckResult{
		Statuses: []DependencyStatus{
			{Dependency: Dependency{Path: "dep"}, Changed: true},
		},
	}

	output := FormatResumeDrift(p, check)
	// Should not show "Stored chunks may be stale" when no queries.
	if strings.Contains(output, "Stored chunks may be stale") {
		t.Errorf("should not show stale message when no queries: %s", output)
	}
	// Should still show recommendation.
	if !strings.Contains(output, "Recommendation") {
		t.Errorf("missing recommendation: %s", output)
	}
}

// ---------------------------------------------------------------------------
// writeDependencyStatuses
// ---------------------------------------------------------------------------

func TestWriteDependencyStatuses(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	statuses := []DependencyStatus{
		{Dependency: Dependency{Path: "a"}, Changed: false},
		{Dependency: Dependency{Path: "b"}, Changed: true},
		{Dependency: Dependency{Path: "c"}, Changed: true, Missing: true},
	}

	writeDependencyStatuses(&b, statuses, "content changed")
	output := b.String()

	if !strings.Contains(output, "a") || !strings.Contains(output, "unchanged") {
		t.Errorf("missing unchanged dep: %s", output)
	}
	if !strings.Contains(output, "b") || !strings.Contains(output, "content changed") {
		t.Errorf("missing changed dep: %s", output)
	}
	if !strings.Contains(output, "c") || !strings.Contains(output, "not found in compiled output") {
		t.Errorf("missing dep should show 'not found': %s", output)
	}
}

// ---------------------------------------------------------------------------
// writePlanHeader
// ---------------------------------------------------------------------------

func TestWritePlanHeader(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	p := &Plan{
		Name:        "Test Plan",
		Status:      StatusInProgress,
		CurrentStep: 2,
		Steps:       []Step{{ID: 1}, {ID: 2}, {ID: 3}},
	}

	writePlanHeader(&b, p)
	output := b.String()

	if !strings.Contains(output, "Plan: Test Plan") {
		t.Errorf("missing plan name: %s", output)
	}
	if !strings.Contains(output, "step 2 of 3") {
		t.Errorf("missing step info: %s", output)
	}
}

func TestWritePlanHeaderNoCurrentStep(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	p := &Plan{
		Name:        "Draft",
		Status:      StatusDraft,
		CurrentStep: 0,
		Steps:       []Step{{ID: 1}},
	}

	writePlanHeader(&b, p)
	output := b.String()

	if strings.Contains(output, "step 0") {
		t.Errorf("should not show step info when CurrentStep=0: %s", output)
	}
}
