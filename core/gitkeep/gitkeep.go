package gitkeep

import (
	"os"
	"path/filepath"
)

const fileName = ".gitkeep"

// Write creates a .gitkeep file in the given directory. The directory must
// already exist. If the file already exists it is left untouched.
func Write(dir string) error {
	p := filepath.Join(dir, fileName)
	if _, err := os.Stat(p); err == nil {
		return nil // already exists
	}
	return os.WriteFile(p, nil, 0o644)
}

// Clean removes .gitkeep files from documentation directories that contain
// other files or subdirectories. It walks the immediate children of docsDir
// (e.g., docs/topics/, docs/prompts/) and checks each child directory. If a
// child directory has a .gitkeep AND at least one other entry, the .gitkeep is
// removed.
//
// The function also processes the package/ directory at the same level as
// docsDir if it exists.
func Clean(docsDir string) error {
	if err := cleanDir(docsDir); err != nil {
		return err
	}

	// Also clean the package/ directory if present.
	pkgDir := filepath.Join(filepath.Dir(docsDir), "package")
	if info, err := os.Stat(pkgDir); err == nil && info.IsDir() {
		if err := cleanDir(pkgDir); err != nil {
			return err
		}
	}

	return nil
}

// cleanDir scans the immediate subdirectories of dir for .gitkeep files that
// can be removed because the directory now contains other content.
func cleanDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		subdir := filepath.Join(dir, entry.Name())
		if err := cleanSubdir(subdir); err != nil {
			return err
		}
	}

	return nil
}

// cleanSubdir removes the .gitkeep in a single directory if the directory
// has at least one other entry (file or subdirectory).
func cleanSubdir(dir string) error {
	keepPath := filepath.Join(dir, fileName)
	if _, err := os.Stat(keepPath); os.IsNotExist(err) {
		return nil // no .gitkeep to clean
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Count entries that are not .gitkeep.
	other := 0
	for _, e := range entries {
		if e.Name() != fileName {
			other++
		}
	}

	if other > 0 {
		return os.Remove(keepPath)
	}

	return nil
}
