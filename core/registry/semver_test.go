package registry

import (
	"testing"
)

func TestResolveVersionLatest(t *testing.T) {
	t.Parallel()

	available := []string{"v1.0.0", "v2.3.1", "v2.4.0", "v1.5.0"}
	got, err := ResolveVersion(available, "latest")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "v2.4.0" {
		t.Errorf("got %q, want %q", got, "v2.4.0")
	}
}

func TestResolveVersionExact(t *testing.T) {
	t.Parallel()

	available := []string{"v1.0.0", "v2.3.1", "v2.4.0"}
	got, err := ResolveVersion(available, "2.3.1")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "v2.3.1" {
		t.Errorf("got %q, want %q", got, "v2.3.1")
	}
}

func TestResolveVersionExactWithPrefix(t *testing.T) {
	t.Parallel()

	available := []string{"v1.0.0", "v2.3.1"}
	got, err := ResolveVersion(available, "v2.3.1")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "v2.3.1" {
		t.Errorf("got %q, want %q", got, "v2.3.1")
	}
}

func TestResolveVersionMinRange(t *testing.T) {
	t.Parallel()

	available := []string{"v0.9.0", "v1.0.0", "v1.3.0", "v2.0.0"}
	got, err := ResolveVersion(available, ">=1.0.0")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	// Should return highest >= 1.0.0.
	if got != "v2.0.0" {
		t.Errorf("got %q, want %q", got, "v2.0.0")
	}
}

func TestResolveVersionMinRangeNoMatch(t *testing.T) {
	t.Parallel()

	available := []string{"v0.5.0", "v0.9.0"}
	_, err := ResolveVersion(available, ">=1.0.0")
	if err == nil {
		t.Error("expected error when no version >= minimum")
	}
}

func TestResolveVersionExactNotFound(t *testing.T) {
	t.Parallel()

	available := []string{"v1.0.0", "v2.0.0"}
	_, err := ResolveVersion(available, "3.0.0")
	if err == nil {
		t.Error("expected error when exact version not found")
	}
}

func TestResolveVersionNoValidTags(t *testing.T) {
	t.Parallel()

	available := []string{"not-semver", "abc", "release-1"}
	_, err := ResolveVersion(available, "latest")
	if err == nil {
		t.Error("expected error when no valid semver tags")
	}
}

func TestResolveVersionEmptyTags(t *testing.T) {
	t.Parallel()

	_, err := ResolveVersion(nil, "latest")
	if err == nil {
		t.Error("expected error for empty tag list")
	}
}

func TestResolveVersionMixedTags(t *testing.T) {
	t.Parallel()

	// Mix of valid and invalid tags.
	available := []string{"v1.0.0", "not-valid", "v2.0.0", "release-3"}
	got, err := ResolveVersion(available, "latest")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "v2.0.0" {
		t.Errorf("got %q, want %q", got, "v2.0.0")
	}
}

func TestFilterValidSemver(t *testing.T) {
	t.Parallel()

	tags := []string{"v1.0.0", "invalid", "v2.3.1", "release", "1.5.0"}
	valid := filterValidSemver(tags)
	if len(valid) != 3 {
		t.Fatalf("len = %d, want 3: %v", len(valid), valid)
	}
}

func TestVersionsCompatible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v1, v2 string
		want   bool
	}{
		{"1.0.0", "1.5.0", true},
		{"2.3.1", "2.4.0", true},
		{"1.0.0", "2.0.0", false},
		{"v1.0.0", "1.9.9", true},
		{"invalid", "1.0.0", false},
		{"1.0.0", "invalid", false},
	}

	for _, tt := range tests {
		got := VersionsCompatible(tt.v1, tt.v2)
		if got != tt.want {
			t.Errorf("VersionsCompatible(%q, %q) = %v, want %v", tt.v1, tt.v2, got, tt.want)
		}
	}
}
