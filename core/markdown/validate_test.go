package markdown

import (
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateFile
// ---------------------------------------------------------------------------

func TestValidateFile_WithHeadings(t *testing.T) {
	doc := Parse([]byte("# Title\n\nSome content.\n"))
	result := ValidateFile(doc, true)

	if !result.OK() {
		t.Error("expected OK for file with headings")
	}
	if result.HasWarnings() {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestValidateFile_NoHeadings_Required(t *testing.T) {
	doc := Parse([]byte("Just text without headings.\n"))
	result := ValidateFile(doc, true)

	if !result.OK() {
		t.Error("no-heading should be a warning, not an error")
	}
	if !result.HasWarnings() {
		t.Error("expected warning for file without headings")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestValidateFile_NoHeadings_NotRequired(t *testing.T) {
	doc := Parse([]byte("Just text without headings.\n"))
	result := ValidateFile(doc, false)

	if result.HasWarnings() {
		t.Error("should not warn when headings are not required")
	}
}

func TestValidateFile_EmptyDocument(t *testing.T) {
	doc := Parse([]byte(""))
	result := ValidateFile(doc, true)

	if !result.HasWarnings() {
		t.Error("expected warning for empty document with require_headings=true")
	}
}

// ---------------------------------------------------------------------------
// ValidationResult methods
// ---------------------------------------------------------------------------

func TestValidationResult_OK(t *testing.T) {
	r := &ValidationResult{}
	if !r.OK() {
		t.Error("empty result should be OK")
	}

	r.Warnings = append(r.Warnings, "a warning")
	if !r.OK() {
		t.Error("result with only warnings should still be OK")
	}

	r.Errors = append(r.Errors, "an error")
	if r.OK() {
		t.Error("result with errors should not be OK")
	}
}

func TestValidationResult_Merge(t *testing.T) {
	a := &ValidationResult{
		Warnings: []string{"w1"},
		Errors:   []string{"e1"},
	}
	b := &ValidationResult{
		Warnings: []string{"w2", "w3"},
		Errors:   []string{"e2"},
	}

	a.Merge(b)

	if len(a.Warnings) != 3 {
		t.Errorf("expected 3 warnings, got %d", len(a.Warnings))
	}
	if len(a.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(a.Errors))
	}
}

func TestValidationResult_MergeNil(t *testing.T) {
	a := &ValidationResult{Warnings: []string{"w1"}}
	a.Merge(nil) // should not panic
	if len(a.Warnings) != 1 {
		t.Error("merge nil should not change warnings")
	}
}
