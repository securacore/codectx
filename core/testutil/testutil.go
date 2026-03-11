// Package testutil provides shared test helper functions used across
// multiple test packages in the codectx project.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/project"
)

// MustWriteFile creates a file with the given content, creating parent
// directories as needed. It calls t.Fatal on any error.
func MustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), project.DirPerm); err != nil {
		t.Fatalf("creating directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), project.FilePerm); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
