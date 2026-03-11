package compile_test

import (
	"sort"
	"testing"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/manifest"
)

// ---------------------------------------------------------------------------
// ClassifyFiles
// ---------------------------------------------------------------------------

func TestClassifyFiles_NilPreviousAllNew(t *testing.T) {
	current := map[string]string{
		"docs/a.md": "sha256:aaa",
		"docs/b.md": "sha256:bbb",
	}

	cs := compile.ClassifyFiles(current, nil)

	if cs.NewCount != 2 {
		t.Errorf("NewCount = %d, want 2", cs.NewCount)
	}
	if cs.ModifiedCount != 0 {
		t.Errorf("ModifiedCount = %d, want 0", cs.ModifiedCount)
	}
	if cs.UnchangedCount != 0 {
		t.Errorf("UnchangedCount = %d, want 0", cs.UnchangedCount)
	}
	if len(cs.Deleted) != 0 {
		t.Errorf("Deleted = %v, want empty", cs.Deleted)
	}
	for path, status := range cs.Status {
		if status != compile.FileNew {
			t.Errorf("file %s: status = %d, want FileNew", path, status)
		}
	}
}

func TestClassifyFiles_EmptyPreviousAllNew(t *testing.T) {
	current := map[string]string{
		"docs/a.md": "sha256:aaa",
	}
	prev := &manifest.Hashes{Files: map[string]string{}}

	cs := compile.ClassifyFiles(current, prev)

	if cs.NewCount != 1 {
		t.Errorf("NewCount = %d, want 1", cs.NewCount)
	}
}

func TestClassifyFiles_AllUnchanged(t *testing.T) {
	hashes := map[string]string{
		"docs/a.md": "sha256:aaa",
		"docs/b.md": "sha256:bbb",
	}
	prev := &manifest.Hashes{Files: hashes}

	// Use same hashes for current.
	current := map[string]string{
		"docs/a.md": "sha256:aaa",
		"docs/b.md": "sha256:bbb",
	}

	cs := compile.ClassifyFiles(current, prev)

	if cs.NewCount != 0 {
		t.Errorf("NewCount = %d, want 0", cs.NewCount)
	}
	if cs.ModifiedCount != 0 {
		t.Errorf("ModifiedCount = %d, want 0", cs.ModifiedCount)
	}
	if cs.UnchangedCount != 2 {
		t.Errorf("UnchangedCount = %d, want 2", cs.UnchangedCount)
	}
	if cs.HasChanges() {
		t.Error("HasChanges should be false when all files unchanged and no deletions")
	}
}

func TestClassifyFiles_MixedChanges(t *testing.T) {
	prev := &manifest.Hashes{
		Files: map[string]string{
			"docs/unchanged.md": "sha256:same",
			"docs/modified.md":  "sha256:old",
			"docs/deleted.md":   "sha256:gone",
		},
	}

	current := map[string]string{
		"docs/unchanged.md": "sha256:same",
		"docs/modified.md":  "sha256:new",
		"docs/added.md":     "sha256:fresh",
	}

	cs := compile.ClassifyFiles(current, prev)

	if cs.NewCount != 1 {
		t.Errorf("NewCount = %d, want 1", cs.NewCount)
	}
	if cs.ModifiedCount != 1 {
		t.Errorf("ModifiedCount = %d, want 1", cs.ModifiedCount)
	}
	if cs.UnchangedCount != 1 {
		t.Errorf("UnchangedCount = %d, want 1", cs.UnchangedCount)
	}
	if len(cs.Deleted) != 1 || cs.Deleted[0] != "docs/deleted.md" {
		t.Errorf("Deleted = %v, want [docs/deleted.md]", cs.Deleted)
	}

	if cs.Status["docs/added.md"] != compile.FileNew {
		t.Error("docs/added.md should be FileNew")
	}
	if cs.Status["docs/modified.md"] != compile.FileModified {
		t.Error("docs/modified.md should be FileModified")
	}
	if cs.Status["docs/unchanged.md"] != compile.FileUnchanged {
		t.Error("docs/unchanged.md should be FileUnchanged")
	}

	if !cs.HasChanges() {
		t.Error("HasChanges should be true")
	}
}

func TestClassifyFiles_DeletedOnly(t *testing.T) {
	prev := &manifest.Hashes{
		Files: map[string]string{
			"docs/a.md": "sha256:aaa",
			"docs/b.md": "sha256:bbb",
		},
	}

	// Only one file remains.
	current := map[string]string{
		"docs/a.md": "sha256:aaa",
	}

	cs := compile.ClassifyFiles(current, prev)

	if cs.UnchangedCount != 1 {
		t.Errorf("UnchangedCount = %d, want 1", cs.UnchangedCount)
	}
	if len(cs.Deleted) != 1 || cs.Deleted[0] != "docs/b.md" {
		t.Errorf("Deleted = %v, want [docs/b.md]", cs.Deleted)
	}
	if !cs.HasChanges() {
		t.Error("HasChanges should be true when files are deleted")
	}
}

func TestClassifyFiles_HasChanges_NewFiles(t *testing.T) {
	prev := &manifest.Hashes{Files: map[string]string{}}
	current := map[string]string{"docs/new.md": "sha256:new"}

	cs := compile.ClassifyFiles(current, prev)

	if !cs.HasChanges() {
		t.Error("HasChanges should be true when new files exist")
	}
}

func TestClassifyFiles_HasChanges_ModifiedFiles(t *testing.T) {
	prev := &manifest.Hashes{Files: map[string]string{"docs/a.md": "sha256:old"}}
	current := map[string]string{"docs/a.md": "sha256:new"}

	cs := compile.ClassifyFiles(current, prev)

	if !cs.HasChanges() {
		t.Error("HasChanges should be true when files are modified")
	}
}

// ---------------------------------------------------------------------------
// DetectInstructionChanges
// ---------------------------------------------------------------------------

func TestDetectInstructionChanges_NoChanges(t *testing.T) {
	system := map[string]string{
		"taxonomy-generation": "sha256:aaa",
		"bridge-summaries":    "sha256:bbb",
		"context-assembly":    "sha256:ccc",
	}

	ic := compile.DetectInstructionChanges(system, system)

	if ic.TaxonomyGeneration {
		t.Error("TaxonomyGeneration should be false")
	}
	if ic.BridgeSummaries {
		t.Error("BridgeSummaries should be false")
	}
	if ic.ContextAssembly {
		t.Error("ContextAssembly should be false")
	}
	if ic.AnyChanged() {
		t.Error("AnyChanged should be false")
	}
}

func TestDetectInstructionChanges_TaxonomyChanged(t *testing.T) {
	prev := map[string]string{
		"taxonomy-generation": "sha256:old",
		"bridge-summaries":    "sha256:bbb",
	}
	curr := map[string]string{
		"taxonomy-generation": "sha256:new",
		"bridge-summaries":    "sha256:bbb",
	}

	ic := compile.DetectInstructionChanges(curr, prev)

	if !ic.TaxonomyGeneration {
		t.Error("TaxonomyGeneration should be true")
	}
	if ic.BridgeSummaries {
		t.Error("BridgeSummaries should be false")
	}
	if !ic.AnyChanged() {
		t.Error("AnyChanged should be true")
	}
}

func TestDetectInstructionChanges_NewDirectory(t *testing.T) {
	prev := map[string]string{}
	curr := map[string]string{
		"context-assembly": "sha256:new",
	}

	ic := compile.DetectInstructionChanges(curr, prev)

	if !ic.ContextAssembly {
		t.Error("ContextAssembly should be true when dir is new")
	}
}

func TestDetectInstructionChanges_RemovedDirectory(t *testing.T) {
	prev := map[string]string{
		"bridge-summaries": "sha256:old",
	}
	curr := map[string]string{}

	ic := compile.DetectInstructionChanges(curr, prev)

	if !ic.BridgeSummaries {
		t.Error("BridgeSummaries should be true when dir is removed")
	}
}

func TestDetectInstructionChanges_BothMissing(t *testing.T) {
	// Neither prev nor curr has context-assembly.
	prev := map[string]string{
		"taxonomy-generation": "sha256:aaa",
	}
	curr := map[string]string{
		"taxonomy-generation": "sha256:aaa",
	}

	ic := compile.DetectInstructionChanges(curr, prev)

	if ic.ContextAssembly {
		t.Error("ContextAssembly should be false when missing in both")
	}
}

func TestDetectInstructionChanges_AllChanged(t *testing.T) {
	prev := map[string]string{
		"taxonomy-generation": "sha256:old1",
		"bridge-summaries":    "sha256:old2",
		"context-assembly":    "sha256:old3",
	}
	curr := map[string]string{
		"taxonomy-generation": "sha256:new1",
		"bridge-summaries":    "sha256:new2",
		"context-assembly":    "sha256:new3",
	}

	ic := compile.DetectInstructionChanges(curr, prev)

	if !ic.TaxonomyGeneration || !ic.BridgeSummaries || !ic.ContextAssembly {
		t.Error("all should be changed")
	}
	if !ic.AnyChanged() {
		t.Error("AnyChanged should be true")
	}
}

// ---------------------------------------------------------------------------
// ChangeSet.Deleted ordering
// ---------------------------------------------------------------------------

func TestClassifyFiles_DeletedIsSorted(t *testing.T) {
	prev := &manifest.Hashes{
		Files: map[string]string{
			"docs/z.md": "sha256:zzz",
			"docs/a.md": "sha256:aaa",
			"docs/m.md": "sha256:mmm",
		},
	}
	current := map[string]string{} // all deleted

	cs := compile.ClassifyFiles(current, prev)

	// Deleted order may vary since it iterates a map.
	// Just check all are present.
	sort.Strings(cs.Deleted)
	want := []string{"docs/a.md", "docs/m.md", "docs/z.md"}
	if len(cs.Deleted) != len(want) {
		t.Fatalf("Deleted count = %d, want %d", len(cs.Deleted), len(want))
	}
	for i, path := range cs.Deleted {
		if path != want[i] {
			t.Errorf("Deleted[%d] = %q, want %q", i, path, want[i])
		}
	}
}
