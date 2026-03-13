package registry

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/securacore/codectx/core/project"
	"golang.org/x/mod/semver"
)

// TagLister provides version tags for a package. Implemented by GitClient
// (local) and GitHubClient (remote).
type TagLister interface {
	// AvailableTags returns the semver tags for a package identified by author/name.
	// Tags should include the "v" prefix (e.g. "v1.0.0").
	AvailableTags(ctx context.Context, dk DepKey, registry string) ([]string, error)
}

// PackageConfigReader reads the codectx.yml from a resolved package to
// discover transitive dependencies.
type PackageConfigReader interface {
	// ReadDeps returns the dependency map from a package's codectx.yml.
	// The map key is "name@author" and the value is a version constraint string
	// (e.g. ">=1.0.0", "latest", "2.0.0").
	ReadDeps(ctx context.Context, dk DepKey, version string, registry string) (map[string]string, error)
}

// ResolvedPackage represents a single package that has been fully resolved
// to a specific version.
type ResolvedPackage struct {
	// Key is the parsed dependency key.
	Key DepKey

	// ResolvedVersion is the exact version chosen (without "v" prefix).
	ResolvedVersion string

	// ResolvedTag is the git tag (with "v" prefix).
	ResolvedTag string

	// Source is "direct" or "transitive".
	Source string

	// RequiredBy lists the packages that required this transitive dependency.
	// Each entry is "name@author:version". Empty for direct dependencies.
	RequiredBy []string
}

// Conflict represents an incompatible version requirement detected during
// dependency resolution.
type Conflict struct {
	// PackageRef is the "name@author" of the conflicting package.
	PackageRef string

	// Versions maps the requester ("name@author:version" or "direct") to the
	// version constraint they requested.
	Versions map[string]string
}

// ResolveResult contains the full output of dependency resolution.
type ResolveResult struct {
	// Packages is the flat list of resolved packages (direct + transitive).
	Packages map[string]*ResolvedPackage

	// Conflicts lists any incompatible version requirements that could not
	// be automatically resolved.
	Conflicts []Conflict
}

// Resolve performs flat dependency resolution for a project's dependencies.
//
// Algorithm:
//  1. Parse each direct dependency key from codectx.yml
//  2. Resolve each to the best matching version tag
//  3. Read transitive dependencies from each package's codectx.yml
//  4. Recursively resolve transitive deps (BFS, max 10 levels deep)
//  5. For shared transitive deps, pick the highest compatible version
//  6. Report incompatible version conflicts as warnings
//
// The resolver does NOT download packages — it only determines which
// versions to install. The caller (install/update commands) handles
// actual git operations.
func Resolve(
	ctx context.Context,
	deps map[string]*project.DependencyConfig,
	registry string,
	tags TagLister,
	configs PackageConfigReader,
) (*ResolveResult, error) {
	result := &ResolveResult{
		Packages: make(map[string]*ResolvedPackage),
	}

	// Track version constraints per package ref for conflict detection.
	// Key: "name@author", Value: map of requester -> version constraint.
	constraints := make(map[string]map[string]string)

	// Step 1: Resolve direct dependencies.
	for key := range deps {
		dk, err := ParseDepKey(key)
		if err != nil {
			return nil, fmt.Errorf("parsing dependency key %q: %w", key, err)
		}

		resolved, err := resolveOne(ctx, dk, dk.Version, registry, tags)
		if err != nil {
			return nil, fmt.Errorf("resolving %s: %w", key, err)
		}

		ref := dk.PackageRef()
		resolved.Source = SourceDirect
		result.Packages[ref] = resolved

		if constraints[ref] == nil {
			constraints[ref] = make(map[string]string)
		}
		constraints[ref]["direct"] = dk.Version
	}

	// Step 2: Resolve transitive dependencies (BFS, max depth 10).
	const maxDepth = 10
	type queueItem struct {
		pkg   *ResolvedPackage
		depth int
	}

	queue := make([]queueItem, 0, len(result.Packages))
	for _, pkg := range result.Packages {
		queue = append(queue, queueItem{pkg: pkg, depth: 0})
	}

	visited := make(map[string]bool)

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.depth >= maxDepth {
			continue
		}

		ref := item.pkg.Key.PackageRef()
		if visited[ref] {
			continue
		}
		visited[ref] = true

		// Read transitive deps from this package.
		transDeps, err := configs.ReadDeps(ctx, item.pkg.Key, item.pkg.ResolvedVersion, registry)
		if err != nil {
			// Not an error — package may have no dependencies.
			continue
		}

		requiredBy := item.pkg.Key.PackageRef() + ":" + item.pkg.ResolvedVersion

		for depRef, constraint := range transDeps {
			name, author, parseErr := ParsePackageRef(depRef)
			if parseErr != nil {
				continue
			}

			transRef := depRef
			if constraints[transRef] == nil {
				constraints[transRef] = make(map[string]string)
			}
			constraints[transRef][requiredBy] = constraint

			dk := DepKey{Name: name, Author: author, Version: constraint}

			existing, exists := result.Packages[transRef]
			if exists {
				// Already resolved — check compatibility.
				existingTag := normalizeTag(existing.ResolvedVersion)
				newTag := normalizeTag(constraint)

				// If constraint is a range (like ">=1.0.0"), extract the version.
				if strings.HasPrefix(constraint, ">=") {
					newTag = normalizeTag(constraint[2:])
				}

				if !VersionsCompatible(existing.ResolvedVersion, VersionFromTag(newTag)) {
					// Incompatible — will be reported as conflict.
					continue
				}

				// Compatible — keep the higher version.
				// If existing resolved version is >= the new constraint, keep it.
				if semverCompare(existingTag, newTag) >= 0 {
					if existing.Source == SourceTransitive {
						existing.RequiredBy = appendUnique(existing.RequiredBy, requiredBy)
					}
					continue
				}

				// New constraint needs a higher version — re-resolve.
			}

			resolved, resolveErr := resolveOne(ctx, dk, constraint, registry, tags)
			if resolveErr != nil {
				continue
			}

			resolved.Source = SourceTransitive
			resolved.RequiredBy = []string{requiredBy}

			// If a direct dependency exists for this ref, don't override.
			if exists && existing.Source == SourceDirect {
				existing.RequiredBy = appendUnique(existing.RequiredBy, requiredBy)
				continue
			}

			result.Packages[transRef] = resolved
			queue = append(queue, queueItem{pkg: resolved, depth: item.depth + 1})
		}
	}

	// Step 3: Detect conflicts.
	for ref, reqs := range constraints {
		if len(reqs) <= 1 {
			continue
		}

		// Collect all resolved version strings.
		versions := make([]string, 0, len(reqs))
		for _, v := range reqs {
			versions = append(versions, extractVersion(v))
		}

		// Check pairwise compatibility.
		compatible := true
		for i := 0; i < len(versions); i++ {
			for j := i + 1; j < len(versions); j++ {
				if !VersionsCompatible(versions[i], versions[j]) {
					compatible = false
					break
				}
			}
			if !compatible {
				break
			}
		}

		if !compatible {
			result.Conflicts = append(result.Conflicts, Conflict{
				PackageRef: ref,
				Versions:   reqs,
			})
		}
	}

	// Sort conflicts for deterministic output.
	sort.Slice(result.Conflicts, func(i, j int) bool {
		return result.Conflicts[i].PackageRef < result.Conflicts[j].PackageRef
	})

	return result, nil
}

// resolveOne resolves a single dependency to its best matching version tag.
func resolveOne(
	ctx context.Context,
	dk DepKey,
	constraint string,
	registry string,
	tags TagLister,
) (*ResolvedPackage, error) {
	available, err := tags.AvailableTags(ctx, dk, registry)
	if err != nil {
		return nil, fmt.Errorf("listing tags for %s: %w", dk.PackageRef(), err)
	}

	resolvedTag, err := ResolveVersion(available, constraint)
	if err != nil {
		return nil, fmt.Errorf("resolving version %q for %s: %w", constraint, dk.PackageRef(), err)
	}

	return &ResolvedPackage{
		Key:             DepKey{Name: dk.Name, Author: dk.Author, Version: VersionFromTag(resolvedTag)},
		ResolvedVersion: VersionFromTag(resolvedTag),
		ResolvedTag:     resolvedTag,
	}, nil
}

// extractVersion extracts a plain version string from a constraint.
// ">=1.0.0" -> "1.0.0", "latest" -> "latest", "2.3.1" -> "2.3.1"
func extractVersion(constraint string) string {
	if strings.HasPrefix(constraint, ">=") {
		return constraint[2:]
	}
	return constraint
}

// semverCompare compares two version strings using proper semver ordering.
// Returns -1 if a < b, 0 if a == b, +1 if a > b.
// Both inputs are normalized to include the "v" prefix.
func semverCompare(a, b string) int {
	return semver.Compare(normalizeTag(a), normalizeTag(b))
}

// appendUnique appends s to the slice if not already present.
func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}

// ToLockFile converts a ResolveResult into a LockFile.
// The commitSHAs map provides the git commit SHA for each "name@author" ref.
func ToLockFile(result *ResolveResult, commitSHAs map[string]string, registry string) *LockFile {
	lf := &LockFile{
		LockfileVersion: LockVersion,
		ResolvedAt:      time.Now().UTC().Format(time.RFC3339),
		Packages:        make(map[string]*LockedPackage),
	}

	for ref, pkg := range result.Packages {
		lp := &LockedPackage{
			ResolvedVersion: pkg.ResolvedVersion,
			Repo:            registry + "/" + pkg.Key.Author + "/" + RepoPrefix + pkg.Key.Name,
			Commit:          commitSHAs[ref],
			Source:          pkg.Source,
		}
		if len(pkg.RequiredBy) > 0 {
			lp.RequiredBy = pkg.RequiredBy
		}
		lf.Packages[ref] = lp
	}

	return lf
}
