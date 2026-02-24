package ui

import (
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Spin runs fn while displaying an animated spinner with the given message.
// In non-TTY mode (piped output), prints the message once without animation.
// The spinner is cleared from the terminal when fn completes.
func Spin(msg string, fn func()) {
	if !IsTTY() {
		Step(msg)
		fn()
		return
	}

	_ = spinner.New().
		Title(msg).
		TitleStyle(lipgloss.NewStyle()).
		Style(dimStyle).
		Type(spinner.MiniDot).
		Action(fn).
		Run()
}

// SpinErr runs fn while displaying a spinner, returning fn's error.
// This is a convenience wrapper for actions that return an error.
func SpinErr(msg string, fn func() error) error {
	var err error
	Spin(msg, func() { err = fn() })
	return err
}
