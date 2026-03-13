package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildHistoryPath_EmptyDocFile(t *testing.T) {
	got := buildHistoryPath("/some/dir", "")
	if got != "" {
		t.Errorf("expected empty path for empty docFile, got %q", got)
	}
}

func TestBuildHistoryPath_RelativePath(t *testing.T) {
	// Create a temp dir structure and chdir into it.
	tmpDir := t.TempDir()
	histDir := filepath.Join(tmpDir, ".codectx", "history")
	if err := os.MkdirAll(filepath.Join(histDir, "docs"), 0755); err != nil {
		t.Fatal(err)
	}

	// Save and restore the working directory.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatal(err)
		}
	})

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	got := buildHistoryPath(histDir, "1700000000000000000.abc123456789.md")
	if got == "" {
		t.Fatal("expected non-empty path")
	}

	// Should be relative since we're in the parent dir.
	if filepath.IsAbs(got) {
		t.Errorf("expected relative path, got %q", got)
	}

	// Should contain the docs subdirectory.
	if !strings.Contains(got, "docs") {
		t.Errorf("path should contain 'docs', got %q", got)
	}
}

func TestOutputDocument_FileMode(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.md")

	content := []byte("# Test Document")
	summary := "Summary text"

	err := outputDocument(content, summary, filePath)
	if err != nil {
		t.Fatalf("outputDocument: %v", err)
	}

	// Verify file was written.
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

func TestOutputDocument_FileMode_InvalidPath(t *testing.T) {
	err := outputDocument([]byte("content"), "summary", "/nonexistent/deep/path/file.md")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestOutputDocument_StdoutMode(t *testing.T) {
	// In stdout mode (empty filePath), the function writes to os.Stdout
	// and os.Stderr. We can verify it doesn't error.
	// For a thorough test we'd capture stdout/stderr, but the key thing
	// is that it doesn't panic or error.
	origStdout := os.Stdout
	origStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = wOut
	os.Stderr = wErr
	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	})

	outErr := outputDocument([]byte("doc content"), "summary text", "")

	_ = wOut.Close()
	_ = wErr.Close()

	if outErr != nil {
		t.Fatalf("outputDocument: %v", outErr)
	}

	// Read captured stdout.
	buf := make([]byte, 1024)
	n, _ := rOut.Read(buf)
	stdout := string(buf[:n])
	if !strings.Contains(stdout, "doc content") {
		t.Errorf("stdout should contain document, got %q", stdout)
	}

	// Read captured stderr.
	n, _ = rErr.Read(buf)
	stderr := string(buf[:n])
	if !strings.Contains(stderr, "summary text") {
		t.Errorf("stderr should contain summary, got %q", stderr)
	}
}
