package session

import (
	"strings"
	"testing"

	codectx "github.com/securacore/codectx/core/context"
	"github.com/securacore/codectx/core/project"
)

// ---------------------------------------------------------------------------
// renderSessionList
// ---------------------------------------------------------------------------

func TestRenderSessionList_BasicOutput(t *testing.T) {
	assembly := &codectx.AssemblyResult{
		TotalTokens: 19300,
		Budget:      30000,
		Entries: []codectx.EntryResult{
			{Reference: "foundation/coding-standards", Tokens: 8200},
			{Reference: "foundation/error-handling", Tokens: 5000},
			{Reference: "foundation/architecture", Tokens: 6100},
		},
	}

	got := renderSessionList(assembly, 30000)

	// Header should contain token counts.
	if !strings.Contains(got, "Always-loaded session context") {
		t.Error("expected header")
	}
	if !strings.Contains(got, "19,300") {
		t.Error("expected total tokens")
	}
	if !strings.Contains(got, "30,000") {
		t.Error("expected budget")
	}

	// Each entry should appear.
	if !strings.Contains(got, "foundation/coding-standards") {
		t.Error("expected first entry reference")
	}
	if !strings.Contains(got, "8,200") {
		t.Error("expected first entry tokens")
	}
	if !strings.Contains(got, "foundation/error-handling") {
		t.Error("expected second entry reference")
	}
}

func TestRenderSessionList_EmptyEntries(t *testing.T) {
	assembly := &codectx.AssemblyResult{
		TotalTokens: 0,
		Budget:      30000,
		Entries:     []codectx.EntryResult{},
	}

	got := renderSessionList(assembly, 30000)

	if !strings.Contains(got, "Always-loaded session context") {
		t.Error("expected header even with no entries")
	}
}

func TestRenderSessionList_BudgetExceeded(t *testing.T) {
	assembly := &codectx.AssemblyResult{
		TotalTokens: 35000,
		Budget:      30000,
		Entries: []codectx.EntryResult{
			{Reference: "foundation/large-doc", Tokens: 35000},
		},
	}

	got := renderSessionList(assembly, 30000)

	if !strings.Contains(got, "Budget exceeded") {
		t.Error("expected budget exceeded warning")
	}
}

func TestRenderSessionList_WithinBudget(t *testing.T) {
	assembly := &codectx.AssemblyResult{
		TotalTokens: 25000,
		Budget:      30000,
		Entries: []codectx.EntryResult{
			{Reference: "foundation/doc", Tokens: 25000},
		},
	}

	got := renderSessionList(assembly, 30000)

	if strings.Contains(got, "Budget exceeded") {
		t.Error("should not show budget exceeded when within budget")
	}
}

// ---------------------------------------------------------------------------
// computeSessionTotal
// ---------------------------------------------------------------------------

func TestComputeSessionTotal_NilSession(t *testing.T) {
	cfg := &project.Config{}
	total, err := computeSessionTotal(cfg, "/root", "/packages", "cl100k_base", 30000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 for nil session, got %d", total)
	}
}

func TestComputeSessionTotal_EmptyAlwaysLoaded(t *testing.T) {
	cfg := &project.Config{
		Session: &project.SessionConfig{
			AlwaysLoaded: []string{},
			Budget:       30000,
		},
	}
	total, err := computeSessionTotal(cfg, "/root", "/packages", "cl100k_base", 30000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 for empty always_loaded, got %d", total)
	}
}

// ---------------------------------------------------------------------------
// isDuplicate
// ---------------------------------------------------------------------------

func TestIsDuplicate_Found(t *testing.T) {
	list := []string{"foundation/coding-standards", "foundation/error-handling"}
	if !isDuplicate(list, "foundation/coding-standards") {
		t.Error("expected duplicate to be found")
	}
}

func TestIsDuplicate_NotFound(t *testing.T) {
	list := []string{"foundation/coding-standards", "foundation/error-handling"}
	if isDuplicate(list, "foundation/architecture") {
		t.Error("expected no duplicate")
	}
}

func TestIsDuplicate_EmptyList(t *testing.T) {
	if isDuplicate(nil, "anything") {
		t.Error("expected no duplicate in nil list")
	}
	if isDuplicate([]string{}, "anything") {
		t.Error("expected no duplicate in empty list")
	}
}

// ---------------------------------------------------------------------------
// removeRef
// ---------------------------------------------------------------------------

func TestRemoveRef_Found(t *testing.T) {
	list := []string{"a", "b", "c"}
	filtered, found := removeRef(list, "b")
	if !found {
		t.Fatal("expected found=true")
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 items, got %d", len(filtered))
	}
	if filtered[0] != "a" || filtered[1] != "c" {
		t.Errorf("expected [a, c], got %v", filtered)
	}
}

func TestRemoveRef_NotFound(t *testing.T) {
	list := []string{"a", "b", "c"}
	filtered, found := removeRef(list, "d")
	if found {
		t.Error("expected found=false")
	}
	if len(filtered) != 3 {
		t.Errorf("expected 3 items, got %d", len(filtered))
	}
}

func TestRemoveRef_EmptyList(t *testing.T) {
	filtered, found := removeRef(nil, "a")
	if found {
		t.Error("expected found=false for nil list")
	}
	if len(filtered) != 0 {
		t.Errorf("expected 0 items, got %d", len(filtered))
	}
}

func TestRemoveRef_FirstOccurrenceOnly(t *testing.T) {
	list := []string{"a", "b", "a", "c"}
	filtered, found := removeRef(list, "a")
	if !found {
		t.Fatal("expected found=true")
	}
	// Should only remove the first "a".
	if len(filtered) != 3 {
		t.Fatalf("expected 3 items, got %d", len(filtered))
	}
	if filtered[0] != "b" || filtered[1] != "a" || filtered[2] != "c" {
		t.Errorf("expected [b, a, c], got %v", filtered)
	}
}

func TestRemoveRef_SingleItem(t *testing.T) {
	list := []string{"only"}
	filtered, found := removeRef(list, "only")
	if !found {
		t.Fatal("expected found=true")
	}
	if len(filtered) != 0 {
		t.Errorf("expected 0 items, got %d", len(filtered))
	}
}
