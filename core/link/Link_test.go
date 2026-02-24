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

func TestLink_contentFormat(t *testing.T) {
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

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := "Read [README.md](" + filepath.Join(outputDir, "README.md") + ") before continuing.\n"
	assert.Equal(t, expected, string(content))
}

func TestLink_failsSubDirCreation(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".codectx")

	// Place a regular file where MkdirAll needs to create a directory.
	blocker := filepath.Join(dir, "blocked")
	require.NoError(t, os.WriteFile(blocker, []byte("not a dir"), 0o644))

	tools := []Tool{
		{Name: "Test", File: "instructions.md", SubDir: filepath.Join(blocker, "subdir")},
	}

	_, err := Link(tools, outputDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create directory")
}

func TestLink_failsBackupRename(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denial as root")
	}
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	// Create an existing file that should be backed up.
	existingPath := filepath.Join(dir, "TEST.md")
	require.NoError(t, os.WriteFile(existingPath, []byte("original"), 0o644))

	// Make the parent directory read-only so Rename (backup) fails.
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	tools := []Tool{
		{Name: "Test", File: existingPath},
	}

	_, err := Link(tools, outputDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup")
}

func TestTools_count(t *testing.T) {
	assert.Len(t, Tools, 5)

	names := make(map[string]bool)
	for _, tool := range Tools {
		names[tool.Name] = true
	}

	assert.True(t, names["Claude Code"], "should contain Claude Code")
	assert.True(t, names["Agents"], "should contain Agents")
	assert.True(t, names["Cursor"], "should contain Cursor")
	assert.True(t, names["Windsurf"], "should contain Windsurf")
	assert.True(t, names["GitHub Copilot"], "should contain GitHub Copilot")
}

func TestLink_failsWriteFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denial as root")
	}
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".codectx")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	// Create a directory that cannot be written to.
	readOnlyDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0o755))
	require.NoError(t, os.Chmod(readOnlyDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o755) })

	filePath := filepath.Join(readOnlyDir, "TEST.md")
	tools := []Tool{
		{Name: "Test", File: filePath},
	}

	_, err := Link(tools, outputDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
}

func TestTools_entries(t *testing.T) {
	expected := []struct {
		Name   string
		File   string
		SubDir string
	}{
		{Name: "Claude Code", File: "CLAUDE.md", SubDir: ""},
		{Name: "Agents", File: "AGENTS.md", SubDir: ""},
		{Name: "Cursor", File: ".cursorrules", SubDir: ""},
		{Name: "Windsurf", File: ".windsurfrules", SubDir: ""},
		{Name: "GitHub Copilot", File: "copilot-instructions.md", SubDir: ".github"},
	}

	require.Len(t, Tools, len(expected))
	for i, exp := range expected {
		assert.Equal(t, exp.Name, Tools[i].Name, "tool %d Name", i)
		assert.Equal(t, exp.File, Tools[i].File, "tool %d File", i)
		assert.Equal(t, exp.SubDir, Tools[i].SubDir, "tool %d SubDir", i)
	}
}
