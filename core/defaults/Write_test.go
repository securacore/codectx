package defaults

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntries_count(t *testing.T) {
	entries := Entries()
	assert.Len(t, entries, len(docs), "Entries should return one entry per default doc")
}

func TestEntries_uniqueIDs(t *testing.T) {
	entries := Entries()
	seen := map[string]bool{}
	for _, e := range entries {
		assert.False(t, seen[e.ID], "duplicate entry ID: %s", e.ID)
		seen[e.ID] = true
	}
}

func TestEntries_loadValues(t *testing.T) {
	entries := Entries()
	byID := map[string]string{}
	for _, e := range entries {
		byID[e.ID] = e.Load
	}

	// Philosophy is load: always; everything else is load: documentation.
	assert.Equal(t, "always", byID["philosophy"])
	assert.Equal(t, "documentation", byID["documentation"])
	assert.Equal(t, "documentation", byID["markdown"])
	assert.Equal(t, "documentation", byID["specs"])
	assert.Equal(t, "documentation", byID["ai-authoring"])
}

func TestEntries_pathFormat(t *testing.T) {
	entries := Entries()
	for _, e := range entries {
		assert.True(t, len(e.Path) > 0, "entry %s should have a path", e.ID)
		assert.Contains(t, e.Path, "foundation/", "entry %s path should be under foundation/", e.ID)
		assert.Contains(t, e.Path, "README.md", "entry %s path should end with README.md", e.ID)
	}
}

func TestEntries_specPaths(t *testing.T) {
	entries := Entries()
	for _, e := range entries {
		assert.True(t, len(e.Spec) > 0, "entry %s should have a spec path", e.ID)
		assert.Contains(t, e.Spec, "spec/README.md", "entry %s spec should contain spec/README.md", e.ID)
	}
}

func TestEntries_descriptions(t *testing.T) {
	entries := Entries()
	for _, e := range entries {
		assert.NotEmpty(t, e.Description, "entry %s should have a description", e.ID)
	}
}

func TestWriteAll_createsAllFiles(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir)
	require.NoError(t, err)

	for _, doc := range docs {
		for _, file := range files {
			path := filepath.Join(dir, doc.Dir, file)
			info, err := os.Stat(path)
			require.NoError(t, err, "file %s/%s should exist", doc.Dir, file)
			assert.Greater(t, info.Size(), int64(0), "file %s/%s should not be empty", doc.Dir, file)
		}
	}
}

func TestWriteAll_contentMatchesEmbedded(t *testing.T) {
	dir := t.TempDir()

	err := WriteAll(dir)
	require.NoError(t, err)

	for _, doc := range docs {
		for _, file := range files {
			srcPath := filepath.Join("content", doc.Dir, file)
			embedded, err := content.ReadFile(srcPath)
			require.NoError(t, err)

			written, err := os.ReadFile(filepath.Join(dir, doc.Dir, file))
			require.NoError(t, err)

			assert.Equal(t, embedded, written, "written %s/%s should match embedded", doc.Dir, file)
		}
	}
}

func TestWriteAll_createsNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "foundation")

	err := WriteAll(nested)
	require.NoError(t, err)

	// Verify subdirectories were created.
	for _, doc := range docs {
		info, err := os.Stat(filepath.Join(nested, doc.Dir))
		require.NoError(t, err, "directory %s should exist", doc.Dir)
		assert.True(t, info.IsDir())

		info, err = os.Stat(filepath.Join(nested, doc.Dir, "spec"))
		require.NoError(t, err, "directory %s/spec should exist", doc.Dir)
		assert.True(t, info.IsDir())
	}
}

func TestWriteAll_idempotent(t *testing.T) {
	dir := t.TempDir()

	// First call.
	err := WriteAll(dir)
	require.NoError(t, err)

	// Capture contents after first write.
	firstWrite := map[string][]byte{}
	for _, doc := range docs {
		for _, file := range files {
			path := filepath.Join(dir, doc.Dir, file)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			firstWrite[doc.Dir+"/"+file] = data
		}
	}

	// Second call should succeed without error.
	err = WriteAll(dir)
	require.NoError(t, err)

	// Files should be unchanged (skipped, not overwritten).
	for _, doc := range docs {
		for _, file := range files {
			path := filepath.Join(dir, doc.Dir, file)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			key := doc.Dir + "/" + file
			assert.Equal(t, firstWrite[key], data, "%s should be unchanged after second write", key)
		}
	}
}

func TestWriteAll_doesNotOverwriteModifiedFiles(t *testing.T) {
	dir := t.TempDir()

	// First write.
	err := WriteAll(dir)
	require.NoError(t, err)

	// Modify a file.
	modifiedPath := filepath.Join(dir, "philosophy", "README.md")
	customContent := []byte("# My Custom Philosophy\nUser-owned content.\n")
	require.NoError(t, os.WriteFile(modifiedPath, customContent, 0o644))

	// Second write should skip existing files.
	err = WriteAll(dir)
	require.NoError(t, err)

	// Verify the modified file was NOT overwritten.
	data, err := os.ReadFile(modifiedPath)
	require.NoError(t, err)
	assert.Equal(t, customContent, data, "user-modified file should not be overwritten")
}

func TestWriteAll_failsOnReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denial as root")
	}
	// Use a path nested under a regular file, which makes MkdirAll fail.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocked")
	require.NoError(t, os.WriteFile(blocker, []byte("not a dir"), 0o644))

	err := WriteAll(filepath.Join(blocker, "foundation"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create directory")
}

func TestDocs_registryConsistency(t *testing.T) {
	// Verify that docs registry matches the embed directives: every doc
	// has readable embedded content for both README.md and spec/README.md.
	for _, doc := range docs {
		for _, file := range files {
			srcPath := filepath.Join("content", doc.Dir, file)
			data, err := content.ReadFile(srcPath)
			assert.NoError(t, err, "embedded file %s should be readable", srcPath)
			assert.Greater(t, len(data), 0, "embedded file %s should not be empty", srcPath)
		}
	}
}
