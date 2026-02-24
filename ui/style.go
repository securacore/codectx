// Package ui provides styled terminal output for the codectx CLI.
// All user-facing output should use this package instead of raw fmt.Print.
package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// Color palette — adaptive for light and dark terminal backgrounds.
// Exported for use by packages that build custom TUI components (e.g.,
// bubbletea models) and need visual consistency with the rest of the CLI.
var (
	Green  = lipgloss.AdaptiveColor{Light: "34", Dark: "78"}
	Yellow = lipgloss.AdaptiveColor{Light: "172", Dark: "220"}
	Red    = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}
	Dim    = lipgloss.AdaptiveColor{Light: "247", Dark: "243"}
	Accent = lipgloss.AdaptiveColor{Light: "30", Dark: "87"}
)

// Styles built from the palette. Exported for use by bubbletea TUI
// components that need visual consistency with CLI output. The output
// functions in this package use these same styles internally.
var (
	GreenStyle  = lipgloss.NewStyle().Foreground(Green)
	YellowStyle = lipgloss.NewStyle().Foreground(Yellow)
	RedStyle    = lipgloss.NewStyle().Foreground(Red)
	DimStyle    = lipgloss.NewStyle().Foreground(Dim)
	BoldStyle   = lipgloss.NewStyle().Bold(true)
	AccentStyle = lipgloss.NewStyle().Foreground(Accent)
)

// Unicode symbols used throughout the CLI.
const (
	SymbolDone    = "\u2713" // checkmark
	SymbolFail    = "\u2717" // ballot x
	SymbolWarn    = "!"
	SymbolBullet  = "\u2022" // bullet
	SymbolSpinner = "\u25CB" // circle (fallback for non-TTY)

	// Activation state indicators for TUI views.
	SymbolActive   = "\u25CF" // ● filled circle  — all entries active
	SymbolPartial  = "\u25D0" // ◐ half circle    — some entries active
	SymbolInactive = "\u25CB" // ○ empty circle   — no entries active
)

// IsTTY reports whether stdout is connected to a terminal.
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}
