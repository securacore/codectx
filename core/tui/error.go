package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// ErrorMsg represents a structured, actionable error message that answers
// three questions: what happened, why, and what the user should do next.
type ErrorMsg struct {
	// Title is the short error summary displayed next to the ✗ icon.
	Title string

	// Detail provides additional context explaining the error.
	// Each string is rendered as a separate line. Optional.
	Detail []string

	// Suggestions are actionable next steps the user can take.
	// Commands are rendered in accent color. Optional.
	Suggestions []Suggestion
}

// Suggestion is a single actionable next step with a description and
// an optional command example.
type Suggestion struct {
	// Text describes what the user should do.
	Text string

	// Command is the CLI command to run. Rendered in accent color.
	// Optional — some suggestions are descriptive without a specific command.
	Command string
}

// Render formats the error message for terminal display. The output follows
// a consistent pattern:
//
//	✗ Error: <title>
//
//	  <detail line 1>
//	  <detail line 2>
//
//	  <suggestion text>
//	    <command>
func (e ErrorMsg) Render() string {
	return renderMessage(ErrorIcon(), StyleError, "Error: "+e.Title, e.Detail, e.Suggestions)
}

// WarnMsg represents a non-fatal warning with the same structure as ErrorMsg.
type WarnMsg struct {
	// Title is the short warning summary displayed next to the ⚠ icon.
	Title string

	// Detail provides additional context. Optional.
	Detail []string
}

// Render formats the warning message for terminal display.
func (w WarnMsg) Render() string {
	return renderMessage(Warning(), StyleWarning, w.Title, w.Detail, nil)
}

// renderMessage is the shared rendering core for ErrorMsg and WarnMsg.
// It composes the icon, styled title, detail lines, and optional suggestions
// into a consistent terminal output format.
func renderMessage(icon string, titleStyle lipgloss.Style, title string, detail []string, suggestions []Suggestion) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s %s\n", icon, titleStyle.Render(title))

	if len(detail) > 0 {
		b.WriteString("\n")
		for _, line := range detail {
			fmt.Fprintf(&b, "%s%s\n", Indent(1), line)
		}
	}

	if len(suggestions) > 0 {
		b.WriteString("\n")
		for _, s := range suggestions {
			fmt.Fprintf(&b, "%s%s\n", Indent(1), s.Text)
			if s.Command != "" {
				fmt.Fprintf(&b, "%s%s\n", Indent(2), StyleCommand.Render(s.Command))
			}
		}
	}

	b.WriteString("\n")
	return b.String()
}
