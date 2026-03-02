package packagetpl

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Options configures the package template writer.
type Options struct {
	// AIBin is the AI tool binary name (e.g., "claude" or "opencode").
	// Defaults to "opencode" if empty.
	AIBin string
}

// WriteAll writes all embedded package template files to the target directory.
// Template placeholders are replaced with values from opts:
//   - {{AI_BIN}} in settings.just is replaced with opts.AIBin
//
// Existing files are not overwritten.
func WriteAll(dir string, opts Options) error {
	if opts.AIBin == "" {
		opts.AIBin = "opencode"
	}

	return fs.WalkDir(content, "content", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip the "content/" prefix to get the relative path.
		rel, err := filepath.Rel("content", path)
		if err != nil {
			return err
		}

		dst := filepath.Join(dir, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}

		// Skip if the destination already exists.
		if _, err := os.Stat(dst); err == nil {
			return nil
		}

		data, err := content.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded template %s: %w", path, err)
		}

		// Apply template substitutions.
		text := string(data)
		text = strings.ReplaceAll(text, "{{AI_BIN}}", opts.AIBin)

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", dst, err)
		}

		// Determine file permissions: executable for bin/ scripts.
		perm := os.FileMode(0o644)
		if rel == filepath.Join("bin", "release") {
			perm = 0o755
		}

		if err := os.WriteFile(dst, []byte(text), perm); err != nil {
			return fmt.Errorf("write template %s: %w", dst, err)
		}

		return nil
	})
}
