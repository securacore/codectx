package link

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLink_createsEntryPoints(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	filePath := filepath.Join(dir, "TEST.md")
	tools := []Tool{
		{Name: "Test Tool", File: filePath},
	}

	results, err := Link(tools, outputDir)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, filePath, results[0].Path)
	assert.Empty(t, results[0].BackedUp)

	// Verify content references README.md.
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Read [README.md]")
}

func TestLink_withSubDir(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	subDir := filepath.Join(dir, ".github")
	tools := []Tool{
		{Name: "Copilot", File: "copilot-instructions.md", SubDir: subDir},
	}

	results, err := Link(tools, outputDir)
	require.NoError(t, err)
	require.Len(t, results, 1)

	expectedPath := filepath.Join(subDir, "copilot-instructions.md")
	assert.Equal(t, expectedPath, results[0].Path)

	// Verify subdirectory was created.
	info, err := os.Stat(subDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify content references README.md.
	content, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Read [README.md]")
	assert.Contains(t, string(content), "README.md")
}

func TestLink_backupsExistingFile(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	// Create an existing file.
	existingPath := filepath.Join(dir, "TEST.md")
	err := os.WriteFile(existingPath, []byte("original content"), 0o644)
	require.NoError(t, err)

	tools := []Tool{
		{Name: "Test Tool", File: existingPath},
	}

	results, err := Link(tools, outputDir)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Verify backup was created.
	assert.NotEmpty(t, results[0].BackedUp)
	assert.Contains(t, results[0].BackedUp, ".bak")

	// Verify backup content matches original.
	backupContent, err := os.ReadFile(results[0].BackedUp)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(backupContent))

	// Verify new file has the entry point content.
	newContent, err := os.ReadFile(existingPath)
	require.NoError(t, err)
	assert.Contains(t, string(newContent), "Read [README.md]")
}

func TestLink_multipleTools(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	file1 := filepath.Join(dir, "TOOL1.md")
	file2 := filepath.Join(dir, "TOOL2.md")
	tools := []Tool{
		{Name: "Tool 1", File: file1},
		{Name: "Tool 2", File: file2},
	}

	results, err := Link(tools, outputDir)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Both files should exist.
	_, err = os.Stat(file1)
	assert.NoError(t, err)
	_, err = os.Stat(file2)
	assert.NoError(t, err)
}

func TestLink_emptyTools(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".codectx")

	results, err := Link([]Tool{}, outputDir)
	require.NoError(t, err)
	assert.Empty(t, results)
}
