package markdown

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WalkedFile represents a markdown file discovered by WalkFiles.
type WalkedFile struct {
	// AbsPath is the absolute file path on disk.
	AbsPath string

	// RelPath is the path relative to the walk root, using forward slashes.
	RelPath string
}

// WalkFiles walks a directory tree and returns all markdown files, skipping
// hidden directories (those starting with '.'). Results are sorted
// alphabetically by relative path.
//
// If root does not exist or is not a directory, an error is returned.
func WalkFiles(root string) ([]WalkedFile, error) {
	var files []WalkedFile

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories.
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only collect markdown files.
		if !IsMarkdown(d.Name()) {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}

		files = append(files, WalkedFile{
			AbsPath: path,
			RelPath: filepath.ToSlash(rel),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})

	return files, nil
}
