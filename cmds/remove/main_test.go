package remove

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
)

func TestParseRef_FullKey(t *testing.T) {
	ref, err := parseRef("react-patterns@community:latest")
	if err != nil {
		t.Fatalf("parseRef: %v", err)
	}
	if ref != "react-patterns@community" {
		t.Errorf("expected %q, got %q", "react-patterns@community", ref)
	}
}

func TestParseRef_ShortRef(t *testing.T) {
	ref, err := parseRef("react-patterns@community")
	if err != nil {
		t.Fatalf("parseRef: %v", err)
	}
	if ref != "react-patterns@community" {
		t.Errorf("expected %q, got %q", "react-patterns@community", ref)
	}
}

func TestParseRef_Invalid(t *testing.T) {
	_, err := parseRef("invalid")
	if err == nil {
		t.Error("expected error for invalid ref")
	}
}

func TestParseRef_EmptyOrg(t *testing.T) {
	_, err := parseRef("name@")
	if err == nil {
		t.Error("expected error for empty org")
	}
}

func TestFindDepByRef_Found(t *testing.T) {
	cfg := &project.Config{
		Dependencies: map[string]*project.DependencyConfig{
			"react-patterns@community:latest": {Active: true},
		},
	}

	key, found := findDepByRef(cfg, "react-patterns@community")
	if !found {
		t.Fatal("expected to find dependency")
	}
	if key != "react-patterns@community:latest" {
		t.Errorf("expected key %q, got %q", "react-patterns@community:latest", key)
	}
}

func TestFindDepByRef_NotFound(t *testing.T) {
	cfg := &project.Config{
		Dependencies: map[string]*project.DependencyConfig{
			"react-patterns@community:latest": {Active: true},
		},
	}

	_, found := findDepByRef(cfg, "other@acme")
	if found {
		t.Error("expected not to find dependency")
	}
}

func TestFindDepByRef_EmptyDeps(t *testing.T) {
	cfg := &project.Config{
		Dependencies: map[string]*project.DependencyConfig{},
	}

	_, found := findDepByRef(cfg, "react-patterns@community")
	if found {
		t.Error("expected not to find dependency in empty deps")
	}
}

func TestRemoveFromProjectConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", "")
	cfg.Dependencies["react-patterns@community:latest"] = &project.DependencyConfig{Active: true}
	cfgPath := filepath.Join(dir, project.ConfigFileName)
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	if err := removeFromProjectConfig(dir, &cfg, "react-patterns@community:latest"); err != nil {
		t.Fatalf("removeFromProjectConfig: %v", err)
	}

	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	if _, ok := loaded.Dependencies["react-patterns@community:latest"]; ok {
		t.Error("expected dependency to be removed")
	}
}

func TestRemoveFromPackageManifest(t *testing.T) {
	dir := t.TempDir()

	manifest := project.DefaultPackageManifest("test-pkg", "testorg", "")
	manifest.Dependencies["react-patterns@community"] = ">=2.0.0"
	manifestPath := project.PackageConfigPath(dir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		t.Fatal(err)
	}

	if err := removeFromPackageManifest(dir, "react-patterns@community"); err != nil {
		t.Fatalf("removeFromPackageManifest: %v", err)
	}

	loaded, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	if _, ok := loaded.Dependencies["react-patterns@community"]; ok {
		t.Error("expected dependency to be removed from package manifest")
	}
}

func TestRemoveDependency_ProjectOnly(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", project.TypePackage)
	cfg.Dependencies["react-patterns@community:latest"] = &project.DependencyConfig{Active: true}
	cfgPath := filepath.Join(dir, project.ConfigFileName)
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	// Also create package manifest with the dep.
	manifest := project.DefaultPackageManifest("test", "testorg", "")
	manifest.Dependencies["react-patterns@community"] = ">=1.0.0"
	manifestPath := project.PackageConfigPath(dir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		t.Fatal(err)
	}

	err := removeDependency(dir, &cfg, "react-patterns@community", "react-patterns@community:latest", rmProject, true, true)
	if err != nil {
		t.Fatalf("removeDependency: %v", err)
	}

	// Root config should not have the dep.
	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if _, ok := loaded.Dependencies["react-patterns@community:latest"]; ok {
		t.Error("expected dep removed from root config")
	}

	// Package manifest should still have the dep.
	loadedManifest, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}
	if _, ok := loadedManifest.Dependencies["react-patterns@community"]; !ok {
		t.Error("expected dep to remain in package manifest")
	}
}

func TestRemoveDependency_PackageOnly(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", project.TypePackage)
	cfg.Dependencies["react-patterns@community:latest"] = &project.DependencyConfig{Active: true}
	cfgPath := filepath.Join(dir, project.ConfigFileName)
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	manifest := project.DefaultPackageManifest("test", "testorg", "")
	manifest.Dependencies["react-patterns@community"] = ">=1.0.0"
	manifestPath := project.PackageConfigPath(dir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		t.Fatal(err)
	}

	err := removeDependency(dir, &cfg, "react-patterns@community", "react-patterns@community:latest", rmPackage, true, true)
	if err != nil {
		t.Fatalf("removeDependency: %v", err)
	}

	// Root config should still have the dep.
	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if _, ok := loaded.Dependencies["react-patterns@community:latest"]; !ok {
		t.Error("expected dep to remain in root config")
	}

	// Package manifest should not have the dep.
	loadedManifest, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}
	if _, ok := loadedManifest.Dependencies["react-patterns@community"]; ok {
		t.Error("expected dep removed from package manifest")
	}
}

func TestRemoveDependency_Both(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", project.TypePackage)
	cfg.Dependencies["react-patterns@community:latest"] = &project.DependencyConfig{Active: true}
	cfgPath := filepath.Join(dir, project.ConfigFileName)
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	manifest := project.DefaultPackageManifest("test", "testorg", "")
	manifest.Dependencies["react-patterns@community"] = ">=1.0.0"
	manifestPath := project.PackageConfigPath(dir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		t.Fatal(err)
	}

	err := removeDependency(dir, &cfg, "react-patterns@community", "react-patterns@community:latest", rmBoth, true, true)
	if err != nil {
		t.Fatalf("removeDependency: %v", err)
	}

	// Both should be removed.
	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if _, ok := loaded.Dependencies["react-patterns@community:latest"]; ok {
		t.Error("expected dep removed from root config")
	}

	loadedManifest, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}
	if _, ok := loadedManifest.Dependencies["react-patterns@community"]; ok {
		t.Error("expected dep removed from package manifest")
	}
}

func TestRemoveLockEntry(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, registry.LockFileName)

	// Create a lock file with some packages.
	lf := &registry.LockFile{
		LockfileVersion: 1,
		ResolvedAt:      "2025-01-01T00:00:00Z",
		Packages: map[string]*registry.LockedPackage{
			"react-patterns@community": {
				ResolvedVersion: "2.0.0",
				Repo:            "github.com/community/codectx-react-patterns",
				Commit:          "abc123",
				Source:          registry.SourceDirect,
			},
			"other@acme": {
				ResolvedVersion: "1.0.0",
				Repo:            "github.com/acme/codectx-other",
				Commit:          "def456",
				Source:          registry.SourceDirect,
			},
		},
	}
	if err := registry.SaveLock(lockPath, lf); err != nil {
		t.Fatal(err)
	}

	removeLockEntry(lockPath, "react-patterns@community")

	loaded, err := registry.LoadLock(lockPath)
	if err != nil {
		t.Fatalf("loading lock: %v", err)
	}

	if _, ok := loaded.Packages["react-patterns@community"]; ok {
		t.Error("expected package to be removed from lock file")
	}
	if _, ok := loaded.Packages["other@acme"]; !ok {
		t.Error("expected other package to remain in lock file")
	}
}

func TestRemoveLockEntry_PrunesOrphanedTransitive(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, registry.LockFileName)

	lf := &registry.LockFile{
		LockfileVersion: 1,
		ResolvedAt:      "2025-01-01T00:00:00Z",
		Packages: map[string]*registry.LockedPackage{
			"react-patterns@community": {
				ResolvedVersion: "2.0.0",
				Repo:            "github.com/community/codectx-react-patterns",
				Commit:          "abc123",
				Source:          registry.SourceDirect,
			},
			"js-fundamentals@community": {
				ResolvedVersion: "1.0.0",
				Repo:            "github.com/community/codectx-js-fundamentals",
				Commit:          "ghi789",
				Source:          registry.SourceTransitive,
				RequiredBy:      []string{"react-patterns@community:2.0.0"},
			},
		},
	}
	if err := registry.SaveLock(lockPath, lf); err != nil {
		t.Fatal(err)
	}

	removeLockEntry(lockPath, "react-patterns@community")

	loaded, err := registry.LoadLock(lockPath)
	if err != nil {
		t.Fatalf("loading lock: %v", err)
	}

	if _, ok := loaded.Packages["react-patterns@community"]; ok {
		t.Error("expected direct package to be removed")
	}
	if _, ok := loaded.Packages["js-fundamentals@community"]; ok {
		t.Error("expected orphaned transitive package to be pruned")
	}
}

func TestRemoveLockEntry_KeepsSharedTransitive(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, registry.LockFileName)

	lf := &registry.LockFile{
		LockfileVersion: 1,
		ResolvedAt:      "2025-01-01T00:00:00Z",
		Packages: map[string]*registry.LockedPackage{
			"react-patterns@community": {
				ResolvedVersion: "2.0.0",
				Repo:            "github.com/community/codectx-react-patterns",
				Commit:          "abc123",
				Source:          registry.SourceDirect,
			},
			"vue-patterns@community": {
				ResolvedVersion: "1.0.0",
				Repo:            "github.com/community/codectx-vue-patterns",
				Commit:          "def456",
				Source:          registry.SourceDirect,
			},
			"js-fundamentals@community": {
				ResolvedVersion: "1.0.0",
				Repo:            "github.com/community/codectx-js-fundamentals",
				Commit:          "ghi789",
				Source:          registry.SourceTransitive,
				RequiredBy: []string{
					"react-patterns@community:2.0.0",
					"vue-patterns@community:1.0.0",
				},
			},
		},
	}
	if err := registry.SaveLock(lockPath, lf); err != nil {
		t.Fatal(err)
	}

	removeLockEntry(lockPath, "react-patterns@community")

	loaded, err := registry.LoadLock(lockPath)
	if err != nil {
		t.Fatalf("loading lock: %v", err)
	}

	if _, ok := loaded.Packages["react-patterns@community"]; ok {
		t.Error("expected direct package to be removed")
	}

	jf, ok := loaded.Packages["js-fundamentals@community"]
	if !ok {
		t.Fatal("expected shared transitive package to remain")
	}

	if len(jf.RequiredBy) != 1 || jf.RequiredBy[0] != "vue-patterns@community:1.0.0" {
		t.Errorf("expected RequiredBy to contain only vue-patterns, got %v", jf.RequiredBy)
	}
}

func TestRemoveLockEntry_NoLockFile(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "nonexistent.lock")

	// Should not panic or error — just no-op.
	removeLockEntry(lockPath, "react-patterns@community")
}

func TestExtractRef(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"react-patterns@community:2.0.0", "react-patterns@community"},
		{"standards@acme:latest", "standards@acme"},
		{"invalid-string", "invalid-string"}, // falls back to input
	}

	for _, tt := range tests {
		got := extractRef(tt.input)
		if got != tt.want {
			t.Errorf("extractRef(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRemoveFromProjectConfig_DiskCleanup(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", "")
	cfg.Dependencies["react-patterns@community:latest"] = &project.DependencyConfig{Active: true}
	cfg.Dependencies["other@acme:1.0.0"] = &project.DependencyConfig{Active: true}
	cfgPath := filepath.Join(dir, project.ConfigFileName)
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	if err := removeFromProjectConfig(dir, &cfg, "react-patterns@community:latest"); err != nil {
		t.Fatalf("removeFromProjectConfig: %v", err)
	}

	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	// Removed dep should be gone.
	if _, ok := loaded.Dependencies["react-patterns@community:latest"]; ok {
		t.Error("expected removed dep to be gone")
	}

	// Other dep should remain.
	if _, ok := loaded.Dependencies["other@acme:1.0.0"]; !ok {
		t.Error("expected other dep to remain")
	}
}

func TestRemovePackageDir(t *testing.T) {
	// Test that we can remove an installed package directory.
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "packages", "react-patterns@community")
	if err := os.MkdirAll(pkgDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}

	// Write a dummy file in it.
	dummyFile := filepath.Join(pkgDir, "codectx.yml")
	if err := os.WriteFile(dummyFile, []byte("name: react-patterns"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	// Remove it.
	if err := os.RemoveAll(pkgDir); err != nil {
		t.Fatalf("removing package dir: %v", err)
	}

	if _, err := os.Stat(pkgDir); !os.IsNotExist(err) {
		t.Error("expected package directory to be removed")
	}
}
