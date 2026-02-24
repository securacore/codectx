package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/lock"
	"github.com/securacore/codectx/core/manifest"
)

const lockFile = "codectx.lock"
const packageFile = "package.yml"

// Compile builds the compiled documentation set from all active sources.
// It reads the local package manifest and any installed package manifests,
// filters entries by activation state, copies files to the output directory,
// builds a unified manifest, and generates a lock file.
func Compile(cfg *config.Config) (*Result, error) {
	docsDir := cfg.DocsDir()
	outputDir := cfg.OutputDir()

	// Load local package manifest.
	localManifestPath := filepath.Join(docsDir, packageFile)
	localManifest, err := manifest.Load(localManifestPath)
	if err != nil {
		return nil, fmt.Errorf("load local manifest: %w", err)
	}

	// Clean and recreate output directory.
	if err := os.RemoveAll(outputDir); err != nil {
		return nil, fmt.Errorf("clean output directory %s: %w", outputDir, err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", outputDir, err)
	}

	// Build the unified manifest starting from local package metadata.
	unified := &manifest.Manifest{
		Name:        cfg.Name,
		Author:      localManifest.Author,
		Version:     localManifest.Version,
		Description: localManifest.Description,
	}

	totalCopied := 0

	// Initialize deduplication tracking.
	seen := make(map[string]seenEntry)
	var report DeduplicationReport

	// Phase 1: Combine local package.
	// All local entries are included (the local package is always fully active).
	copied, err := copyManifestFiles(localManifest, docsDir, outputDir)
	if err != nil {
		return nil, fmt.Errorf("copy local package files: %w", err)
	}
	totalCopied += copied
	mergeManifestDedup(unified, localManifest, docsDir, docsDir, "local", seen)

	// Phase 1 continued: Combine installed packages.
	lck := &lock.Lock{
		CompiledAt: time.Now().Format("2006-01-02"),
	}

	packagesProcessed := 0
	for _, pkg := range cfg.Packages {
		if pkg.Active.IsNone() {
			lck.Packages = append(lck.Packages, lock.LockedPackage{
				Name:    pkg.Name,
				Author:  pkg.Author,
				Version: pkg.Version,
				Source:  pkg.Source,
				Active:  pkg.Active,
			})
			continue
		}

		// Load installed package manifest.
		pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", pkg.Name, pkg.Author))
		pkgManifestPath := filepath.Join(pkgDir, packageFile)

		pkgManifest, err := manifest.Load(pkgManifestPath)
		if err != nil {
			return nil, fmt.Errorf("load package %s@%s manifest: %w", pkg.Name, pkg.Author, err)
		}

		// Filter entries by activation state.
		filtered := filterManifest(pkgManifest, pkg.Active)

		// Copy activated files to output.
		copied, err := copyManifestFiles(filtered, pkgDir, outputDir)
		if err != nil {
			return nil, fmt.Errorf("copy package %s@%s files: %w", pkg.Name, pkg.Author, err)
		}
		totalCopied += copied

		// Phase 2: Align by merging activated entries with dedup.
		pkgLabel := fmt.Sprintf("%s@%s", pkg.Name, pkg.Author)
		events := mergeManifestDedup(unified, filtered, outputDir, pkgDir, pkgLabel, seen)
		for _, ev := range events {
			if ev.Reason == "duplicate" {
				report.Duplicates = append(report.Duplicates, ev)
			} else {
				report.Conflicts = append(report.Conflicts, ev)
			}
		}
		packagesProcessed++

		lck.Packages = append(lck.Packages, lock.LockedPackage{
			Name:    pkg.Name,
			Author:  pkg.Author,
			Version: pkgManifest.Version,
			Source:  pkg.Source,
			Active:  pkg.Active,
		})
	}

	// Write unified manifest to output directory.
	unifiedPath := filepath.Join(outputDir, packageFile)
	if err := manifest.Write(unifiedPath, unified); err != nil {
		return nil, fmt.Errorf("write unified manifest: %w", err)
	}

	// Generate and write compiled README.md.
	readmeContent := generateReadme(unified)
	readmePath := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
		return nil, fmt.Errorf("write compiled README.md: %w", err)
	}

	// Write lock file.
	if err := lock.Write(lockFile, lck); err != nil {
		return nil, fmt.Errorf("write lock file: %w", err)
	}

	return &Result{
		OutputDir:   outputDir,
		FilesCopied: totalCopied,
		Packages:    packagesProcessed,
		Dedup:       report,
	}, nil
}
