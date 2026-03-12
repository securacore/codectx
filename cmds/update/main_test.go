package update

import (
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/testutil"
)

func TestClassifyChanges_AllNew(t *testing.T) {
	t.Parallel()

	result := &registry.ResolveResult{
		Packages: map[string]*registry.ResolvedPackage{
			"react-patterns@community": {
				ResolvedVersion: "2.3.1",
				Source:          registry.SourceDirect,
			},
			"utils@org": {
				ResolvedVersion: "1.0.0",
				Source:          registry.SourceTransitive,
			},
		},
	}

	entries := classifyChanges(result, nil)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.Status != statusNew {
			t.Errorf("%s: expected status %q, got %q", e.Ref, statusNew, e.Status)
		}
		if e.OldVersion != "" {
			t.Errorf("%s: expected empty old version, got %q", e.Ref, e.OldVersion)
		}
	}
}

func TestClassifyChanges_AllUnchanged(t *testing.T) {
	t.Parallel()

	result := &registry.ResolveResult{
		Packages: map[string]*registry.ResolvedPackage{
			"react-patterns@community": {
				ResolvedVersion: "2.3.1",
				Source:          registry.SourceDirect,
			},
		},
	}
	oldLock := &registry.LockFile{
		Packages: map[string]*registry.LockedPackage{
			"react-patterns@community": {
				ResolvedVersion: "2.3.1",
			},
		},
	}

	entries := classifyChanges(result, oldLock)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != statusUnchanged {
		t.Errorf("expected status %q, got %q", statusUnchanged, entries[0].Status)
	}
	if entries[0].OldVersion != "2.3.1" {
		t.Errorf("expected old version %q, got %q", "2.3.1", entries[0].OldVersion)
	}
}

func TestClassifyChanges_Mixed(t *testing.T) {
	t.Parallel()

	result := &registry.ResolveResult{
		Packages: map[string]*registry.ResolvedPackage{
			"react-patterns@community": {
				ResolvedVersion: "2.4.0",
				Source:          registry.SourceDirect,
			},
			"utils@org": {
				ResolvedVersion: "1.0.0",
				Source:          registry.SourceTransitive,
			},
			"new-pkg@other": {
				ResolvedVersion: "1.0.0",
				Source:          registry.SourceDirect,
			},
		},
	}
	oldLock := &registry.LockFile{
		Packages: map[string]*registry.LockedPackage{
			"react-patterns@community": {
				ResolvedVersion: "2.3.1",
			},
			"utils@org": {
				ResolvedVersion: "1.0.0",
			},
		},
	}

	entries := classifyChanges(result, oldLock)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Entries are sorted by ref.
	statusMap := make(map[string]changeStatus)
	for _, e := range entries {
		statusMap[e.Ref] = e.Status
	}

	if statusMap["react-patterns@community"] != statusUpdated {
		t.Errorf("react-patterns should be updated, got %q", statusMap["react-patterns@community"])
	}
	if statusMap["utils@org"] != statusUnchanged {
		t.Errorf("utils should be unchanged, got %q", statusMap["utils@org"])
	}
	if statusMap["new-pkg@other"] != statusNew {
		t.Errorf("new-pkg should be new, got %q", statusMap["new-pkg@other"])
	}
}

func TestClassifyChanges_Sorted(t *testing.T) {
	t.Parallel()

	result := &registry.ResolveResult{
		Packages: map[string]*registry.ResolvedPackage{
			"zebra@org": {ResolvedVersion: "1.0.0", Source: registry.SourceDirect},
			"alpha@org": {ResolvedVersion: "1.0.0", Source: registry.SourceDirect},
			"mid@org":   {ResolvedVersion: "1.0.0", Source: registry.SourceDirect},
		},
	}

	entries := classifyChanges(result, nil)

	if entries[0].Ref != "alpha@org" {
		t.Errorf("first entry should be alpha@org, got %q", entries[0].Ref)
	}
	if entries[1].Ref != "mid@org" {
		t.Errorf("second entry should be mid@org, got %q", entries[1].Ref)
	}
	if entries[2].Ref != "zebra@org" {
		t.Errorf("third entry should be zebra@org, got %q", entries[2].Ref)
	}
}

func TestCountChanged(t *testing.T) {
	t.Parallel()

	entries := []changeEntry{
		{Status: statusNew},
		{Status: statusUpdated},
		{Status: statusUnchanged},
		{Status: statusUnchanged},
	}

	got := countChanged(entries)
	if got != 2 {
		t.Errorf("expected 2 changed, got %d", got)
	}
}

func TestCountChanged_NoneChanged(t *testing.T) {
	t.Parallel()

	entries := []changeEntry{
		{Status: statusUnchanged},
		{Status: statusUnchanged},
	}

	got := countChanged(entries)
	if got != 0 {
		t.Errorf("expected 0 changed, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// shouldAutoCompile
// ---------------------------------------------------------------------------

func TestShouldAutoCompile_SkipFlag(t *testing.T) {
	t.Parallel()

	dir, cfg := testutil.SetupProjectWithPrefs(t, project.BoolPtr(true))
	got := shouldAutoCompile(dir, cfg, false, true)
	if got {
		t.Error("expected false when --no-compile is set")
	}
}

func TestShouldAutoCompile_ForceFlag(t *testing.T) {
	t.Parallel()

	dir, cfg := testutil.SetupProjectWithPrefs(t, project.BoolPtr(false))
	got := shouldAutoCompile(dir, cfg, true, false)
	if !got {
		t.Error("expected true when --compile is set, even with auto_compile: false")
	}
}

func TestShouldAutoCompile_ConfigDefault(t *testing.T) {
	t.Parallel()

	// auto_compile is nil (not set) — should default to true.
	dir, cfg := testutil.SetupProjectWithPrefs(t, nil)
	got := shouldAutoCompile(dir, cfg, false, false)
	if !got {
		t.Error("expected true when auto_compile is not set (default)")
	}
}

func TestShouldAutoCompile_ConfigExplicitTrue(t *testing.T) {
	t.Parallel()

	dir, cfg := testutil.SetupProjectWithPrefs(t, project.BoolPtr(true))
	got := shouldAutoCompile(dir, cfg, false, false)
	if !got {
		t.Error("expected true when auto_compile is explicitly true")
	}
}

func TestShouldAutoCompile_ConfigExplicitFalse(t *testing.T) {
	t.Parallel()

	dir, cfg := testutil.SetupProjectWithPrefs(t, project.BoolPtr(false))
	got := shouldAutoCompile(dir, cfg, false, false)
	if got {
		t.Error("expected false when auto_compile is explicitly false")
	}
}

func TestShouldAutoCompile_SkipOverridesForce(t *testing.T) {
	t.Parallel()

	// If both flags are set, --no-compile wins (checked first).
	dir, cfg := testutil.SetupProjectWithPrefs(t, project.BoolPtr(true))
	got := shouldAutoCompile(dir, cfg, true, true)
	if got {
		t.Error("expected false when --no-compile is set, even with --compile")
	}
}

func TestShouldAutoCompile_MissingPrefsFile(t *testing.T) {
	t.Parallel()

	// Project dir without preferences.yml — should default to compiling.
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs")
	if err := cfg.WriteToFile(filepath.Join(dir, project.ConfigFileName)); err != nil {
		t.Fatal(err)
	}

	got := shouldAutoCompile(dir, &cfg, false, false)
	if !got {
		t.Error("expected true when preferences.yml is missing (graceful fallback)")
	}
}
