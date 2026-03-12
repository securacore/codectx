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

func TestRenderTemplate_ContainsStopDirective(t *testing.T) {
	content := renderTemplate("path/to/context.md")

	if !strings.Contains(content, "STOP") {
		t.Error("expected STOP directive in rendered template")
	}
}

func TestRenderTemplate_ContainsMarkdownLink(t *testing.T) {
	content := renderTemplate("docs/.codectx/compiled/context.md")

	if !strings.Contains(content, "[context](docs/.codectx/compiled/context.md)") {
		t.Error("expected markdown link to context.md")
	}
}

func TestRenderTemplate_ContainsCodectxHeading(t *testing.T) {
	content := renderTemplate("path/to/context.md")

	if !strings.Contains(content, "# codectx") {
		t.Error("expected codectx heading")
	}
}

func TestRenderTemplate_ForwardSlashPaths(t *testing.T) {
	// Verify that a path with forward slashes is preserved as-is.
	content := renderTemplate("docs/.codectx/compiled/context.md")

	if !strings.Contains(content, "docs/.codectx/compiled/context.md") {
		t.Error("expected forward-slash path to be preserved")
	}
}

func TestRenderTemplate_DoNotProceed(t *testing.T) {
	content := renderTemplate("path/to/context.md")

	if !strings.Contains(content, "Do not proceed") {
		t.Error("expected 'Do not proceed' directive in template")
	}
}
