package manifest

import (
	"os"
	"path/filepath"
)

// Sync synchronizes a manifest with the filesystem. It performs three steps:
//
//  1. Discovery: scans standard directories and adds entries found on disk
//     that are not already declared in the manifest (merge-missing).
//  2. Stale removal: removes entries whose primary file no longer exists on disk.
//  3. Relationship inference: parses all markdown files for cross-entry links
//     and rebuilds depends_on/required_by from the link graph. Links are the
//     sole source of truth — all existing relationships are replaced.
//
// The returned manifest preserves all metadata (name, author, version, description)
// and any entry fields not affected by sync (e.g., load, description).
func Sync(pkgDir string, existing *Manifest) *Manifest {
	// Step 1: Discover new entries from disk.
	result := Discover(pkgDir, existing)

	// Step 2: Remove stale entries whose primary file is missing.
	result.Foundation = removeStaleFoundation(pkgDir, result.Foundation)
	result.Application = removeStaleApplication(pkgDir, result.Application)
	result.Topics = removeStaleTopics(pkgDir, result.Topics)
	result.Prompts = removeStalePrompts(pkgDir, result.Prompts)
	result.Plans = removeStalePlans(pkgDir, result.Plans)

	// Step 3: Infer relationships from markdown links.
	inferRelationships(pkgDir, result)

	return result
}

// removeStaleFoundation filters out foundation entries whose Path does not exist on disk.
func removeStaleFoundation(pkgDir string, entries []FoundationEntry) []FoundationEntry {
	if entries == nil {
		return nil
	}
	var kept []FoundationEntry
	for _, e := range entries {
		if fileExists(filepath.Join(pkgDir, e.Path)) {
			kept = append(kept, e)
		}
	}
	return kept
}

// removeStaleApplication filters out application entries whose Path does not exist on disk.
func removeStaleApplication(pkgDir string, entries []ApplicationEntry) []ApplicationEntry {
	if entries == nil {
		return nil
	}
	var kept []ApplicationEntry
	for _, e := range entries {
		if fileExists(filepath.Join(pkgDir, e.Path)) {
			kept = append(kept, e)
		}
	}
	return kept
}

// removeStaleTopics filters out topic entries whose Path does not exist on disk.
func removeStaleTopics(pkgDir string, entries []TopicEntry) []TopicEntry {
	if entries == nil {
		return nil
	}
	var kept []TopicEntry
	for _, e := range entries {
		if fileExists(filepath.Join(pkgDir, e.Path)) {
			kept = append(kept, e)
		}
	}
	return kept
}

// removeStalePrompts filters out prompt entries whose Path does not exist on disk.
func removeStalePrompts(pkgDir string, entries []PromptEntry) []PromptEntry {
	if entries == nil {
		return nil
	}
	var kept []PromptEntry
	for _, e := range entries {
		if fileExists(filepath.Join(pkgDir, e.Path)) {
			kept = append(kept, e)
		}
	}
	return kept
}

// removeStalePlans filters out plan entries whose Path does not exist on disk.
func removeStalePlans(pkgDir string, entries []PlanEntry) []PlanEntry {
	if entries == nil {
		return nil
	}
	var kept []PlanEntry
	for _, e := range entries {
		if fileExists(filepath.Join(pkgDir, e.Path)) {
			kept = append(kept, e)
		}
	}
	return kept
}

// fileExists returns true if the given path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
