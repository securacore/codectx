package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotFound is returned when no codectx.yml is found walking up from the
// starting directory to the filesystem root.
var ErrNotFound = errors.New("codectx.yml not found (walked to filesystem root)")

// Discover walks up from the given directory looking for a codectx.yml file.
// Returns the absolute path to the directory containing codectx.yml.
// Returns ErrNotFound if the filesystem root is reached without finding one.
func Discover(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	for {
		candidate := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root.
			return "", ErrNotFound
		}
		dir = parent
	}
}

// RootDir resolves the documentation root directory for a project.
// projectDir is the directory containing codectx.yml.
// Returns the absolute path to the documentation root.
func RootDir(projectDir string, cfg *Config) string {
	root := cfg.Root
	if root == "" {
		root = DefaultRoot
	}
	return filepath.Join(projectDir, root)
}
