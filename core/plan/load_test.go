package plan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/testutil"
)

// samplePlanYAML is a complete plan.yml matching the spec's example schema.
const samplePlanYAML = `name: "Authentication System Migration"
status: "in-progress"
created: "2025-03-01T00:00:00Z"
updated: "2025-03-09T12:00:00Z"

dependencies:
  - path: "foundation/architecture-principles"
    hash: "sha256:a1b2c3"
  - path: "topics/authentication/jwt-tokens"
    hash: "sha256:d4e5f6"
  - path: "topics/authentication/oauth"
    hash: "sha256:g7h8i9"

steps:
  - id: 1
    title: "Audit current JWT implementation"
    status: "completed"
    completed_at: "2025-03-02T14:00:00Z"
    notes: "Found 3 services using deprecated token format"
    queries:
      - "jwt token implementation current"
      - "token validation service audit"
    chunks:
      - "obj:a1b2c3.01,obj:a1b2c3.02,obj:a1b2c3.03,spec:f7g8h9.01"

  - id: 2
    title: "Design new token schema"
    status: "completed"
    completed_at: "2025-03-05T10:00:00Z"
    queries:
      - "jwt token schema design"
      - "refresh token lifecycle"
    chunks:
      - "obj:a1b2c3.03,obj:a1b2c3.04,obj:d4e5f6.02,spec:f7g8h9.02"

  - id: 3
    title: "Implement token service refactor"
    status: "in-progress"
    started_at: "2025-03-07T09:00:00Z"
    notes: "User service and payment service updated. Order service remaining."
    queries:
      - "token service refactor implementation"
      - "order service authentication"
    chunks:
      - "obj:a1b2c3.04,obj:d4e5f6.02,obj:d4e5f6.03"
      - "obj:x9y8z7.01,spec:x9y8z7.01"

  - id: 4
    title: "Migration testing"
    status: "pending"
    blocked_by: [3]

  - id: 5
    title: "Production rollout"
    status: "pending"
    blocked_by: [4]

current_step: 3
`

func TestLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	planPath := filepath.Join(dir, PlanFile)
	testutil.MustWriteFile(t, planPath, samplePlanYAML)

	p, err := Load(planPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Top-level fields.
	if p.Name != "Authentication System Migration" {
		t.Errorf("Name = %q, want %q", p.Name, "Authentication System Migration")
	}
	if p.Status != StatusInProgress {
		t.Errorf("Status = %q, want %q", p.Status, StatusInProgress)
	}
	if p.Created != "2025-03-01T00:00:00Z" {
		t.Errorf("Created = %q, want %q", p.Created, "2025-03-01T00:00:00Z")
	}
	if p.CurrentStep != 3 {
		t.Errorf("CurrentStep = %d, want 3", p.CurrentStep)
	}

	// Dependencies.
	if len(p.Dependencies) != 3 {
		t.Fatalf("Dependencies count = %d, want 3", len(p.Dependencies))
	}
	if p.Dependencies[0].Path != "foundation/architecture-principles" {
		t.Errorf("Dependencies[0].Path = %q", p.Dependencies[0].Path)
	}
	if p.Dependencies[0].Hash != "sha256:a1b2c3" {
		t.Errorf("Dependencies[0].Hash = %q", p.Dependencies[0].Hash)
	}

	// Steps.
	if len(p.Steps) != 5 {
		t.Fatalf("Steps count = %d, want 5", len(p.Steps))
	}

	// Step 1 — completed.
	s1 := p.Steps[0]
	if s1.ID != 1 || s1.Status != StepCompleted {
		t.Errorf("Step 1: ID=%d, Status=%q", s1.ID, s1.Status)
	}
	if s1.CompletedAt != "2025-03-02T14:00:00Z" {
		t.Errorf("Step 1 CompletedAt = %q", s1.CompletedAt)
	}
	if len(s1.Queries) != 2 {
		t.Errorf("Step 1 Queries count = %d, want 2", len(s1.Queries))
	}
	if len(s1.Chunks) != 1 {
		t.Errorf("Step 1 Chunks count = %d, want 1", len(s1.Chunks))
	}

	// Step 3 — in-progress with multiple chunks.
	s3 := p.Steps[2]
	if s3.ID != 3 || s3.Status != StepInProgress {
		t.Errorf("Step 3: ID=%d, Status=%q", s3.ID, s3.Status)
	}
	if s3.StartedAt != "2025-03-07T09:00:00Z" {
		t.Errorf("Step 3 StartedAt = %q", s3.StartedAt)
	}
	if len(s3.Chunks) != 2 {
		t.Errorf("Step 3 Chunks count = %d, want 2", len(s3.Chunks))
	}

	// Step 4 — pending with blocked_by.
	s4 := p.Steps[3]
	if s4.Status != StepPending {
		t.Errorf("Step 4 Status = %q, want %q", s4.Status, StepPending)
	}
	if len(s4.BlockedBy) != 1 || s4.BlockedBy[0] != 3 {
		t.Errorf("Step 4 BlockedBy = %v, want [3]", s4.BlockedBy)
	}
}

func TestLoadNonExistent(t *testing.T) {
	t.Parallel()

	_, err := Load("/nonexistent/path/plan.yml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	planPath := filepath.Join(dir, PlanFile)
	testutil.MustWriteFile(t, planPath, "{{invalid yaml")

	_, err := Load(planPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadMinimal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	planPath := filepath.Join(dir, PlanFile)
	testutil.MustWriteFile(t, planPath, `name: "Minimal Plan"
status: "draft"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 0
`)

	p, err := Load(planPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Slices should be non-nil.
	if p.Dependencies == nil {
		t.Error("Dependencies should be non-nil")
	}
	if p.Steps == nil {
		t.Error("Steps should be non-nil")
	}
	if len(p.Dependencies) != 0 {
		t.Errorf("Dependencies count = %d, want 0", len(p.Dependencies))
	}
	if len(p.Steps) != 0 {
		t.Errorf("Steps count = %d, want 0", len(p.Steps))
	}
}

func TestSaveRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	planPath := filepath.Join(dir, PlanFile)

	original := &Plan{
		Name:    "Test Plan",
		Status:  StatusInProgress,
		Created: "2025-06-01T00:00:00Z",
		Updated: "2025-06-09T12:00:00Z",
		Dependencies: []Dependency{
			{Path: "foundation/overview", Hash: "sha256:abc123"},
		},
		Steps: []Step{
			{
				ID:          1,
				Title:       "First step",
				Status:      StepCompleted,
				CompletedAt: "2025-06-02T10:00:00Z",
				Notes:       "Done",
				Queries:     []string{"search term"},
				Chunks:      []string{"obj:abc123.01,obj:abc123.02"},
				BlockedBy:   []int{},
			},
			{
				ID:        2,
				Title:     "Second step",
				Status:    StepInProgress,
				StartedAt: "2025-06-05T09:00:00Z",
				Queries:   []string{},
				Chunks:    []string{},
				BlockedBy: []int{},
			},
		},
		CurrentStep: 2,
	}

	if err := Save(planPath, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(planPath)
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}

	// Verify round-trip fidelity.
	if loaded.Name != original.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, original.Name)
	}
	if loaded.Status != original.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, original.Status)
	}
	if loaded.CurrentStep != original.CurrentStep {
		t.Errorf("CurrentStep = %d, want %d", loaded.CurrentStep, original.CurrentStep)
	}
	if len(loaded.Dependencies) != 1 {
		t.Fatalf("Dependencies count = %d, want 1", len(loaded.Dependencies))
	}
	if loaded.Dependencies[0].Hash != "sha256:abc123" {
		t.Errorf("Dependency hash = %q", loaded.Dependencies[0].Hash)
	}
	if len(loaded.Steps) != 2 {
		t.Fatalf("Steps count = %d, want 2", len(loaded.Steps))
	}
	if loaded.Steps[0].CompletedAt != "2025-06-02T10:00:00Z" {
		t.Errorf("Step 1 CompletedAt = %q", loaded.Steps[0].CompletedAt)
	}
}

func TestDiscover(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "docs")

	// Create two plan directories, one without plan.yml.
	testutil.MustWriteFile(t, filepath.Join(rootDir, "plans", "auth-migration", PlanFile), samplePlanYAML)
	testutil.MustWriteFile(t, filepath.Join(rootDir, "plans", "db-refactor", PlanFile), `name: "DB Refactor"
status: "draft"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 0
`)
	// Directory without plan.yml.
	if err := os.MkdirAll(filepath.Join(rootDir, "plans", "no-plan-here"), 0755); err != nil {
		t.Fatal(err)
	}

	plans, err := Discover(rootDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(plans) != 2 {
		t.Fatalf("plans count = %d, want 2", len(plans))
	}
	if _, ok := plans["auth-migration"]; !ok {
		t.Error("missing auth-migration plan")
	}
	if _, ok := plans["db-refactor"]; !ok {
		t.Error("missing db-refactor plan")
	}
}

func TestDiscoverNoPlansDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	plans, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected no plans, got %d", len(plans))
	}
}

func TestFindPlanByName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "docs")
	testutil.MustWriteFile(t, filepath.Join(rootDir, "plans", "auth-migration", PlanFile), samplePlanYAML)

	name, path, err := FindPlan(rootDir, "auth-migration")
	if err != nil {
		t.Fatalf("FindPlan failed: %v", err)
	}
	if name != "auth-migration" {
		t.Errorf("name = %q, want %q", name, "auth-migration")
	}
	if path == "" {
		t.Error("path should not be empty")
	}
}

func TestFindPlanAutoDetect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "docs")
	testutil.MustWriteFile(t, filepath.Join(rootDir, "plans", "only-plan", PlanFile), `name: "Only Plan"
status: "draft"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 0
`)

	name, _, err := FindPlan(rootDir, "")
	if err != nil {
		t.Fatalf("FindPlan failed: %v", err)
	}
	if name != "only-plan" {
		t.Errorf("name = %q, want %q", name, "only-plan")
	}
}

func TestFindPlanMultipleRequiresName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "docs")
	testutil.MustWriteFile(t, filepath.Join(rootDir, "plans", "plan-a", PlanFile), `name: "A"
status: "draft"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 0
`)
	testutil.MustWriteFile(t, filepath.Join(rootDir, "plans", "plan-b", PlanFile), `name: "B"
status: "draft"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 0
`)

	_, _, err := FindPlan(rootDir, "")
	if err == nil {
		t.Error("expected error when multiple plans exist and no name specified")
	}
}

func TestFindPlanNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootDir := filepath.Join(dir, "docs")
	testutil.MustWriteFile(t, filepath.Join(rootDir, "plans", "auth", PlanFile), `name: "Auth"
status: "draft"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 0
`)

	_, _, err := FindPlan(rootDir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan name")
	}
}

func TestFindPlanNoPlans(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, _, err := FindPlan(dir, "")
	if err == nil {
		t.Error("expected error when no plans exist")
	}
}
