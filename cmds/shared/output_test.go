package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputDocument_FileMode(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.md")

	err := OutputDocument(OutputDocumentParams{
		Content:  []byte("# Test Document"),
		FilePath: filePath,
		Footer:   "Summary text",
	})
	if err != nil {
		t.Fatalf("OutputDocument: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if string(data) != "# Test Document" {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

func TestOutputDocument_FileMode_WithHeaderAndFooter(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.md")

	err := OutputDocument(OutputDocumentParams{
		Content:  []byte("# Test Document"),
		FilePath: filePath,
		Header:   "header text",
		Footer:   "footer text",
	})
	if err != nil {
		t.Fatalf("OutputDocument: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if string(data) != "# Test Document" {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

func TestOutputDocument_FileMode_InvalidPath(t *testing.T) {
	err := OutputDocument(OutputDocumentParams{
		Content:  []byte("content"),
		FilePath: "/nonexistent/deep/path/file.md",
		Footer:   "summary",
	})
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestOutputDocument_StdoutMode_FooterOnly(t *testing.T) {
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

	outErr := OutputDocument(OutputDocumentParams{
		Content: []byte("doc content"),
		Footer:  "summary text",
	})

	_ = wOut.Close()
	_ = wErr.Close()

	if outErr != nil {
		t.Fatalf("OutputDocument: %v", outErr)
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

func TestOutputDocument_StdoutMode_HeaderAndFooter(t *testing.T) {
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

	outErr := OutputDocument(OutputDocumentParams{
		Content: []byte("doc content"),
		Header:  "header text",
		Footer:  "footer text",
	})

	_ = wOut.Close()
	_ = wErr.Close()

	if outErr != nil {
		t.Fatalf("OutputDocument: %v", outErr)
	}

	buf := make([]byte, 4096)
	n, _ := rOut.Read(buf)
	stdout := string(buf[:n])
	if !strings.Contains(stdout, "doc content") {
		t.Errorf("stdout should contain document, got %q", stdout)
	}

	n, _ = rErr.Read(buf)
	stderr := string(buf[:n])
	if !strings.Contains(stderr, "header text") {
		t.Errorf("stderr should contain header, got %q", stderr)
	}
	if !strings.Contains(stderr, "footer text") {
		t.Errorf("stderr should contain footer, got %q", stderr)
	}
}
