package tui_test

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/tui"
)

func TestErrorMsg_Render_TitleOnly(t *testing.T) {
	msg := tui.ErrorMsg{
		Title: "something went wrong",
	}

	result := msg.Render()

	if !strings.Contains(result, "something went wrong") {
		t.Error("expected error title in output")
	}
	if !strings.Contains(result, "Error:") {
		t.Error("expected 'Error:' prefix in output")
	}
}

func TestErrorMsg_Render_WithDetail(t *testing.T) {
	msg := tui.ErrorMsg{
		Title:  "file not found",
		Detail: []string{"The file codectx.yml does not exist.", "Check your current directory."},
	}

	result := msg.Render()

	if !strings.Contains(result, "codectx.yml does not exist") {
		t.Error("expected detail line 1 in output")
	}
	if !strings.Contains(result, "Check your current directory") {
		t.Error("expected detail line 2 in output")
	}
}

func TestErrorMsg_Render_WithSuggestions(t *testing.T) {
	msg := tui.ErrorMsg{
		Title: "not initialized",
		Suggestions: []tui.Suggestion{
			{Text: "Initialize a new project:", Command: "codectx init"},
			{Text: "Or create in a new directory:", Command: "codectx init my-project"},
		},
	}

	result := msg.Render()

	if !strings.Contains(result, "Initialize a new project") {
		t.Error("expected suggestion text in output")
	}
	if !strings.Contains(result, "codectx init") {
		t.Error("expected suggestion command in output")
	}
	if !strings.Contains(result, "codectx init my-project") {
		t.Error("expected second suggestion command in output")
	}
}

func TestErrorMsg_Render_SuggestionWithoutCommand(t *testing.T) {
	msg := tui.ErrorMsg{
		Title: "test error",
		Suggestions: []tui.Suggestion{
			{Text: "Check your network connection."},
		},
	}

	result := msg.Render()

	if !strings.Contains(result, "Check your network connection") {
		t.Error("expected suggestion text without command")
	}
}

func TestErrorMsg_Render_FullError(t *testing.T) {
	msg := tui.ErrorMsg{
		Title:  "codectx.yml already exists in this directory",
		Detail: []string{"This project has already been initialized."},
		Suggestions: []tui.Suggestion{
			{Text: "To reinitialize, remove the existing config first:", Command: "rm codectx.yml"},
			{Text: "To compile the existing project:", Command: "codectx compile"},
		},
	}

	result := msg.Render()

	// Verify all sections are present.
	if !strings.Contains(result, "already exists") {
		t.Error("expected title")
	}
	if !strings.Contains(result, "already been initialized") {
		t.Error("expected detail")
	}
	if !strings.Contains(result, "rm codectx.yml") {
		t.Error("expected first suggestion command")
	}
	if !strings.Contains(result, "codectx compile") {
		t.Error("expected second suggestion command")
	}
}

func TestWarnMsg_Render_TitleOnly(t *testing.T) {
	msg := tui.WarnMsg{
		Title: "something might be wrong",
	}

	result := msg.Render()

	if !strings.Contains(result, "something might be wrong") {
		t.Error("expected warning title in output")
	}
}

func TestProjectNotFoundError_ContainsExpectedContent(t *testing.T) {
	result := tui.ProjectNotFoundError()

	if !strings.Contains(result, "No codectx project found") {
		t.Error("expected title in output")
	}
	if !strings.Contains(result, "codectx.yml") {
		t.Error("expected codectx.yml mention in output")
	}
	if !strings.Contains(result, "codectx init") {
		t.Error("expected init suggestion in output")
	}
}

func TestWarnMsg_Render_WithDetail(t *testing.T) {
	msg := tui.WarnMsg{
		Title:  "nested project detected",
		Detail: []string{"A codectx project exists in a parent directory.", "Continuing will create a nested project."},
	}

	result := msg.Render()

	if !strings.Contains(result, "nested project") {
		t.Error("expected warning title")
	}
	if !strings.Contains(result, "parent directory") {
		t.Error("expected detail line 1")
	}
	if !strings.Contains(result, "nested project") {
		t.Error("expected detail line 2")
	}
}

func TestWarnMsg_Render_WithSuggestions(t *testing.T) {
	msg := tui.WarnMsg{
		Title: "session context exceeds budget",
		Suggestions: []tui.Suggestion{
			{Text: "Remove an entry:", Command: "codectx session rm <ref>"},
			{Text: "Or increase the budget in codectx.yml"},
		},
	}

	result := msg.Render()

	if !strings.Contains(result, "session context exceeds budget") {
		t.Error("expected warning title in output")
	}
	if !strings.Contains(result, "Remove an entry") {
		t.Error("expected first suggestion text")
	}
	if !strings.Contains(result, "codectx session rm <ref>") {
		t.Error("expected first suggestion command")
	}
	if !strings.Contains(result, "increase the budget") {
		t.Error("expected second suggestion text (no command)")
	}
}

func TestWarnMsg_Render_Full(t *testing.T) {
	msg := tui.WarnMsg{
		Title:  "3 validation warning(s)",
		Detail: []string{"Missing description in api.md", "Empty section in setup.md"},
		Suggestions: []tui.Suggestion{
			{Text: "Fix the warnings and recompile:", Command: "codectx compile"},
		},
	}

	result := msg.Render()

	if !strings.Contains(result, "3 validation warning(s)") {
		t.Error("expected title")
	}
	if !strings.Contains(result, "Missing description") {
		t.Error("expected detail line 1")
	}
	if !strings.Contains(result, "Empty section") {
		t.Error("expected detail line 2")
	}
	if !strings.Contains(result, "Fix the warnings") {
		t.Error("expected suggestion text")
	}
	if !strings.Contains(result, "codectx compile") {
		t.Error("expected suggestion command")
	}
}
