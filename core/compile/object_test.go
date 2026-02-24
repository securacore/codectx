package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ContentHash ---

func TestContentHash_deterministic(t *testing.T) {
	data := []byte("hello world")
	h1 := ContentHash(data)
	h2 := ContentHash(data)
	assert.Equal(t, h1, h2)
}

func TestContentHash_length(t *testing.T) {
	h := ContentHash([]byte("test content"))
	assert.Len(t, h, 16)
}

func TestContentHash_differentContent(t *testing.T) {
	h1 := ContentHash([]byte("version A"))
	h2 := ContentHash([]byte("version B"))
	assert.NotEqual(t, h1, h2)
}

func TestContentHash_empty(t *testing.T) {
	h := ContentHash([]byte{})
	assert.Len(t, h, 16)
	assert.NotEmpty(t, h)
}

// --- ObjectPath ---

func TestObjectPath(t *testing.T) {
	assert.Equal(t, "objects/a1b2c3d4e5f67890.md", ObjectPath("a1b2c3d4e5f67890"))
}

// --- ObjectStore.Store ---

func TestStore_writesFile(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	content := []byte("# Philosophy\nCore philosophy document.\n")
	hash, err := store.Store(content)
	require.NoError(t, err)
	assert.Len(t, hash, 16)

	// Verify file exists at expected path.
	path := filepath.Join(dir, "objects", hash+".md")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestStore_idempotent(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	content := []byte("same content")
	h1, err := store.Store(content)
	require.NoError(t, err)

	h2, err := store.Store(content)
	require.NoError(t, err)

	assert.Equal(t, h1, h2)

	// Only one file should exist.
	entries, err := os.ReadDir(filepath.Join(dir, "objects"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestStore_differentContent(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	h1, err := store.Store([]byte("content A"))
	require.NoError(t, err)
	h2, err := store.Store([]byte("content B"))
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2)

	entries, err := os.ReadDir(filepath.Join(dir, "objects"))
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestStore_createsDirectory(t *testing.T) {
	dir := t.TempDir()
	objDir := filepath.Join(dir, "nested", "objects")
	store := NewObjectStore(objDir)

	_, err := store.Store([]byte("content"))
	require.NoError(t, err)

	info, err := os.Stat(objDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// --- ObjectStore.Read ---

func TestRead_existingObject(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	content := []byte("readable content")
	hash, err := store.Store(content)
	require.NoError(t, err)

	data, err := store.Read(hash)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestRead_missingObject(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	_, err := store.Read("nonexistent1234")
	assert.Error(t, err)
}

// --- ObjectStore.List ---

func TestList_empty(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	hashes, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, hashes)
}

func TestList_nonexistentDir(t *testing.T) {
	store := NewObjectStore("/nonexistent/objects")
	hashes, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, hashes)
}

func TestList_withObjects(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	h1, err := store.Store([]byte("A"))
	require.NoError(t, err)
	h2, err := store.Store([]byte("B"))
	require.NoError(t, err)

	hashes, err := store.List()
	require.NoError(t, err)
	assert.Len(t, hashes, 2)
	assert.True(t, hashes[h1])
	assert.True(t, hashes[h2])
}

// --- ObjectStore.Prune ---

func TestPrune_removesOrphans(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	h1, err := store.Store([]byte("keep this"))
	require.NoError(t, err)
	_, err = store.Store([]byte("remove this"))
	require.NoError(t, err)

	active := map[string]bool{h1: true}
	removed, err := store.Prune(active)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	// Verify only the active object remains.
	hashes, err := store.List()
	require.NoError(t, err)
	assert.Len(t, hashes, 1)
	assert.True(t, hashes[h1])
}

func TestPrune_emptyActive(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	_, err := store.Store([]byte("A"))
	require.NoError(t, err)
	_, err = store.Store([]byte("B"))
	require.NoError(t, err)

	removed, err := store.Prune(map[string]bool{})
	require.NoError(t, err)
	assert.Equal(t, 2, removed)

	hashes, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, hashes)
}

func TestPrune_nonexistentDir(t *testing.T) {
	store := NewObjectStore("/nonexistent/objects")
	removed, err := store.Prune(map[string]bool{})
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
}

func TestList_skipsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	objDir := filepath.Join(dir, "objects")
	store := NewObjectStore(objDir)

	h1, err := store.Store([]byte("A"))
	require.NoError(t, err)

	// Create a subdirectory inside the objects dir.
	require.NoError(t, os.MkdirAll(filepath.Join(objDir, "subdir"), 0o755))

	hashes, err := store.List()
	require.NoError(t, err)
	// Only the file should appear, not the subdirectory.
	assert.Len(t, hashes, 1)
	assert.True(t, hashes[h1])
}

func TestPrune_skipsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	objDir := filepath.Join(dir, "objects")
	store := NewObjectStore(objDir)

	h1, err := store.Store([]byte("keep"))
	require.NoError(t, err)

	// Create a subdirectory inside the objects dir.
	subdir := filepath.Join(objDir, "nested")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	active := map[string]bool{h1: true}
	removed, err := store.Prune(active)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)

	// Subdirectory should still exist (not removed by Prune).
	_, err = os.Stat(subdir)
	assert.NoError(t, err)
}

func TestStore_failsMkdirAll(t *testing.T) {
	// Create a file where the objects directory needs to be — MkdirAll will fail.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "objects")
	require.NoError(t, os.WriteFile(blocker, []byte("not a dir"), 0o644))

	store := NewObjectStore(blocker)
	_, err := store.Store([]byte("should fail"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create objects directory")
}

func TestPrune_failsRemove(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denial as root")
	}
	dir := t.TempDir()
	objDir := filepath.Join(dir, "objects")
	store := NewObjectStore(objDir)

	// Store an object, then make the directory read-only so Remove fails.
	hash, err := store.Store([]byte("orphan"))
	require.NoError(t, err)

	require.NoError(t, os.Chmod(objDir, 0o555))
	t.Cleanup(func() { os.Chmod(objDir, 0o755) })

	_, err = store.Prune(map[string]bool{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove orphan")
	_ = hash
}

func TestPrune_allActive(t *testing.T) {
	dir := t.TempDir()
	store := NewObjectStore(filepath.Join(dir, "objects"))

	h1, err := store.Store([]byte("A"))
	require.NoError(t, err)
	h2, err := store.Store([]byte("B"))
	require.NoError(t, err)

	active := map[string]bool{h1: true, h2: true}
	removed, err := store.Prune(active)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)

	hashes, err := store.List()
	require.NoError(t, err)
	assert.Len(t, hashes, 2)
}
