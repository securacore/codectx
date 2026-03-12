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
// Unexported: consumers use the Style* variables or rendered icon functions.

var colorSuccess = lightDark(lipgloss.Color("#16A34A"), lipgloss.Color("#4ADE80"))
var colorWarning = lightDark(lipgloss.Color("#CA8A04"), lipgloss.Color("#FACC15"))
var colorError = lightDark(lipgloss.Color("#DC2626"), lipgloss.Color("#F87171"))
var colorMuted = lightDark(lipgloss.Color("#6B7280"), lipgloss.Color("#9CA3AF"))
var colorAccent = lightDark(lipgloss.Color("#0891B2"), lipgloss.Color("#22D3EE"))

// --- Styles ---
// Pre-built lipgloss styles for common output patterns.

// StyleWarning renders text in the warning color (yellow).
var StyleWarning = lipgloss.NewStyle().Foreground(colorWarning)

// styleError renders text in the error color (red). Unexported: only used
// by renderMessage internally.
var styleError = lipgloss.NewStyle().Foreground(colorError)

// StyleMuted renders text in the muted color (gray).
var StyleMuted = lipgloss.NewStyle().Foreground(colorMuted)

// StyleAccent renders text in the accent color (cyan).
var StyleAccent = lipgloss.NewStyle().Foreground(colorAccent)

// StyleBold renders text in bold.
var StyleBold = lipgloss.NewStyle().Bold(true)

// styleErrorIcon renders the error icon in error color, bold. Unexported:
// only consumed by ErrorIcon().
var styleErrorIcon = lipgloss.NewStyle().Foreground(colorError).Bold(true)

// styleSuccessIcon renders the success icon in success color, bold. Unexported:
// only consumed by Success().
var styleSuccessIcon = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)

// styleWarningIcon renders the warning icon in warning color, bold. Unexported:
// only consumed by Warning().
var styleWarningIcon = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)

// StyleCommand renders inline command references — accent colored for visibility.
var StyleCommand = lipgloss.NewStyle().Foreground(colorAccent)

// StylePath renders file paths — accent colored for visibility.
var StylePath = StyleAccent

// --- Rendered Strings ---
// Pre-rendered icon strings for direct use in fmt output.

// Success renders the ✓ icon in success style.
func Success() string { return styleSuccessIcon.Render(IconSuccess) }

// Warning renders the ⚠ icon in warning style.
func Warning() string { return styleWarningIcon.Render(IconWarning) }

// Error() would shadow the builtin — use ErrorIcon instead.

// ErrorIcon renders the ✗ icon in error style.
func ErrorIcon() string { return styleErrorIcon.Render(IconError) }

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
