package gitkeep

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Write ---

func TestWrite_createsGitkeep(t *testing.T) {
	dir := t.TempDir()

	err := Write(dir)
	require.NoError(t, err)

	path := filepath.Join(dir, ".gitkeep")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size(), ".gitkeep should be empty")
}

func TestWrite_idempotent(t *testing.T) {
	dir := t.TempDir()

	// First write.
	err := Write(dir)
	require.NoError(t, err)

	// Second write should succeed without error (file already exists).
	err = Write(dir)
	require.NoError(t, err)

	// File should still exist and be empty.
	info, err := os.Stat(filepath.Join(dir, ".gitkeep"))
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size())
}

func TestWrite_doesNotOverwriteExisting(t *testing.T) {
	dir := t.TempDir()

	// Pre-create .gitkeep with some content (unusual but defensive).
	path := filepath.Join(dir, ".gitkeep")
	require.NoError(t, os.WriteFile(path, []byte("custom"), 0o644))

	err := Write(dir)
	require.NoError(t, err)

	// Content should be preserved (not overwritten).
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "custom", string(data))
}

func TestWrite_failsIfDirDoesNotExist(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")

	err := Write(dir)
	require.Error(t, err)
}

// --- Clean ---

func TestClean_removesGitkeepWhenSiblingsExist(t *testing.T) {
	docsDir := t.TempDir()

	// Create a subdirectory with .gitkeep and a real file.
	sub := filepath.Join(docsDir, "topics")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, ".gitkeep"), nil, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(sub, "react"), 0o755))

	err := Clean(docsDir)
	require.NoError(t, err)

	// .gitkeep should be removed because "react" directory exists.
	_, err = os.Stat(filepath.Join(sub, ".gitkeep"))
	assert.True(t, os.IsNotExist(err), ".gitkeep should be removed")
}

func TestClean_preservesGitkeepWhenAlone(t *testing.T) {
	docsDir := t.TempDir()

	// Create a subdirectory with only .gitkeep.
	sub := filepath.Join(docsDir, "topics")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, ".gitkeep"), nil, 0o644))

	err := Clean(docsDir)
	require.NoError(t, err)

	// .gitkeep should still exist (no siblings).
	_, err = os.Stat(filepath.Join(sub, ".gitkeep"))
	assert.NoError(t, err, ".gitkeep should be preserved when alone")
}

func TestClean_handlesNoGitkeep(t *testing.T) {
	docsDir := t.TempDir()

	// Create a subdirectory without .gitkeep.
	sub := filepath.Join(docsDir, "topics")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(sub, "react"), 0o755))

	// Should not error.
	err := Clean(docsDir)
	require.NoError(t, err)
}

func TestClean_handlesNonexistentDocsDir(t *testing.T) {
	// Passing a non-existent directory should not error.
	err := Clean(filepath.Join(t.TempDir(), "nonexistent"))
	require.NoError(t, err)
}

func TestClean_handlesEmptyDocsDir(t *testing.T) {
	docsDir := t.TempDir()

	// Empty docs dir — nothing to clean.
	err := Clean(docsDir)
	require.NoError(t, err)
}

func TestClean_multipleSubdirectories(t *testing.T) {
	docsDir := t.TempDir()

	// topics: has .gitkeep + real content → remove .gitkeep
	topics := filepath.Join(docsDir, "topics")
	require.NoError(t, os.MkdirAll(topics, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(topics, ".gitkeep"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(topics, "react.md"), []byte("# React"), 0o644))

	// prompts: has .gitkeep only → keep .gitkeep
	prompts := filepath.Join(docsDir, "prompts")
	require.NoError(t, os.MkdirAll(prompts, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(prompts, ".gitkeep"), nil, 0o644))

	// plans: has .gitkeep + subdirectory → remove .gitkeep
	plans := filepath.Join(docsDir, "plans")
	require.NoError(t, os.MkdirAll(plans, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(plans, ".gitkeep"), nil, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(plans, "migrate"), 0o755))

	err := Clean(docsDir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(topics, ".gitkeep"))
	assert.True(t, os.IsNotExist(err), "topics/.gitkeep should be removed")

	_, err = os.Stat(filepath.Join(prompts, ".gitkeep"))
	assert.NoError(t, err, "prompts/.gitkeep should be preserved")

	_, err = os.Stat(filepath.Join(plans, ".gitkeep"))
	assert.True(t, os.IsNotExist(err), "plans/.gitkeep should be removed")
}

func TestClean_alsoProcessesPackageDir(t *testing.T) {
	// Create a structure: parent/docs/ and parent/package/.
	parent := t.TempDir()
	docsDir := filepath.Join(parent, "docs")
	pkgDir := filepath.Join(parent, "package")

	// docs/topics has .gitkeep + content.
	topics := filepath.Join(docsDir, "topics")
	require.NoError(t, os.MkdirAll(topics, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(topics, ".gitkeep"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(topics, "react.md"), []byte("# React"), 0o644))

	// package/topics has .gitkeep + content.
	pkgTopics := filepath.Join(pkgDir, "topics")
	require.NoError(t, os.MkdirAll(pkgTopics, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgTopics, ".gitkeep"), nil, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgTopics, "api"), 0o755))

	// package/prompts has .gitkeep only.
	pkgPrompts := filepath.Join(pkgDir, "prompts")
	require.NoError(t, os.MkdirAll(pkgPrompts, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgPrompts, ".gitkeep"), nil, 0o644))

	err := Clean(docsDir)
	require.NoError(t, err)

	// docs/topics .gitkeep removed.
	_, err = os.Stat(filepath.Join(topics, ".gitkeep"))
	assert.True(t, os.IsNotExist(err), "docs/topics/.gitkeep should be removed")

	// package/topics .gitkeep removed.
	_, err = os.Stat(filepath.Join(pkgTopics, ".gitkeep"))
	assert.True(t, os.IsNotExist(err), "package/topics/.gitkeep should be removed")

	// package/prompts .gitkeep preserved.
	_, err = os.Stat(filepath.Join(pkgPrompts, ".gitkeep"))
	assert.NoError(t, err, "package/prompts/.gitkeep should be preserved")
}

func TestClean_skipsPackageDirIfMissing(t *testing.T) {
	parent := t.TempDir()
	docsDir := filepath.Join(parent, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	// No package/ directory — should not error.
	err := Clean(docsDir)
	require.NoError(t, err)
}

func TestClean_skipsNonDirectoryEntries(t *testing.T) {
	docsDir := t.TempDir()

	// Place a regular file directly in docsDir (not a subdirectory).
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "manifest.yml"), []byte("name: test"), 0o644))

	// Create a subdirectory with .gitkeep only.
	sub := filepath.Join(docsDir, "topics")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, ".gitkeep"), nil, 0o644))

	// Should handle files in docsDir gracefully (skip them).
	err := Clean(docsDir)
	require.NoError(t, err)

	// .gitkeep should still be there (no siblings except itself).
	_, err = os.Stat(filepath.Join(sub, ".gitkeep"))
	assert.NoError(t, err)
}
