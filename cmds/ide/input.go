package ide

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/securacore/codectx/ui"
)

// newTextArea creates a styled textarea for user input.
func newTextArea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Prompt = "  > "
	ta.CharLimit = 0 // No limit
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = ui.DimStyle
	ta.ShowLineNumbers = false
	ta.Focus()
	return ta
}

// inputUpdate handles textarea key events and returns whether the user
// pressed Enter to send a message (without shift held).
func inputUpdate(ta *textarea.Model, msg tea.KeyMsg) (string, bool) {
	switch msg.Type {
	case tea.KeyEnter:
		// Shift+Enter inserts a newline. Plain Enter sends.
		if msg.Alt {
			// Alt+Enter also inserts newline as fallback.
			break
		}
		text := ta.Value()
		if text == "" {
			return "", false
		}
		ta.Reset()
		return text, true
	}
	return "", false
}
