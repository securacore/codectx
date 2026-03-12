// Package registry implements the codectx package manager.
//
// It provides package discovery, resolution, installation, and publishing
// for documentation packages hosted on GitHub. Packages follow the
// codectx-[name] naming convention and use git tags for versioning.
//
// Key types:
//   - DepKey: parsed dependency key (name@org:version)
//   - LockFile: codectx.lock schema for deterministic installs
//   - SearchResult: GitHub search result for package discovery
//
// Key operations:
//   - Resolve: flatten direct + transitive dependencies with semver resolution
//   - Install: clone/checkout packages to .codectx/packages/
//   - Search: query GitHub for codectx packages
//   - Publish: validate structure, tag, and push
package registry

import (
	"fmt"
	"strings"
)

// RepoPrefix is the naming convention prefix for codectx packages on GitHub.
const RepoPrefix = "codectx-"

// LatestVersion is the special version string that resolves to the
// highest semver tag on the repository.
const LatestVersion = "latest"

// DepKey represents a parsed dependency key from codectx.yml.
//
// The full key format is "name@org:version" where:
//   - name: package name (e.g. "react-patterns")
//   - org: GitHub organization or user (e.g. "community")
//   - version: semver version or "latest" (e.g. "2.3.1", "latest")
//
// Examples:
//
//	"react-patterns@community:latest" -> {Name: "react-patterns", Org: "community", Version: "latest"}
//	"company-standards@acme:2.0.0"    -> {Name: "company-standards", Org: "acme", Version: "2.0.0"}
type DepKey struct {
	// Name is the package name without the codectx- prefix.
	Name string

	// Org is the GitHub organization or user.
	Org string

	// Version is the semver version string or "latest".
	Version string
}

// ParseDepKey parses a dependency map key into its components.
// The expected format is "name@org:version".
//
// Returns an error if the key is malformed (missing @ or :).
func ParseDepKey(key string) (DepKey, error) {
	atIdx := strings.Index(key, "@")
	if atIdx < 1 {
		return DepKey{}, fmt.Errorf("invalid dependency key %q: missing '@' separator", key)
	}

	afterAt := key[atIdx+1:]
	colonIdx := strings.Index(afterAt, ":")
	if colonIdx < 1 {
		return DepKey{}, fmt.Errorf("invalid dependency key %q: missing ':' version separator", key)
	}

	name := key[:atIdx]
	org := afterAt[:colonIdx]
	version := afterAt[colonIdx+1:]

	if name == "" {
		return DepKey{}, fmt.Errorf("invalid dependency key %q: empty name", key)
	}
	if org == "" {
		return DepKey{}, fmt.Errorf("invalid dependency key %q: empty org", key)
	}
	if version == "" {
		return DepKey{}, fmt.Errorf("invalid dependency key %q: empty version", key)
	}

	return DepKey{Name: name, Org: org, Version: version}, nil
}

// String returns the canonical key representation "name@org:version".
func (dk DepKey) String() string {
	return dk.Name + "@" + dk.Org + ":" + dk.Version
}

// PackageRef returns the short reference "name@org" without version.
// Used in lock file keys and session context references.
func (dk DepKey) PackageRef() string {
	return dk.Name + "@" + dk.Org
}

// RepoName returns the GitHub repository name using the codectx- prefix convention.
// Example: DepKey{Name: "react-patterns"} -> "codectx-react-patterns"
func (dk DepKey) RepoName() string {
	return RepoPrefix + dk.Name
}

// RepoURL returns the full GitHub repository URL.
// Example: DepKey{Name: "react-patterns", Org: "community"} with registry "github.com"
// returns "https://github.com/community/codectx-react-patterns".
func (dk DepKey) RepoURL(registry string) string {
	return "https://" + registry + "/" + dk.Org + "/" + dk.RepoName()
}

// ParsePackageRef parses a short package reference "name@org" into name and org.
// This format is used in lock file keys and session context references.
func ParsePackageRef(ref string) (name, org string, err error) {
	name, org, found := strings.Cut(ref, "@")
	if !found || name == "" {
		return "", "", fmt.Errorf("invalid package reference %q: missing '@' separator", ref)
	}
	if org == "" {
		return "", "", fmt.Errorf("invalid package reference %q: empty org", ref)
	}

	return name, org, nil
}

// GitTag returns the git tag for a version string.
// Prepends "v" prefix if not already present.
// Example: "2.3.1" -> "v2.3.1", "v2.3.1" -> "v2.3.1"
func GitTag(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

// VersionFromTag strips the "v" prefix from a git tag to get the semver version.
// Example: "v2.3.1" -> "2.3.1", "2.3.1" -> "2.3.1"
func VersionFromTag(tag string) string {
	return strings.TrimPrefix(tag, "v")
}
