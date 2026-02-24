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

// Styles built from the palette (package-private; used by output functions).
var (
	greenStyle  = lipgloss.NewStyle().Foreground(Green)
	yellowStyle = lipgloss.NewStyle().Foreground(Yellow)
	redStyle    = lipgloss.NewStyle().Foreground(Red)
	dimStyle    = lipgloss.NewStyle().Foreground(Dim)
	boldStyle   = lipgloss.NewStyle().Bold(true)
)

// Unicode symbols used throughout the CLI.
const (
	SymbolDone    = "\u2713" // checkmark
	SymbolFail    = "\u2717" // ballot x
	SymbolWarn    = "!"
	SymbolBullet  = "\u2022" // bullet
	SymbolSpinner = "\u25CB" // circle (fallback for non-TTY)
)

// IsTTY reports whether stdout is connected to a terminal.
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}
