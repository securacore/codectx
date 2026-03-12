package tui_test

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/tui"
)

func TestIcons_AreNotEmpty(t *testing.T) {
	icons := map[string]string{
		"IconSuccess": tui.IconSuccess,
		"IconWarning": tui.IconWarning,
		"IconError":   tui.IconError,
		"IconArrow":   tui.IconArrow,
		"IconIndent":  tui.IconIndent,
	}
	for name, icon := range icons {
		if icon == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}

func TestTreeCharacters_AreNotEmpty(t *testing.T) {
	chars := map[string]string{
		"TreePipe":   tui.TreePipe,
		"TreeBranch": tui.TreeBranch,
		"TreeCorner": tui.TreeCorner,
		"TreeSpace":  tui.TreeSpace,
	}
	for name, ch := range chars {
		if ch == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}

func TestSuccess_RendersNonEmpty(t *testing.T) {
	result := tui.Success()
	if result == "" {
		t.Error("Success() should render non-empty string")
	}
}

func TestWarning_RendersNonEmpty(t *testing.T) {
	result := tui.Warning()
	if result == "" {
		t.Error("Warning() should render non-empty string")
	}
}

func TestErrorIcon_RendersNonEmpty(t *testing.T) {
	result := tui.ErrorIcon()
	if result == "" {
		t.Error("ErrorIcon() should render non-empty string")
	}
}

func TestArrow_RendersNonEmpty(t *testing.T) {
	result := tui.Arrow()
	if result == "" {
		t.Error("Arrow() should render non-empty string")
	}
}

func TestArrow_ContainsIcon(t *testing.T) {
	result := tui.Arrow()
	if !strings.Contains(result, tui.IconArrow) {
		t.Errorf("Arrow() should contain IconArrow %q, got %q", tui.IconArrow, result)
	}
}

func TestIndent_ProducesCorrectLength(t *testing.T) {
	tests := []struct {
		level    int
		expected int
	}{
		{0, 0},
		{1, 2},
		{2, 4},
		{3, 6},
	}

	for _, tt := range tests {
		result := tui.Indent(tt.level)
		if len(result) != tt.expected {
			t.Errorf("Indent(%d): expected length %d, got %d (%q)",
				tt.level, tt.expected, len(result), result)
		}
	}
}

func TestIndent_UsesSpaces(t *testing.T) {
	result := tui.Indent(3)
	trimmed := strings.TrimLeft(result, " ")
	if trimmed != "" {
		t.Errorf("Indent should use only spaces, got %q", result)
	}
}

func TestIndent_NegativeReturnsEmpty(t *testing.T) {
	result := tui.Indent(-1)
	if result != "" {
		t.Errorf("Indent(-1) should return empty string, got %q", result)
	}
}

func TestIndent_ZeroReturnsEmpty(t *testing.T) {
	result := tui.Indent(0)
	if result != "" {
		t.Errorf("Indent(0) should return empty string, got %q", result)
	}
}
