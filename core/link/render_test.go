package link

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// renderTemplate (internal)
// ---------------------------------------------------------------------------

func TestRenderTemplate_ContainsContextPath(t *testing.T) {
	content := renderTemplate("docs/.codectx/compiled/context.md")

	if !strings.Contains(content, "docs/.codectx/compiled/context.md") {
		t.Error("expected context path in rendered template")
	}
}

func TestRenderTemplate_ContainsCodectxMarker(t *testing.T) {
	content := renderTemplate("path/to/context.md")

	if !strings.Contains(content, codectxMarker) {
		t.Error("expected codectx marker in rendered template")
	}
}

func TestRenderTemplate_ContainsQueryCommand(t *testing.T) {
	content := renderTemplate("path/to/context.md")

	if !strings.Contains(content, "codectx query") {
		t.Error("expected query command in rendered template")
	}
}

func TestRenderTemplate_ContainsGenerateCommand(t *testing.T) {
	content := renderTemplate("path/to/context.md")

	if !strings.Contains(content, "codectx generate") {
		t.Error("expected generate command in rendered template")
	}
}

func TestRenderTemplate_ContainsProjectInstructionsHeading(t *testing.T) {
	content := renderTemplate("path/to/context.md")

	if !strings.Contains(content, "# Project Instructions") {
		t.Error("expected Project Instructions heading")
	}
}

func TestRenderTemplate_ForwardSlashPaths(t *testing.T) {
	// Verify that a path with forward slashes is preserved as-is.
	content := renderTemplate("docs/.codectx/compiled/context.md")

	if !strings.Contains(content, "docs/.codectx/compiled/context.md") {
		t.Error("expected forward-slash path to be preserved")
	}
}
