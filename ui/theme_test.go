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
