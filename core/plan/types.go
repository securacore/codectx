// Package plan implements plan state tracking for codectx.
//
// A plan is a structured workflow stored in plan.yaml within a plan directory
// (e.g. docs/plans/auth-migration/plan.yml). It tracks multi-step tasks with
// dependency hash checking for documentation drift detection and chunk replay
// for instant context reconstruction.
//
// This package provides:
//   - Plan, Step, and Dependency types mapping to the plan.yaml schema
//   - Loading and writing plan.yaml files
//   - Dependency hash checking against compiled hashes.yml
//   - Status reporting and resume logic
package plan

// PlanFile is the filename for plan state files within plan directories.
const PlanFile = "plan.yml"

// Status represents the overall plan status.
type Status string

const (
	// StatusDraft indicates a plan that has been created but not started.
	StatusDraft Status = "draft"

	// StatusInProgress indicates a plan with active work.
	StatusInProgress Status = "in-progress"

	// StatusBlocked indicates a plan that cannot proceed due to blockers.
	StatusBlocked Status = "blocked"

	// StatusCompleted indicates a plan that has been fully completed.
	StatusCompleted Status = "completed"
)

// StepStatus represents the status of an individual plan step.
type StepStatus string

const (
	// StepCompleted indicates a step that has been finished.
	StepCompleted StepStatus = "completed"

	// StepInProgress indicates a step currently being worked on.
	StepInProgress StepStatus = "in-progress"

	// StepPending indicates a step not yet started.
	StepPending StepStatus = "pending"
)

// Plan represents the full plan.yaml schema.
// It tracks a multi-step workflow with dependency hashes for drift detection
// and per-step chunk references for context replay.
type Plan struct {
	// Name is the human-readable plan name.
	Name string `yaml:"name"`

	// Status is the overall plan status (draft, in-progress, blocked, completed).
	Status Status `yaml:"status"`

	// Created is the ISO 8601 timestamp of when the plan was created.
	Created string `yaml:"created"`

	// Updated is the ISO 8601 timestamp of when the plan was last updated.
	Updated string `yaml:"updated"`

	// Dependencies lists the documentation paths this plan depends on,
	// each with a content hash at the time the plan was last updated.
	Dependencies []Dependency `yaml:"dependencies,omitempty"`

	// Steps lists the ordered steps in this plan.
	Steps []Step `yaml:"steps"`

	// CurrentStep is the ID of the step currently being worked on.
	CurrentStep int `yaml:"current_step"`
}

// Dependency tracks a documentation path and its content hash at the time
// the plan was last updated. Used for drift detection during plan resumption.
type Dependency struct {
	// Path is the documentation path relative to the docs root
	// (e.g. "foundation/architecture-principles", "topics/authentication/jwt-tokens").
	Path string `yaml:"path"`

	// Hash is the content hash in "sha256:<hex>" format, recorded at the time
	// the plan was last updated.
	Hash string `yaml:"hash"`
}

// Step represents a single step in a plan.
type Step struct {
	// ID is the numeric step identifier (1-based).
	ID int `yaml:"id"`

	// Title is the human-readable step description.
	Title string `yaml:"title"`

	// Status is the step status (completed, in-progress, pending).
	Status StepStatus `yaml:"status"`

	// CompletedAt is the ISO 8601 timestamp of when the step was completed.
	// Only set for completed steps.
	CompletedAt string `yaml:"completed_at,omitempty"`

	// StartedAt is the ISO 8601 timestamp of when work began on this step.
	// Only set for in-progress steps.
	StartedAt string `yaml:"started_at,omitempty"`

	// Notes contains free-form notes about the step's progress or findings.
	Notes string `yaml:"notes,omitempty"`

	// Queries lists the search terms the AI used to find relevant documentation
	// for this step. Preserved for re-search when hashes change.
	Queries []string `yaml:"queries,omitempty"`

	// Chunks lists the codectx generate calls made during this step.
	// Each entry is a comma-delimited string of chunk IDs (one generate call).
	// Directly replayable if dependency hashes haven't changed.
	Chunks []string `yaml:"chunks,omitempty"`

	// BlockedBy lists step IDs that must be completed before this step can start.
	BlockedBy []int `yaml:"blocked_by,omitempty"`
}

// CurrentStepEntry returns the step matching the plan's CurrentStep ID.
// Returns nil if no step matches or if the plan has no current step.
func (p *Plan) CurrentStepEntry() *Step {
	if p.CurrentStep <= 0 {
		return nil
	}
	for i := range p.Steps {
		if p.Steps[i].ID == p.CurrentStep {
			return &p.Steps[i]
		}
	}
	return nil
}

// Progress returns counts of completed, in-progress, and pending steps.
func (p *Plan) Progress() (completed, inProgress, pending int) {
	for _, s := range p.Steps {
		switch s.Status {
		case StepCompleted:
			completed++
		case StepInProgress:
			inProgress++
		case StepPending:
			pending++
		}
	}
	return completed, inProgress, pending
}

// BlockedSteps returns all steps that have a blocked_by field referencing
// steps that are not yet completed.
func (p *Plan) BlockedSteps() []Step {
	// Build a set of completed step IDs.
	completedIDs := make(map[int]bool, len(p.Steps))
	for _, s := range p.Steps {
		if s.Status == StepCompleted {
			completedIDs[s.ID] = true
		}
	}

	var blocked []Step
	for _, s := range p.Steps {
		if len(s.BlockedBy) == 0 {
			continue
		}
		for _, depID := range s.BlockedBy {
			if !completedIDs[depID] {
				blocked = append(blocked, s)
				break
			}
		}
	}
	return blocked
}

// StepByID returns the step with the given ID, or nil if not found.
func (p *Plan) StepByID(id int) *Step {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			return &p.Steps[i]
		}
	}
	return nil
}
