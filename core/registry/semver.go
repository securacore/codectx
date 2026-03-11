package registry

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

// ResolveVersion finds the best matching version from a list of available
// version tags given a constraint string.
//
// Constraint types:
//   - "latest": returns the highest valid semver version
//   - Exact version (e.g. "2.3.1"): matches exactly (with or without "v" prefix)
//   - Minimum range (e.g. ">=1.0.0"): returns highest version >= the minimum
//
// Available tags should include the "v" prefix (e.g. "v2.3.1").
// Returns the matching tag (with "v" prefix) or an error if no match is found.
func ResolveVersion(available []string, constraint string) (string, error) {
	// Filter to valid semver tags and sort descending (highest first).
	valid := filterValidSemver(available)
	if len(valid) == 0 {
		return "", fmt.Errorf("no valid semver tags found")
	}

	// Sort descending.
	sort.Slice(valid, func(i, j int) bool {
		return semver.Compare(valid[i], valid[j]) > 0
	})

	switch {
	case constraint == LatestVersion:
		return valid[0], nil

	case strings.HasPrefix(constraint, ">="):
		return resolveMinVersion(valid, constraint[2:])

	default:
		return resolveExactVersion(valid, constraint)
	}
}

// resolveExactVersion finds an exact version match.
func resolveExactVersion(sorted []string, version string) (string, error) {
	target := normalizeTag(version)
	if !semver.IsValid(target) {
		return "", fmt.Errorf("invalid version %q", version)
	}
	for _, tag := range sorted {
		if semver.Compare(tag, target) == 0 {
			return tag, nil
		}
	}
	return "", fmt.Errorf("version %q not found in available tags", version)
}

// resolveMinVersion finds the highest version that is >= the minimum.
func resolveMinVersion(sorted []string, minVersion string) (string, error) {
	minTag := normalizeTag(minVersion)
	if !semver.IsValid(minTag) {
		return "", fmt.Errorf("invalid minimum version %q", minVersion)
	}
	for _, tag := range sorted {
		if semver.Compare(tag, minTag) >= 0 {
			return tag, nil
		}
	}
	return "", fmt.Errorf("no version found >= %q", minVersion)
}

// filterValidSemver filters a list of tags to only those that are valid semver.
// Normalizes tags to include the "v" prefix if missing.
func filterValidSemver(tags []string) []string {
	var valid []string
	for _, tag := range tags {
		normalized := normalizeTag(tag)
		if semver.IsValid(normalized) {
			valid = append(valid, normalized)
		}
	}
	return valid
}

// normalizeTag ensures a version string has the "v" prefix required by
// the Go semver package.
func normalizeTag(tag string) string {
	if strings.HasPrefix(tag, "v") {
		return tag
	}
	return "v" + tag
}

// VersionsCompatible checks if two semver version constraints are compatible.
// Currently checks if they share the same major version (documentation packages
// are additive, so minor/patch differences are acceptable).
//
// Both versions should be plain version strings (e.g. "2.3.1", not ranges).
func VersionsCompatible(v1, v2 string) bool {
	t1 := normalizeTag(v1)
	t2 := normalizeTag(v2)
	if !semver.IsValid(t1) || !semver.IsValid(t2) {
		return false
	}
	return semver.Major(t1) == semver.Major(t2)
}
