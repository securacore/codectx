package manifest

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// LinkPattern matches markdown links: [text](url)
// Captures the URL portion (group 1). Excludes URLs starting with http://, https://,
// or # (fragment-only links). Only matches .md file targets, optionally followed
// by a #fragment.
var LinkPattern = regexp.MustCompile(`\[(?:[^\]]*)\]\(([^)]+\.md(?:#[^)]*)?)\)`)

// extractLinks reads a markdown file and returns all relative link targets
// (the URL portion of [text](url) links). Only .md file targets are returned.
// Absolute URLs (http://, https://) and fragment-only links (#...) are excluded.
// Returns nil if the file cannot be read.
func extractLinks(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var links []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		matches := LinkPattern.FindAllStringSubmatch(scanner.Text(), -1)
		for _, m := range matches {
			target := m[1]
			// Skip absolute URLs and fragment-only links.
			if strings.HasPrefix(target, "http://") ||
				strings.HasPrefix(target, "https://") ||
				strings.HasPrefix(target, "#") {
				continue
			}
			// Strip any fragment from the path.
			if idx := strings.Index(target, "#"); idx >= 0 {
				target = target[:idx]
			}
			if target == "" {
				continue
			}
			if !seen[target] {
				seen[target] = true
				links = append(links, target)
			}
		}
	}

	return links
}

// ResolveLink resolves a relative markdown link target to a path relative
// to pkgDir. sourceFile is the docs-relative path of the file containing the
// link (e.g., "topics/react/README.md"). target is the raw link target
// (e.g., "../../foundation/philosophy.md").
//
// Returns the cleaned docs-relative path, or "" if the resolution escapes
// the package directory or produces an invalid path.
func ResolveLink(sourceFile, target string) string {
	// Resolve relative to the directory containing the source file.
	sourceDir := filepath.Dir(sourceFile)
	resolved := filepath.Join(sourceDir, target)
	resolved = filepath.Clean(resolved)

	// Reject paths that escape the package root.
	if strings.HasPrefix(resolved, "..") {
		return ""
	}

	return resolved
}

// pathToEntryID maps a docs-relative file path to the entry ID that owns it.
// It builds an index from all manifest entries: the entry's Path, Spec, and
// Files all map to the entry's ID.
func pathToEntryID(m *Manifest) map[string]string {
	index := make(map[string]string)

	for _, e := range m.Foundation {
		index[e.Path] = e.ID
		if e.Spec != "" {
			index[e.Spec] = e.ID
		}
		for _, f := range e.Files {
			index[f] = e.ID
		}
	}
	for _, e := range m.Application {
		index[e.Path] = e.ID
		if e.Spec != "" {
			index[e.Spec] = e.ID
		}
		for _, f := range e.Files {
			index[f] = e.ID
		}
	}
	for _, e := range m.Topics {
		index[e.Path] = e.ID
		if e.Spec != "" {
			index[e.Spec] = e.ID
		}
		for _, f := range e.Files {
			index[f] = e.ID
		}
	}
	for _, e := range m.Prompts {
		index[e.Path] = e.ID
	}
	for _, e := range m.Plans {
		index[e.Path] = e.ID
	}

	return index
}

// inferRelationships scans all markdown files referenced by the manifest,
// extracts cross-entry links, and builds bidirectional depends_on/required_by
// relationships. Links within the same entry (intra-entry) are ignored.
//
// All existing depends_on/required_by are cleared and fully rebuilt from links.
// This makes links the sole source of truth for relationships.
func inferRelationships(pkgDir string, m *Manifest) {
	index := pathToEntryID(m)

	// Collect all files to scan, grouped by owning entry ID.
	type fileRef struct {
		entryID string
		relPath string // docs-relative
	}
	var files []fileRef

	for _, e := range m.Foundation {
		files = append(files, fileRef{e.ID, e.Path})
		if e.Spec != "" {
			files = append(files, fileRef{e.ID, e.Spec})
		}
		for _, f := range e.Files {
			files = append(files, fileRef{e.ID, f})
		}
	}
	for _, e := range m.Application {
		files = append(files, fileRef{e.ID, e.Path})
		if e.Spec != "" {
			files = append(files, fileRef{e.ID, e.Spec})
		}
		for _, f := range e.Files {
			files = append(files, fileRef{e.ID, f})
		}
	}
	for _, e := range m.Topics {
		files = append(files, fileRef{e.ID, e.Path})
		if e.Spec != "" {
			files = append(files, fileRef{e.ID, e.Spec})
		}
		for _, f := range e.Files {
			files = append(files, fileRef{e.ID, f})
		}
	}
	for _, e := range m.Prompts {
		files = append(files, fileRef{e.ID, e.Path})
	}
	for _, e := range m.Plans {
		files = append(files, fileRef{e.ID, e.Path})
	}

	// Build the raw dependency graph: sourceID -> set of targetIDs.
	deps := make(map[string]map[string]bool)

	for _, fr := range files {
		absPath := filepath.Join(pkgDir, fr.relPath)
		links := extractLinks(absPath)
		for _, link := range links {
			resolved := ResolveLink(fr.relPath, link)
			if resolved == "" {
				continue
			}
			targetID, ok := index[resolved]
			if !ok {
				continue // link target is not a known entry
			}
			if targetID == fr.entryID {
				continue // intra-entry link, skip
			}
			if deps[fr.entryID] == nil {
				deps[fr.entryID] = make(map[string]bool)
			}
			deps[fr.entryID][targetID] = true
		}
	}

	// Build the reverse map: targetID -> set of sourceIDs (required_by).
	revDeps := make(map[string]map[string]bool)
	for srcID, targets := range deps {
		for tgtID := range targets {
			if revDeps[tgtID] == nil {
				revDeps[tgtID] = make(map[string]bool)
			}
			revDeps[tgtID][srcID] = true
		}
	}

	// Apply to manifest entries, clearing and rebuilding.
	applyDeps := func(id string) ([]string, []string) {
		var dependsOn, requiredBy []string
		if d, ok := deps[id]; ok {
			dependsOn = sortedKeys(d)
		}
		if r, ok := revDeps[id]; ok {
			requiredBy = sortedKeys(r)
		}
		return dependsOn, requiredBy
	}

	for i := range m.Foundation {
		m.Foundation[i].DependsOn, m.Foundation[i].RequiredBy = applyDeps(m.Foundation[i].ID)
	}
	for i := range m.Application {
		m.Application[i].DependsOn, m.Application[i].RequiredBy = applyDeps(m.Application[i].ID)
	}
	for i := range m.Topics {
		m.Topics[i].DependsOn, m.Topics[i].RequiredBy = applyDeps(m.Topics[i].ID)
	}
	for i := range m.Prompts {
		m.Prompts[i].DependsOn, m.Prompts[i].RequiredBy = applyDeps(m.Prompts[i].ID)
	}
	for i := range m.Plans {
		m.Plans[i].DependsOn, m.Plans[i].RequiredBy = applyDeps(m.Plans[i].ID)
	}
}

// sortedKeys returns the keys of a set as a sorted slice.
func sortedKeys(s map[string]bool) []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
