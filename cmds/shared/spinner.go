package shared

import (
	"os"

	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/x/term"
)

// RunWithSpinner wraps an action with a TUI spinner when running in an
// interactive terminal. In non-TTY environments (piped output, AI tool
// subprocesses, CI), the action is executed directly without the spinner
// to avoid bubbletea's "error opening TTY" crash.
//
// This is the single entry point for spinner usage across all commands.
// Commands should call this instead of spinner.New() directly.
func RunWithSpinner(title string, action func()) error {
	if !term.IsTerminal(os.Stdin.Fd()) {
		action()
		return nil
	}

	return spinner.New().
		Title(title).
		Action(action).
		Run()
}
