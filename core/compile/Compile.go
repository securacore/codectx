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
const manifestFile = "manifest.yml"

// Compile builds the compiled documentation set from all active sources.
// It loads manifests, merges entries with deduplication, stores files as
// content-addressed objects, builds a compiled manifest with provenance
// tracking, prunes orphaned objects, and generates a lock file.
func Compile(cfg *config.Config) (*Result, error) {
	docsDir := cfg.DocsDir()
	outputDir := cfg.OutputDir()

	// Check if inputs have changed since last compile.
	fingerprint, fpErr := computeFingerprint(cfg)
	if fpErr == nil && fingerprint != "" {
		stored := loadFingerprint(outputDir)
		if stored == fingerprint {
			return &Result{
				OutputDir: outputDir,
				UpToDate:  true,
			}, nil
		}
	}

	// Load local package manifest.
	localManifestPath := filepath.Join(docsDir, manifestFile)
	localManifest, err := manifest.Load(localManifestPath)
	if err != nil {
		return nil, fmt.Errorf("load local manifest: %w", err)
	}

	// Sync: discover new entries, remove stale, infer relationships from links.
	localManifest = manifest.Sync(docsDir, localManifest)

	// Write the synced manifest back so the source stays current.
	if err := manifest.Write(localManifestPath, localManifest); err != nil {
		return nil, fmt.Errorf("write synced local manifest: %w", err)
	}

	// Clean output directory, preserving user preferences.
	if err := cleanOutputDir(outputDir); err != nil {
		return nil, fmt.Errorf("clean output directory %s: %w", outputDir, err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", outputDir, err)
	}

	// Initialize object store.
	store := NewObjectStore(filepath.Join(outputDir, "objects"))

	// Build the unified manifest starting from local package metadata.
	unified := &manifest.Manifest{
		Name:        cfg.Name,
		Author:      localManifest.Author,
		Version:     localManifest.Version,
		Description: localManifest.Description,
	}

	// Initialize deduplication tracking.
	seen := make(map[string]seenEntry)
	var report DeduplicationReport

	// Track source directories for provenance-aware object storage.
	srcDirs := map[string]string{"local": docsDir}

	// Merge local package entries.
	mergeManifestDedup(unified, localManifest, docsDir, docsDir, "local", seen)

	// Merge installed packages.
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
		pkgManifestPath := filepath.Join(pkgDir, manifestFile)

		pkgManifest, err := manifest.Load(pkgManifestPath)
		if err != nil {
			return nil, fmt.Errorf("load package %s@%s manifest: %w", pkg.Name, pkg.Author, err)
		}

		// Discover entries from disk that aren't declared in the manifest.
		pkgManifest = manifest.Discover(pkgDir, pkgManifest)

		// Filter entries by activation state.
		filtered := filterManifest(pkgManifest, pkg.Active)

		// Track source directory for this package.
		pkgLabel := fmt.Sprintf("%s@%s", pkg.Name, pkg.Author)
		srcDirs[pkgLabel] = pkgDir

		// Merge with deduplication.
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

	// Build provenance map from merge tracking.
	provenance := make(map[string]string, len(seen))
	for key, s := range seen {
		provenance[key] = s.pkg
	}

	// Store all winning entry files as content-addressed objects.
	pathToHash, objectsStored, err := storeObjects(store, unified, srcDirs, provenance)
	if err != nil {
		return nil, fmt.Errorf("store objects: %w", err)
	}

	// Copy plan state files (mutable, not content-addressed).
	if err := copyStateFiles(unified, srcDirs, provenance, outputDir); err != nil {
		return nil, fmt.Errorf("copy state files: %w", err)
	}

	// Convert to compiled manifest with object references.
	cm := toCompiledManifest(unified, pathToHash, provenance)

	// Generate heuristics sidecar (needed for decomposition thresholds and README).
	objectsDir := filepath.Join(outputDir, "objects")
	h := generateHeuristics(unified, pathToHash, provenance, objectsDir)

	// Decompose if thresholds are exceeded.
	if shouldDecompose(h) {
		if err := decompose(cm, h, outputDir); err != nil {
			return nil, fmt.Errorf("decompose manifest: %w", err)
		}
	}

	// Write compiled manifest.
	manifestPath := filepath.Join(outputDir, "manifest.yml")
	if err := WriteCompiledManifest(manifestPath, cm); err != nil {
		return nil, fmt.Errorf("write compiled manifest: %w", err)
	}

	// Write heuristics sidecar.
	heuristicsPath := filepath.Join(outputDir, heuristicsFile)
	if err := WriteHeuristics(heuristicsPath, h); err != nil {
		return nil, fmt.Errorf("write heuristics: %w", err)
	}

	// Generate and write compiled README.md.
	readmeContent := generateReadme(unified, h)
	readmePath := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
		return nil, fmt.Errorf("write compiled README.md: %w", err)
	}

	// Prune orphaned objects no longer referenced by any entry.
	activeHashes := collectActiveHashes(pathToHash)
	objectsPruned, err := store.Prune(activeHashes)
	if err != nil {
		return nil, fmt.Errorf("prune orphaned objects: %w", err)
	}

	// Write lock file.
	if err := lock.Write(lockFile, lck); err != nil {
		return nil, fmt.Errorf("write lock file: %w", err)
	}

	// Save fingerprint for incremental compilation.
	if fpErr == nil && fingerprint != "" {
		_ = saveFingerprint(outputDir, fingerprint)
	}

	return &Result{
		OutputDir:     outputDir,
		ObjectsStored: objectsStored,
		ObjectsPruned: objectsPruned,
		Packages:      packagesProcessed,
		Dedup:         report,
	}, nil
}

// storeObjects iterates over the unified manifest and stores each referenced
// file through the ObjectStore. It resolves the correct source directory for
// each entry using the provenance map. Returns the pathToHash map and count.
func storeObjects(
	store *ObjectStore,
	unified *manifest.Manifest,
	srcDirs map[string]string,
	provenance map[string]string,
) (map[string]string, int, error) {
	pathToHash := make(map[string]string)
	stored := 0

	storeFile := func(section, id, relPath string) error {
		if relPath == "" {
			return nil
		}
		if _, ok := pathToHash[relPath]; ok {
			return nil // already stored (same path, same content)
		}

		srcDir := srcDirs[provenance[section+":"+id]]
		if srcDir == "" {
			return nil
		}

		data, err := os.ReadFile(filepath.Join(srcDir, relPath))
		if err != nil {
			if os.IsNotExist(err) {
				return nil // skip missing files
			}
			return fmt.Errorf("read %s from %s: %w", relPath, srcDir, err)
		}

		hash, err := store.Store(data)
		if err != nil {
			return err
		}

		pathToHash[relPath] = hash
		stored++
		return nil
	}

	for _, e := range unified.Foundation {
		if err := storeFile("foundation", e.ID, e.Path); err != nil {
			return nil, stored, err
		}
	}

	for _, e := range unified.Application {
		if err := storeFile("application", e.ID, e.Path); err != nil {
			return nil, stored, err
		}
		if err := storeFile("application", e.ID, e.Spec); err != nil {
			return nil, stored, err
		}
		for _, f := range e.Files {
			if err := storeFile("application", e.ID, f); err != nil {
				return nil, stored, err
			}
		}
	}

	for _, e := range unified.Topics {
		if err := storeFile("topics", e.ID, e.Path); err != nil {
			return nil, stored, err
		}
		if err := storeFile("topics", e.ID, e.Spec); err != nil {
			return nil, stored, err
		}
		for _, f := range e.Files {
			if err := storeFile("topics", e.ID, f); err != nil {
				return nil, stored, err
			}
		}
	}

	for _, e := range unified.Prompts {
		if err := storeFile("prompts", e.ID, e.Path); err != nil {
			return nil, stored, err
		}
	}

	for _, e := range unified.Plans {
		if err := storeFile("plans", e.ID, e.Path); err != nil {
			return nil, stored, err
		}
	}

	return pathToHash, stored, nil
}

// copyStateFiles copies plan state files to outputDir/state/{id}.yml.
// State files are mutable and not stored in the object store.
func copyStateFiles(
	unified *manifest.Manifest,
	srcDirs map[string]string,
	provenance map[string]string,
	outputDir string,
) error {
	for _, e := range unified.Plans {
		if e.State == "" {
			continue
		}

		srcDir := srcDirs[provenance["plans:"+e.ID]]
		if srcDir == "" {
			continue
		}

		srcPath := filepath.Join(srcDir, e.State)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		dstPath := filepath.Join(outputDir, "state", e.ID+".yml")
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy state %s: %w", e.ID, err)
		}
	}
	return nil
}

// collectActiveHashes extracts the set of hashes currently referenced
// by compiled entries. Used for pruning orphaned objects.
func collectActiveHashes(pathToHash map[string]string) map[string]bool {
	active := make(map[string]bool, len(pathToHash))
	for _, hash := range pathToHash {
		active[hash] = true
	}
	return active
}

// preservedFiles lists files in the output directory that should survive
// a clean. These are user-specific and not part of compiled output.
var preservedFiles = map[string]bool{
	"preferences.yml": true,
	".fingerprint":    true,
}

// cleanOutputDir removes all contents of dir except preserved files.
// If the directory does not exist, this is a no-op.
func cleanOutputDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read output directory: %w", err)
	}

	for _, e := range entries {
		if preservedFiles[e.Name()] {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("remove %s: %w", e.Name(), err)
		}
	}

	return nil
}
