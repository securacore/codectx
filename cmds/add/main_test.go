package add

import (
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
)

func TestToSemverRange_Latest(t *testing.T) {
	got := toSemverRange("latest")
	if got != "latest" {
		t.Errorf("toSemverRange(\"latest\") = %q, want %q", got, "latest")
	}
}

func TestToSemverRange_PlainVersion(t *testing.T) {
	got := toSemverRange("2.3.1")
	if got != ">=2.3.1" {
		t.Errorf("toSemverRange(\"2.3.1\") = %q, want %q", got, ">=2.3.1")
	}
}

func TestToSemverRange_AlreadyRange(t *testing.T) {
	got := toSemverRange(">=1.0.0")
	if got != ">=1.0.0" {
		t.Errorf("toSemverRange(\">=1.0.0\") = %q, want %q", got, ">=1.0.0")
	}
}

func TestToSemverRange_LessThan(t *testing.T) {
	got := toSemverRange("<3.0.0")
	if got != "<3.0.0" {
		t.Errorf("toSemverRange(\"<3.0.0\") = %q, want %q", got, "<3.0.0")
	}
}

func TestIsDuplicate_Found(t *testing.T) {
	cfg := &project.Config{
		Dependencies: map[string]*project.DependencyConfig{
			"react-patterns@community:latest": {Active: true},
		},
	}

	if !isDuplicate(cfg, "react-patterns@community") {
		t.Error("expected isDuplicate to return true for existing dep")
	}
}

func TestIsDuplicate_NotFound(t *testing.T) {
	cfg := &project.Config{
		Dependencies: map[string]*project.DependencyConfig{
			"react-patterns@community:latest": {Active: true},
		},
	}

	if isDuplicate(cfg, "other-pkg@acme") {
		t.Error("expected isDuplicate to return false for non-existing dep")
	}
}

func TestIsDuplicate_EmptyDeps(t *testing.T) {
	cfg := &project.Config{
		Dependencies: map[string]*project.DependencyConfig{},
	}

	if isDuplicate(cfg, "react-patterns@community") {
		t.Error("expected isDuplicate to return false for empty deps")
	}
}

func TestIsDuplicate_NilDeps(t *testing.T) {
	cfg := &project.Config{}

	if isDuplicate(cfg, "react-patterns@community") {
		t.Error("expected isDuplicate to return false for nil deps")
	}
}

func TestAddToProjectConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", "")
	cfgPath := dir + "/" + project.ConfigFileName
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	err := addToProjectConfig(dir, &cfg, "react-patterns@community:latest", true)
	if err != nil {
		t.Fatalf("addToProjectConfig: %v", err)
	}

	// Reload and verify.
	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	dep, ok := loaded.Dependencies["react-patterns@community:latest"]
	if !ok {
		t.Fatal("expected dependency to be added")
	}
	if !dep.Active {
		t.Error("expected dependency to be active")
	}
}

func TestAddToProjectConfig_Inactive(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", "")
	cfgPath := dir + "/" + project.ConfigFileName
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	err := addToProjectConfig(dir, &cfg, "react-patterns@community:latest", false)
	if err != nil {
		t.Fatalf("addToProjectConfig: %v", err)
	}

	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	dep, ok := loaded.Dependencies["react-patterns@community:latest"]
	if !ok {
		t.Fatal("expected dependency to be added")
	}
	if dep.Active {
		t.Error("expected dependency to be inactive")
	}
}

func TestAddToPackageManifest(t *testing.T) {
	dir := t.TempDir()

	// Create package/codectx.yml.
	manifest := project.DefaultPackageManifest("test-pkg", "testorg", "A test package")
	manifestPath := project.PackageConfigPath(dir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		t.Fatal(err)
	}

	dk := registry.DepKey{Name: "react-patterns", Org: "community", Version: "2.0.0"}
	if err := addToPackageManifest(dir, dk); err != nil {
		t.Fatalf("addToPackageManifest: %v", err)
	}

	// Reload and verify.
	loaded, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	constraint, ok := loaded.Dependencies["react-patterns@community"]
	if !ok {
		t.Fatal("expected dependency to be added to package manifest")
	}
	if constraint != ">=2.0.0" {
		t.Errorf("expected constraint %q, got %q", ">=2.0.0", constraint)
	}
}

func TestAddToPackageManifest_Latest(t *testing.T) {
	dir := t.TempDir()

	manifest := project.DefaultPackageManifest("test-pkg", "testorg", "")
	manifestPath := project.PackageConfigPath(dir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		t.Fatal(err)
	}

	dk := registry.DepKey{Name: "standards", Org: "acme", Version: "latest"}
	if err := addToPackageManifest(dir, dk); err != nil {
		t.Fatalf("addToPackageManifest: %v", err)
	}

	loaded, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	constraint, ok := loaded.Dependencies["standards@acme"]
	if !ok {
		t.Fatal("expected dependency to be added to package manifest")
	}
	if constraint != "latest" {
		t.Errorf("expected constraint %q, got %q", "latest", constraint)
	}
}

func TestApplyDependency_TargetProject(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", "")
	cfgPath := dir + "/" + project.ConfigFileName
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	dk := registry.DepKey{Name: "react-patterns", Org: "community", Version: "latest"}
	if err := applyDependency(dir, &cfg, dk, targetProject, false); err != nil {
		t.Fatalf("applyDependency: %v", err)
	}

	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	if _, ok := loaded.Dependencies["react-patterns@community:latest"]; !ok {
		t.Error("expected dependency in root config")
	}
}

func TestApplyDependency_TargetPackage(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", project.TypePackage)
	cfgPath := dir + "/" + project.ConfigFileName
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	// Create package manifest.
	manifest := project.DefaultPackageManifest("test", "testorg", "")
	manifestPath := project.PackageConfigPath(dir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		t.Fatal(err)
	}

	dk := registry.DepKey{Name: "react-patterns", Org: "community", Version: "2.0.0"}
	if err := applyDependency(dir, &cfg, dk, targetPackage, false); err != nil {
		t.Fatalf("applyDependency: %v", err)
	}

	// Verify root config (should be inactive).
	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	dep, ok := loaded.Dependencies["react-patterns@community:2.0.0"]
	if !ok {
		t.Fatal("expected dependency in root config for install")
	}
	if dep.Active {
		t.Error("expected dependency to be inactive in root config for package-only target")
	}

	// Verify package manifest.
	loadedManifest, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	constraint, ok := loadedManifest.Dependencies["react-patterns@community"]
	if !ok {
		t.Fatal("expected dependency in package manifest")
	}
	if constraint != ">=2.0.0" {
		t.Errorf("expected constraint %q, got %q", ">=2.0.0", constraint)
	}
}

func TestApplyDependency_TargetBoth(t *testing.T) {
	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", project.TypePackage)
	cfgPath := dir + "/" + project.ConfigFileName
	if err := cfg.WriteToFile(cfgPath); err != nil {
		t.Fatal(err)
	}

	manifest := project.DefaultPackageManifest("test", "testorg", "")
	manifestPath := project.PackageConfigPath(dir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		t.Fatal(err)
	}

	dk := registry.DepKey{Name: "react-patterns", Org: "community", Version: "1.5.0"}
	if err := applyDependency(dir, &cfg, dk, targetBoth, false); err != nil {
		t.Fatalf("applyDependency: %v", err)
	}

	// Verify root config (should be active).
	loaded, err := project.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	dep, ok := loaded.Dependencies["react-patterns@community:1.5.0"]
	if !ok {
		t.Fatal("expected dependency in root config")
	}
	if !dep.Active {
		t.Error("expected dependency to be active for both target")
	}

	// Verify package manifest.
	loadedManifest, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	constraint, ok := loadedManifest.Dependencies["react-patterns@community"]
	if !ok {
		t.Fatal("expected dependency in package manifest")
	}
	if constraint != ">=1.5.0" {
		t.Errorf("expected constraint %q, got %q", ">=1.5.0", constraint)
	}
}
