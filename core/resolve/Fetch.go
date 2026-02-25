package resolve

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Fetch downloads a resolved package into destDir. It first attempts to
// fetch a release asset (package.tar.gz) from GitHub Releases. If no
// release asset is available, it falls back to a shallow git clone.
// The destination directory must not already exist.
func Fetch(resolved *ResolvedPackage, destDir string) error {
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("destination %s already exists", destDir)
	}

	// Try release asset first, fall back to git clone.
	err := fetchReleaseAsset(resolved, destDir)
	if err != nil {
		_ = os.RemoveAll(destDir)
		err = fetchGitClone(resolved, destDir)
	}
	if err != nil {
		_ = os.RemoveAll(destDir)
		return err
	}

	// Verify manifest.yml exists regardless of fetch method.
	if _, err := os.Stat(filepath.Join(destDir, "manifest.yml")); os.IsNotExist(err) {
		_ = os.RemoveAll(destDir)
		return fmt.Errorf("fetched package has no manifest.yml at root")
	}

	return nil
}

// fetchGitClone performs a shallow git clone of the resolved package
// into destDir.
func fetchGitClone(resolved *ResolvedPackage, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create destination %s: %w", destDir, err)
	}

	_, err := git.PlainClone(destDir, false, &git.CloneOptions{
		URL:           resolved.Source,
		ReferenceName: plumbing.NewTagReferenceName(resolved.Tag),
		Depth:         1,
	})
	if err != nil {
		return fmt.Errorf("clone %s at tag %s: %w", resolved.Source, resolved.Tag, err)
	}

	return nil
}
