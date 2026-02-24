package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme returns a custom huh theme that matches the codectx color palette.
// Apply it to forms with: huh.NewForm(...).WithTheme(ui.Theme())
func Theme() *huh.Theme {
	t := huh.ThemeBase()

	normalFg := lipgloss.AdaptiveColor{Light: "235", Dark: "252"}
	dimFg := lipgloss.AdaptiveColor{Light: "247", Dark: "243"}
	borderFg := lipgloss.AdaptiveColor{Light: "250", Dark: "238"}

	// Focused field styles.
	t.Focused.Base = t.Focused.Base.BorderForeground(borderFg)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(Accent).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(Accent).Bold(true).MarginBottom(1)
	t.Focused.Description = t.Focused.Description.Foreground(dimFg)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(Red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(Red)

	// Select styles.
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(Accent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(Accent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(Accent)
	t.Focused.Option = t.Focused.Option.Foreground(normalFg)

	// Multi-select styles.
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(Accent)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(Green)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(Green).SetString(SymbolDone + " ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(normalFg)
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(dimFg).SetString(SymbolBullet + " ")

	// Button styles.
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Foreground(lipgloss.AdaptiveColor{Light: "255", Dark: "232"}).
		Background(Accent)
	t.Focused.BlurredButton = t.Focused.BlurredButton.
		Foreground(normalFg).
		Background(lipgloss.AdaptiveColor{Light: "252", Dark: "237"})

	// Text input styles.
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(Accent)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(dimFg)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(Accent)

	// Blurred field styles (inherit focused, hide border).
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	// Group styles.
	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}
