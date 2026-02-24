package compile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/manifest"
)

// copyFile copies a single file from src to dst, creating parent directories.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", dst, err)
	}

	return os.WriteFile(dst, data, 0o644)
}

// copyManifestFiles copies all files referenced by a manifest's entries
// from srcRoot to dstRoot. Returns the number of files copied.
func copyManifestFiles(m *manifest.Manifest, srcRoot, dstRoot string) (int, error) {
	paths := collectFilePaths(m)
	copied := 0

	for _, p := range paths {
		src := filepath.Join(srcRoot, p)
		dst := filepath.Join(dstRoot, p)

		// Skip files that do not exist on disk. Entries may reference
		// files that have not been created yet (e.g., empty plan dirs).
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}

		if err := copyFile(src, dst); err != nil {
			return copied, fmt.Errorf("copy %s: %w", p, err)
		}
		copied++
	}

	return copied, nil
}

// collectFilePaths returns all file paths referenced by a manifest's entries.
// Paths are relative to the package root.
func collectFilePaths(m *manifest.Manifest) []string {
	var paths []string

	for _, e := range m.Foundation {
		paths = append(paths, e.Path)
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
		if e.State != "" {
			paths = append(paths, e.State)
		}
	}

	return paths
}
