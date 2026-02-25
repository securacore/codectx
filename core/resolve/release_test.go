package resolve

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- parseGitHubRepo ---

func TestParseGitHubRepo_httpsWithGitSuffix(t *testing.T) {
	owner, repo, ok := parseGitHubRepo("https://github.com/securacore/codectx-react.git")
	require.True(t, ok)
	assert.Equal(t, "securacore", owner)
	assert.Equal(t, "codectx-react", repo)
}

func TestParseGitHubRepo_httpsWithoutGitSuffix(t *testing.T) {
	owner, repo, ok := parseGitHubRepo("https://github.com/securacore/codectx-react")
	require.True(t, ok)
	assert.Equal(t, "securacore", owner)
	assert.Equal(t, "codectx-react", repo)
}

func TestParseGitHubRepo_wwwGitHub(t *testing.T) {
	owner, repo, ok := parseGitHubRepo("https://www.github.com/org/codectx-go.git")
	require.True(t, ok)
	assert.Equal(t, "org", owner)
	assert.Equal(t, "codectx-go", repo)
}

func TestParseGitHubRepo_nonGitHub(t *testing.T) {
	_, _, ok := parseGitHubRepo("https://gitlab.com/org/codectx-react.git")
	assert.False(t, ok)
}

func TestParseGitHubRepo_localPath(t *testing.T) {
	_, _, ok := parseGitHubRepo("/tmp/bare.git")
	assert.False(t, ok)
}

func TestParseGitHubRepo_invalidURL(t *testing.T) {
	_, _, ok := parseGitHubRepo("://bad")
	assert.False(t, ok)
}

func TestParseGitHubRepo_missingRepo(t *testing.T) {
	_, _, ok := parseGitHubRepo("https://github.com/org")
	assert.False(t, ok)
}

// --- findReleaseAsset ---

func TestFindReleaseAsset_success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"assets": [
				{"name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt"},
				{"name": "package.tar.gz", "browser_download_url": "https://example.com/package.tar.gz"}
			]
		}`))
	}))
	defer ts.Close()

	url, err := findReleaseAsset(ts.URL)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/package.tar.gz", url)
}

func TestFindReleaseAsset_noPackageAsset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"assets": [
				{"name": "other.tar.gz", "browser_download_url": "https://example.com/other.tar.gz"}
			]
		}`))
	}))
	defer ts.Close()

	_, err := findReleaseAsset(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), releaseAssetName)
}

func TestFindReleaseAsset_noRelease(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := findReleaseAsset(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFindReleaseAsset_emptyAssets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"assets": []}`))
	}))
	defer ts.Close()

	_, err := findReleaseAsset(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), releaseAssetName)
}

func TestFindReleaseAsset_invalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer ts.Close()

	_, err := findReleaseAsset(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestFindReleaseAsset_networkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	_, err := findReleaseAsset(ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "release lookup")
}

// --- extractTarGz ---

// buildTarGz creates an in-memory gzipped tarball with the given file entries.
func buildTarGz(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0o644,
		})
		require.NoError(t, err)
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return &buf
}

func TestExtractTarGz_success(t *testing.T) {
	tarball := buildTarGz(t, map[string]string{
		"manifest.yml":           "name: test\n",
		"topics/react/README.md": "# React\n",
	})

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	err := extractTarGz(tarball, destDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(destDir, "manifest.yml"))
	require.NoError(t, err)
	assert.Equal(t, "name: test\n", string(content))

	content, err = os.ReadFile(filepath.Join(destDir, "topics", "react", "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# React\n", string(content))
}

func TestExtractTarGz_directoryEntry(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Explicit directory entry.
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     "topics/",
		Mode:     0o755,
	}))

	// File inside the directory.
	content := "# Go\n"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "topics/README.md",
		Size:     int64(len(content)),
		Mode:     0o644,
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	require.NoError(t, extractTarGz(&buf, destDir))

	info, err := os.Stat(filepath.Join(destDir, "topics"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestExtractTarGz_directoryTraversal(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	_ = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "../../../etc/evil",
		Size:     5,
		Mode:     0o644,
	})
	_, _ = tw.Write([]byte("evil\n"))
	_ = tw.Close()
	_ = gw.Close()

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	err := extractTarGz(&buf, destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "escapes destination")
}

func TestExtractTarGz_dotEntry(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// A "." directory entry (produced by tar -C dir .).
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     "./",
		Mode:     0o755,
	}))

	content := "name: test\n"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "./manifest.yml",
		Size:     int64(len(content)),
		Mode:     0o644,
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	require.NoError(t, extractTarGz(&buf, destDir))

	data, err := os.ReadFile(filepath.Join(destDir, "manifest.yml"))
	require.NoError(t, err)
	assert.Equal(t, "name: test\n", string(data))
}

func TestExtractTarGz_invalidGzip(t *testing.T) {
	buf := bytes.NewBufferString("not gzip data")

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	err := extractTarGz(buf, destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gzip")
}

func TestExtractTarGz_skipsSymlinks(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Regular file.
	content := "name: test\n"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "manifest.yml",
		Size:     int64(len(content)),
		Mode:     0o644,
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)

	// Symlink — should be skipped.
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     "link.yml",
		Linkname: "manifest.yml",
	}))

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	require.NoError(t, extractTarGz(&buf, destDir))

	// Regular file should exist.
	_, err = os.Stat(filepath.Join(destDir, "manifest.yml"))
	assert.NoError(t, err)

	// Symlink should not exist.
	_, err = os.Stat(filepath.Join(destDir, "link.yml"))
	assert.True(t, os.IsNotExist(err))
}

// --- downloadAndExtract ---

func TestDownloadAndExtract_success(t *testing.T) {
	tarball := buildTarGz(t, map[string]string{
		"manifest.yml": "name: test\nversion: 1.0.0\n",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tarball.Bytes())
	}))
	defer ts.Close()

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	err := downloadAndExtract(ts.URL, destDir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(destDir, "manifest.yml"))
	assert.NoError(t, err)
}

func TestDownloadAndExtract_httpError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	err := downloadAndExtract(ts.URL, destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestDownloadAndExtract_networkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	destDir := filepath.Join(t.TempDir(), "extracted")
	require.NoError(t, os.MkdirAll(destDir, 0o755))

	err := downloadAndExtract(ts.URL, destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download asset")
}
