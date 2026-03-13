package registry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// maxArchiveSize is the maximum allowed total extracted size (500MB).
const maxArchiveSize = 500 * 1024 * 1024

// maxSingleFile is the maximum size of a single file in the archive (50MB).
const maxSingleFile = 50 * 1024 * 1024

// PackageArchiveName is the standard archive filename attached to GitHub Releases.
const PackageArchiveName = "package.tar.gz"

// HTTPClient is the interface for HTTP requests, matching *http.Client.
// Allows test injection.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DownloadArchive downloads a file from the given URL and saves it to destPath.
// Uses the provided HTTP client (which may include auth headers).
func DownloadArchive(ctx context.Context, client HTTPClient, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s returned status %d", url, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", destPath, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, io.LimitReader(resp.Body, maxArchiveSize)); err != nil {
		return fmt.Errorf("writing archive: %w", err)
	}

	return nil
}

// ExtractPackageArchive extracts a package tar.gz archive to the destination
// directory, preserving directory structure.
//
// The archive is expected to contain the contents of a package/ directory:
// codectx.yml, foundation/, topics/, plans/, prompts/. The archive contents
// are extracted directly into destDir.
//
// Security measures:
//   - Path traversal ("../") is rejected
//   - Individual file size is capped at maxSingleFile
//   - Total extracted size is capped at maxArchiveSize
//   - Only regular files and directories are extracted
func ExtractPackageArchive(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	var totalSize int64

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		// Clean the path and reject traversal.
		name := filepath.Clean(header.Name)
		if strings.Contains(name, "..") {
			return fmt.Errorf("path traversal detected: %s", header.Name)
		}

		// Strip leading "./" if present.
		name = strings.TrimPrefix(name, "./")
		if name == "." || name == "" {
			continue
		}

		target := filepath.Join(destDir, name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", name, err)
			}

		case tar.TypeReg:
			if header.Size > maxSingleFile {
				return fmt.Errorf("file %s exceeds maximum size (%d bytes)", name, header.Size)
			}

			totalSize += header.Size
			if totalSize > maxArchiveSize {
				return fmt.Errorf("archive exceeds maximum total size (%d bytes)", maxArchiveSize)
			}

			// Ensure parent directory exists.
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent for %s: %w", name, err)
			}

			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)&0755|0644)
			if err != nil {
				return fmt.Errorf("creating %s: %w", name, err)
			}

			if _, err := io.Copy(out, io.LimitReader(tr, maxSingleFile+1)); err != nil {
				_ = out.Close()
				return fmt.Errorf("extracting %s: %w", name, err)
			}
			_ = out.Close()

		default:
			// Skip symlinks, devices, etc.
			continue
		}
	}

	return nil
}

// InstallPackageFromArchive downloads a package archive from a URL and
// extracts it to the destination directory. This is the primary installation
// method for codectx packages.
//
// The destDir will be created if it doesn't exist. If it already exists,
// it is removed first to ensure a clean install.
func InstallPackageFromArchive(ctx context.Context, client HTTPClient, archiveURL, destDir string) error {
	// Remove existing installation if present.
	if _, err := os.Stat(destDir); err == nil {
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("removing existing package at %s: %w", destDir, err)
		}
	}

	// Create a temp file for the download.
	tmpFile, err := os.CreateTemp("", "codectx-pkg-*.tar.gz")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	// Download.
	if err := DownloadArchive(ctx, client, archiveURL, tmpPath); err != nil {
		return err
	}

	// Create destination.
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination %s: %w", destDir, err)
	}

	// Extract.
	if err := ExtractPackageArchive(tmpPath, destDir); err != nil {
		// Clean up on extraction failure.
		_ = os.RemoveAll(destDir)
		return fmt.Errorf("extracting package: %w", err)
	}

	return nil
}

// AuthenticatedHTTPClient returns an *http.Client-compatible wrapper that
// adds a GitHub token as a Bearer authorization header to all requests.
// If token is empty, returns a plain http.Client.
func AuthenticatedHTTPClient(token string) HTTPClient {
	if token == "" {
		return http.DefaultClient
	}
	return &tokenTransport{
		token:   token,
		wrapped: http.DefaultTransport,
	}
}

// tokenTransport is an http.RoundTripper that adds a Bearer token to requests.
type tokenTransport struct {
	token   string
	wrapped http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.token)
	return t.wrapped.RoundTrip(req2)
}

// Do implements HTTPClient by using the token transport.
func (t *tokenTransport) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{Transport: t}
	return client.Do(req)
}
