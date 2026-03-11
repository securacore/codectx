package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/testutil"
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

// ---------------------------------------------------------------------------
// validateDir
// ---------------------------------------------------------------------------

func Test_validateDir_AllPass(t *testing.T) {
	dir := t.TempDir()

	// Create a directory with README.md that has a heading.
	topicDir := filepath.Join(dir, "topic-a")
	mustMkdir(t, topicDir)
	mustWriteFile(t, filepath.Join(topicDir, "README.md"), "# Topic A\n\nContent.\n")
	mustWriteFile(t, filepath.Join(topicDir, "details.md"), "# Details\n\nMore content.\n")

	result, err := validateDir(dir, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.OK() {
		t.Errorf("expected OK, got errors: %v", result.Errors)
	}
	if result.HasWarnings() {
		t.Errorf("expected no warnings, got: %v", result.Warnings)
	}
}

func Test_validateDir_MissingReadme(t *testing.T) {
	dir := t.TempDir()

	// Create a directory with .md files but no README.md.
	topicDir := filepath.Join(dir, "topic-b")
	mustMkdir(t, topicDir)
	mustWriteFile(t, filepath.Join(topicDir, "guide.md"), "# Guide\n\nContent.\n")

	result, err := validateDir(dir, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasWarnings() {
		t.Error("expected warning for missing README.md")
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "README.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about README.md, got: %v", result.Warnings)
	}
}

func Test_validateDir_ReadmeNotRequired(t *testing.T) {
	dir := t.TempDir()

	topicDir := filepath.Join(dir, "topic-c")
	mustMkdir(t, topicDir)
	mustWriteFile(t, filepath.Join(topicDir, "guide.md"), "# Guide\n\nContent.\n")

	result, err := validateDir(dir, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasWarnings() {
		t.Errorf("expected no warnings when readme not required, got: %v", result.Warnings)
	}
}

func Test_validateDir_FileWithNoHeadings(t *testing.T) {
	dir := t.TempDir()

	topicDir := filepath.Join(dir, "topic-d")
	mustMkdir(t, topicDir)
	mustWriteFile(t, filepath.Join(topicDir, "README.md"), "# Topic D\n")
	mustWriteFile(t, filepath.Join(topicDir, "notes.md"), "Just text, no headings.\n")

	result, err := validateDir(dir, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasWarnings() {
		t.Error("expected warning for file without headings")
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "notes.md") && strings.Contains(w, "no headings") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about notes.md having no headings, got: %v", result.Warnings)
	}
}

func Test_validateDir_SkipsHiddenDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create .codectx/ with files that would fail validation.
	hiddenDir := filepath.Join(dir, ".codectx")
	mustMkdir(t, hiddenDir)
	mustWriteFile(t, filepath.Join(hiddenDir, "ai.yml"), "not markdown")

	// Create a valid topic directory.
	topicDir := filepath.Join(dir, "topic-e")
	mustMkdir(t, topicDir)
	mustWriteFile(t, filepath.Join(topicDir, "README.md"), "# Topic\n\nContent.\n")

	result, err := validateDir(dir, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasWarnings() {
		t.Errorf("should skip hidden dirs, got warnings: %v", result.Warnings)
	}
}

func Test_validateDir_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := validateDir(dir, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.OK() {
		t.Error("empty directory should pass validation")
	}
}

func Test_validateDir_MultipleIssues(t *testing.T) {
	dir := t.TempDir()

	// Dir without README.
	dirA := filepath.Join(dir, "dir-a")
	mustMkdir(t, dirA)
	mustWriteFile(t, filepath.Join(dirA, "guide.md"), "No heading here.\n")

	// Dir without README.
	dirB := filepath.Join(dir, "dir-b")
	mustMkdir(t, dirB)
	mustWriteFile(t, filepath.Join(dirB, "notes.md"), "Also no heading.\n")

	result, err := validateDir(dir, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have at least 4 warnings: 2 missing READMEs + 2 missing headings.
	if len(result.Warnings) < 4 {
		t.Errorf("expected at least 4 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func Test_validateDir_DirWithOnlyReadme(t *testing.T) {
	dir := t.TempDir()

	topicDir := filepath.Join(dir, "topic-f")
	mustMkdir(t, topicDir)
	mustWriteFile(t, filepath.Join(topicDir, "README.md"), "# Topic F\n")

	result, err := validateDir(dir, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Has a README with a heading — everything should pass.
	if result.HasWarnings() {
		t.Errorf("expected no warnings, got: %v", result.Warnings)
	}
}

func Test_validateDir_NonexistentDir(t *testing.T) {
	_, err := validateDir("/nonexistent/path/12345", true, true)
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, project.DirPerm); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	testutil.MustWriteFile(t, path, content)
}
