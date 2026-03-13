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

	dk := registry.DepKey{Name: "react-patterns", Author: "community", Version: "2.0.0"}
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

	dk := registry.DepKey{Name: "standards", Author: "acme", Version: "latest"}
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

	dk := registry.DepKey{Name: "react-patterns", Author: "community", Version: "latest"}
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

	dk := registry.DepKey{Name: "react-patterns", Author: "community", Version: "2.0.0"}
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

// --- Tests for new flexible parsing helpers ---

func TestFilterExactName(t *testing.T) {
	results := []registry.SearchResult{
		{Name: "react", Author: "community"},
		{Name: "react-patterns", Author: "community"},
		{Name: "react", Author: "acme"},
		{Name: "react-testing", Author: "someone"},
	}

	filtered := filterExactName(results, "react")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 results, got %d", len(filtered))
	}
	if filtered[0].Author != "community" {
		t.Errorf("first result author = %q, want %q", filtered[0].Author, "community")
	}
	if filtered[1].Author != "acme" {
		t.Errorf("second result author = %q, want %q", filtered[1].Author, "acme")
	}
}

func TestFilterExactName_NoMatch(t *testing.T) {
	results := []registry.SearchResult{
		{Name: "react-patterns", Author: "community"},
	}

	filtered := filterExactName(results, "react")
	if len(filtered) != 0 {
		t.Fatalf("expected 0 results, got %d", len(filtered))
	}
}

func TestFilterInstallable(t *testing.T) {
	results := []registry.SearchResult{
		{Name: "react", Author: "community", HasRelease: true},
		{Name: "react", Author: "acme", HasRelease: false},
		{Name: "react", Author: "someone", HasRelease: true},
	}

	filtered, hidden := filterInstallable(results)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 installable, got %d", len(filtered))
	}
	if hidden != 1 {
		t.Errorf("expected 1 hidden, got %d", hidden)
	}
	if filtered[0].Author != "community" || filtered[1].Author != "someone" {
		t.Error("unexpected filter order")
	}
}

func TestFilterInstallable_AllInstallable(t *testing.T) {
	results := []registry.SearchResult{
		{Name: "react", Author: "community", HasRelease: true},
	}

	filtered, hidden := filterInstallable(results)
	if len(filtered) != 1 {
		t.Fatalf("expected 1, got %d", len(filtered))
	}
	if hidden != 0 {
		t.Errorf("expected 0 hidden, got %d", hidden)
	}
}

func TestFilterInstallable_NoneInstallable(t *testing.T) {
	results := []registry.SearchResult{
		{Name: "react", Author: "acme", HasRelease: false},
	}

	filtered, hidden := filterInstallable(results)
	if len(filtered) != 0 {
		t.Fatalf("expected 0, got %d", len(filtered))
	}
	if hidden != 1 {
		t.Errorf("expected 1 hidden, got %d", hidden)
	}
}

func TestAuthorSuggestions(t *testing.T) {
	results := []registry.SearchResult{
		{Name: "react", Author: "community"},
		{Name: "react", Author: "acme"},
	}
	partial := registry.PartialDepKey{Name: "react", Version: "2.0.0"}

	suggestions := authorSuggestions(results, partial)
	// 1 header + 2 command suggestions.
	if len(suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(suggestions))
	}
	if suggestions[0].Text != "Specify the author explicitly:" {
		t.Errorf("unexpected header: %q", suggestions[0].Text)
	}
	if suggestions[1].Command != "codectx add react@community:2.0.0" {
		t.Errorf("unexpected command 1: %q", suggestions[1].Command)
	}
	if suggestions[2].Command != "codectx add react@acme:2.0.0" {
		t.Errorf("unexpected command 2: %q", suggestions[2].Command)
	}
}

func TestAuthorSuggestions_NoVersion(t *testing.T) {
	results := []registry.SearchResult{
		{Name: "react", Author: "community"},
	}
	partial := registry.PartialDepKey{Name: "react"}

	suggestions := authorSuggestions(results, partial)
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(suggestions))
	}
	// Should default to "latest" when no version specified.
	if suggestions[1].Command != "codectx add react@community:latest" {
		t.Errorf("unexpected command: %q", suggestions[1].Command)
	}
}

func TestPluralize(t *testing.T) {
	if got := pluralize(1, "package", "packages"); got != "package" {
		t.Errorf("pluralize(1) = %q, want %q", got, "package")
	}
	if got := pluralize(0, "package", "packages"); got != "packages" {
		t.Errorf("pluralize(0) = %q, want %q", got, "packages")
	}
	if got := pluralize(5, "package", "packages"); got != "packages" {
		t.Errorf("pluralize(5) = %q, want %q", got, "packages")
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

	dk := registry.DepKey{Name: "react-patterns", Author: "community", Version: "1.5.0"}
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
