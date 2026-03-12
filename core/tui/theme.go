// Package tui provides the shared visual theme, output formatting, and error
// display patterns used by all codectx CLI commands. Every command is a consumer
// of this package — colors, icons, and styles are defined here once and used
// everywhere for visual consistency.
//
// Interactive components (prompts, selects, spinners) use charmbracelet/huh
// directly, configured with styles from this package. This package does NOT
// wrap huh — it provides the theme that huh components are styled with.
package tui

import (
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

// Icons used throughout the CLI for status indication.
const (
	IconSuccess = "✓"
	IconWarning = "⚠"
	IconError   = "✗"
	IconArrow   = "->"
	IconIndent  = "  "
)

// Tree drawing characters for directory structure output.
const (
	TreePipe   = "│   "
	TreeBranch = "├── "
	TreeCorner = "└── "
	TreeSpace  = "    "
)

// hasDark caches the dark background detection result. lipgloss detects
// the terminal's background color to select appropriate colors.
var hasDark = lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

// lightDark returns the appropriate color based on terminal background.
var lightDark = lipgloss.LightDark(hasDark)

// --- Color Palette ---
// All colors adapt to light/dark terminal backgrounds.

// ColorSuccess is used for completed operations, checkmarks, and positive outcomes.
var ColorSuccess = lightDark(lipgloss.Color("#16A34A"), lipgloss.Color("#4ADE80"))

// ColorWarning is used for drift detection, budget alerts, and caution states.
var ColorWarning = lightDark(lipgloss.Color("#CA8A04"), lipgloss.Color("#FACC15"))

// ColorError is used for failed operations, validation errors, and critical states.
var ColorError = lightDark(lipgloss.Color("#DC2626"), lipgloss.Color("#F87171"))

// ColorMuted is used for secondary information, timestamps, and pending states.
var ColorMuted = lightDark(lipgloss.Color("#6B7280"), lipgloss.Color("#9CA3AF"))

// ColorAccent is used for highlighted terms, file paths, chunk IDs, and commands.
var ColorAccent = lightDark(lipgloss.Color("#0891B2"), lipgloss.Color("#22D3EE"))

// --- Styles ---
// Pre-built lipgloss styles for common output patterns.

// StyleWarning renders text in the warning color (yellow).
var StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)

// StyleError renders text in the error color (red).
var StyleError = lipgloss.NewStyle().Foreground(ColorError)

// StyleMuted renders text in the muted color (gray).
var StyleMuted = lipgloss.NewStyle().Foreground(ColorMuted)

// StyleAccent renders text in the accent color (cyan).
var StyleAccent = lipgloss.NewStyle().Foreground(ColorAccent)

// StyleBold renders text in bold.
var StyleBold = lipgloss.NewStyle().Bold(true)

// StyleErrorIcon renders the error icon (✗) in error color, bold.
var StyleErrorIcon = lipgloss.NewStyle().Foreground(ColorError).Bold(true)

// StyleSuccessIcon renders the success icon (✓) in success color, bold.
var StyleSuccessIcon = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)

// StyleWarningIcon renders the warning icon (⚠) in warning color, bold.
var StyleWarningIcon = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)

// StyleCommand renders inline command references — accent colored for visibility.
var StyleCommand = lipgloss.NewStyle().Foreground(ColorAccent)

// StylePath renders file paths — accent colored for visibility.
var StylePath = StyleAccent

// --- Rendered Strings ---
// Pre-rendered icon strings for direct use in fmt output.

// Success renders the ✓ icon in success style.
func Success() string { return StyleSuccessIcon.Render(IconSuccess) }

// Warning renders the ⚠ icon in warning style.
func Warning() string { return StyleWarningIcon.Render(IconWarning) }

// Error() would shadow the builtin — use ErrorIcon instead.

// ErrorIcon renders the ✗ icon in error style.
func ErrorIcon() string { return StyleErrorIcon.Render(IconError) }

// Arrow renders the -> icon in accent style for progress/status indication.
func Arrow() string { return StyleAccent.Render(IconArrow) }

// --- Indentation Helpers ---

// Indent returns a string with n levels of indentation (2 spaces each).
// Returns an empty string for zero or negative values.
func Indent(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(IconIndent, n)
}
