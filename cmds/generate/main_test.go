package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	buf := make([]byte, 1024)
	n, _ := rOut.Read(buf)
	stdout := string(buf[:n])
	if !strings.Contains(stdout, "doc content") {
		t.Errorf("stdout should contain document, got %q", stdout)
	}

	n, _ = rErr.Read(buf)
	stderr := string(buf[:n])
	if !strings.Contains(stderr, "summary text") {
		t.Errorf("stderr should contain summary, got %q", stderr)
	}
}
