package registry

import (
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/testutil"
)

const sampleLockYAML = `lockfile_version: 1
resolved_at: "2025-03-09T12:00:00Z"

packages:
  react-patterns@community:
    resolved_version: "2.3.1"
    repo: "github.com/community/codectx-react-patterns"
    commit: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
    source: "direct"

  company-standards@acme:
    resolved_version: "2.0.0"
    repo: "github.com/acme/codectx-company-standards"
    commit: "d4e5f6g7h8i9d4e5f6g7h8i9d4e5f6g7h8i9d4e5"
    source: "direct"

  javascript-fundamentals@community:
    resolved_version: "1.3.0"
    repo: "github.com/community/codectx-javascript-fundamentals"
    commit: "g7h8i9j0k1l2g7h8i9j0k1l2g7h8i9j0k1l2g7h8"
    source: "transitive"
    required_by:
      - "react-patterns@community:2.3.1"
`

func TestLoadLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)
	testutil.MustWriteFile(t, path, sampleLockYAML)

	lf, err := LoadLock(path)
	if err != nil {
		t.Fatalf("LoadLock: %v", err)
	}

	if lf.LockfileVersion != 1 {
		t.Errorf("LockfileVersion = %d, want 1", lf.LockfileVersion)
	}
	if lf.ResolvedAt != "2025-03-09T12:00:00Z" {
		t.Errorf("ResolvedAt = %q", lf.ResolvedAt)
	}
	if len(lf.Packages) != 3 {
		t.Fatalf("Packages count = %d, want 3", len(lf.Packages))
	}

	// Direct package.
	rp := lf.Packages["react-patterns@community"]
	if rp == nil {
		t.Fatal("missing react-patterns@community")
	}
	if rp.ResolvedVersion != "2.3.1" {
		t.Errorf("ResolvedVersion = %q", rp.ResolvedVersion)
	}
	if rp.Source != SourceDirect {
		t.Errorf("Source = %q, want %q", rp.Source, SourceDirect)
	}

	// Transitive package.
	jf := lf.Packages["javascript-fundamentals@community"]
	if jf == nil {
		t.Fatal("missing javascript-fundamentals@community")
	}
	if jf.Source != SourceTransitive {
		t.Errorf("Source = %q, want %q", jf.Source, SourceTransitive)
	}
	if len(jf.RequiredBy) != 1 || jf.RequiredBy[0] != "react-patterns@community:2.3.1" {
		t.Errorf("RequiredBy = %v", jf.RequiredBy)
	}
}

func TestLoadLockNonExistent(t *testing.T) {
	t.Parallel()

	_, err := LoadLock("/nonexistent/codectx.lock")
	if err == nil {
		t.Error("expected error for nonexistent lock file")
	}
}

func TestLoadLockInvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)
	testutil.MustWriteFile(t, path, "{{invalid yaml")

	_, err := LoadLock(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestSaveLockRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)

	original := &LockFile{
		LockfileVersion: LockVersion,
		ResolvedAt:      "2025-06-01T00:00:00Z",
		Packages: map[string]*LockedPackage{
			"test-pkg@org": {
				ResolvedVersion: "1.0.0",
				Repo:            "github.com/org/codectx-test-pkg",
				Commit:          "abc123",
				Source:          SourceDirect,
			},
		},
	}

	if err := SaveLock(path, original); err != nil {
		t.Fatalf("SaveLock: %v", err)
	}

	loaded, err := LoadLock(path)
	if err != nil {
		t.Fatalf("LoadLock after SaveLock: %v", err)
	}

	if loaded.LockfileVersion != original.LockfileVersion {
		t.Errorf("LockfileVersion = %d", loaded.LockfileVersion)
	}
	if loaded.ResolvedAt != original.ResolvedAt {
		t.Errorf("ResolvedAt = %q", loaded.ResolvedAt)
	}
	if len(loaded.Packages) != 1 {
		t.Fatalf("Packages count = %d", len(loaded.Packages))
	}
	pkg := loaded.Packages["test-pkg@org"]
	if pkg == nil {
		t.Fatal("missing test-pkg@org")
	}
	if pkg.ResolvedVersion != "1.0.0" || pkg.Commit != "abc123" || pkg.Source != SourceDirect {
		t.Errorf("package mismatch: %+v", pkg)
	}
}

func TestNewLockFile(t *testing.T) {
	t.Parallel()

	lf := NewLockFile()
	if lf.LockfileVersion != LockVersion {
		t.Errorf("LockfileVersion = %d, want %d", lf.LockfileVersion, LockVersion)
	}
	if lf.ResolvedAt == "" {
		t.Error("ResolvedAt should not be empty")
	}
	if lf.Packages == nil {
		t.Error("Packages should be non-nil")
	}
}

func TestLockCurrent(t *testing.T) {
	t.Parallel()

	lf := &LockFile{
		Packages: map[string]*LockedPackage{
			"react-patterns@community": {ResolvedVersion: "2.3.1"},
			"acme-standards@acme":      {ResolvedVersion: "1.0.0"},
		},
	}

	// All deps present in lock.
	deps := map[string]*project.DependencyConfig{
		"react-patterns@community:latest": {Active: true},
		"acme-standards@acme:1.0.0":       {Active: true},
	}
	if !LockCurrent(lf, deps) {
		t.Error("LockCurrent should return true when all deps are in lock")
	}

	// Add a dep not in lock.
	deps["new-pkg@org:1.0.0"] = &project.DependencyConfig{Active: true}
	if LockCurrent(lf, deps) {
		t.Error("LockCurrent should return false when dep is missing from lock")
	}
}

func TestLockCurrentInvalidKey(t *testing.T) {
	t.Parallel()

	lf := &LockFile{
		Packages: map[string]*LockedPackage{},
	}
	deps := map[string]*project.DependencyConfig{
		"invalid-key-no-at": {Active: true},
	}
	if LockCurrent(lf, deps) {
		t.Error("LockCurrent should return false for invalid key")
	}
}

func TestLockCurrent_ExtraPackagesInLock(t *testing.T) {
	t.Parallel()

	// Lock has more packages than deps — this should still return true
	// since LockCurrent only checks that all deps are present in the lock,
	// not that the lock has no extras.
	lf := &LockFile{
		Packages: map[string]*LockedPackage{
			"react-patterns@community":          {ResolvedVersion: "2.3.1"},
			"acme-standards@acme":               {ResolvedVersion: "1.0.0"},
			"javascript-fundamentals@community": {ResolvedVersion: "1.3.0"},
		},
	}

	deps := map[string]*project.DependencyConfig{
		"react-patterns@community:latest": {Active: true},
	}
	if !LockCurrent(lf, deps) {
		t.Error("LockCurrent should return true even when lock has extra packages")
	}
}

func TestLockCurrent_EmptyDeps(t *testing.T) {
	t.Parallel()

	lf := &LockFile{
		Packages: map[string]*LockedPackage{
			"react-patterns@community": {ResolvedVersion: "2.3.1"},
		},
	}

	deps := map[string]*project.DependencyConfig{}
	if !LockCurrent(lf, deps) {
		t.Error("LockCurrent should return true for empty deps map")
	}
}

func TestSortedPackageRefs(t *testing.T) {
	t.Parallel()

	lf := &LockFile{
		Packages: map[string]*LockedPackage{
			"zebra@org":   {},
			"alpha@org":   {},
			"middle@org2": {},
		},
	}

	refs := lf.SortedPackageRefs()
	if len(refs) != 3 {
		t.Fatalf("len = %d, want 3", len(refs))
	}
	if refs[0] != "alpha@org" || refs[1] != "middle@org2" || refs[2] != "zebra@org" {
		t.Errorf("refs = %v, expected sorted order", refs)
	}
}
