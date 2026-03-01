package link

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	corelink "github.com/securacore/codectx/core/link"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "link", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}

func TestRun_missingConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// No codectx.yml exists.
	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_invalidConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Write invalid YAML.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, shared.ConfigFile),
		[]byte("{{{{not valid"), 0o644))

	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_missingCompiledOutput(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Write valid codectx.yml but no compiled output.
	cfg := &config.Config{
		Name:     "test-project",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, shared.ConfigFile), cfg))

	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compiled output not found")
	assert.Contains(t, err.Error(), "codectx compile")
}

func TestRun_missingCompiledOutputCustomDir(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	customOutput := filepath.Join(dir, "custom-output")
	cfg := &config.Config{
		Name: "test-project",
		Config: &config.BuildConfig{
			OutputDir: customOutput,
		},
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, shared.ConfigFile), cfg))

	err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compiled output not found")
	assert.Contains(t, err.Error(), customOutput)
}

// --- selectTools ---

func TestSelectTools_allIndices(t *testing.T) {
	tools := []corelink.Tool{
		{Name: "Claude", File: "CLAUDE.md"},
		{Name: "Cursor", File: ".cursor/rules/codectx.mdc", SubDir: ".cursor/rules"},
		{Name: "Windsurf", File: ".windsurfrules"},
	}

	selected := selectTools(tools, []int{0, 1, 2})
	require.Len(t, selected, 3)
	assert.Equal(t, "Claude", selected[0].Name)
	assert.Equal(t, "Cursor", selected[1].Name)
	assert.Equal(t, "Windsurf", selected[2].Name)
}

func TestSelectTools_subset(t *testing.T) {
	tools := []corelink.Tool{
		{Name: "Claude", File: "CLAUDE.md"},
		{Name: "Cursor", File: ".cursor/rules/codectx.mdc", SubDir: ".cursor/rules"},
		{Name: "Windsurf", File: ".windsurfrules"},
	}

	selected := selectTools(tools, []int{0, 2})
	require.Len(t, selected, 2)
	assert.Equal(t, "Claude", selected[0].Name)
	assert.Equal(t, "Windsurf", selected[1].Name)
}

func TestSelectTools_empty(t *testing.T) {
	tools := []corelink.Tool{
		{Name: "Claude", File: "CLAUDE.md"},
	}
	selected := selectTools(tools, nil)
	assert.Empty(t, selected)
}

// --- detectExistingFiles ---

func TestDetectExistingFiles_noCollisions(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	tools := []corelink.Tool{
		{Name: "Claude", File: "CLAUDE.md"},
		{Name: "Windsurf", File: ".windsurfrules"},
	}

	collisions := detectExistingFiles(tools)
	assert.Empty(t, collisions)
}

func TestDetectExistingFiles_withCollisions(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Create files that will collide.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("existing"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".windsurfrules"), []byte("existing"), 0o644))

	tools := []corelink.Tool{
		{Name: "Claude", File: "CLAUDE.md"},
		{Name: "Cursor", File: "codectx.mdc", SubDir: ".cursor/rules"},
		{Name: "Windsurf", File: ".windsurfrules"},
	}

	collisions := detectExistingFiles(tools)
	require.Len(t, collisions, 2)
	assert.Equal(t, "CLAUDE.md", collisions[0])
	assert.Equal(t, ".windsurfrules", collisions[1])
}

func TestDetectExistingFiles_withSubDir(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	// Create the subdirectory and file.
	subDir := filepath.Join(dir, ".cursor", "rules")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "codectx.mdc"), []byte("existing"), 0o644))

	tools := []corelink.Tool{
		{Name: "Cursor", File: "codectx.mdc", SubDir: ".cursor/rules"},
	}

	collisions := detectExistingFiles(tools)
	require.Len(t, collisions, 1)
	assert.Equal(t, filepath.Join(".cursor/rules", "codectx.mdc"), collisions[0])
}

// --- printLinkResults ---

// captureStdout runs fn and returns whatever it writes to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestPrintLinkResults_basic(t *testing.T) {
	results := []corelink.LinkResult{
		{Tool: corelink.Tool{Name: "Claude"}, Path: "CLAUDE.md"},
		{Tool: corelink.Tool{Name: "Windsurf"}, Path: ".windsurfrules"},
	}

	out := captureStdout(t, func() { printLinkResults(results) })
	assert.Contains(t, out, "Linked")
	assert.Contains(t, out, "CLAUDE.md")
	assert.Contains(t, out, ".windsurfrules")
}

func TestPrintLinkResults_withBackup(t *testing.T) {
	results := []corelink.LinkResult{
		{Tool: corelink.Tool{Name: "Claude"}, Path: "CLAUDE.md", BackedUp: "CLAUDE.md.bak-20250101-120000"},
	}

	out := captureStdout(t, func() { printLinkResults(results) })
	assert.Contains(t, out, "Linked")
	assert.Contains(t, out, "CLAUDE.md")
	assert.Contains(t, out, "backed up to")
	assert.Contains(t, out, "CLAUDE.md.bak-20250101-120000")
}

func TestPrintLinkResults_empty(t *testing.T) {
	out := captureStdout(t, func() { printLinkResults(nil) })
	assert.Contains(t, out, "Linked")
}
