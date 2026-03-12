package shared

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/registry"
)

func TestPrintConflicts(t *testing.T) {
	t.Run("no conflicts produces no output", func(t *testing.T) {
		// Capture stdout.
		origStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = w
		t.Cleanup(func() { os.Stdout = origStdout })

		PrintConflicts(nil)

		_ = w.Close()
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r); err != nil {
			t.Fatal(err)
		}

		if buf.Len() != 0 {
			t.Errorf("expected no output for nil conflicts, got: %s", buf.String())
		}
	})

	t.Run("prints conflict warnings", func(t *testing.T) {
		origStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = w
		t.Cleanup(func() { os.Stdout = origStdout })

		PrintConflicts([]registry.Conflict{
			{
				PackageRef: "pkg@org",
				Versions: map[string]string{
					"dep-a": ">=1.0.0",
					"dep-b": ">=2.0.0",
				},
			},
		})

		_ = w.Close()
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r); err != nil {
			t.Fatal(err)
		}

		output := buf.String()
		if output == "" {
			t.Fatal("expected output, got empty string")
		}
		if !bytes.Contains([]byte(output), []byte("pkg@org")) {
			t.Errorf("output should contain package ref, got: %s", output)
		}
	})
}

func TestSaveLockOrError(t *testing.T) {
	t.Run("saves lock file successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "codectx.lock")

		result := &registry.ResolveResult{
			Packages: map[string]*registry.ResolvedPackage{},
		}

		err := SaveLockOrError(lockPath, result, map[string]string{}, "github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created.
		if _, err := os.Stat(lockPath); err != nil {
			t.Fatalf("lock file should exist: %v", err)
		}
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		result := &registry.ResolveResult{
			Packages: map[string]*registry.ResolvedPackage{},
		}

		err := SaveLockOrError("/nonexistent/deep/path/codectx.lock", result, map[string]string{}, "github.com")
		if err == nil {
			t.Fatal("expected error for invalid path, got nil")
		}
	})
}
