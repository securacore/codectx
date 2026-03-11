package context_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/context"
)

// ---------------------------------------------------------------------------
// WriteContextMD
// ---------------------------------------------------------------------------

func TestWriteContextMD_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	compiledDir := filepath.Join(dir, "compiled")

	result := &context.AssemblyResult{
		Content:     "## Coding Standards\n\nFollow these rules.\n",
		TotalTokens: 100,
		Budget:      30000,
		Utilization: 0.33,
	}

	if err := context.WriteContextMD(compiledDir, result); err != nil {
		t.Fatalf("WriteContextMD: %v", err)
	}

	path := context.ContextPath(compiledDir)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading context.md: %v", err)
	}

	content := string(data)

	// Verify header.
	if !strings.Contains(content, "# Project Engineering Context") {
		t.Error("expected main heading")
	}
	if !strings.Contains(content, "session.always_loaded") {
		t.Error("expected source reference")
	}
	if !strings.Contains(content, "100 / 30,000 budget") {
		t.Error("expected token count and budget")
	}
	if !strings.Contains(content, "Compiled:") {
		t.Error("expected compiled timestamp")
	}

	// Verify content.
	if !strings.Contains(content, "## Coding Standards") {
		t.Error("expected assembled content")
	}
	if !strings.Contains(content, "Follow these rules.") {
		t.Error("expected paragraph content")
	}
}

func TestWriteContextMD_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	compiledDir := filepath.Join(dir, "nested", "compiled")

	result := &context.AssemblyResult{
		Content:     "## Test\n\nContent.\n",
		TotalTokens: 10,
		Budget:      1000,
	}

	if err := context.WriteContextMD(compiledDir, result); err != nil {
		t.Fatalf("WriteContextMD: %v", err)
	}

	if _, err := os.Stat(context.ContextPath(compiledDir)); err != nil {
		t.Errorf("expected context.md to exist: %v", err)
	}
}

func TestWriteContextMD_NilResult(t *testing.T) {
	err := context.WriteContextMD(t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected error for nil result")
	}
}

func TestWriteContextMD_EmptyContent(t *testing.T) {
	dir := t.TempDir()

	result := &context.AssemblyResult{
		Budget: 30000,
	}

	if err := context.WriteContextMD(dir, result); err != nil {
		t.Fatalf("WriteContextMD: %v", err)
	}

	data, err := os.ReadFile(context.ContextPath(dir))
	if err != nil {
		t.Fatalf("reading context.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# Project Engineering Context") {
		t.Error("expected header even with empty content")
	}
	if !strings.Contains(content, "0 / 30,000 budget") {
		t.Errorf("expected zero token count, got:\n%s", content)
	}
}

func TestWriteContextMD_ZeroBudget(t *testing.T) {
	dir := t.TempDir()

	result := &context.AssemblyResult{
		Content:     "## Test\n\nContent.\n",
		TotalTokens: 50,
		Budget:      0,
	}

	if err := context.WriteContextMD(dir, result); err != nil {
		t.Fatalf("WriteContextMD: %v", err)
	}

	data, err := os.ReadFile(context.ContextPath(dir))
	if err != nil {
		t.Fatalf("reading context.md: %v", err)
	}

	content := string(data)
	// With zero budget, should show token count without "/ budget" suffix.
	if strings.Contains(content, "budget") {
		t.Errorf("expected no budget display with zero budget, got:\n%s", content)
	}
	if !strings.Contains(content, "50") {
		t.Error("expected token count")
	}
}

func TestContextPath(t *testing.T) {
	path := context.ContextPath("/foo/bar/compiled")
	if path != "/foo/bar/compiled/context.md" {
		t.Errorf("expected /foo/bar/compiled/context.md, got %q", path)
	}
}
