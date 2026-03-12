// Package selfupdate provides functionality to update the codectx binary
// from GitHub Releases. It checks for new versions, downloads the appropriate
// archive, verifies its SHA-256 checksum, and atomically replaces the running
// binary.
package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

// repo is the GitHub owner/repo for codectx releases.
const repo = "securacore/codectx"

// releaseAPIURL is the GitHub API endpoint for the latest release.
// Declared as a variable so tests can override it.
var releaseAPIURL = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

// githubRelease is the subset of GitHub's release API response we need.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// httpClient is the interface used for HTTP requests, allowing test injection.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// CheckLatest queries the GitHub Releases API for the latest version tag.
// Returns the version string without the "v" prefix.
func CheckLatest(ctx context.Context, client httpClient) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	// Use GitHub token if available for higher rate limits.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decoding release response: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in release response")
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

// NeedsUpdate compares the current version against the latest.
// Returns true if the latest version is newer. Both versions should
// be without "v" prefix.
func NeedsUpdate(current, latest string) bool {
	if current == "dev" || current == "" {
		return true
	}
	// semver.Compare requires "v" prefix.
	return semver.Compare("v"+current, "v"+latest) < 0
}

// archiveURL builds the download URL for the release archive.
func archiveURL(version string) string {
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/v%s/codectx.%s.%s.%s.tar.gz",
		repo, version, version, runtime.GOOS, runtime.GOARCH,
	)
}

// checksumsURL builds the download URL for the checksums file.
func checksumsURL(version string) string {
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/v%s/checksums.txt",
		repo, version,
	)
}

// archiveName returns the expected archive filename for checksum lookup.
func archiveName(version string) string {
	return fmt.Sprintf("codectx.%s.%s.%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
}

// Download fetches the release archive and checksums, verifies the checksum,
// extracts the binary, and returns the path to the extracted binary in a
// temporary directory. The caller is responsible for cleaning up the temp dir.
func Download(ctx context.Context, client httpClient, version string) (binaryPath string, tempDir string, err error) {
	tempDir, err = os.MkdirTemp("", "codectx-update-*")
	if err != nil {
		return "", "", fmt.Errorf("creating temp dir: %w", err)
	}

	// Clean up the temp dir on any error after this point.
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tempDir)
			tempDir = ""
			binaryPath = ""
		}
	}()

	// Download checksums.
	expectedHash, err := downloadChecksum(ctx, client, version)
	if err != nil {
		return "", "", err
	}

	// Download archive.
	archPath := filepath.Join(tempDir, archiveName(version))
	if err = downloadFile(ctx, client, archiveURL(version), archPath); err != nil {
		return "", "", fmt.Errorf("downloading archive: %w", err)
	}

	// Verify checksum.
	actualHash, err := fileHash(archPath)
	if err != nil {
		return "", "", fmt.Errorf("computing checksum: %w", err)
	}
	if actualHash != expectedHash {
		return "", "", fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	// Extract the archive.
	if err = extractTarGz(archPath, tempDir); err != nil {
		return "", "", fmt.Errorf("extracting archive: %w", err)
	}

	binaryPath = filepath.Join(tempDir, "codectx")
	if _, err = os.Stat(binaryPath); err != nil {
		return "", "", fmt.Errorf("binary not found in archive: %w", err)
	}

	return binaryPath, tempDir, nil
}

// Replace atomically replaces the current binary with the new one.
// It copies the new binary to a temp file next to the current one, then
// renames it in place.
func Replace(currentBinary, newBinary string) error {
	// Resolve symlinks to get the real binary path.
	realPath, err := filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	// Get the permissions of the current binary.
	info, err := os.Stat(realPath)
	if err != nil {
		return fmt.Errorf("stat current binary: %w", err)
	}

	// Write to a temp file in the same directory (ensures same filesystem for rename).
	dir := filepath.Dir(realPath)
	tmp, err := os.CreateTemp(dir, "codectx-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	src, err := os.Open(newBinary)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("opening new binary: %w", err)
	}

	if _, err := io.Copy(tmp, src); err != nil {
		_ = src.Close()
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("copying new binary: %w", err)
	}
	_ = src.Close()
	_ = tmp.Close()

	if err := os.Chmod(tmpPath, info.Mode()); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmpPath, realPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

// downloadChecksum fetches checksums.txt and extracts the hash for our archive.
func downloadChecksum(ctx context.Context, client httpClient, version string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsURL(version), nil)
	if err != nil {
		return "", fmt.Errorf("creating checksum request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching checksums: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading checksums: %w", err)
	}

	name := archiveName(version)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == name {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("checksum not found for %s", name)
}

// downloadFile downloads a URL to a local file.
func downloadFile(ctx context.Context, client httpClient, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// fileHash computes the SHA-256 hex digest of a file.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
