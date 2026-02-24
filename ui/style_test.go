package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Symbol constants ---

func TestSymbolConstants_nonEmpty(t *testing.T) {
	symbols := map[string]string{
		"SymbolDone":     SymbolDone,
		"SymbolFail":     SymbolFail,
		"SymbolWarn":     SymbolWarn,
		"SymbolBullet":   SymbolBullet,
		"SymbolSpinner":  SymbolSpinner,
		"SymbolActive":   SymbolActive,
		"SymbolPartial":  SymbolPartial,
		"SymbolInactive": SymbolInactive,
	}
	for name, val := range symbols {
		assert.NotEmpty(t, val, "%s should not be empty", name)
	}
}

func TestActivationSymbols_distinct(t *testing.T) {
	assert.NotEqual(t, SymbolActive, SymbolPartial)
	assert.NotEqual(t, SymbolActive, SymbolInactive)
	assert.NotEqual(t, SymbolPartial, SymbolInactive)
}

// --- Exported styles render without panic ---

func TestGreenStyle_renders(t *testing.T) {
	result := GreenStyle.Render("test")
	assert.Contains(t, result, "test")
}

func TestYellowStyle_renders(t *testing.T) {
	result := YellowStyle.Render("test")
	assert.Contains(t, result, "test")
}

func TestRedStyle_renders(t *testing.T) {
	result := RedStyle.Render("test")
	assert.Contains(t, result, "test")
}

func TestDimStyle_renders(t *testing.T) {
	result := DimStyle.Render("test")
	assert.Contains(t, result, "test")
}

func TestBoldStyle_renders(t *testing.T) {
	result := BoldStyle.Render("test")
	assert.Contains(t, result, "test")
}

func TestAccentStyle_renders(t *testing.T) {
	result := AccentStyle.Render("test")
	assert.Contains(t, result, "test")
}

// --- Adaptive colors are defined ---

func TestAdaptiveColors_defined(t *testing.T) {
	colors := map[string]struct{ Light, Dark string }{
		"Green":  {Green.Light, Green.Dark},
		"Yellow": {Yellow.Light, Yellow.Dark},
		"Red":    {Red.Light, Red.Dark},
		"Dim":    {Dim.Light, Dim.Dark},
		"Accent": {Accent.Light, Accent.Dark},
	}
	for name, c := range colors {
		assert.NotEmpty(t, c.Light, "%s.Light should not be empty", name)
		assert.NotEmpty(t, c.Dark, "%s.Dark should not be empty", name)
	}
}
