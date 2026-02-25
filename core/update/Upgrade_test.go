package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- archiveName ---

func TestArchiveName(t *testing.T) {
	name := archiveName("0.5.0")
	expected := fmt.Sprintf("codectx_0.5.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	assert.Equal(t, expected, name)
}

func TestArchiveName_differentVersion(t *testing.T) {
	name := archiveName("1.2.3")
	assert.Contains(t, name, "codectx_1.2.3_")
	assert.Contains(t, name, ".tar.gz")
}

// --- releaseDownloadURL ---

func TestReleaseDownloadURL(t *testing.T) {
	url := releaseDownloadURL("v0.5.0")
	assert.Equal(t, "https://github.com/securacore/codectx/releases/download/v0.5.0", url)
}

// --- findChecksum ---

func TestFindChecksum_success(t *testing.T) {
	content := "abc123  codectx_0.5.0_linux_amd64.tar.gz\ndef456  codectx_0.5.0_darwin_arm64.tar.gz\n"
	hash, err := findChecksum(content, "codectx_0.5.0_linux_amd64.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, "abc123", hash)
}

func TestFindChecksum_secondEntry(t *testing.T) {
	content := "abc123  codectx_0.5.0_linux_amd64.tar.gz\ndef456  codectx_0.5.0_darwin_arm64.tar.gz\n"
	hash, err := findChecksum(content, "codectx_0.5.0_darwin_arm64.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, "def456", hash)
}

func TestFindChecksum_notFound(t *testing.T) {
	content := "abc123  codectx_0.5.0_linux_amd64.tar.gz\n"
	_, err := findChecksum(content, "codectx_0.5.0_darwin_arm64.tar.gz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no checksum")
}

func TestFindChecksum_emptyContent(t *testing.T) {
	_, err := findChecksum("", "codectx_0.5.0_linux_amd64.tar.gz")
	assert.Error(t, err)
}

func TestFindChecksum_blankLines(t *testing.T) {
	content := "\nabc123  test.tar.gz\n\n"
	hash, err := findChecksum(content, "test.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, "abc123", hash)
}

// --- hashFile ---

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world\n")
	require.NoError(t, os.WriteFile(path, content, 0o644))

	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	actual, err := hashFile(path)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestHashFile_missing(t *testing.T) {
	_, err := hashFile("/nonexistent/file")
	assert.Error(t, err)
}

// --- verifyChecksum ---

func TestVerifyChecksum_valid(t *testing.T) {
	dir := t.TempDir()

	archiveContent := []byte("fake archive content")
	archivePath := filepath.Join(dir, "archive.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, archiveContent, 0o644))

	h := sha256.Sum256(archiveContent)
	hash := hex.EncodeToString(h[:])
	checksumContent := fmt.Sprintf("%s  archive.tar.gz\n", hash)
	checksumPath := filepath.Join(dir, "checksums.txt")
	require.NoError(t, os.WriteFile(checksumPath, []byte(checksumContent), 0o644))

	err := verifyChecksum(archivePath, checksumPath, "archive.tar.gz")
	assert.NoError(t, err)
}

func TestVerifyChecksum_mismatch(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "archive.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, []byte("content"), 0o644))

	checksumContent := "0000000000000000000000000000000000000000000000000000000000000000  archive.tar.gz\n"
	checksumPath := filepath.Join(dir, "checksums.txt")
	require.NoError(t, os.WriteFile(checksumPath, []byte(checksumContent), 0o644))

	err := verifyChecksum(archivePath, checksumPath, "archive.tar.gz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestVerifyChecksum_missingEntry(t *testing.T) {
	dir := t.TempDir()

	archivePath := filepath.Join(dir, "archive.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, []byte("content"), 0o644))

	checksumContent := "abc123  other.tar.gz\n"
	checksumPath := filepath.Join(dir, "checksums.txt")
	require.NoError(t, os.WriteFile(checksumPath, []byte(checksumContent), 0o644))

	err := verifyChecksum(archivePath, checksumPath, "archive.tar.gz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no checksum")
}

// --- extractBinary ---

// buildTestArchive creates a tar.gz archive with the given file entries.
func buildTestArchive(t *testing.T, files map[string][]byte) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "archive.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0o755,
		})
		require.NoError(t, err)
		_, err = tw.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())
	return path
}

func TestExtractBinary_success(t *testing.T) {
	archivePath := buildTestArchive(t, map[string][]byte{
		"codectx": []byte("#!/bin/sh\necho hello\n"),
	})

	dir := t.TempDir()
	binaryPath, err := extractBinary(archivePath, dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "codectx"), binaryPath)

	content, err := os.ReadFile(binaryPath)
	require.NoError(t, err)
	assert.Equal(t, "#!/bin/sh\necho hello\n", string(content))

	info, err := os.Stat(binaryPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&0o111 != 0, "binary should be executable")
}

func TestExtractBinary_skipsNonBinary(t *testing.T) {
	archivePath := buildTestArchive(t, map[string][]byte{
		"README.md": []byte("# Test\n"),
		"codectx":   []byte("binary content"),
	})

	dir := t.TempDir()
	binaryPath, err := extractBinary(archivePath, dir)
	require.NoError(t, err)

	content, err := os.ReadFile(binaryPath)
	require.NoError(t, err)
	assert.Equal(t, "binary content", string(content))
}

func TestExtractBinary_notFound(t *testing.T) {
	archivePath := buildTestArchive(t, map[string][]byte{
		"README.md": []byte("# Test\n"),
	})

	dir := t.TempDir()
	_, err := extractBinary(archivePath, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExtractBinary_invalidArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.tar.gz")
	require.NoError(t, os.WriteFile(path, []byte("not a gzip"), 0o644))

	_, err := extractBinary(path, t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gzip")
}

// --- downloadToTemp ---

func TestDownloadToTemp_success(t *testing.T) {
	content := []byte("file content")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path, err := downloadToTemp(ts.URL, dir, "test.txt")
	require.NoError(t, err)

	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, actual)
	assert.Equal(t, filepath.Join(dir, "test.txt"), path)
}

func TestDownloadToTemp_httpError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	dir := t.TempDir()
	_, err := downloadToTemp(ts.URL, dir, "test.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDownloadToTemp_networkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	dir := t.TempDir()
	_, err := downloadToTemp(ts.URL, dir, "test.txt")
	assert.Error(t, err)
}

// --- replaceFile ---

func TestReplaceFile_success(t *testing.T) {
	dir := t.TempDir()

	targetPath := filepath.Join(dir, "codectx")
	require.NoError(t, os.WriteFile(targetPath, []byte("old version"), 0o755))

	sourcePath := filepath.Join(t.TempDir(), "codectx-new")
	require.NoError(t, os.WriteFile(sourcePath, []byte("new version"), 0o755))

	err := replaceFile(sourcePath, targetPath)
	require.NoError(t, err)

	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, "new version", string(content))

	info, err := os.Stat(targetPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&0o111 != 0, "should be executable")
}

func TestReplaceFile_missingSource(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "codectx")
	require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0o755))

	err := replaceFile("/nonexistent/binary", targetPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read new binary")

	// Original should be untouched.
	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, "old", string(content))
}

func TestReplaceFile_preservesContent(t *testing.T) {
	dir := t.TempDir()

	// Write a larger binary to verify complete content transfer.
	bigContent := make([]byte, 1024*1024)
	for i := range bigContent {
		bigContent[i] = byte(i % 256)
	}

	targetPath := filepath.Join(dir, "codectx")
	require.NoError(t, os.WriteFile(targetPath, []byte("old"), 0o755))

	sourcePath := filepath.Join(t.TempDir(), "codectx-new")
	require.NoError(t, os.WriteFile(sourcePath, bigContent, 0o755))

	require.NoError(t, replaceFile(sourcePath, targetPath))

	actual, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, bigContent, actual)
}

// --- replaceBinary ---

func TestReplaceBinary_success(t *testing.T) {
	// Create a fake executable target.
	dir := t.TempDir()
	fakeExec := filepath.Join(dir, "codectx")
	require.NoError(t, os.WriteFile(fakeExec, []byte("old binary"), 0o755))

	// Override resolveExecutable to point at our fake binary.
	origResolve := resolveExecutable
	resolveExecutable = func() (string, error) { return fakeExec, nil }
	t.Cleanup(func() { resolveExecutable = origResolve })

	// Create the new binary source.
	newBin := filepath.Join(t.TempDir(), "codectx-new")
	require.NoError(t, os.WriteFile(newBin, []byte("new binary"), 0o755))

	err := replaceBinary(newBin)
	require.NoError(t, err)

	content, err := os.ReadFile(fakeExec)
	require.NoError(t, err)
	assert.Equal(t, "new binary", string(content))
}

func TestReplaceBinary_resolveError(t *testing.T) {
	origResolve := resolveExecutable
	resolveExecutable = func() (string, error) {
		return "", fmt.Errorf("resolve symlinks: cannot resolve")
	}
	t.Cleanup(func() { resolveExecutable = origResolve })

	err := replaceBinary("/some/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolve symlinks")
}

// --- Upgrade (integration) ---

// buildReleaseArchive creates a tar.gz archive containing a codectx binary
// with the given content, and returns the archive bytes and its SHA-256 hash.
func buildReleaseArchive(t *testing.T, binaryContent []byte) (archiveBytes []byte, hash string) {
	t.Helper()

	path := buildTestArchive(t, map[string][]byte{
		"codectx": binaryContent,
	})
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	h := sha256.Sum256(data)
	return data, hex.EncodeToString(h[:])
}

// startReleaseServer starts an httptest server that serves a release archive
// and checksums.txt for the given version. Returns the server and the
// expected archive name.
func startReleaseServer(t *testing.T, version string, binaryContent []byte) *httptest.Server {
	t.Helper()

	archive := archiveName(version)
	archiveData, checksum := buildReleaseArchive(t, binaryContent)
	checksumTxt := fmt.Sprintf("%s  %s\n", checksum, archive)

	archivePath := "/" + archive
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case archivePath:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(archiveData)
		case "/checksums.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(checksumTxt))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestUpgrade_success(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho upgraded\n")
	ts := startReleaseServer(t, "1.0.0", binaryContent)

	// Override download base to use the test server.
	origBase := upgradeDownloadBase
	upgradeDownloadBase = func(tag string) string { return ts.URL }
	t.Cleanup(func() { upgradeDownloadBase = origBase })

	// Create a fake executable target.
	dir := t.TempDir()
	fakeExec := filepath.Join(dir, "codectx")
	require.NoError(t, os.WriteFile(fakeExec, []byte("old"), 0o755))

	origResolve := resolveExecutable
	resolveExecutable = func() (string, error) { return fakeExec, nil }
	t.Cleanup(func() { resolveExecutable = origResolve })

	err := Upgrade("v1.0.0")
	require.NoError(t, err)

	// Verify the binary was replaced with the new content.
	content, err := os.ReadFile(fakeExec)
	require.NoError(t, err)
	assert.Equal(t, string(binaryContent), string(content))

	// Verify the binary is executable.
	info, err := os.Stat(fakeExec)
	require.NoError(t, err)
	assert.True(t, info.Mode()&0o111 != 0, "replaced binary should be executable")
}

func TestUpgrade_stripsVPrefix(t *testing.T) {
	// Verify that Upgrade("v1.0.0") correctly strips the "v" prefix
	// and requests the right archive name (codectx_1.0.0_...).
	binaryContent := []byte("new binary")
	ts := startReleaseServer(t, "1.0.0", binaryContent)

	origBase := upgradeDownloadBase
	upgradeDownloadBase = func(tag string) string { return ts.URL }
	t.Cleanup(func() { upgradeDownloadBase = origBase })

	dir := t.TempDir()
	fakeExec := filepath.Join(dir, "codectx")
	require.NoError(t, os.WriteFile(fakeExec, []byte("old"), 0o755))

	origResolve := resolveExecutable
	resolveExecutable = func() (string, error) { return fakeExec, nil }
	t.Cleanup(func() { resolveExecutable = origResolve })

	// Should work with "v" prefix.
	err := Upgrade("v1.0.0")
	require.NoError(t, err)

	content, err := os.ReadFile(fakeExec)
	require.NoError(t, err)
	assert.Equal(t, string(binaryContent), string(content))
}

func TestUpgrade_checksumMismatch(t *testing.T) {
	// Serve an archive with a wrong checksum.
	version := "1.0.0"
	archive := archiveName(version)
	archivePath := buildTestArchive(t, map[string][]byte{
		"codectx": []byte("binary"),
	})
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	// Checksum deliberately wrong.
	checksumTxt := fmt.Sprintf("0000000000000000000000000000000000000000000000000000000000000000  %s\n", archive)

	archiveURLPath := "/" + archive
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case archiveURLPath:
			_, _ = w.Write(archiveData)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksumTxt))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	origBase := upgradeDownloadBase
	upgradeDownloadBase = func(tag string) string { return ts.URL }
	t.Cleanup(func() { upgradeDownloadBase = origBase })

	dir := t.TempDir()
	fakeExec := filepath.Join(dir, "codectx")
	require.NoError(t, os.WriteFile(fakeExec, []byte("old"), 0o755))

	origResolve := resolveExecutable
	resolveExecutable = func() (string, error) { return fakeExec, nil }
	t.Cleanup(func() { resolveExecutable = origResolve })

	err = Upgrade("v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	// Original binary should be untouched.
	content, err := os.ReadFile(fakeExec)
	require.NoError(t, err)
	assert.Equal(t, "old", string(content))
}

func TestUpgrade_archiveDownloadFails(t *testing.T) {
	// Server returns 404 for the archive.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(ts.Close)

	origBase := upgradeDownloadBase
	upgradeDownloadBase = func(tag string) string { return ts.URL }
	t.Cleanup(func() { upgradeDownloadBase = origBase })

	err := Upgrade("v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download archive")
}

func TestUpgrade_checksumDownloadFails(t *testing.T) {
	// Server serves the archive but 404s the checksums.
	version := "1.0.0"
	archive := archiveName(version)
	archivePath := buildTestArchive(t, map[string][]byte{
		"codectx": []byte("binary"),
	})
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/"+archive {
			_, _ = w.Write(archiveData)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(ts.Close)

	origBase := upgradeDownloadBase
	upgradeDownloadBase = func(tag string) string { return ts.URL }
	t.Cleanup(func() { upgradeDownloadBase = origBase })

	err = Upgrade("v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download checksums")
}

func TestUpgrade_archiveNoBinary(t *testing.T) {
	// Serve an archive that doesn't contain the codectx binary.
	version := "1.0.0"
	archive := archiveName(version)
	archivePath := buildTestArchive(t, map[string][]byte{
		"README.md": []byte("# not a binary"),
	})
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	h := sha256.Sum256(archiveData)
	checksumTxt := fmt.Sprintf("%s  %s\n", hex.EncodeToString(h[:]), archive)

	archiveURLPath := "/" + archive
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case archiveURLPath:
			_, _ = w.Write(archiveData)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksumTxt))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	origBase := upgradeDownloadBase
	upgradeDownloadBase = func(tag string) string { return ts.URL }
	t.Cleanup(func() { upgradeDownloadBase = origBase })

	dir := t.TempDir()
	fakeExec := filepath.Join(dir, "codectx")
	require.NoError(t, os.WriteFile(fakeExec, []byte("old"), 0o755))

	origResolve := resolveExecutable
	resolveExecutable = func() (string, error) { return fakeExec, nil }
	t.Cleanup(func() { resolveExecutable = origResolve })

	err = Upgrade("v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpgrade_replaceBinaryFails(t *testing.T) {
	binaryContent := []byte("new binary")
	ts := startReleaseServer(t, "1.0.0", binaryContent)

	origBase := upgradeDownloadBase
	upgradeDownloadBase = func(tag string) string { return ts.URL }
	t.Cleanup(func() { upgradeDownloadBase = origBase })

	// resolveExecutable returns a path in a non-existent directory so
	// replaceFile's CreateTemp fails.
	origResolve := resolveExecutable
	resolveExecutable = func() (string, error) {
		return "/nonexistent-dir/codectx", nil
	}
	t.Cleanup(func() { resolveExecutable = origResolve })

	err := Upgrade("v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create temp file")
}
