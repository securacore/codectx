package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheme_returnsNonNil(t *testing.T) {
	theme := Theme()
	require.NotNil(t, theme)
}

func TestTheme_focusedStyles(t *testing.T) {
	theme := Theme()

	// SelectedPrefix should contain the checkmark symbol.
	selectedPrefix := theme.Focused.SelectedPrefix.Render("")
	assert.Contains(t, selectedPrefix, SymbolDone)

	// UnselectedPrefix should contain the bullet symbol.
	unselectedPrefix := theme.Focused.UnselectedPrefix.Render("")
	assert.Contains(t, unselectedPrefix, SymbolBullet)
}

func TestTheme_blurredInheritsFocused(t *testing.T) {
	theme := Theme()

	// Blurred Title should match Focused Title (inherited via t.Blurred = t.Focused).
	focusedTitle := theme.Focused.Title.GetForeground()
	blurredTitle := theme.Blurred.Title.GetForeground()
	assert.Equal(t, focusedTitle, blurredTitle)
}

func TestTheme_groupStyles(t *testing.T) {
	theme := Theme()

	// Group Title and Description should match Focused counterparts.
	assert.Equal(t, theme.Focused.Title.GetForeground(), theme.Group.Title.GetForeground())
	assert.Equal(t, theme.Focused.Description.GetForeground(), theme.Group.Description.GetForeground())
}

func TestTheme_buttonStyles(t *testing.T) {
	theme := Theme()

	// Focused button should have a background color.
	focusedBg := theme.Focused.FocusedButton.GetBackground()
	assert.NotNil(t, focusedBg, "focused button should have a background color")

	// Blurred button should have a background color.
	blurredBg := theme.Focused.BlurredButton.GetBackground()
	assert.NotNil(t, blurredBg, "blurred button should have a background color")
}

func TestTheme_textInputStyles(t *testing.T) {
	theme := Theme()

	// Cursor style should have foreground set.
	cursorFg := theme.Focused.TextInput.Cursor.GetForeground()
	assert.NotNil(t, cursorFg, "cursor should have foreground color")

	// Prompt style should have foreground set.
	promptFg := theme.Focused.TextInput.Prompt.GetForeground()
	assert.NotNil(t, promptFg, "prompt should have foreground color")
}

func TestTheme_blurredIndicatorsEmpty(t *testing.T) {
	theme := Theme()

	// Blurred next/prev indicators are set to empty style.
	nextRendered := theme.Blurred.NextIndicator.Render("x")
	prevRendered := theme.Blurred.PrevIndicator.Render("x")
	// They should render as just "x" with no additional styling content,
	// but at minimum they should not crash.
	assert.Contains(t, nextRendered, "x")
	assert.Contains(t, prevRendered, "x")
}
