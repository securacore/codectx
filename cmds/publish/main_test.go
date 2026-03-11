package publish

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePackageStructure_Valid(t *testing.T) {
	t.Parallel()

	for _, dir := range validPackageDirs {
		dir := dir
		t.Run(dir, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
				t.Fatalf("mkdir %s: %v", dir, err)
			}

			if err := validatePackageStructure(root); err != nil {
				t.Errorf("expected valid for %s dir, got: %v", dir, err)
			}
		})
	}
}

func TestValidatePackageStructure_Empty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	err := validatePackageStructure(root)
	if err == nil {
		t.Error("expected error for directory with no package content dirs")
	}
}

func TestValidatePackageStructure_FileNotDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// Create "foundation" as a file, not a directory.
	if err := os.WriteFile(filepath.Join(root, "foundation"), []byte("not a dir"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := validatePackageStructure(root)
	if err == nil {
		t.Error("expected error when 'foundation' is a file not a directory")
	}
}

func TestValidatePackageStructure_IrrelevantDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// Create directories that are NOT in the valid list.
	for _, dir := range []string{"src", "lib", "node_modules"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	err := validatePackageStructure(root)
	if err == nil {
		t.Error("expected error for directory with only irrelevant dirs")
	}
}

func TestValidatePackageStructure_MultipleDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, dir := range []string{"foundation", "topics", "plans"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	if err := validatePackageStructure(root); err != nil {
		t.Errorf("expected valid for multiple valid dirs, got: %v", err)
	}
}
