package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxExtractSize is the maximum allowed size for extracted files (100MB).
const maxExtractSize = 100 * 1024 * 1024

// extractTarGz extracts a .tar.gz archive to the destination directory.
// Only regular files are extracted; symlinks and other special types are skipped.
// Path traversal attempts (../) are rejected.
func extractTarGz(archivePath, destDir string) error {
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

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		// Only extract regular files.
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Protect against path traversal.
		name := filepath.Clean(header.Name)
		if strings.Contains(name, "..") {
			return fmt.Errorf("path traversal detected: %s", header.Name)
		}

		target := filepath.Join(destDir, filepath.Base(name))

		if header.Size > maxExtractSize {
			return fmt.Errorf("file %s exceeds maximum size (%d bytes)", name, maxExtractSize)
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("creating %s: %w", target, err)
		}

		if _, err := io.Copy(out, io.LimitReader(tr, maxExtractSize)); err != nil {
			_ = out.Close()
			return fmt.Errorf("extracting %s: %w", name, err)
		}
		_ = out.Close()
	}

	return nil
}
