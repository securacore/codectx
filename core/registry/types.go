// Package registry implements the codectx package manager.
//
// It provides package discovery, resolution, installation, and publishing
// for documentation packages hosted on GitHub. Packages follow the
// codectx-[name] naming convention and use git tags for versioning.
//
// Key types:
//   - DepKey: parsed dependency key (name@author:version)
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
// The full key format is "name@author:version" where:
//   - name: package name (e.g. "react-patterns")
//   - author: GitHub username or organization (e.g. "community")
//   - version: semver version or "latest" (e.g. "2.3.1", "latest")
//
// Examples:
//
//	"react-patterns@community:latest" -> {Name: "react-patterns", Author: "community", Version: "latest"}
//	"company-standards@acme:2.0.0"    -> {Name: "company-standards", Author: "acme", Version: "2.0.0"}
type DepKey struct {
	// Name is the package name without the codectx- prefix.
	Name string

	// Author is the GitHub username or organization.
	Author string

	// Version is the semver version string or "latest".
	Version string
}

// ParseDepKey parses a dependency map key into its components.
// The expected format is "name@author:version".
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
	author := afterAt[:colonIdx]
	version := afterAt[colonIdx+1:]

	if name == "" {
		return DepKey{}, fmt.Errorf("invalid dependency key %q: empty name", key)
	}
	if author == "" {
		return DepKey{}, fmt.Errorf("invalid dependency key %q: empty author", key)
	}
	if version == "" {
		return DepKey{}, fmt.Errorf("invalid dependency key %q: empty version", key)
	}

	return DepKey{Name: name, Author: author, Version: version}, nil
}

// String returns the canonical key representation "name@author:version".
func (dk DepKey) String() string {
	return dk.Name + "@" + dk.Author + ":" + dk.Version
}

// PackageRef returns the short reference "name@author" without version.
// Used in lock file keys and session context references.
func (dk DepKey) PackageRef() string {
	return dk.Name + "@" + dk.Author
}

// RepoName returns the GitHub repository name using the codectx- prefix convention.
// Example: DepKey{Name: "react-patterns"} -> "codectx-react-patterns"
func (dk DepKey) RepoName() string {
	return RepoPrefix + dk.Name
}

// RepoURL returns the full GitHub repository URL.
// Example: DepKey{Name: "react-patterns", Author: "community"} with registry "github.com"
// returns "https://github.com/community/codectx-react-patterns".
func (dk DepKey) RepoURL(registry string) string {
	return "https://" + registry + "/" + dk.Author + "/" + dk.RepoName()
}

// ParsePackageRef parses a short package reference "name@author" into name and author.
// This format is used in lock file keys and session context references.
func ParsePackageRef(ref string) (name, author string, err error) {
	name, author, found := strings.Cut(ref, "@")
	if !found || name == "" {
		return "", "", fmt.Errorf("invalid package reference %q: missing '@' separator", ref)
	}
	if author == "" {
		return "", "", fmt.Errorf("invalid package reference %q: empty author", ref)
	}

	return name, author, nil
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

// PartialDepKey represents a partially specified dependency reference.
// At minimum, Name is always present. Author and Version may be empty,
// indicating they need to be resolved interactively or via search.
//
// Supported formats:
//
//	"react"                 -> {Name: "react"}
//	"react@community"      -> {Name: "react", Author: "community"}
//	"react@community:2.0"  -> {Name: "react", Author: "community", Version: "2.0"}
//	"react:2.0"            -> {Name: "react", Version: "2.0"}
type PartialDepKey struct {
	// Name is the package name (always present).
	Name string

	// Author is the GitHub username or organization (may be empty).
	Author string

	// Version is the version constraint (may be empty).
	Version string
}

// ParsePartialDepKey parses a flexible dependency reference that may omit
// the author and/or version components. Returns an error only if the
// input is empty or has an empty name component.
func ParsePartialDepKey(input string) (PartialDepKey, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return PartialDepKey{}, fmt.Errorf("empty dependency reference")
	}

	var name, author, version string

	atIdx := strings.Index(input, "@")
	colonIdx := strings.Index(input, ":")

	switch {
	case atIdx >= 0 && colonIdx >= 0 && colonIdx > atIdx:
		// "name@author:version"
		name = input[:atIdx]
		author = input[atIdx+1 : colonIdx]
		version = input[colonIdx+1:]

	case atIdx >= 0 && colonIdx < 0:
		// "name@author"
		name = input[:atIdx]
		author = input[atIdx+1:]

	case atIdx < 0 && colonIdx >= 0:
		// "name:version"
		name = input[:colonIdx]
		version = input[colonIdx+1:]

	default:
		// "name"
		name = input
	}

	if name == "" {
		return PartialDepKey{}, fmt.Errorf("invalid dependency reference %q: empty name", input)
	}

	return PartialDepKey{Name: name, Author: author, Version: version}, nil
}

// IsComplete reports whether the partial key has all components needed
// to construct a full DepKey.
func (pk PartialDepKey) IsComplete() bool {
	return pk.Name != "" && pk.Author != "" && pk.Version != ""
}

// ToDepKey converts a partial key to a full DepKey.
// Missing version defaults to "latest". Author must be non-empty.
func (pk PartialDepKey) ToDepKey() (DepKey, error) {
	if pk.Author == "" {
		return DepKey{}, fmt.Errorf("cannot convert partial key %q to DepKey: missing author", pk.Name)
	}
	version := pk.Version
	if version == "" {
		version = LatestVersion
	}
	return DepKey{Name: pk.Name, Author: pk.Author, Version: version}, nil
}
