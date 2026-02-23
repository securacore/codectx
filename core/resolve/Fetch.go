package resolve

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Fetch clones the resolved package at its exact tag into destDir.
// It performs a shallow clone (depth 1) to minimize footprint.
// The destination directory must not already exist.
func Fetch(resolved *ResolvedPackage, destDir string) error {
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("destination %s already exists", destDir)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create destination %s: %w", destDir, err)
	}

	_, err := git.PlainClone(destDir, false, &git.CloneOptions{
		URL:           resolved.Source,
		ReferenceName: plumbing.NewTagReferenceName(resolved.Tag),
		Depth:         1,
	})
	if err != nil {
		// Clean up on failure.
		os.RemoveAll(destDir)
		return fmt.Errorf("clone %s at tag %s: %w", resolved.Source, resolved.Tag, err)
	}

	// Verify package.yml exists in the cloned repo.
	if _, err := os.Stat(fmt.Sprintf("%s/package.yml", destDir)); os.IsNotExist(err) {
		os.RemoveAll(destDir)
		return fmt.Errorf("cloned package has no package.yml at root")
	}

	return nil
}
