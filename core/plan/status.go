package plan

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/tui"
)

// FormatStatus renders the plan status output matching the spec format:
//
//	Plan: Authentication System Migration
//	Status: in-progress (step 3 of 5)
//	Progress: 2 steps completed, 1 in progress, 2 pending
//
//	Current step: Implement token service refactor
//	  Started: 2025-03-07T09:00:00Z
//	  Notes: User service and payment service updated. Order service remaining.
//	  Stored queries:
//	    - "token service refactor implementation"
//	    - "order service authentication"
//
//	Dependencies:
//	  ✓ foundation/architecture-principles — unchanged
//	  ⚠ topics/authentication/jwt-tokens — content changed since last update
//	  ✓ topics/authentication/oauth — unchanged
//
//	Blocked steps:
//	  Step 4 (Migration testing) — blocked by step 3
//	  Step 5 (Production rollout) — blocked by step 4
func FormatStatus(p *Plan, check *CheckResult) string {
	var b strings.Builder

	completed, inProgress, pending := p.Progress()

	// Header.
	writePlanHeader(&b, p)

	// Progress.
	fmt.Fprintf(&b, "Progress: %s\n", formatProgress(completed, inProgress, pending))

	// Current step details.
	if current := p.CurrentStepEntry(); current != nil {
		b.WriteString("\n")
		fmt.Fprintf(&b, "Current step: %s\n", current.Title)
		if current.StartedAt != "" {
			fmt.Fprintf(&b, "  Started: %s\n", current.StartedAt)
		}
		if current.Notes != "" {
			fmt.Fprintf(&b, "  Notes: %s\n", current.Notes)
		}
		if len(current.Queries) > 0 {
			b.WriteString("  Stored queries:\n")
			for _, q := range current.Queries {
				fmt.Fprintf(&b, "    - %q\n", q)
			}
		}
	}

	// Dependencies.
	if check != nil && len(check.Statuses) > 0 {
		b.WriteString("\nDependencies:\n")
		writeDependencyStatuses(&b, check.Statuses, "content changed since last update")
	}

	// Blocked steps.
	blocked := p.BlockedSteps()
	if len(blocked) > 0 {
		b.WriteString("\nBlocked steps:\n")
		for _, s := range blocked {
			blockerNames := formatBlockers(p, s.BlockedBy)
			fmt.Fprintf(&b, "  Step %d (%s) — blocked by %s\n", s.ID, s.Title, blockerNames)
		}
	}

	return b.String()
}

// FormatResumeMatch renders the output for plan resume when all hashes match.
//
//	Plan: Authentication System Migration
//	Status: in-progress (step 3 of 5)
//	Dependencies: all unchanged ✓
//
//	Replaying context for step 3...
//	Generated: /tmp/codectx/auth-migration-step3.1741532400.md (1,847 tokens)
//	Contains: obj:a1b2c3.04, obj:d4e5f6.02, obj:d4e5f6.03, obj:x9y8z7.01, spec:x9y8z7.01
//
//	Current step: Implement token service refactor
//	Notes: User service and payment service updated. Order service remaining.
func FormatResumeMatch(p *Plan, generateOutputs []string) string {
	var b strings.Builder

	writePlanHeader(&b, p)
	fmt.Fprintf(&b, "Dependencies: all unchanged %s\n", tui.Success())

	if len(generateOutputs) > 0 {
		fmt.Fprintf(&b, "\nReplaying context for step %d...\n", p.CurrentStep)
		for _, out := range generateOutputs {
			b.WriteString(out)
		}
	}

	if current := p.CurrentStepEntry(); current != nil {
		fmt.Fprintf(&b, "\nCurrent step: %s\n", current.Title)
		if current.Notes != "" {
			fmt.Fprintf(&b, "Notes: %s\n", current.Notes)
		}
	}

	return b.String()
}

// FormatResumeDrift renders the output for plan resume when hashes have changed.
//
//	Plan: Authentication System Migration
//	Status: in-progress (step 3 of 5)
//
//	Documentation changes since last update:
//	  ⚠ topics/authentication/jwt-tokens — content changed
//	  ✓ foundation/architecture-principles — unchanged
//	  ✓ topics/authentication/oauth — unchanged
//
//	Stored chunks may be stale. Stored queries for current step:
//	  - "token service refactor implementation"
//	  - "order service authentication"
//
//	Recommendation: Re-run stored queries to refresh context with updated documentation.
func FormatResumeDrift(p *Plan, check *CheckResult) string {
	var b strings.Builder

	writePlanHeader(&b, p)

	// Documentation changes.
	b.WriteString("\nDocumentation changes since last update:\n")
	writeDependencyStatuses(&b, check.Statuses, "content changed")

	// Stored queries.
	if current := p.CurrentStepEntry(); current != nil && len(current.Queries) > 0 {
		b.WriteString("\nStored chunks may be stale. Stored queries for current step:\n")
		for _, q := range current.Queries {
			fmt.Fprintf(&b, "  - %q\n", q)
		}
	}

	b.WriteString("\nRecommendation: Re-run stored queries to refresh context with updated documentation.\n")

	return b.String()
}

// writePlanHeader writes the common plan name and status line to the builder.
func writePlanHeader(b *strings.Builder, p *Plan) {
	total := len(p.Steps)
	fmt.Fprintf(b, "Plan: %s\n", p.Name)
	fmt.Fprintf(b, "Status: %s", p.Status)
	if p.CurrentStep > 0 {
		fmt.Fprintf(b, " (step %d of %d)", p.CurrentStep, total)
	}
	b.WriteString("\n")
}

// writeDependencyStatuses writes dependency status lines to the builder.
// The changedSuffix parameter controls the text shown for changed dependencies
// (e.g. "content changed" or "content changed since last update").
func writeDependencyStatuses(b *strings.Builder, statuses []DependencyStatus, changedSuffix string) {
	for _, ds := range statuses {
		if ds.Changed {
			suffix := changedSuffix
			if ds.Missing {
				suffix = "not found in compiled output"
			}
			fmt.Fprintf(b, "  %s %s — %s\n", tui.Warning(), ds.Dependency.Path, suffix)
		} else {
			fmt.Fprintf(b, "  %s %s — unchanged\n", tui.Success(), ds.Dependency.Path)
		}
	}
}

// formatProgress builds a human-readable progress summary.
func formatProgress(completed, inProgress, pending int) string {
	var parts []string
	if completed > 0 {
		parts = append(parts, fmt.Sprintf("%d %s completed", completed, pluralize("step", completed)))
	}
	if inProgress > 0 {
		parts = append(parts, fmt.Sprintf("%d in progress", inProgress))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if len(parts) == 0 {
		return "no steps"
	}
	return strings.Join(parts, ", ")
}

// formatBlockers renders blocked_by step IDs as "step N" references.
func formatBlockers(p *Plan, blockedBy []int) string {
	names := make([]string, 0, len(blockedBy))
	for _, id := range blockedBy {
		if s := p.StepByID(id); s != nil {
			names = append(names, fmt.Sprintf("step %d", id))
		} else {
			names = append(names, fmt.Sprintf("step %d (unknown)", id))
		}
	}
	return strings.Join(names, ", ")
}

// pluralize returns singular or plural form based on count.
func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}
