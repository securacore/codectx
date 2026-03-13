package registry

import (
	"context"
	"testing"

	"github.com/securacore/codectx/core/project"
)

// mockTagLister provides canned tag responses for testing.
type mockTagLister struct {
	// tags maps "name@author" to available tags.
	tags map[string][]string
}

func (m *mockTagLister) AvailableTags(_ context.Context, dk DepKey, _ string) ([]string, error) {
	ref := dk.PackageRef()
	if tags, ok := m.tags[ref]; ok {
		return tags, nil
	}
	return nil, nil
}

// mockConfigReader provides canned package dependency responses for testing.
type mockConfigReader struct {
	// deps maps "name@author:version" to its transitive dependency map.
	deps map[string]map[string]string
}

func (m *mockConfigReader) ReadDeps(_ context.Context, dk DepKey, version string, _ string) (map[string]string, error) {
	key := dk.PackageRef() + ":" + version
	if deps, ok := m.deps[key]; ok {
		return deps, nil
	}
	return nil, nil
}

func TestResolve_DirectDeps(t *testing.T) {
	tags := &mockTagLister{
		tags: map[string][]string{
			"react-patterns@community": {"v2.3.1", "v2.3.0", "v2.2.0", "v1.0.0"},
			"company-standards@acme":   {"v2.0.0", "v1.0.0"},
		},
	}
	configs := &mockConfigReader{deps: map[string]map[string]string{}}

	deps := map[string]*project.DependencyConfig{
		"react-patterns@community:latest": {Active: true},
		"company-standards@acme:2.0.0":    {Active: true},
	}

	result, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(result.Packages))
	}

	rp := result.Packages["react-patterns@community"]
	if rp == nil {
		t.Fatal("react-patterns@community not found")
	}
	if rp.ResolvedVersion != "2.3.1" {
		t.Errorf("react-patterns version: got %q, want %q", rp.ResolvedVersion, "2.3.1")
	}
	if rp.Source != SourceDirect {
		t.Errorf("react-patterns source: got %q, want %q", rp.Source, SourceDirect)
	}

	cs := result.Packages["company-standards@acme"]
	if cs == nil {
		t.Fatal("company-standards@acme not found")
	}
	if cs.ResolvedVersion != "2.0.0" {
		t.Errorf("company-standards version: got %q, want %q", cs.ResolvedVersion, "2.0.0")
	}

	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(result.Conflicts))
	}
}

func TestResolve_TransitiveDeps(t *testing.T) {
	tags := &mockTagLister{
		tags: map[string][]string{
			"react-patterns@community":          {"v2.3.1", "v2.3.0"},
			"javascript-fundamentals@community": {"v1.3.0", "v1.2.0", "v1.1.0"},
		},
	}
	configs := &mockConfigReader{
		deps: map[string]map[string]string{
			"react-patterns@community:2.3.1": {
				"javascript-fundamentals@community": ">=1.0.0",
			},
		},
	}

	deps := map[string]*project.DependencyConfig{
		"react-patterns@community:latest": {Active: true},
	}

	result, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Packages) != 2 {
		t.Fatalf("expected 2 packages (1 direct + 1 transitive), got %d", len(result.Packages))
	}

	jf := result.Packages["javascript-fundamentals@community"]
	if jf == nil {
		t.Fatal("javascript-fundamentals@community not found")
	}
	if jf.ResolvedVersion != "1.3.0" {
		t.Errorf("js-fundamentals version: got %q, want %q", jf.ResolvedVersion, "1.3.0")
	}
	if jf.Source != SourceTransitive {
		t.Errorf("js-fundamentals source: got %q, want %q", jf.Source, SourceTransitive)
	}
	if len(jf.RequiredBy) != 1 || jf.RequiredBy[0] != "react-patterns@community:2.3.1" {
		t.Errorf("js-fundamentals required_by: got %v", jf.RequiredBy)
	}
}

func TestResolve_SharedTransitive(t *testing.T) {
	// Two direct deps both depend on the same transitive dep.
	// Should resolve to the highest compatible version.
	tags := &mockTagLister{
		tags: map[string][]string{
			"react-patterns@community":          {"v2.0.0"},
			"vue-patterns@community":            {"v1.0.0"},
			"javascript-fundamentals@community": {"v1.3.0", "v1.2.0", "v1.1.0"},
		},
	}
	configs := &mockConfigReader{
		deps: map[string]map[string]string{
			"react-patterns@community:2.0.0": {
				"javascript-fundamentals@community": ">=1.1.0",
			},
			"vue-patterns@community:1.0.0": {
				"javascript-fundamentals@community": ">=1.2.0",
			},
		},
	}

	deps := map[string]*project.DependencyConfig{
		"react-patterns@community:latest": {Active: true},
		"vue-patterns@community:latest":   {Active: true},
	}

	result, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 packages total.
	if len(result.Packages) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(result.Packages))
	}

	jf := result.Packages["javascript-fundamentals@community"]
	if jf == nil {
		t.Fatal("javascript-fundamentals@community not found")
	}
	// Should resolve to 1.3.0 (highest compatible with both >=1.1.0 and >=1.2.0).
	if jf.ResolvedVersion != "1.3.0" {
		t.Errorf("js-fundamentals version: got %q, want %q", jf.ResolvedVersion, "1.3.0")
	}
	if jf.Source != SourceTransitive {
		t.Errorf("js-fundamentals source: got %q, want %q", jf.Source, SourceTransitive)
	}

	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(result.Conflicts))
	}
}

func TestResolve_IncompatibleConflict(t *testing.T) {
	// Two packages require different major versions of the same dep.
	tags := &mockTagLister{
		tags: map[string][]string{
			"pkg-a@org":    {"v1.0.0"},
			"pkg-b@org":    {"v1.0.0"},
			"shared@other": {"v2.0.0", "v1.0.0"},
		},
	}
	configs := &mockConfigReader{
		deps: map[string]map[string]string{
			"pkg-a@org:1.0.0": {
				"shared@other": ">=1.0.0",
			},
			"pkg-b@org:1.0.0": {
				"shared@other": ">=2.0.0",
			},
		},
	}

	deps := map[string]*project.DependencyConfig{
		"pkg-a@org:latest": {Active: true},
		"pkg-b@org:latest": {Active: true},
	}

	result, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect a conflict on shared@other.
	if len(result.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(result.Conflicts))
	}
	if result.Conflicts[0].PackageRef != "shared@other" {
		t.Errorf("conflict ref: got %q, want %q", result.Conflicts[0].PackageRef, "shared@other")
	}
}

func TestResolve_DirectOverridesTransitive(t *testing.T) {
	// Direct dep takes precedence over transitive.
	tags := &mockTagLister{
		tags: map[string][]string{
			"react-patterns@community":          {"v2.0.0"},
			"javascript-fundamentals@community": {"v1.3.0", "v1.2.0"},
		},
	}
	configs := &mockConfigReader{
		deps: map[string]map[string]string{
			"react-patterns@community:2.0.0": {
				"javascript-fundamentals@community": ">=1.0.0",
			},
		},
	}

	// User has both as direct deps.
	deps := map[string]*project.DependencyConfig{
		"react-patterns@community:latest":         {Active: true},
		"javascript-fundamentals@community:1.2.0": {Active: true},
	}

	result, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jf := result.Packages["javascript-fundamentals@community"]
	if jf == nil {
		t.Fatal("javascript-fundamentals@community not found")
	}
	// Direct dep pins to 1.2.0, should stay as direct.
	if jf.ResolvedVersion != "1.2.0" {
		t.Errorf("version: got %q, want %q", jf.ResolvedVersion, "1.2.0")
	}
	if jf.Source != SourceDirect {
		t.Errorf("source: got %q, want %q", jf.Source, SourceDirect)
	}
}

func TestResolve_NoDeps(t *testing.T) {
	tags := &mockTagLister{tags: map[string][]string{}}
	configs := &mockConfigReader{deps: map[string]map[string]string{}}

	deps := map[string]*project.DependencyConfig{}

	result, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(result.Packages))
	}
}

func TestResolve_InvalidDepKey(t *testing.T) {
	tags := &mockTagLister{tags: map[string][]string{}}
	configs := &mockConfigReader{deps: map[string]map[string]string{}}

	deps := map[string]*project.DependencyConfig{
		"invalid-key": {Active: true},
	}

	_, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err == nil {
		t.Fatal("expected error for invalid dep key")
	}
}

func TestToLockFile(t *testing.T) {
	result := &ResolveResult{
		Packages: map[string]*ResolvedPackage{
			"react-patterns@community": {
				Key:             DepKey{Name: "react-patterns", Author: "community", Version: "2.3.1"},
				ResolvedVersion: "2.3.1",
				ResolvedTag:     "v2.3.1",
				Source:          SourceDirect,
			},
			"javascript-fundamentals@community": {
				Key:             DepKey{Name: "javascript-fundamentals", Author: "community", Version: "1.3.0"},
				ResolvedVersion: "1.3.0",
				ResolvedTag:     "v1.3.0",
				Source:          SourceTransitive,
				RequiredBy:      []string{"react-patterns@community:2.3.1"},
			},
		},
	}

	commitSHAs := map[string]string{
		"react-patterns@community":          "abc123",
		"javascript-fundamentals@community": "def456",
	}

	lf := ToLockFile(result, commitSHAs, "github.com")

	if lf.LockfileVersion != LockVersion {
		t.Errorf("lockfile version: got %d, want %d", lf.LockfileVersion, LockVersion)
	}

	if len(lf.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(lf.Packages))
	}

	rp := lf.Packages["react-patterns@community"]
	if rp == nil {
		t.Fatal("react-patterns@community not found in lock")
	}
	if rp.ResolvedVersion != "2.3.1" {
		t.Errorf("version: got %q, want %q", rp.ResolvedVersion, "2.3.1")
	}
	if rp.Repo != "github.com/community/codectx-react-patterns" {
		t.Errorf("repo: got %q, want %q", rp.Repo, "github.com/community/codectx-react-patterns")
	}
	if rp.Commit != "abc123" {
		t.Errorf("commit: got %q, want %q", rp.Commit, "abc123")
	}
	if rp.Source != SourceDirect {
		t.Errorf("source: got %q, want %q", rp.Source, SourceDirect)
	}

	jf := lf.Packages["javascript-fundamentals@community"]
	if jf == nil {
		t.Fatal("javascript-fundamentals@community not found in lock")
	}
	if jf.Source != SourceTransitive {
		t.Errorf("source: got %q, want %q", jf.Source, SourceTransitive)
	}
	if len(jf.RequiredBy) != 1 || jf.RequiredBy[0] != "react-patterns@community:2.3.1" {
		t.Errorf("required_by: got %v", jf.RequiredBy)
	}
}

func TestResolve_TagListingReturnsNoTags(t *testing.T) {
	// When the tag lister returns no tags for a direct dependency,
	// resolution should fail.
	tags := &mockTagLister{
		tags: map[string][]string{
			"react-patterns@community": {},
		},
	}
	configs := &mockConfigReader{deps: map[string]map[string]string{}}

	deps := map[string]*project.DependencyConfig{
		"react-patterns@community:latest": {Active: true},
	}

	_, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err == nil {
		t.Fatal("expected error when tag list is empty for a direct dep")
	}
}

func TestResolve_TransitiveResolutionFails(t *testing.T) {
	// When a transitive dependency has no valid semver tags, it should not
	// be added to the result (silently skipped, not a hard error).
	tags := &mockTagLister{
		tags: map[string][]string{
			"react-patterns@community": {"v2.0.0"},
			"bad-dep@org":              {"not-semver", "also-invalid"},
		},
	}
	configs := &mockConfigReader{
		deps: map[string]map[string]string{
			"react-patterns@community:2.0.0": {
				"bad-dep@org": ">=1.0.0",
			},
		},
	}

	deps := map[string]*project.DependencyConfig{
		"react-patterns@community:latest": {Active: true},
	}

	result, err := Resolve(context.Background(), deps, "github.com", tags, configs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Direct dep should still be resolved.
	if _, ok := result.Packages["react-patterns@community"]; !ok {
		t.Error("direct dep react-patterns@community should still be resolved")
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{">=1.0.0", "1.0.0"},
		{"latest", "latest"},
		{"2.3.1", "2.3.1"},
	}
	for _, tt := range tests {
		got := extractVersion(tt.input)
		if got != tt.want {
			t.Errorf("extractVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSemverCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},  // normalized
		{"1.9.0", "1.10.0", -1}, // proper semver, not string comparison
		{"v0.0.1", "v0.0.2", -1},
		{"v10.0.0", "v9.0.0", 1},
	}

	for _, tt := range tests {
		got := semverCompare(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("semverCompare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAppendUnique(t *testing.T) {
	s := []string{"a", "b"}
	s = appendUnique(s, "c")
	if len(s) != 3 {
		t.Errorf("expected 3 elements, got %d", len(s))
	}
	s = appendUnique(s, "b") // duplicate
	if len(s) != 3 {
		t.Errorf("expected 3 elements after duplicate, got %d", len(s))
	}
}
