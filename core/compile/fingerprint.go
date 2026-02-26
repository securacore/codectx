package compile

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"sort"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
)

const fingerprintFile = ".fingerprint"

// computeFingerprint hashes all compilation inputs into a single
// deterministic fingerprint. Any change to config, manifests, source
// files, or activation state produces a different fingerprint.
func computeFingerprint(cfg *config.Config) (string, error) {
	h := sha256.New()

	// Hash the config file content.
	configData, err := os.ReadFile("codectx.yml")
	if err != nil {
		return "", fmt.Errorf("read config for fingerprint: %w", err)
	}
	h.Write(configData)

	docsDir := cfg.DocsDir()

	// Hash the local package manifest.
	localManifestPath := filepath.Join(docsDir, manifestFile)
	if err := hashFile(h, localManifestPath); err != nil {
		return "", fmt.Errorf("hash local manifest: %w", err)
	}

	// Hash all local documentation files referenced by the manifest.
	localManifest, err := manifest.Load(localManifestPath)
	if err != nil {
		return "", fmt.Errorf("load local manifest for fingerprint: %w", err)
	}
	localManifest = manifest.Sync(docsDir, localManifest)
	if err := hashManifestFiles(h, localManifest, docsDir); err != nil {
		return "", fmt.Errorf("hash local files: %w", err)
	}

	// Hash installed package manifests and their files.
	for _, pkg := range cfg.Packages {
		pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", pkg.Name, pkg.Author))
		pkgManifestPath := filepath.Join(pkgDir, manifestFile)

		if err := hashFile(h, pkgManifestPath); err != nil {
			// Package directory may not exist yet (not fetched).
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("hash package %s@%s manifest: %w", pkg.Name, pkg.Author, err)
		}

		if pkg.Active.IsNone() {
			continue
		}

		pkgManifest, err := manifest.Load(pkgManifestPath)
		if err != nil {
			continue
		}
		pkgManifest = manifest.Discover(pkgDir, pkgManifest)

		if err := hashManifestFiles(h, pkgManifest, pkgDir); err != nil {
			return "", fmt.Errorf("hash package %s@%s files: %w", pkg.Name, pkg.Author, err)
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// hashFile reads a file and writes its content to the hash.
func hashFile(h hash.Hash, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Include the path in the hash so moving a file changes the fingerprint.
	h.Write([]byte(path))
	h.Write(data)
	return nil
}

// hashManifestFiles collects all file paths referenced by a manifest
// and hashes their contents in sorted order for determinism.
func hashManifestFiles(h hash.Hash, m *manifest.Manifest, baseDir string) error {
	var paths []string

	for _, e := range m.Foundation {
		paths = append(paths, e.Path)
		if e.Spec != "" {
			paths = append(paths, e.Spec)
		}
		paths = append(paths, e.Files...)
	}
	for _, e := range m.Application {
		paths = append(paths, e.Path)
		if e.Spec != "" {
			paths = append(paths, e.Spec)
		}
		paths = append(paths, e.Files...)
	}
	for _, e := range m.Topics {
		paths = append(paths, e.Path)
		if e.Spec != "" {
			paths = append(paths, e.Spec)
		}
		paths = append(paths, e.Files...)
	}
	for _, e := range m.Prompts {
		paths = append(paths, e.Path)
	}
	for _, e := range m.Plans {
		paths = append(paths, e.Path)
		if e.PlanState != "" {
			paths = append(paths, e.PlanState)
		}
	}

	// Sort for deterministic ordering.
	sort.Strings(paths)

	for _, p := range paths {
		fullPath := filepath.Join(baseDir, p)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", fullPath, err)
		}
		h.Write([]byte(p))
		h.Write(data)
	}

	return nil
}

// loadFingerprint reads the stored fingerprint from the output directory.
// Returns empty string if the file does not exist.
func loadFingerprint(outputDir string) string {
	data, err := os.ReadFile(filepath.Join(outputDir, fingerprintFile))
	if err != nil {
		return ""
	}
	return string(data)
}

// saveFingerprint writes the fingerprint to the output directory.
func saveFingerprint(outputDir, fingerprint string) error {
	return os.WriteFile(filepath.Join(outputDir, fingerprintFile), []byte(fingerprint), 0o644)
}
