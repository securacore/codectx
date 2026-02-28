package packagetpl

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// embeddedFiles collects all file paths from the embedded FS (relative to
// content/). Directories are excluded.
func embeddedFiles(t *testing.T) []string {
	t.Helper()
	var paths []string
	err := fs.WalkDir(content, "content", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel("content", path)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, rel)
		return nil
	})
	require.NoError(t, err)
	return paths
}

// --- WriteAll: basic ---

func TestWriteAll_createsAllFiles(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "claude"})
	require.NoError(t, err)

	// Every embedded file should exist on disk.
	for _, rel := range embeddedFiles(t) {
		path := filepath.Join(dir, rel)
		info, err := os.Stat(path)
		require.NoError(t, err, "file %s should exist", rel)
		assert.Greater(t, info.Size(), int64(0), "file %s should not be empty", rel)
	}
}

func TestWriteAll_createsDirectories(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "opencode"})
	require.NoError(t, err)

	// Spot-check key directories exist.
	dirs := []string{
		"bin",
		"bin/just",
		"bin/just/root",
		"bin/just/ai",
		"bin/just/claude",
		"bin/just/opencode",
		".github",
		".github/workflows",
		"docs",
		"docs/prompts",
		"docs/prompts/save",
	}
	for _, d := range dirs {
		info, err := os.Stat(filepath.Join(dir, d))
		require.NoError(t, err, "directory %s should exist", d)
		assert.True(t, info.IsDir(), "%s should be a directory", d)
	}
}

// --- WriteAll: template substitution ---

func TestWriteAll_substitutesAIBin(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "claude"})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "bin", "just", "settings.just"))
	require.NoError(t, err)

	assert.Contains(t, string(data), `ai_bin := "claude"`)
	assert.NotContains(t, string(data), "{{AI_BIN}}")
}

func TestWriteAll_defaultAIBin(t *testing.T) {
	dir := t.TempDir()

	// Empty AIBin should default to "opencode".
	err := WriteAll(dir, Options{})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "bin", "just", "settings.just"))
	require.NoError(t, err)

	assert.Contains(t, string(data), `ai_bin := "opencode"`)
	assert.NotContains(t, string(data), "{{AI_BIN}}")
}

func TestWriteAll_noResidualPlaceholders(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "claude"})
	require.NoError(t, err)

	// Walk all written files and ensure no {{AI_BIN}} placeholders remain.
	for _, rel := range embeddedFiles(t) {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)
		assert.NotContains(t, string(data), "{{AI_BIN}}",
			"file %s should not contain residual placeholder", rel)
	}
}

// --- WriteAll: file permissions ---

func TestWriteAll_binReleaseIsExecutable(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "opencode"})
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(dir, "bin", "release"))
	require.NoError(t, err)
	// Check the executable bit is set (0o755).
	assert.True(t, info.Mode().Perm()&0o111 != 0,
		"bin/release should be executable, got %o", info.Mode().Perm())
}

func TestWriteAll_regularFilesNotExecutable(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "opencode"})
	require.NoError(t, err)

	// Check that non-bin files are 0o644 (not executable).
	regularFiles := []string{
		".justfile",
		"devbox.json",
		filepath.Join(".github", "workflows", "release.yml"),
		filepath.Join("bin", "just", "settings.just"),
	}
	for _, rel := range regularFiles {
		info, err := os.Stat(filepath.Join(dir, rel))
		require.NoError(t, err, "file %s should exist", rel)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm(),
			"file %s should have 0644 permissions", rel)
	}
}

// --- WriteAll: idempotent / skip existing ---

func TestWriteAll_skipsExistingFiles(t *testing.T) {
	dir := t.TempDir()

	// First write.
	err := WriteAll(dir, Options{AIBin: "claude"})
	require.NoError(t, err)

	// Modify a file to prove it won't be overwritten.
	justfile := filepath.Join(dir, ".justfile")
	customContent := []byte("# My custom justfile\n")
	require.NoError(t, os.WriteFile(justfile, customContent, 0o644))

	// Second write with a different AIBin.
	err = WriteAll(dir, Options{AIBin: "opencode"})
	require.NoError(t, err)

	// Custom file should be preserved.
	data, err := os.ReadFile(justfile)
	require.NoError(t, err)
	assert.Equal(t, customContent, data, ".justfile should not be overwritten")
}

func TestWriteAll_idempotent(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "claude"})
	require.NoError(t, err)

	// Capture file count.
	var count1 int
	_ = filepath.Walk(dir, func(_ string, _ os.FileInfo, _ error) error {
		count1++
		return nil
	})

	// Second call should succeed.
	err = WriteAll(dir, Options{AIBin: "claude"})
	require.NoError(t, err)

	// File count should be the same.
	var count2 int
	_ = filepath.Walk(dir, func(_ string, _ os.FileInfo, _ error) error {
		count2++
		return nil
	})
	assert.Equal(t, count1, count2, "file count should remain the same after second write")
}

// --- WriteAll: content integrity ---

func TestWriteAll_contentMatchesEmbedded(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "opencode"})
	require.NoError(t, err)

	for _, rel := range embeddedFiles(t) {
		srcPath := filepath.Join("content", rel)
		embedded, err := content.ReadFile(srcPath)
		require.NoError(t, err)

		written, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err)

		// Apply the same substitution to expected content.
		expected := strings.ReplaceAll(string(embedded), "{{AI_BIN}}", "opencode")
		assert.Equal(t, expected, string(written),
			"written %s should match embedded after substitution", rel)
	}
}

// --- WriteAll: nested path creation ---

func TestWriteAll_createsNestedPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "deep", "nested", "path")

	err := WriteAll(dir, Options{AIBin: "opencode"})
	require.NoError(t, err)

	// Verify at least one file was created.
	_, err = os.Stat(filepath.Join(dir, ".justfile"))
	assert.NoError(t, err)
}

// --- WriteAll: expected template files ---

func TestWriteAll_expectedTemplateFiles(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir, Options{AIBin: "opencode"})
	require.NoError(t, err)

	// Key files that must always be present.
	expected := []string{
		".justfile",
		"devbox.json",
		"bin/release",
		"bin/just/settings.just",
		"bin/just/root/.mod.just",
		"bin/just/ai/.mod.just",
		"bin/just/claude/.mod.just",
		"bin/just/opencode/.mod.just",
		".github/workflows/release.yml",
		"docs/prompts/save/README.md",
	}
	for _, f := range expected {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.NoError(t, err, "expected template file %s should exist", f)
	}
}

// --- Embed: content FS is populated ---

func TestEmbed_contentFSIsPopulated(t *testing.T) {
	files := embeddedFiles(t)
	assert.Greater(t, len(files), 10,
		"embedded content should have at least 10 files, got %d", len(files))
}

func TestEmbed_contentReadable(t *testing.T) {
	for _, rel := range embeddedFiles(t) {
		srcPath := filepath.Join("content", rel)
		data, err := content.ReadFile(srcPath)
		assert.NoError(t, err, "should be able to read %s", srcPath)
		assert.Greater(t, len(data), 0, "%s should not be empty", srcPath)
	}
}
