package project_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/project"
)

func TestEnsureGitignore_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()

	if err := project.EnsureGitignore(dir, "docs"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	expected := []string{
		"docs/.codectx/compiled/",
		"docs/.codectx/packages/",
		"docs/.codectx/ai.local.yml",
		"!docs/.codectx/ai.yml",
		"!docs/.codectx/preferences.yml",
	}
	for _, e := range expected {
		if !strings.Contains(content, e) {
			t.Errorf("expected .gitignore to contain %q", e)
		}
	}
}

func TestEnsureGitignore_PreservesExisting(t *testing.T) {
	dir := t.TempDir()

	existing := "# Custom ignores\nnode_modules/\n*.log\nbuild/\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := project.EnsureGitignore(dir, "docs"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// Existing entries preserved.
	if !strings.Contains(content, "node_modules/") {
		t.Error("expected node_modules/ to be preserved")
	}
	if !strings.Contains(content, "*.log") {
		t.Error("expected *.log to be preserved")
	}
	if !strings.Contains(content, "build/") {
		t.Error("expected build/ to be preserved")
	}

	// codectx entries present.
	if !strings.Contains(content, "docs/.codectx/compiled/") {
		t.Error("expected codectx entries to be added")
	}

	// Existing content comes before managed block.
	nodeIdx := strings.Index(content, "node_modules/")
	managedIdx := strings.Index(content, "# codectx managed entries")
	if nodeIdx > managedIdx {
		t.Error("expected existing entries to come before managed block")
	}
}

func TestEnsureGitignore_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Run three times.
	for i := range 3 {
		if err := project.EnsureGitignore(dir, "docs"); err != nil {
			t.Fatalf("run %d: unexpected error: %v", i+1, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// Each managed pattern should appear exactly once.
	patterns := []string{
		"docs/.codectx/compiled/",
		"docs/.codectx/packages/",
		"docs/.codectx/ai.local.yml",
		"!docs/.codectx/ai.yml",
		"!docs/.codectx/preferences.yml",
	}
	for _, p := range patterns {
		count := strings.Count(content, p)
		if count != 1 {
			t.Errorf("expected exactly 1 occurrence of %q, got %d", p, count)
		}
	}

	// Section header should appear exactly once.
	headerCount := strings.Count(content, "# codectx managed entries")
	if headerCount != 1 {
		t.Errorf("expected exactly 1 section header, got %d", headerCount)
	}
}

func TestEnsureGitignore_DeduplicatesExisting(t *testing.T) {
	dir := t.TempDir()

	// Simulate someone manually adding a codectx pattern.
	existing := "# Existing\nnode_modules/\ndocs/.codectx/compiled/\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := project.EnsureGitignore(dir, "docs"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// The pattern should appear exactly once (in the managed block).
	count := strings.Count(content, "docs/.codectx/compiled/")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of compiled/, got %d", count)
	}
}

func TestEnsureGitignore_ResolvesConflicts(t *testing.T) {
	dir := t.TempDir()

	// Simulate conflicting rules: existing ignores what we want to force-include.
	existing := "# Bad rule\ndocs/.codectx/ai.yml\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := project.EnsureGitignore(dir, "docs"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// The conflicting ignore rule (without !) should be removed.
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "docs/.codectx/ai.yml" {
			t.Error("expected conflicting ignore rule 'docs/.codectx/ai.yml' to be removed")
		}
	}

	// The force-include should be present.
	if !strings.Contains(content, "!docs/.codectx/ai.yml") {
		t.Error("expected force-include !docs/.codectx/ai.yml to be present")
	}
}

func TestEnsureGitignore_CustomRoot(t *testing.T) {
	dir := t.TempDir()

	if err := project.EnsureGitignore(dir, "custom"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "custom/.codectx/compiled/") {
		t.Error("expected custom root path")
	}
	if strings.Contains(content, "docs/.codectx/") {
		t.Error("expected no default 'docs' root when custom root used")
	}
}

func TestEnsureGitignore_DefaultRoot(t *testing.T) {
	dir := t.TempDir()

	if err := project.EnsureGitignore(dir, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "docs/.codectx/compiled/") {
		t.Error("expected default 'docs' root")
	}
}

func TestEnsureGitignore_IgnoreRulesBeforeNegations(t *testing.T) {
	dir := t.TempDir()

	if err := project.EnsureGitignore(dir, "docs"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}

	content := string(data)

	// Ignore rules must come before negation rules for gitignore to work.
	compiledIdx := strings.Index(content, "docs/.codectx/compiled/")
	forceIdx := strings.Index(content, "!docs/.codectx/ai.yml")

	if compiledIdx == -1 || forceIdx == -1 {
		t.Fatal("expected both ignore and negation patterns to exist")
	}

	if compiledIdx > forceIdx {
		t.Error("expected ignore patterns to come before negation patterns")
	}
}

func TestEnsureGitignore_ReplacesManagedBlock(t *testing.T) {
	dir := t.TempDir()

	// First run.
	if err := project.EnsureGitignore(dir, "docs"); err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Manually append content after the managed block.
	path := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	modified := string(data) + "\n# User addition\nmy-custom-ignore/\n"
	if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
		t.Fatal(err)
	}

	// Second run — should replace managed block but keep user addition.
	if err := project.EnsureGitignore(dir, "docs"); err != nil {
		t.Fatalf("second run: %v", err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result)

	if !strings.Contains(content, "my-custom-ignore/") {
		t.Error("expected user addition after managed block to be preserved")
	}

	// Managed block should not be duplicated.
	count := strings.Count(content, "# codectx managed entries")
	if count != 1 {
		t.Errorf("expected exactly 1 managed block, got %d", count)
	}
}
