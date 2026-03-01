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

// collectFilePaths returns all file paths referenced by a manifest's entries.
// Paths are relative to the package root.
func collectFilePaths(m *manifest.Manifest) []string {
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

	return paths
}
