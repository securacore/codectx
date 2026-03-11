// Package compile implements the codectx compilation pipeline. It orchestrates
// source file discovery, markdown parsing, token counting, chunking, BM25
// indexing, and manifest generation — producing the compiled artifacts that
// the AI consumes at runtime.
//
// The package follows the same pattern as core/scaffold: business logic lives
// here, and the CLI layer in cmds/compile is a thin wrapper that handles TUI
// and config assembly.
package compile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/project"
)

// sourceFilesFromWalked converts generic WalkedFile results into SourceFile
// structs with paths relative to rootDir.
func sourceFilesFromWalked(walked []markdown.WalkedFile, rootDir string) ([]SourceFile, error) {
	sources := make([]SourceFile, 0, len(walked))
	for _, wf := range walked {
		rel, err := filepath.Rel(rootDir, wf.AbsPath)
		if err != nil {
			return nil, err
		}
		sources = append(sources, SourceFile{
			Path:    filepath.ToSlash(rel),
			AbsPath: wf.AbsPath,
			IsSpec:  strings.HasSuffix(wf.RelPath, ".spec.md"),
		})
	}
	return sources, nil
}

// SourceFile represents a discovered markdown file to be compiled.
type SourceFile struct {
	// Path is the file path relative to the documentation root.
	// Uses forward slashes for consistency (matches chunk.ClassifySource).
	Path string

	// AbsPath is the absolute file path on disk.
	AbsPath string

	// IsSpec is true if the file ends with .spec.md.
	IsSpec bool
}

// DiscoverSources walks the documentation root and collects all markdown files
// to be compiled. It skips hidden directories (starting with '.') except for
// .codectx/packages/ which contains installed dependency documentation.
//
// The rootDir parameter is the absolute path to the documentation root
// (e.g., /path/to/project/docs). All returned paths are relative to rootDir.
//
// Active dependencies are resolved from the config's Dependencies map —
// only packages marked as active are included.
func DiscoverSources(rootDir string, activeDeps map[string]bool) ([]SourceFile, error) {
	// Walk the documentation root for local docs and system docs.
	walked, err := markdown.WalkFiles(rootDir)
	if err != nil {
		return nil, err
	}

	sources := make([]SourceFile, 0, len(walked))
	for _, wf := range walked {
		sources = append(sources, SourceFile{
			Path:    wf.RelPath,
			AbsPath: wf.AbsPath,
			IsSpec:  strings.HasSuffix(wf.RelPath, ".spec.md"),
		})
	}

	// Walk active packages from .codectx/packages/.
	packagesDir := filepath.Join(rootDir, project.CodectxDir, project.PackagesDir)
	if info, err := os.Stat(packagesDir); err == nil && info.IsDir() {
		pkgSources, err := discoverPackages(rootDir, packagesDir, activeDeps)
		if err != nil {
			return nil, err
		}
		sources = append(sources, pkgSources...)
	}

	// Sort for deterministic output.
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Path < sources[j].Path
	})

	return sources, nil
}

// discoverPackages walks the packages directory and returns SourceFiles for
// active packages. Package paths are relative to rootDir (e.g.,
// ".codectx/packages/react-patterns/topics/hooks.md").
func discoverPackages(rootDir, packagesDir string, activeDeps map[string]bool) ([]SourceFile, error) {
	var sources []SourceFile

	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgName := entry.Name()

		// Skip inactive packages.
		if !activeDeps[pkgName] {
			continue
		}

		pkgDir := filepath.Join(packagesDir, pkgName)
		walked, err := markdown.WalkFiles(pkgDir)
		if err != nil {
			return nil, err
		}
		pkgSources, err := sourceFilesFromWalked(walked, rootDir)
		if err != nil {
			return nil, err
		}
		sources = append(sources, pkgSources...)
	}

	return sources, nil
}
