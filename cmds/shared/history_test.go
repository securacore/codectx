package shared

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/history"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/testutil"
)

func TestResolveHistoryDir(t *testing.T) {
	t.Run("returns history dir for valid project", func(t *testing.T) {
		// Create a temp project with codectx.yml.
		tmpDir := t.TempDir()

		// Resolve symlinks (macOS /var -> /private/var) for accurate comparison.
		tmpDir, err := filepath.EvalSymlinks(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(tmpDir, project.ConfigFileName)
		cfg := project.DefaultConfig("test-project", "docs", "")
		if err := cfg.WriteToFile(configPath); err != nil {
			t.Fatal(err)
		}

		// Change to the project dir so DiscoverProject can find it.
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(origDir) })

		histDir, projectDir, returnedCfg, err := ResolveHistoryDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if projectDir != tmpDir {
			t.Errorf("projectDir = %q, want %q", projectDir, tmpDir)
		}

		if returnedCfg == nil {
			t.Fatal("cfg should not be nil")
		}

		expectedHistDir := filepath.Join(tmpDir, "docs", ".codectx", "history")
		if histDir != expectedHistDir {
			t.Errorf("histDir = %q, want %q", histDir, expectedHistDir)
		}
	})

	t.Run("returns error when no project found", func(t *testing.T) {
		tmpDir := t.TempDir()

		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(origDir) })

		_, _, _, err = ResolveHistoryDir()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// BuildHistoryPath
// ---------------------------------------------------------------------------

func TestBuildHistoryPath_EmptyDocFile(t *testing.T) {
	got := BuildHistoryPath("/some/hist/dir", "")
	if got != "" {
		t.Errorf("expected empty string for empty docFile, got %q", got)
	}
}

func TestBuildHistoryPath_ReturnsRelativePath(t *testing.T) {
	dir := t.TempDir()
	histDir := filepath.Join(dir, "docs", ".codectx", "history")
	docFile := "123456.abc123.md"

	// Create the doc file so it exists.
	docsDir := filepath.Join(histDir, history.DocsDir)
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, docFile), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Chdir to the temp dir to get a relative path.
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	got := BuildHistoryPath(histDir, docFile)

	// Should be a relative path, not absolute.
	if filepath.IsAbs(got) {
		t.Errorf("expected relative path, got absolute: %q", got)
	}
	if got == "" {
		t.Error("expected non-empty path")
	}
}

// ---------------------------------------------------------------------------
// ResolveTopN
// ---------------------------------------------------------------------------

func TestResolveTopN_FlagValueOverrides(t *testing.T) {
	got := ResolveTopN(5, "/nonexistent", nil)
	if got != 5 {
		t.Errorf("ResolveTopN(5, ...) = %d, want 5", got)
	}
}

func TestResolveTopN_FlagValueZeroUsesDefault(t *testing.T) {
	got := ResolveTopN(0, "/nonexistent", nil)
	if got != project.DefaultResultsCount {
		t.Errorf("ResolveTopN(0, ...) = %d, want %d", got, project.DefaultResultsCount)
	}
}

func TestResolveTopN_NegativeUsesDefault(t *testing.T) {
	got := ResolveTopN(-1, "/nonexistent", nil)
	if got != project.DefaultResultsCount {
		t.Errorf("ResolveTopN(-1, ...) = %d, want %d", got, project.DefaultResultsCount)
	}
}

func TestResolveTopN_AIConfigOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	root := "docs"

	testutil.MustWriteFile(t, filepath.Join(dir, root, "codectx.yml"), `name: test
author: test
version: "0.1.0"
root: docs
`)
	testutil.MustWriteFile(t, filepath.Join(dir, root, project.CodectxDir, "ai.yml"), `consumption:
  results_count: 25
`)

	cfg := &project.Config{Root: root}
	got := ResolveTopN(0, dir, cfg)

	if got != 25 {
		t.Errorf("ResolveTopN(0, ...) with AI config = %d, want 25", got)
	}
}

func TestResolveTopN_FlagOverridesAIConfig(t *testing.T) {
	dir := t.TempDir()
	root := "docs"

	testutil.MustWriteFile(t, filepath.Join(dir, root, "codectx.yml"), `name: test
author: test
version: "0.1.0"
root: docs
`)
	testutil.MustWriteFile(t, filepath.Join(dir, root, project.CodectxDir, "ai.yml"), `consumption:
  results_count: 25
`)

	cfg := &project.Config{Root: root}
	got := ResolveTopN(3, dir, cfg)

	if got != 3 {
		t.Errorf("ResolveTopN(3, ...) with AI config = %d, want 3", got)
	}
}

// ---------------------------------------------------------------------------
// WarnHistory
// ---------------------------------------------------------------------------

func TestWarnHistory(t *testing.T) {
	t.Run("writes warning to stderr", func(t *testing.T) {
		// Capture stderr.
		origStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stderr = w
		t.Cleanup(func() { os.Stderr = origStderr })

		WarnHistory("save document", fmt.Errorf("disk full"))

		_ = w.Close()
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r); err != nil {
			t.Fatal(err)
		}

		output := buf.String()
		if output == "" {
			t.Fatal("expected stderr output, got empty string")
		}

		// Verify the output contains expected content.
		if !bytes.Contains([]byte(output), []byte("save document")) {
			t.Errorf("output should contain action name, got: %s", output)
		}
		if !bytes.Contains([]byte(output), []byte("disk full")) {
			t.Errorf("output should contain error message, got: %s", output)
		}
	})
}
