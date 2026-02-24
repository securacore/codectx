package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// githubRepo is the GitHub repository for codectx releases.
	githubRepo = "securacore/codectx"

	// upgradeTimeout is the HTTP timeout for downloading release assets.
	upgradeTimeout = 60 * time.Second
)

// Upgrade downloads and installs the specified version, replacing the
// running binary. It downloads the platform-appropriate archive from
// GitHub Releases, verifies its SHA-256 checksum, extracts the binary,
// and atomically replaces the current executable.
func Upgrade(tag string) error {
	version := strings.TrimPrefix(tag, "v")
	archive := archiveName(version)
	baseURL := releaseDownloadURL(tag)

	// Create a temp directory for all downloads.
	tmpDir, err := os.MkdirTemp("", "codectx-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Download archive and checksums.
	archivePath, err := downloadToTemp(baseURL+"/"+archive, tmpDir, archive)
	if err != nil {
		return fmt.Errorf("download archive: %w", err)
	}

	checksumPath, err := downloadToTemp(baseURL+"/checksums.txt", tmpDir, "checksums.txt")
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	// Verify checksum.
	if err := verifyChecksum(archivePath, checksumPath, archive); err != nil {
		return err
	}

	// Extract binary.
	binaryPath, err := extractBinary(archivePath, tmpDir)
	if err != nil {
		return err
	}

	// Replace running binary.
	return replaceBinary(binaryPath)
}

// archiveName returns the platform-appropriate release archive filename.
func archiveName(version string) string {
	return fmt.Sprintf("codectx_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
}

// releaseDownloadURL returns the GitHub release download base URL for a tag.
func releaseDownloadURL(tag string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s", githubRepo, tag)
}

// downloadToTemp downloads a URL to a file in dir with the given name.
// Returns the full path to the downloaded file.
func downloadToTemp(url, dir, name string) (string, error) {
	client := &http.Client{Timeout: upgradeTimeout}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("request %s: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", name, resp.StatusCode)
	}

	destPath := filepath.Join(dir, name)
	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", name, err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("write %s: %w", name, err)
	}

	_ = f.Close()
	return destPath, nil
}

// verifyChecksum reads the checksums file, finds the expected hash for the
// archive, and compares it against the actual SHA-256 of the archive file.
func verifyChecksum(archivePath, checksumPath, archiveName string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}

	expected, err := findChecksum(string(data), archiveName)
	if err != nil {
		return err
	}

	actual, err := hashFile(archivePath)
	if err != nil {
		return fmt.Errorf("hash archive: %w", err)
	}

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// findChecksum parses a checksums.txt file and returns the hash for the
// named file. The format is "hash  filename" (matching sha256sum output).
func findChecksum(content, name string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == name {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no checksum for %s in checksums file", name)
}

// hashFile computes the SHA-256 hash of a file and returns it as a hex string.
func hashFile(path string) (string, error) {
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

// extractBinary extracts the codectx binary from a tar.gz archive into dir.
// Returns the path to the extracted binary.
func extractBinary(archivePath, dir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read tar: %w", err)
		}

		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != "codectx" {
			continue
		}

		destPath := filepath.Join(dir, "codectx")
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", fmt.Errorf("create binary: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return "", fmt.Errorf("write binary: %w", err)
		}
		_ = out.Close()
		return destPath, nil
	}

	return "", fmt.Errorf("binary codectx not found in archive")
}

// replaceBinary atomically replaces the running executable with the binary
// at newPath. It resolves symlinks to find the real path, writes the new
// binary to a temp file in the same directory (ensuring same filesystem),
// and renames it over the original.
func replaceBinary(newPath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	return replaceFile(newPath, execPath)
}

// replaceFile atomically replaces target with the contents of source.
// It writes to a temp file in the same directory as target (ensuring
// same filesystem for atomic rename) and renames over the original.
func replaceFile(source, target string) error {
	newData, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read new binary: %w", err)
	}

	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".codectx-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(newData); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	_ = tmp.Close()

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod: %w", err)
	}

	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace binary: %w", err)
	}

	return nil
}
