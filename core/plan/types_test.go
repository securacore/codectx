package plan

import (
	"testing"
)

func TestCurrentStepEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		plan       Plan
		wantNil    bool
		wantStepID int
	}{
		{
			name:    "no current step",
			plan:    Plan{CurrentStep: 0, Steps: []Step{{ID: 1}}},
			wantNil: true,
		},
		{
			name:    "negative current step",
			plan:    Plan{CurrentStep: -1, Steps: []Step{{ID: 1}}},
			wantNil: true,
		},
		{
			name:       "matching step",
			plan:       Plan{CurrentStep: 2, Steps: []Step{{ID: 1}, {ID: 2}, {ID: 3}}},
			wantStepID: 2,
		},
		{
			name:    "nonexistent step ID",
			plan:    Plan{CurrentStep: 99, Steps: []Step{{ID: 1}, {ID: 2}}},
			wantNil: true,
		},
		{
			name:    "empty steps",
			plan:    Plan{CurrentStep: 1, Steps: []Step{}},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.plan.CurrentStepEntry()
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got step %d", got.ID)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil step, got nil")
			}
			if got.ID != tt.wantStepID {
				t.Errorf("step ID = %d, want %d", got.ID, tt.wantStepID)
			}
		})
	}
}

func TestProgress(t *testing.T) {
	t.Parallel()

	p := Plan{
		Steps: []Step{
			{ID: 1, Status: StepCompleted},
			{ID: 2, Status: StepCompleted},
			{ID: 3, Status: StepInProgress},
			{ID: 4, Status: StepPending},
			{ID: 5, Status: StepPending},
		},
	}

	completed, inProgress, pending := p.Progress()
	if completed != 2 {
		t.Errorf("completed = %d, want 2", completed)
	}
	if inProgress != 1 {
		t.Errorf("inProgress = %d, want 1", inProgress)
	}
	if pending != 2 {
		t.Errorf("pending = %d, want 2", pending)
	}
}

func TestProgressEmpty(t *testing.T) {
	t.Parallel()

	p := Plan{Steps: []Step{}}
	completed, inProgress, pending := p.Progress()
	if completed != 0 || inProgress != 0 || pending != 0 {
		t.Errorf("expected all zeros, got %d/%d/%d", completed, inProgress, pending)
	}
}

func TestBlockedSteps(t *testing.T) {
	t.Parallel()

	p := Plan{
		Steps: []Step{
			{ID: 1, Status: StepCompleted},
			{ID: 2, Status: StepCompleted},
			{ID: 3, Status: StepInProgress},
			{ID: 4, Status: StepPending, BlockedBy: []int{3}},
			{ID: 5, Status: StepPending, BlockedBy: []int{4}},
		},
	}

	blocked := p.BlockedSteps()
	if len(blocked) != 2 {
		t.Fatalf("blocked count = %d, want 2", len(blocked))
	}
	if blocked[0].ID != 4 {
		t.Errorf("blocked[0].ID = %d, want 4", blocked[0].ID)
	}
	if blocked[1].ID != 5 {
		t.Errorf("blocked[1].ID = %d, want 5", blocked[1].ID)
	}
}

func TestBlockedStepsAllCompleted(t *testing.T) {
	t.Parallel()

	p := Plan{
		Steps: []Step{
			{ID: 1, Status: StepCompleted},
			{ID: 2, Status: StepCompleted, BlockedBy: []int{1}},
		},
	}

	blocked := p.BlockedSteps()
	if len(blocked) != 0 {
		t.Errorf("expected no blocked steps when all blockers completed, got %d", len(blocked))
	}
}

func TestBlockedStepsNoBlockers(t *testing.T) {
	t.Parallel()

	p := Plan{
		Steps: []Step{
			{ID: 1, Status: StepPending},
			{ID: 2, Status: StepPending},
		},
	}

	blocked := p.BlockedSteps()
	if len(blocked) != 0 {
		t.Errorf("expected no blocked steps, got %d", len(blocked))
	}
}

func TestStepByID(t *testing.T) {
	t.Parallel()

	p := Plan{
		Steps: []Step{
			{ID: 1, Title: "First"},
			{ID: 2, Title: "Second"},
			{ID: 3, Title: "Third"},
		},
	}

	// Found.
	s := p.StepByID(2)
	if s == nil {
		t.Fatal("expected step 2, got nil")
	}
	if s.Title != "Second" {
		t.Errorf("title = %q, want %q", s.Title, "Second")
	}

	// Not found.
	if p.StepByID(99) != nil {
		t.Error("expected nil for nonexistent step")
	}
}
