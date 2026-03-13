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

// SetupProjectWithPrefs creates a minimal project directory with a
// codectx.yml and preferences.yml containing the given auto_compile
// setting. Returns the project directory and a pointer to the config.
func SetupProjectWithPrefs(t *testing.T, autoCompile *bool) (string, *project.Config) {
	t.Helper()

	dir := t.TempDir()
	cfg := project.DefaultConfig("test", "docs", "")

	if err := cfg.WriteToFile(filepath.Join(dir, project.ConfigFileName)); err != nil {
		t.Fatal(err)
	}

	rootDir := project.RootDir(dir, &cfg)
	codectxDir := filepath.Join(rootDir, project.CodectxDir)
	if err := os.MkdirAll(codectxDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}

	prefs := project.DefaultPreferencesConfig()
	prefs.AutoCompile = autoCompile
	if err := prefs.WriteToFile(filepath.Join(codectxDir, project.PreferencesFile)); err != nil {
		t.Fatal(err)
	}

	return dir, &cfg
}
