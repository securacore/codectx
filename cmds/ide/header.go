package ide

import (
	"fmt"
	"strings"

	coreide "github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/ui"
)

// renderHeader produces the two-line header bar showing session context.
func renderHeader(providerID, category, docID, target string, phase coreide.Phase, width int) string {
	left := ui.BoldStyle.Render("codectx ide")
	if providerID != "" {
		left += "  " + ui.DimStyle.Render("provider:"+providerID)
	}

	var right string
	if category != "" {
		right = ui.AccentStyle.Render(category)
		if docID != "" {
			right += ui.DimStyle.Render(" . ") + ui.BoldStyle.Render(docID)
		}
	}

	line1 := padLine(left, right, width)

	phaseStr := ui.DimStyle.Render("phase:") + ui.AccentStyle.Render(phase.String())
	var targetStr string
	if target != "" {
		targetStr = ui.DimStyle.Render(target)
	}

	line2 := padLine("  "+phaseStr, targetStr, width)

	return fmt.Sprintf("  %s\n%s", line1, line2)
}

// padLine places left and right strings on a single line, padding with spaces.
func padLine(left, right string, width int) string {
	// Use raw lengths for padding calculation (strip ANSI).
	leftLen := visualLen(left)
	rightLen := visualLen(right)

	if right == "" {
		return left
	}

	gap := width - leftLen - rightLen
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

// visualLen approximates the visible length of a styled string by stripping
// ANSI escape sequences. This is a simple approximation.
func visualLen(s string) int {
	inEsc := false
	count := 0
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		count++
	}
	return count
}
