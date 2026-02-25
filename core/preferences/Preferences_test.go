package preferences

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_missingFile(t *testing.T) {
	dir := t.TempDir()
	p, err := Load(dir)
	require.NoError(t, err)
	assert.Nil(t, p.AutoCompile)
}

func TestLoad_emptyFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "preferences.yml"), []byte("{}"), 0o644))

	p, err := Load(dir)
	require.NoError(t, err)
	assert.Nil(t, p.AutoCompile)
}

func TestLoad_autoCompileTrue(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "preferences.yml"),
		[]byte("auto_compile: true\n"),
		0o644,
	))

	p, err := Load(dir)
	require.NoError(t, err)
	require.NotNil(t, p.AutoCompile)
	assert.True(t, *p.AutoCompile)
}

func TestLoad_autoCompileFalse(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "preferences.yml"),
		[]byte("auto_compile: false\n"),
		0o644,
	))

	p, err := Load(dir)
	require.NoError(t, err)
	require.NotNil(t, p.AutoCompile)
	assert.False(t, *p.AutoCompile)
}

func TestLoad_corruptYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "preferences.yml"),
		[]byte("{{invalid"),
		0o644,
	))

	_, err := Load(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse preferences")
}

func TestWriteAndLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	original := &Preferences{AutoCompile: BoolPtr(true)}

	err := Write(dir, original)
	require.NoError(t, err)

	loaded, err := Load(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded.AutoCompile)
	assert.True(t, *loaded.AutoCompile)
}

func TestWrite_createsDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", ".codectx")
	p := &Preferences{AutoCompile: BoolPtr(false)}

	err := Write(dir, p)
	require.NoError(t, err)

	loaded, err := Load(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded.AutoCompile)
	assert.False(t, *loaded.AutoCompile)
}

func TestWrite_nilPreference(t *testing.T) {
	dir := t.TempDir()
	p := &Preferences{} // AutoCompile is nil

	err := Write(dir, p)
	require.NoError(t, err)

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Nil(t, loaded.AutoCompile)
}

func TestLoad_readError(t *testing.T) {
	dir := t.TempDir()
	prefsPath := filepath.Join(dir, "preferences.yml")

	// Create the file, then make it unreadable.
	require.NoError(t, os.WriteFile(prefsPath, []byte("auto_compile: true\n"), 0o644))
	require.NoError(t, os.Chmod(prefsPath, 0o000))
	t.Cleanup(func() { _ = os.Chmod(prefsPath, 0o644) })

	_, err := Load(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read preferences")
}

func TestWrite_mkdirAllError(t *testing.T) {
	// Create a regular file where the output directory should be,
	// so MkdirAll fails.
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "blocked")
	require.NoError(t, os.WriteFile(outputPath, []byte("not a dir"), 0o644))

	p := &Preferences{AutoCompile: BoolPtr(true)}
	err := Write(outputPath, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create output directory")
}

func TestWrite_writeFileError(t *testing.T) {
	// Create a read-only directory so WriteFile fails.
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	require.NoError(t, os.Chmod(outputDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(outputDir, 0o755) })

	p := &Preferences{AutoCompile: BoolPtr(true)}
	err := Write(outputDir, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write preferences")
}

func TestBoolPtr(t *testing.T) {
	truePtr := BoolPtr(true)
	falsePtr := BoolPtr(false)

	require.NotNil(t, truePtr)
	assert.True(t, *truePtr)
	require.NotNil(t, falsePtr)
	assert.False(t, *falsePtr)
}
