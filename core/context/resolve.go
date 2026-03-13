// Package context implements session context assembly for codectx.
// It resolves always_loaded references from codectx.yml to source
// markdown files, strips and normalizes them, and assembles them into
// a single context.md document for AI session startup.
//
// The package handles three reference formats:
//   - Local paths: "foundation/coding-standards" -> walk under docs root
//   - Package-specific: "react-patterns@community/foundation/..." -> walk under .codectx/packages/
//   - Bare package: "company-standards@acme" -> all docs from that package
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/core/markdown"
)

// RefType identifies how an always_loaded reference should be resolved.
type RefType int

const (
	// RefLocal is a local documentation path (e.g., "foundation/coding-standards").
	RefLocal RefType = iota

	// RefPackageSpecific is a package path with a subpath
	// (e.g., "react-patterns@community/foundation/component-principles").
	RefPackageSpecific

	// RefPackageBare is a bare package reference with no subpath
	// (e.g., "company-standards@acme").
	RefPackageBare
)

// String returns a human-readable name for the reference type.
func (rt RefType) String() string {
	switch rt {
	case RefLocal:
		return "local"
	case RefPackageSpecific:
		return "package"
	case RefPackageBare:
		return "package"
	default:
		return "unknown"
	}
}

// ResolvedFile represents a single markdown file resolved from an always_loaded reference.
type ResolvedFile struct {
	// AbsPath is the absolute path to the file on disk.
	AbsPath string

	// RelPath is the path relative to the resolution root (docs root or package dir).
	// Uses forward slashes.
	RelPath string
}

// ResolvedEntry represents a fully resolved always_loaded reference.
type ResolvedEntry struct {
	// Reference is the original always_loaded string from codectx.yml.
	Reference string

	// Type is the resolution type (local, package-specific, or bare-package).
	Type RefType

	// PackageName is the package name for package references.
	// Empty for local references.
	PackageName string

	// Files is the ordered list of resolved markdown files.
	Files []ResolvedFile

	// Title is the display title for this entry in context.md (H2 heading).
	// Derived from the reference path or package name.
	Title string
}

// ParseRef parses an always_loaded reference string into its type and components.
// Returns (refType, packageName, subpath).
//
// Format rules:
//   - Contains "@" -> package reference: "name@author" or "name@author/path/..."
//   - No "@" -> local path
func ParseRef(ref string) (RefType, string, string) {
	atIdx := strings.Index(ref, "@")
	if atIdx < 0 {
		// Local path: "foundation/coding-standards"
		return RefLocal, "", ref
	}

	// Package reference: everything before "@" is the package name.
	pkgName := ref[:atIdx]

	// Everything after "@author" — find the first "/" after the author segment.
	afterAt := ref[atIdx+1:]
	slashIdx := strings.Index(afterAt, "/")

	if slashIdx < 0 {
		// Bare package: "company-standards@acme" — no subpath.
		return RefPackageBare, pkgName, ""
	}

	// Package-specific: "react-patterns@community/foundation/component-principles"
	subpath := afterAt[slashIdx+1:]
	return RefPackageSpecific, pkgName, subpath
}

// Resolve resolves a list of always_loaded references to their source markdown files.
//
// rootDir is the absolute path to the documentation root (e.g., /path/to/project/docs).
// packagesDir is the absolute path to .codectx/packages/.
// alwaysLoaded is the list of references from codectx.yml session.always_loaded.
//
// Each reference is resolved to one or more markdown files. The order of references
// is preserved. Within each reference, files are sorted alphabetically.
func Resolve(rootDir, packagesDir string, alwaysLoaded []string) ([]ResolvedEntry, error) {
	entries := make([]ResolvedEntry, 0, len(alwaysLoaded))

	for _, ref := range alwaysLoaded {
		if ref == "" {
			continue
		}

		entry, err := resolveOne(rootDir, packagesDir, ref)
		if err != nil {
			return nil, fmt.Errorf("resolving %q: %w", ref, err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// resolveOne resolves a single always_loaded reference.
func resolveOne(rootDir, packagesDir, ref string) (ResolvedEntry, error) {
	refType, pkgName, subpath := ParseRef(ref)

	entry := ResolvedEntry{
		Reference:   ref,
		Type:        refType,
		PackageName: pkgName,
	}

	var walkRoot string

	switch refType {
	case RefLocal:
		walkRoot = filepath.Join(rootDir, filepath.FromSlash(subpath))
		entry.Title = titleFromPath(subpath)

	case RefPackageSpecific:
		walkRoot = filepath.Join(packagesDir, pkgName, filepath.FromSlash(subpath))
		entry.Title = titleFromPath(subpath)

	case RefPackageBare:
		walkRoot = filepath.Join(packagesDir, pkgName)
		entry.Title = titleFromPackage(pkgName)
	}

	files, err := walkMarkdown(walkRoot)
	if err != nil {
		return entry, fmt.Errorf("walking %s: %w", walkRoot, err)
	}

	if len(files) == 0 {
		return entry, fmt.Errorf("no markdown files found for %q at %s", ref, walkRoot)
	}

	entry.Files = files
	return entry, nil
}

// walkMarkdown walks a directory and collects all .md files, sorted alphabetically.
// If walkRoot is a single file, it is returned as the sole entry.
func walkMarkdown(walkRoot string) ([]ResolvedFile, error) {
	info, err := os.Stat(walkRoot)
	if err != nil {
		return nil, err
	}

	// If walkRoot is a single file, return it directly.
	if !info.IsDir() {
		if markdown.IsMarkdown(info.Name()) {
			return []ResolvedFile{{
				AbsPath: walkRoot,
				RelPath: filepath.Base(walkRoot),
			}}, nil
		}
		return nil, nil
	}

	// Use the shared markdown file walker.
	walked, walkErr := markdown.WalkFiles(walkRoot)
	if walkErr != nil {
		return nil, walkErr
	}

	files := make([]ResolvedFile, len(walked))
	for i, wf := range walked {
		files[i] = ResolvedFile{
			AbsPath: wf.AbsPath,
			RelPath: wf.RelPath,
		}
	}

	return files, nil
}

// titleFromPath derives a display title from a documentation path.
// E.g., "foundation/coding-standards" -> "Coding Standards"
// E.g., "foundation/error-handling/README.md" -> "Error Handling"
func titleFromPath(path string) string {
	// Use the last meaningful segment.
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimSuffix(path, ".md")
	path = strings.TrimSuffix(path, "/README")

	parts := strings.Split(path, "/")
	last := parts[len(parts)-1]

	return humanize(last)
}

// titleFromPackage derives a display title from a package name.
// E.g., "company-standards" -> "Company Standards"
func titleFromPackage(name string) string {
	return humanize(name)
}

// humanize converts a kebab-case or snake_case string to Title Case.
func humanize(s string) string {
	// Replace hyphens and underscores with spaces.
	s = strings.NewReplacer("-", " ", "_", " ").Replace(s)

	// Title case each word.
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}

	return strings.Join(words, " ")
}
