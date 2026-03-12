package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// mockHTTPClient implements HTTPClient for testing.
type mockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	if err, ok := m.errors[url]; ok {
		return nil, err
	}
	if resp, ok := m.responses[url]; ok {
		return resp, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("not found")),
	}, nil
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func textResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func errorResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func TestCheckLatest(t *testing.T) {
	t.Run("returns version from tag_name", func(t *testing.T) {
		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				releaseAPIURL: jsonResponse(`{"tag_name": "v1.2.3"}`),
			},
		}

		version, err := CheckLatest(context.Background(), client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != "1.2.3" {
			t.Errorf("got %q, want %q", version, "1.2.3")
		}
	})

	t.Run("strips v prefix", func(t *testing.T) {
		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				releaseAPIURL: jsonResponse(`{"tag_name": "v0.5.0"}`),
			},
		}

		version, err := CheckLatest(context.Background(), client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != "0.5.0" {
			t.Errorf("got %q, want %q", version, "0.5.0")
		}
	})

	t.Run("handles tag without v prefix", func(t *testing.T) {
		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				releaseAPIURL: jsonResponse(`{"tag_name": "2.0.0"}`),
			},
		}

		version, err := CheckLatest(context.Background(), client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if version != "2.0.0" {
			t.Errorf("got %q, want %q", version, "2.0.0")
		}
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				releaseAPIURL: errorResponse(http.StatusForbidden),
			},
		}

		_, err := CheckLatest(context.Background(), client)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "403") {
			t.Errorf("error should mention status code: %v", err)
		}
	})

	t.Run("returns error on empty tag_name", func(t *testing.T) {
		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				releaseAPIURL: jsonResponse(`{"tag_name": ""}`),
			},
		}

		_, err := CheckLatest(context.Background(), client)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("error should mention empty tag: %v", err)
		}
	})

	t.Run("returns error on network failure", func(t *testing.T) {
		client := &mockHTTPClient{
			errors: map[string]error{
				releaseAPIURL: fmt.Errorf("connection refused"),
			},
		}

		_, err := CheckLatest(context.Background(), client)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestNeedsUpdate(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"dev", "1.0.0", true},
		{"", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "2.0.0", true},
		{"2.0.0", "1.0.0", false},
		{"1.2.3", "1.2.3", false},
		{"1.2.3", "1.2.4", true},
		{"1.3.0", "1.2.4", false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s_vs_%s", tt.current, tt.latest)
		t.Run(name, func(t *testing.T) {
			got := NeedsUpdate(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("NeedsUpdate(%q, %q) = %v, want %v",
					tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestArchiveURL(t *testing.T) {
	url := archiveURL("1.2.3")
	expected := fmt.Sprintf(
		"https://github.com/%s/releases/download/v1.2.3/codectx.1.2.3.%s.%s.tar.gz",
		repo, runtime.GOOS, runtime.GOARCH,
	)
	if url != expected {
		t.Errorf("got %q, want %q", url, expected)
	}
}

func TestChecksumsURL(t *testing.T) {
	url := checksumsURL("1.2.3")
	expected := fmt.Sprintf(
		"https://github.com/%s/releases/download/v1.2.3/checksums.txt",
		repo,
	)
	if url != expected {
		t.Errorf("got %q, want %q", url, expected)
	}
}

func TestArchiveName(t *testing.T) {
	name := archiveName("1.2.3")
	expected := fmt.Sprintf("codectx.1.2.3.%s.%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	if name != expected {
		t.Errorf("got %q, want %q", name, expected)
	}
}

func TestDownloadChecksum(t *testing.T) {
	name := archiveName("1.0.0")

	t.Run("extracts correct hash", func(t *testing.T) {
		checksumBody := fmt.Sprintf("abc123  %s\ndef456  other-file.tar.gz\n", name)
		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				checksumsURL("1.0.0"): textResponse(checksumBody),
			},
		}

		hash, err := downloadChecksum(context.Background(), client, "1.0.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash != "abc123" {
			t.Errorf("got %q, want %q", hash, "abc123")
		}
	})

	t.Run("returns error when archive not in checksums", func(t *testing.T) {
		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				checksumsURL("1.0.0"): textResponse("abc123  other-file.tar.gz\n"),
			},
		}

		_, err := downloadChecksum(context.Background(), client, "1.0.0")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error should mention not found: %v", err)
		}
	})

	t.Run("returns error on HTTP failure", func(t *testing.T) {
		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				checksumsURL("1.0.0"): errorResponse(http.StatusNotFound),
			},
		}

		_, err := downloadChecksum(context.Background(), client, "1.0.0")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestExtractTarGz(t *testing.T) {
	t.Run("extracts regular files", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")

		// Create a tar.gz with a single file.
		createTarGzWithPaths(t, archivePath, map[string]string{
			"codectx": "#!/bin/sh\necho hello",
		})

		destDir := filepath.Join(tmpDir, "extracted")
		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := extractTarGz(archivePath, destDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(destDir, "codectx"))
		if err != nil {
			t.Fatalf("reading extracted file: %v", err)
		}
		if string(content) != "#!/bin/sh\necho hello" {
			t.Errorf("got %q, want %q", string(content), "#!/bin/sh\necho hello")
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "evil.tar.gz")

		// Manually create a tar.gz with a path traversal entry.
		createTarGzWithPaths(t, archivePath, map[string]string{
			"../../../etc/passwd": "evil content",
		})

		destDir := filepath.Join(tmpDir, "extracted")
		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.Fatal(err)
		}

		err := extractTarGz(archivePath, destDir)
		if err == nil {
			t.Fatal("expected error for path traversal, got nil")
		}
		if !strings.Contains(err.Error(), "path traversal") {
			t.Errorf("error should mention path traversal: %v", err)
		}
	})
}

func TestFileHash(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "testfile")

	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := fileHash(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])
	if hash != expected {
		t.Errorf("got %q, want %q", hash, expected)
	}
}

func TestReplace(t *testing.T) {
	t.Run("replaces binary atomically", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create "current" binary.
		currentPath := filepath.Join(tmpDir, "codectx")
		if err := os.WriteFile(currentPath, []byte("old binary"), 0755); err != nil {
			t.Fatal(err)
		}

		// Create "new" binary.
		newPath := filepath.Join(tmpDir, "codectx-new")
		if err := os.WriteFile(newPath, []byte("new binary"), 0755); err != nil {
			t.Fatal(err)
		}

		if err := Replace(currentPath, newPath); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(currentPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "new binary" {
			t.Errorf("got %q, want %q", string(content), "new binary")
		}

		// Verify permissions preserved.
		info, err := os.Stat(currentPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0111 == 0 {
			t.Error("binary should be executable")
		}
	})

	t.Run("follows symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create real binary.
		realPath := filepath.Join(tmpDir, "real-codectx")
		if err := os.WriteFile(realPath, []byte("old binary"), 0755); err != nil {
			t.Fatal(err)
		}

		// Create symlink.
		linkPath := filepath.Join(tmpDir, "codectx-link")
		if err := os.Symlink(realPath, linkPath); err != nil {
			t.Fatal(err)
		}

		// Create new binary.
		newPath := filepath.Join(tmpDir, "codectx-new")
		if err := os.WriteFile(newPath, []byte("new binary"), 0755); err != nil {
			t.Fatal(err)
		}

		if err := Replace(linkPath, newPath); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The real file should be updated.
		content, err := os.ReadFile(realPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "new binary" {
			t.Errorf("got %q, want %q", string(content), "new binary")
		}
	})
}

// createTarGzWithPaths creates a tar.gz archive preserving exact paths.
func createTarGzWithPaths(t *testing.T, path string, files map[string]string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	gw := gzip.NewWriter(f)
	defer func() { _ = gw.Close() }()

	tw := tar.NewWriter(gw)
	defer func() { _ = tw.Close() }()

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0755,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, content); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDownload(t *testing.T) {
	t.Run("downloads verifies and extracts", func(t *testing.T) {
		version := "1.0.0"
		name := archiveName(version)

		// Create a tar.gz archive in memory.
		archiveData := createTarGzBytes(t, map[string]string{
			"codectx": "fake binary content",
		})

		// Compute its SHA-256.
		h := sha256.Sum256(archiveData)
		checksum := hex.EncodeToString(h[:])
		checksumBody := fmt.Sprintf("%s  %s\n", checksum, name)

		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				checksumsURL(version): textResponse(checksumBody),
				archiveURL(version): {
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(archiveData)),
				},
			},
		}

		binaryPath, tempDir, err := Download(context.Background(), client, version)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		content, err := os.ReadFile(binaryPath)
		if err != nil {
			t.Fatalf("reading extracted binary: %v", err)
		}
		if string(content) != "fake binary content" {
			t.Errorf("got %q, want %q", string(content), "fake binary content")
		}
	})

	t.Run("fails on checksum mismatch", func(t *testing.T) {
		version := "1.0.0"
		name := archiveName(version)

		archiveData := createTarGzBytes(t, map[string]string{
			"codectx": "fake binary content",
		})

		checksumBody := fmt.Sprintf("0000000000000000000000000000000000000000000000000000000000000000  %s\n", name)

		client := &mockHTTPClient{
			responses: map[string]*http.Response{
				checksumsURL(version): textResponse(checksumBody),
				archiveURL(version): {
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(archiveData)),
				},
			},
		}

		_, _, err := Download(context.Background(), client, version)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Errorf("error should mention checksum mismatch: %v", err)
		}
	})
}

func TestDownload_ChecksumDownloadFailure(t *testing.T) {
	version := "1.0.0"

	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			checksumsURL(version): errorResponse(http.StatusNotFound),
		},
	}

	_, tempDir, err := Download(context.Background(), client, version)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Verify temp dir was cleaned up.
	if tempDir != "" {
		t.Errorf("tempDir should be empty on error, got %q", tempDir)
	}
}

func TestDownload_ArchiveDownloadFailure(t *testing.T) {
	version := "1.0.0"
	name := archiveName(version)
	checksumBody := fmt.Sprintf("abc123  %s\n", name)

	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			checksumsURL(version): textResponse(checksumBody),
			archiveURL(version):   errorResponse(http.StatusNotFound),
		},
	}

	_, tempDir, err := Download(context.Background(), client, version)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "downloading archive") {
		t.Errorf("error should mention downloading archive: %v", err)
	}
	if tempDir != "" {
		t.Errorf("tempDir should be empty on error, got %q", tempDir)
	}
}

func TestDownload_BinaryNotInArchive(t *testing.T) {
	version := "1.0.0"
	name := archiveName(version)

	// Create archive with a file named "not-codectx" instead of "codectx".
	archiveData := createTarGzBytes(t, map[string]string{
		"not-codectx": "wrong binary name",
	})

	h := sha256.Sum256(archiveData)
	checksum := hex.EncodeToString(h[:])
	checksumBody := fmt.Sprintf("%s  %s\n", checksum, name)

	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			checksumsURL(version): textResponse(checksumBody),
			archiveURL(version): {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(archiveData)),
			},
		},
	}

	_, tempDir, err := Download(context.Background(), client, version)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "binary not found") {
		t.Errorf("error should mention binary not found: %v", err)
	}
	if tempDir != "" {
		t.Errorf("tempDir should be empty on error, got %q", tempDir)
	}
}

func TestReplace_NonexistentCurrentBinary(t *testing.T) {
	tmpDir := t.TempDir()
	newPath := filepath.Join(tmpDir, "codectx-new")
	if err := os.WriteFile(newPath, []byte("new binary"), 0755); err != nil {
		t.Fatal(err)
	}

	err := Replace(filepath.Join(tmpDir, "nonexistent"), newPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resolving binary path") {
		t.Errorf("error should mention resolving path: %v", err)
	}
}

func TestReplace_NonexistentNewBinary(t *testing.T) {
	tmpDir := t.TempDir()
	currentPath := filepath.Join(tmpDir, "codectx")
	if err := os.WriteFile(currentPath, []byte("old binary"), 0755); err != nil {
		t.Fatal(err)
	}

	err := Replace(currentPath, filepath.Join(tmpDir, "nonexistent-new"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "opening new binary") {
		t.Errorf("error should mention opening new binary: %v", err)
	}
}

func TestExtractTarGz_SkipsNonRegularFiles(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	// Create a tar.gz with a directory entry and a regular file.
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add a directory entry (should be skipped).
	if err := tw.WriteHeader(&tar.Header{
		Name:     "somedir/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}); err != nil {
		t.Fatal(err)
	}

	// Add a regular file.
	content := "hello from regular file"
	if err := tw.WriteHeader(&tar.Header{
		Name:     "codectx",
		Mode:     0755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(tw, content); err != nil {
		t.Fatal(err)
	}

	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	destDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The regular file should exist.
	data, err := os.ReadFile(filepath.Join(destDir, "codectx"))
	if err != nil {
		t.Fatalf("regular file should be extracted: %v", err)
	}
	if string(data) != content {
		t.Errorf("got %q, want %q", string(data), content)
	}

	// The directory entry should NOT create a "somedir" file.
	if _, err := os.Stat(filepath.Join(destDir, "somedir")); err == nil {
		t.Error("directory entry should not be extracted as a file")
	}
}

func TestExtractTarGz_OversizedFile(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	// Create archive with a file header claiming a size > maxExtractSize.
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Write a header with a huge size (no actual body needed since we check before extracting).
	if err := tw.WriteHeader(&tar.Header{
		Name:     "codectx",
		Mode:     0755,
		Size:     maxExtractSize + 1,
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}

	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	destDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}

	err = extractTarGz(archivePath, destDir)
	if err == nil {
		t.Fatal("expected error for oversized file, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("error should mention maximum size: %v", err)
	}
}

func TestCheckLatest_InvalidJSON(t *testing.T) {
	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			releaseAPIURL: jsonResponse(`{invalid json`),
		},
	}

	_, err := CheckLatest(context.Background(), client)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "decoding") {
		t.Errorf("error should mention decoding: %v", err)
	}
}

func TestFileHash_NonexistentFile(t *testing.T) {
	_, err := fileHash("/nonexistent/path/to/file")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

// createTarGzBytes creates a tar.gz archive in memory and returns the bytes.
func createTarGzBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0755,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, content); err != nil {
			t.Fatal(err)
		}
	}

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}
