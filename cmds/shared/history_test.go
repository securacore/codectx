package shared

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/project"
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
