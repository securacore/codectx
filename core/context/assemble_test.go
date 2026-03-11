package context_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/context"
	"github.com/securacore/codectx/core/tokens"
)

// ---------------------------------------------------------------------------
// Assemble
// ---------------------------------------------------------------------------

func TestAssemble_SingleEntry(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "standards", "README.md"),
		"# Coding Standards\n\nFollow these rules.\n\n## Naming\n\nUse camelCase.\n")

	entries, err := context.Resolve(root, "", []string{"foundation/standards"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	result, err := context.Assemble(entries, tokens.Cl100kBase, 30000)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if result.TotalTokens == 0 {
		t.Error("expected non-zero total tokens")
	}
	if result.Budget != 30000 {
		t.Errorf("expected budget 30000, got %d", result.Budget)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry result, got %d", len(result.Entries))
	}
	if result.Entries[0].Tokens == 0 {
		t.Error("expected non-zero entry tokens")
	}
	if result.Entries[0].Title != "Standards" {
		t.Errorf("expected title %q, got %q", "Standards", result.Entries[0].Title)
	}

	// Content should contain the H2 entry header and internal headings.
	if !strings.Contains(result.Content, "## Standards") {
		t.Error("expected H2 entry title in content")
	}
	if !strings.Contains(result.Content, "Follow these rules.") {
		t.Error("expected paragraph content")
	}
}

func TestAssemble_MultipleEntries(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "coding", "README.md"),
		"# Coding\n\nCode well.\n")
	mustWriteFile(t, filepath.Join(root, "foundation", "arch", "README.md"),
		"# Architecture\n\nDesign well.\n")

	entries, err := context.Resolve(root, "", []string{
		"foundation/coding",
		"foundation/arch",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	result, err := context.Assemble(entries, tokens.Cl100kBase, 30000)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 entry results, got %d", len(result.Entries))
	}

	// Both entries should be present.
	if !strings.Contains(result.Content, "## Coding") {
		t.Error("expected first entry in content")
	}
	if !strings.Contains(result.Content, "## Arch") {
		t.Error("expected second entry in content")
	}

	// Order should match input.
	codingIdx := strings.Index(result.Content, "## Coding")
	archIdx := strings.Index(result.Content, "## Arch")
	if codingIdx > archIdx {
		t.Error("expected Coding before Arch in assembled content")
	}
}

func TestAssemble_HeadingLevelShift(t *testing.T) {
	root := t.TempDir()

	// File with H1 and H2 — should be shifted to H3 and H4 under the H2 entry title.
	mustWriteFile(t, filepath.Join(root, "foundation", "standards", "README.md"),
		"# Top Level\n\nContent.\n\n## Second Level\n\nMore content.\n")

	entries, err := context.Resolve(root, "", []string{"foundation/standards"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	result, err := context.Assemble(entries, tokens.Cl100kBase, 30000)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Original H1 "Top Level" should become H3 (under H2 entry title).
	if !strings.Contains(result.Content, "### Top Level") {
		t.Errorf("expected H1 shifted to H3, got:\n%s", result.Content)
	}
	// Original H2 "Second Level" should become H4.
	if !strings.Contains(result.Content, "#### Second Level") {
		t.Errorf("expected H2 shifted to H4, got:\n%s", result.Content)
	}
}

func TestAssemble_BudgetWarning(t *testing.T) {
	root := t.TempDir()

	// Create a file with enough content to exceed a tiny budget.
	content := "# Standards\n\n" + strings.Repeat("This is a long paragraph with many tokens to exceed the budget. ", 50) + "\n"
	mustWriteFile(t, filepath.Join(root, "foundation", "standards", "README.md"), content)

	entries, err := context.Resolve(root, "", []string{"foundation/standards"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Set a very small budget.
	result, err := context.Assemble(entries, tokens.Cl100kBase, 10)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Error("expected budget warning")
	}

	foundBudgetWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "exceeds budget") {
			foundBudgetWarning = true
			break
		}
	}
	if !foundBudgetWarning {
		t.Errorf("expected budget exceeded warning, got: %v", result.Warnings)
	}

	if result.Utilization <= 100 {
		t.Errorf("expected utilization > 100%%, got %.1f%%", result.Utilization)
	}
}

func TestAssemble_NoBudgetWarningUnderLimit(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "standards", "README.md"),
		"# Standards\n\nShort content.\n")

	entries, err := context.Resolve(root, "", []string{"foundation/standards"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	result, err := context.Assemble(entries, tokens.Cl100kBase, 30000)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Warnings) > 0 {
		t.Errorf("expected no warnings, got: %v", result.Warnings)
	}
}

func TestAssemble_EmptyEntries(t *testing.T) {
	result, err := context.Assemble(nil, tokens.Cl100kBase, 30000)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if result.TotalTokens != 0 {
		t.Errorf("expected 0 tokens, got %d", result.TotalTokens)
	}
	if result.Content != "" {
		t.Errorf("expected empty content, got %q", result.Content)
	}
	if result.Budget != 30000 {
		t.Errorf("expected budget 30000, got %d", result.Budget)
	}
}

func TestAssemble_InvalidEncoding(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "foundation", "README.md"), "# Test\n")

	entries, err := context.Resolve(root, "", []string{"foundation"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	_, err = context.Assemble(entries, "invalid-encoding-xyz", 30000)
	if err == nil {
		t.Fatal("expected error for invalid encoding")
	}
}

func TestAssemble_MultipleFilesInEntry(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "standards", "README.md"),
		"# Overview\n\nGeneral standards.\n")
	mustWriteFile(t, filepath.Join(root, "foundation", "standards", "naming.md"),
		"# Naming\n\nUse camelCase.\n")

	entries, err := context.Resolve(root, "", []string{"foundation/standards"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	result, err := context.Assemble(entries, tokens.Cl100kBase, 30000)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if result.Entries[0].FileCount != 2 {
		t.Errorf("expected 2 files, got %d", result.Entries[0].FileCount)
	}

	// Both files should be present in the content.
	if !strings.Contains(result.Content, "General standards.") {
		t.Error("expected README content")
	}
	if !strings.Contains(result.Content, "Use camelCase.") {
		t.Error("expected naming content")
	}
}

func TestAssemble_CodeBlocksPreserved(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "foundation", "README.md"),
		"# Standards\n\nExample:\n\n```go\nfunc main() {}\n```\n")

	entries, err := context.Resolve(root, "", []string{"foundation"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	result, err := context.Assemble(entries, tokens.Cl100kBase, 30000)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if !strings.Contains(result.Content, "```go") {
		t.Error("expected code block with language tag")
	}
	if !strings.Contains(result.Content, "func main() {}") {
		t.Error("expected code block content")
	}
}

func TestAssemble_ZeroBudgetNoWarning(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "foundation", "README.md"), "# Test\n\nContent.\n")

	entries, err := context.Resolve(root, "", []string{"foundation"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Zero budget means no budget enforcement.
	result, err := context.Assemble(entries, tokens.Cl100kBase, 0)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	if len(result.Warnings) > 0 {
		t.Errorf("expected no warnings with zero budget, got: %v", result.Warnings)
	}
}
