package resolve

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// releaseAssetName is the conventional filename for the package tarball
// in GitHub Releases. Package release workflows produce this asset
// containing the package contents with manifest.yml at the archive root.
const releaseAssetName = "package.tar.gz"

// githubRelease maps the subset of the GitHub release response needed
// to locate the package tarball asset.
type githubRelease struct {
	Assets []githubAsset `json:"assets"`
}

// githubAsset maps a single asset entry from a GitHub release.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// fetchReleaseAsset attempts to download and extract a package tarball
// from a GitHub Release matching the resolved tag. It creates destDir
// and extracts the archive contents into it. Returns an error if the
// source is not a GitHub URL, no release exists for the tag, or the
// release has no package.tar.gz asset.
func fetchReleaseAsset(resolved *ResolvedPackage, destDir string) error {
	owner, repo, ok := parseGitHubRepo(resolved.Source)
	if !ok {
		return fmt.Errorf("not a GitHub source")
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s",
		owner, repo, resolved.Tag)

	assetURL, err := findReleaseAsset(apiURL)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create destination %s: %w", destDir, err)
	}

	return downloadAndExtract(assetURL, destDir)
}

// parseGitHubRepo extracts the owner and repository name from a GitHub
// source URL. Returns empty strings and false for non-GitHub URLs.
func parseGitHubRepo(source string) (owner, repo string, ok bool) {
	u, err := url.Parse(source)
	if err != nil {
		return "", "", false
	}

	if u.Host != "github.com" && u.Host != "www.github.com" {
		return "", "", false
	}

	path := strings.Trim(u.Path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return "", "", false
	}

	return parts[0], strings.TrimSuffix(parts[1], ".git"), true
}

// findReleaseAsset queries the GitHub Releases API at the given URL
// and returns the download URL for the package.tar.gz asset.
// Extracted from fetchReleaseAsset for testability.
func findReleaseAsset(apiURL string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("release lookup: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("no release for tag (HTTP %d)", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode release: %w", err)
	}

	for _, asset := range release.Assets {
		if asset.Name == releaseAssetName {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("release has no %s asset", releaseAssetName)
}

// downloadAndExtract downloads a tarball from assetURL and extracts it
// into destDir.
func downloadAndExtract(assetURL, destDir string) error {
	client := &http.Client{Timeout: 60 * time.Second}

	resp, err := client.Get(assetURL)
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download asset: HTTP %d", resp.StatusCode)
	}

	return extractTarGz(resp.Body, destDir)
}

// extractTarGz decompresses and extracts a gzipped tar archive into destDir.
// It validates that no tar entry escapes the destination directory.
// Symlinks and other non-regular entry types are silently skipped.
func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		name := filepath.Clean(hdr.Name)
		if name == "." {
			continue
		}

		target := filepath.Join(destDir, name)
		if !strings.HasPrefix(target, cleanDest) {
			return fmt.Errorf("tar entry %q escapes destination", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create dir: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create parent dir: %w", err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return fmt.Errorf("write file: %w", err)
			}
			_ = f.Close()
		}
	}

	return nil
}
