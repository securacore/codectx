package link

import (
	"os"
	"path/filepath"
	"testing"

	corelink "github.com/securacore/codectx/core/link"
	"github.com/securacore/codectx/core/testutil"
)

func TestSelectAll_ReturnsAllIntegrations(t *testing.T) {
	selected := selectAll()
	allIntegrations := corelink.AllIntegrations()

	if len(selected) != len(allIntegrations) {
		t.Errorf("selectAll() returned %d integrations, want %d", len(selected), len(allIntegrations))
	}

	for i, info := range allIntegrations {
		if selected[i] != info.Type {
			t.Errorf("selectAll()[%d] = %v, want %v", i, selected[i], info.Type)
		}
	}
}

func TestSelectNonInteractive_DefaultsToClaude(t *testing.T) {
	// Empty directory — no tools detected.
	dir := t.TempDir()

	selected := selectNonInteractive(dir)
	if len(selected) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(selected))
	}
	if selected[0] != corelink.Claude {
		t.Errorf("expected Claude, got %v", selected[0])
	}
}

func TestSelectNonInteractive_DetectsClaudeDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	selected := selectNonInteractive(dir)
	found := false
	for _, s := range selected {
		if s == corelink.Claude {
			found = true
		}
	}
	if !found {
		t.Error("expected Claude to be detected")
	}
}

func TestSelectNonInteractive_DetectsCursorDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}

	selected := selectNonInteractive(dir)
	found := false
	for _, s := range selected {
		if s == corelink.Cursor {
			found = true
		}
	}
	if !found {
		t.Error("expected Cursor to be detected")
	}
}

func TestHasExistingFiles_NoFiles(t *testing.T) {
	dir := t.TempDir()
	integrations := []corelink.Integration{corelink.Claude, corelink.Agents}

	if hasExistingFiles(dir, integrations) {
		t.Error("expected no existing files")
	}
}

func TestHasExistingFiles_WithExisting(t *testing.T) {
	dir := t.TempDir()
	testutil.MustWriteFile(t, filepath.Join(dir, "CLAUDE.md"), "existing content")

	if !hasExistingFiles(dir, []corelink.Integration{corelink.Claude}) {
		t.Error("expected existing file to be detected")
	}
}

func TestHasExistingFiles_EmptyList(t *testing.T) {
	dir := t.TempDir()
	if hasExistingFiles(dir, nil) {
		t.Error("expected false for empty integrations list")
	}
}
