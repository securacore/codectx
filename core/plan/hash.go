package plan

import (
	"strings"

	"github.com/securacore/codectx/core/manifest"
)

// DependencyStatus represents whether a plan dependency's content hash
// matches the current compiled state.
type DependencyStatus struct {
	// Dependency is the original dependency from the plan.
	Dependency Dependency

	// CurrentHash is the current content hash from hashes.yml.
	// Empty string if the file is not found in compiled hashes.
	CurrentHash string

	// Changed is true if the current hash differs from the stored hash.
	Changed bool

	// Missing is true if the dependency path was not found in compiled hashes.
	Missing bool
}

// CheckResult holds the results of checking all plan dependencies against
// the current compiled state.
type CheckResult struct {
	// Statuses contains the status of each dependency in plan order.
	Statuses []DependencyStatus

	// AllMatch is true if no dependencies have changed or are missing.
	AllMatch bool

	// ChangedCount is the number of dependencies whose hashes changed.
	ChangedCount int

	// MissingCount is the number of dependencies not found in compiled hashes.
	MissingCount int
}

// CheckDependencies compares each plan dependency's stored hash against the
// current content hashes from the compiled hashes.yml file.
//
// The hashes parameter maps relative source file paths (as stored in hashes.yml)
// to their "sha256:<hex>" content hashes. Plan dependency paths are resolved
// by scanning the hashes map for entries whose path contains the dependency path.
//
// Returns a CheckResult summarizing which dependencies match, changed, or are missing.
func CheckDependencies(deps []Dependency, hashes *manifest.Hashes) *CheckResult {
	result := &CheckResult{
		Statuses: make([]DependencyStatus, len(deps)),
		AllMatch: true,
	}

	for i, dep := range deps {
		status := DependencyStatus{
			Dependency: dep,
		}

		currentHash, found := resolveHash(dep.Path, hashes)
		if !found {
			status.Missing = true
			status.Changed = true
			result.MissingCount++
			result.AllMatch = false
		} else {
			status.CurrentHash = currentHash
			if currentHash != dep.Hash {
				status.Changed = true
				result.ChangedCount++
				result.AllMatch = false
			}
		}

		result.Statuses[i] = status
	}

	return result
}

// resolveHash finds the content hash for a dependency path in the compiled hashes.
//
// Plan dependency paths use the document-relative format (e.g.
// "foundation/architecture-principles" or "topics/authentication/jwt-tokens").
// The hashes.yml file stores paths relative to the docs root with file extensions
// (e.g. "foundation/architecture-principles.md" or
// "topics/authentication/jwt-tokens.md").
//
// Resolution strategy:
//  1. Exact match in hashes.Files
//  2. Try appending common extensions (.md, /README.md)
//  3. Look for any key that starts with the dependency path
func resolveHash(depPath string, hashes *manifest.Hashes) (string, bool) {
	// 1. Exact match.
	if hash, ok := hashes.Files[depPath]; ok {
		return hash, true
	}

	// 2. Try common extensions.
	candidates := []string{
		depPath + ".md",
		depPath + "/README.md",
	}
	for _, candidate := range candidates {
		if hash, ok := hashes.Files[candidate]; ok {
			return hash, true
		}
	}

	// 3. Prefix match — find the lexically first file under this path
	// for deterministic results regardless of map iteration order.
	prefix := depPath + "/"
	var matchPath string
	var matchHash string
	for path, hash := range hashes.Files {
		if strings.HasPrefix(path, prefix) && len(path) > len(prefix) {
			if matchPath == "" || path < matchPath {
				matchPath = path
				matchHash = hash
			}
		}
	}
	if matchPath != "" {
		return matchHash, true
	}

	return "", false
}
