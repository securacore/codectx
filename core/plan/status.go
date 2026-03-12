package plan

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/tui"
)

// FormatStatus renders the plan status output with full TUI styling:
//
//	-> Plan: Authentication System Migration
//	   Status: in-progress (step 3 of 5)
//	   Progress: 2 steps completed, 1 in progress, 2 pending
//
//	   Current step: Implement token service refactor
//	     Started: 2025-03-07T09:00:00Z
//	     Notes: User service and payment service updated. Order service remaining.
//	     Stored queries:
//	       - "token service refactor implementation"
//	       - "order service authentication"
//
//	   Dependencies:
//	     ✓ foundation/architecture-principles — unchanged
//	     ⚠ topics/authentication/jwt-tokens — content changed since last update
//	     ✓ topics/authentication/oauth — unchanged
//
//	   Blocked steps:
//	     Step 4 (Migration testing) — blocked by step 3
//	     Step 5 (Production rollout) — blocked by step 4
func FormatStatus(p *Plan, check *CheckResult) string {
	var b strings.Builder

	completed, inProgress, pending := p.Progress()

	// Header.
	writePlanHeader(&b, p)

	// Progress.
	fmt.Fprintf(&b, "%s%s\n",
		tui.Indent(1),
		tui.KeyValue("Progress", formatProgress(completed, inProgress, pending)),
	)

	// Current step details.
	if current := p.CurrentStepEntry(); current != nil {
		fmt.Fprintf(&b, "\n%s%s %s\n",
			tui.Indent(1),
			tui.StyleMuted.Render("Current step:"),
			tui.StyleBold.Render(current.Title),
		)
		if current.StartedAt != "" {
			fmt.Fprintf(&b, "%s%s\n", tui.Indent(2), tui.KeyValue("Started", current.StartedAt))
		}
		if current.Notes != "" {
			fmt.Fprintf(&b, "%s%s\n", tui.Indent(2), tui.KeyValue("Notes", current.Notes))
		}
		if len(current.Queries) > 0 {
			fmt.Fprintf(&b, "%s%s\n", tui.Indent(2), tui.StyleMuted.Render("Stored queries:"))
			for _, q := range current.Queries {
				fmt.Fprintf(&b, "%s%s %q\n",
					tui.Indent(3),
					tui.StyleMuted.Render("-"),
					q,
				)
			}
		}
	}

	// Dependencies.
	if check != nil && len(check.Statuses) > 0 {
		fmt.Fprintf(&b, "\n%s%s\n", tui.Indent(1), tui.StyleMuted.Render("Dependencies:"))
		writeDependencyStatuses(&b, check.Statuses, "content changed since last update")
	}

	// Blocked steps.
	blocked := p.BlockedSteps()
	if len(blocked) > 0 {
		fmt.Fprintf(&b, "\n%s%s\n", tui.Indent(1), tui.StyleMuted.Render("Blocked steps:"))
		for _, s := range blocked {
			blockerNames := formatBlockers(p, s.BlockedBy)
			fmt.Fprintf(&b, "%sStep %d (%s) \u2014 %s\n",
				tui.Indent(2),
				s.ID,
				tui.StyleBold.Render(s.Title),
				tui.StyleWarning.Render("blocked by "+blockerNames),
			)
		}
	}

	b.WriteString("\n")
	return b.String()
}

// FormatResumeMatch renders the output for plan resume when all hashes match.
//
//	-> Plan: Authentication System Migration
//	   Status: in-progress (step 3 of 5)
//	   Dependencies: all unchanged ✓
//
//	   Replaying context for step 3...
//	   Generated: /tmp/codectx/auth-migration-step3.1741532400.md (1,847 tokens)
//	   Contains: obj:a1b2c3.04, obj:d4e5f6.02, ...
//
//	   Current step: Implement token service refactor
//	   Notes: User service and payment service updated. Order service remaining.
func FormatResumeMatch(p *Plan, generateOutputs []string) string {
	var b strings.Builder

	writePlanHeader(&b, p)
	fmt.Fprintf(&b, "%s%s %s\n",
		tui.Indent(1),
		tui.KeyValue("Dependencies", "all unchanged"),
		tui.Success(),
	)

	if len(generateOutputs) > 0 {
		fmt.Fprintf(&b, "\n%s%s\n",
			tui.Indent(1),
			tui.StyleMuted.Render(fmt.Sprintf("Replaying context for step %d...", p.CurrentStep)),
		)
		for _, out := range generateOutputs {
			b.WriteString(out)
		}
	}

	if current := p.CurrentStepEntry(); current != nil {
		fmt.Fprintf(&b, "\n%s%s %s\n",
			tui.Indent(1),
			tui.StyleMuted.Render("Current step:"),
			tui.StyleBold.Render(current.Title),
		)
		if current.Notes != "" {
			fmt.Fprintf(&b, "%s%s\n", tui.Indent(2), tui.KeyValue("Notes", current.Notes))
		}
	}

	b.WriteString("\n")
	return b.String()
}

// FormatResumeDrift renders the output for plan resume when hashes have changed.
//
//	-> Plan: Authentication System Migration
//	   Status: in-progress (step 3 of 5)
//
//	   Documentation changes since last update:
//	     ⚠ topics/authentication/jwt-tokens — content changed
//	     ✓ foundation/architecture-principles — unchanged
//	     ✓ topics/authentication/oauth — unchanged
//
//	   Stored chunks may be stale. Stored queries for current step:
//	     - "token service refactor implementation"
//	     - "order service authentication"
//
//	   Recommendation: Re-run stored queries to refresh context with updated documentation.
func FormatResumeDrift(p *Plan, check *CheckResult) string {
	var b strings.Builder

	writePlanHeader(&b, p)

	// Documentation changes.
	fmt.Fprintf(&b, "\n%s%s\n",
		tui.Indent(1),
		tui.StyleWarning.Render("Documentation changes since last update:"),
	)
	writeDependencyStatuses(&b, check.Statuses, "content changed")

	// Stored queries.
	if current := p.CurrentStepEntry(); current != nil && len(current.Queries) > 0 {
		fmt.Fprintf(&b, "\n%s%s\n",
			tui.Indent(1),
			tui.StyleWarning.Render("Stored chunks may be stale. Stored queries for current step:"),
		)
		for _, q := range current.Queries {
			fmt.Fprintf(&b, "%s%s %q\n",
				tui.Indent(2),
				tui.StyleMuted.Render("-"),
				q,
			)
		}
	}

	fmt.Fprintf(&b, "\n%s%s %s\n\n",
		tui.Indent(1),
		tui.StyleBold.Render("Recommendation:"),
		"Re-run stored queries to refresh context with updated documentation.",
	)

	return b.String()
}

// writePlanHeader writes the styled plan name and status line to the builder.
func writePlanHeader(b *strings.Builder, p *Plan) {
	total := len(p.Steps)

	fmt.Fprintf(b, "\n%s %s\n",
		tui.Arrow(),
		tui.KeyValue("Plan", tui.StyleBold.Render(p.Name)),
	)

	statusText := string(p.Status)
	if p.CurrentStep > 0 {
		statusText += fmt.Sprintf(" (step %d of %d)", p.CurrentStep, total)
	}
	fmt.Fprintf(b, "%s%s\n",
		tui.Indent(1),
		tui.KeyValue("Status", statusText),
	)
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
			fmt.Fprintf(b, "%s%s %s \u2014 %s\n",
				tui.Indent(2),
				tui.Warning(),
				tui.StylePath.Render(ds.Dependency.Path),
				tui.StyleWarning.Render(suffix),
			)
		} else {
			fmt.Fprintf(b, "%s%s %s \u2014 %s\n",
				tui.Indent(2),
				tui.Success(),
				tui.StylePath.Render(ds.Dependency.Path),
				tui.StyleMuted.Render("unchanged"),
			)
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
