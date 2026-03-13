package registry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// createTestArchive creates a tar.gz file containing the specified files.
// Files are given as name/content pairs.
func createTestArchive(t *testing.T, destPath string, files map[string]string) {
	t.Helper()

	f, err := os.Create(destPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	gz := gzip.NewWriter(f)
	defer func() { _ = gz.Close() }()

	tw := tar.NewWriter(gz)
	defer func() { _ = tw.Close() }()

	// Collect unique directories.
	dirs := map[string]bool{}
	for name := range files {
		dir := filepath.Dir(name)
		for dir != "." && dir != "" {
			dirs[dir] = true
			dir = filepath.Dir(dir)
		}
	}

	// Write directory entries.
	for dir := range dirs {
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     dir + "/",
			Mode:     0755,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Write file entries.
	for name, content := range files {
		data := []byte(content)
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     name,
			Size:     int64(len(data)),
			Mode:     0644,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExtractPackageArchive_BasicFiles(t *testing.T) {
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "test.tar.gz")

	createTestArchive(t, archivePath, map[string]string{
		"codectx.yml":               "name: test-pkg\norg: testorg\nversion: 1.0.0\n",
		"foundation/overview.md":    "# Overview\nTest content\n",
		"topics/getting-started.md": "# Getting Started\nHello world\n",
	})

	destDir := t.TempDir()
	if err := ExtractPackageArchive(archivePath, destDir); err != nil {
		t.Fatalf("ExtractPackageArchive: %v", err)
	}

	// Check that files were extracted.
	assertFileExists(t, filepath.Join(destDir, "codectx.yml"))
	assertFileExists(t, filepath.Join(destDir, "foundation", "overview.md"))
	assertFileExists(t, filepath.Join(destDir, "topics", "getting-started.md"))

	// Verify content.
	data, err := os.ReadFile(filepath.Join(destDir, "codectx.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "name: test-pkg\norg: testorg\nversion: 1.0.0\n" {
		t.Errorf("unexpected content: %s", string(data))
	}
}

func TestExtractPackageArchive_DotSlashPrefix(t *testing.T) {
	// Archives created with `tar -C dir .` have "./" prefixed entries.
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "test.tar.gz")

	createTestArchive(t, archivePath, map[string]string{
		"./codectx.yml":            "name: test\n",
		"./foundation/overview.md": "# Overview\n",
	})

	destDir := t.TempDir()
	if err := ExtractPackageArchive(archivePath, destDir); err != nil {
		t.Fatalf("ExtractPackageArchive: %v", err)
	}

	assertFileExists(t, filepath.Join(destDir, "codectx.yml"))
	assertFileExists(t, filepath.Join(destDir, "foundation", "overview.md"))
}

func TestExtractPackageArchive_RejectsPathTraversal(t *testing.T) {
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "test.tar.gz")

	// Create archive with path traversal.
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	data := []byte("malicious content")
	_ = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "../../../etc/passwd",
		Size:     int64(len(data)),
		Mode:     0644,
	})
	_, _ = tw.Write(data)
	_ = tw.Close()
	_ = gz.Close()
	_ = f.Close()

	destDir := t.TempDir()
	err = ExtractPackageArchive(archivePath, destDir)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestExtractPackageArchive_InvalidArchive(t *testing.T) {
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "bad.tar.gz")

	if err := os.WriteFile(archivePath, []byte("not a tar.gz"), 0644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	err := ExtractPackageArchive(archivePath, destDir)
	if err == nil {
		t.Error("expected error for invalid archive")
	}
}

func TestExtractPackageArchive_NonexistentFile(t *testing.T) {
	destDir := t.TempDir()
	err := ExtractPackageArchive("/nonexistent/path.tar.gz", destDir)
	if err == nil {
		t.Error("expected error for nonexistent archive")
	}
}

func TestDownloadArchive(t *testing.T) {
	content := "test archive content bytes"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte(content))
	}))
	defer ts.Close()

	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "package.tar.gz")

	err := DownloadArchive(context.Background(), http.DefaultClient, ts.URL, destPath)
	if err != nil {
		t.Fatalf("DownloadArchive: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}
}

func TestDownloadArchive_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "package.tar.gz")

	err := DownloadArchive(context.Background(), http.DefaultClient, ts.URL, destPath)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestInstallPackageFromArchive(t *testing.T) {
	// Create a real tar.gz to serve.
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "package.tar.gz")

	createTestArchive(t, archivePath, map[string]string{
		"codectx.yml":            "name: test-pkg\norg: testorg\nversion: 1.0.0\n",
		"foundation/overview.md": "# Overview\n",
	})

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(archiveData)
	}))
	defer ts.Close()

	destDir := filepath.Join(t.TempDir(), "test-pkg@testorg")
	err = InstallPackageFromArchive(context.Background(), http.DefaultClient, ts.URL, destDir)
	if err != nil {
		t.Fatalf("InstallPackageFromArchive: %v", err)
	}

	assertFileExists(t, filepath.Join(destDir, "codectx.yml"))
	assertFileExists(t, filepath.Join(destDir, "foundation", "overview.md"))
}

func TestInstallPackageFromArchive_CleansExistingDir(t *testing.T) {
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "package.tar.gz")

	createTestArchive(t, archivePath, map[string]string{
		"codectx.yml": "name: new\nversion: 2.0.0\n",
	})

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archiveData)
	}))
	defer ts.Close()

	destDir := filepath.Join(t.TempDir(), "pkg")
	// Pre-create with old content.
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "old-file.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := InstallPackageFromArchive(context.Background(), http.DefaultClient, ts.URL, destDir); err != nil {
		t.Fatal(err)
	}

	// Old file should be gone.
	if _, err := os.Stat(filepath.Join(destDir, "old-file.txt")); !os.IsNotExist(err) {
		t.Error("expected old file to be removed")
	}

	// New file should exist.
	assertFileExists(t, filepath.Join(destDir, "codectx.yml"))
}

func TestAuthenticatedHTTPClient_NoToken(t *testing.T) {
	client := AuthenticatedHTTPClient("")
	if client == nil {
		t.Error("expected non-nil client")
	}
	// Should be the default client.
	if _, ok := client.(*http.Client); !ok {
		t.Error("expected *http.Client for empty token")
	}
}

func TestAuthenticatedHTTPClient_WithToken(t *testing.T) {
	var receivedAuth string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := AuthenticatedHTTPClient("test-token-123")

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if receivedAuth != "Bearer test-token-123" {
		t.Errorf("expected auth header %q, got %q", "Bearer test-token-123", receivedAuth)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}
