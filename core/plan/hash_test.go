package plan

import (
	"testing"

	"github.com/securacore/codectx/core/manifest"
)

func TestCheckDependenciesAllMatch(t *testing.T) {
	t.Parallel()

	deps := []Dependency{
		{Path: "foundation/overview", Hash: "sha256:abc123"},
		{Path: "topics/auth/jwt", Hash: "sha256:def456"},
	}

	hashes := &manifest.Hashes{
		Files: map[string]string{
			"foundation/overview.md": "sha256:abc123",
			"topics/auth/jwt.md":     "sha256:def456",
		},
	}

	result := CheckDependencies(deps, hashes)
	if !result.AllMatch {
		t.Error("AllMatch should be true")
	}
	if result.ChangedCount != 0 {
		t.Errorf("ChangedCount = %d, want 0", result.ChangedCount)
	}
	if result.MissingCount != 0 {
		t.Errorf("MissingCount = %d, want 0", result.MissingCount)
	}
	if len(result.Statuses) != 2 {
		t.Fatalf("Statuses count = %d, want 2", len(result.Statuses))
	}
	for _, s := range result.Statuses {
		if s.Changed {
			t.Errorf("dependency %q should not be changed", s.Dependency.Path)
		}
	}
}

func TestCheckDependenciesWithDrift(t *testing.T) {
	t.Parallel()

	deps := []Dependency{
		{Path: "foundation/overview", Hash: "sha256:abc123"},
		{Path: "topics/auth/jwt", Hash: "sha256:old-hash"},
	}

	hashes := &manifest.Hashes{
		Files: map[string]string{
			"foundation/overview.md": "sha256:abc123",
			"topics/auth/jwt.md":     "sha256:new-hash",
		},
	}

	result := CheckDependencies(deps, hashes)
	if result.AllMatch {
		t.Error("AllMatch should be false")
	}
	if result.ChangedCount != 1 {
		t.Errorf("ChangedCount = %d, want 1", result.ChangedCount)
	}
	if result.MissingCount != 0 {
		t.Errorf("MissingCount = %d, want 0", result.MissingCount)
	}

	// First dependency unchanged.
	if result.Statuses[0].Changed {
		t.Error("first dependency should not be changed")
	}
	// Second dependency changed.
	if !result.Statuses[1].Changed {
		t.Error("second dependency should be changed")
	}
	if result.Statuses[1].CurrentHash != "sha256:new-hash" {
		t.Errorf("CurrentHash = %q, want %q", result.Statuses[1].CurrentHash, "sha256:new-hash")
	}
}

func TestCheckDependenciesMissing(t *testing.T) {
	t.Parallel()

	deps := []Dependency{
		{Path: "foundation/overview", Hash: "sha256:abc123"},
		{Path: "topics/nonexistent", Hash: "sha256:xyz"},
	}

	hashes := &manifest.Hashes{
		Files: map[string]string{
			"foundation/overview.md": "sha256:abc123",
		},
	}

	result := CheckDependencies(deps, hashes)
	if result.AllMatch {
		t.Error("AllMatch should be false")
	}
	if result.MissingCount != 1 {
		t.Errorf("MissingCount = %d, want 1", result.MissingCount)
	}
	if !result.Statuses[1].Missing {
		t.Error("second dependency should be marked missing")
	}
	if !result.Statuses[1].Changed {
		t.Error("missing dependency should also be marked changed")
	}
}

func TestCheckDependenciesEmpty(t *testing.T) {
	t.Parallel()

	result := CheckDependencies(nil, &manifest.Hashes{Files: map[string]string{}})
	if !result.AllMatch {
		t.Error("AllMatch should be true for empty dependencies")
	}
	if len(result.Statuses) != 0 {
		t.Errorf("Statuses should be empty, got %d", len(result.Statuses))
	}
}

func TestResolveHashExactMatch(t *testing.T) {
	t.Parallel()

	hashes := &manifest.Hashes{
		Files: map[string]string{
			"foundation/overview": "sha256:exact",
		},
	}

	hash, found := resolveHash("foundation/overview", hashes)
	if !found {
		t.Fatal("expected to find exact match")
	}
	if hash != "sha256:exact" {
		t.Errorf("hash = %q, want %q", hash, "sha256:exact")
	}
}

func TestResolveHashMdExtension(t *testing.T) {
	t.Parallel()

	hashes := &manifest.Hashes{
		Files: map[string]string{
			"topics/auth/jwt.md": "sha256:md-match",
		},
	}

	hash, found := resolveHash("topics/auth/jwt", hashes)
	if !found {
		t.Fatal("expected to find .md match")
	}
	if hash != "sha256:md-match" {
		t.Errorf("hash = %q, want %q", hash, "sha256:md-match")
	}
}

func TestResolveHashReadme(t *testing.T) {
	t.Parallel()

	hashes := &manifest.Hashes{
		Files: map[string]string{
			"topics/auth/README.md": "sha256:readme-match",
		},
	}

	hash, found := resolveHash("topics/auth", hashes)
	if !found {
		t.Fatal("expected to find README.md match")
	}
	if hash != "sha256:readme-match" {
		t.Errorf("hash = %q, want %q", hash, "sha256:readme-match")
	}
}

func TestResolveHashPrefixMatch(t *testing.T) {
	t.Parallel()

	hashes := &manifest.Hashes{
		Files: map[string]string{
			"topics/auth/jwt-tokens/overview.md": "sha256:prefix-match",
		},
	}

	hash, found := resolveHash("topics/auth/jwt-tokens", hashes)
	if !found {
		t.Fatal("expected to find prefix match")
	}
	if hash != "sha256:prefix-match" {
		t.Errorf("hash = %q, want %q", hash, "sha256:prefix-match")
	}
}

func TestResolveHashPrefixMatchDeterministic(t *testing.T) {
	t.Parallel()

	// Multiple files under the same prefix — should return the lexically first.
	hashes := &manifest.Hashes{
		Files: map[string]string{
			"topics/auth/jwt-tokens/overview.md":  "sha256:overview-hash",
			"topics/auth/jwt-tokens/advanced.md":  "sha256:advanced-hash",
			"topics/auth/jwt-tokens/zzz-extra.md": "sha256:zzz-hash",
		},
	}

	hash, found := resolveHash("topics/auth/jwt-tokens", hashes)
	if !found {
		t.Fatal("expected to find prefix match")
	}
	// Should consistently return the lexically first path's hash.
	if hash != "sha256:advanced-hash" {
		t.Errorf("hash = %q, want %q (lexically first path)", hash, "sha256:advanced-hash")
	}
}

func TestResolveHashNotFound(t *testing.T) {
	t.Parallel()

	hashes := &manifest.Hashes{
		Files: map[string]string{
			"topics/other.md": "sha256:other",
		},
	}

	_, found := resolveHash("topics/nonexistent", hashes)
	if found {
		t.Error("expected not to find nonexistent path")
	}
}
