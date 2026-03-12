package link_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/link"
	"github.com/securacore/codectx/core/project"
)

// ---------------------------------------------------------------------------
// AllIntegrations
// ---------------------------------------------------------------------------

func TestAllIntegrations_ReturnsFour(t *testing.T) {
	all := link.AllIntegrations()
	if len(all) != 4 {
		t.Errorf("expected 4 integrations, got %d", len(all))
	}
}

func TestAllIntegrations_UniqueTypes(t *testing.T) {
	all := link.AllIntegrations()
	seen := make(map[link.Integration]bool)
	for _, info := range all {
		if seen[info.Type] {
			t.Errorf("duplicate integration type: %v", info.Type)
		}
		seen[info.Type] = true
	}
}

func TestInfoByType(t *testing.T) {
	info := link.InfoByType(link.Claude)
	if info.Name != "Claude Code" {
		t.Errorf("expected name %q, got %q", "Claude Code", info.Name)
	}
	if info.FilePath != "CLAUDE.md" {
		t.Errorf("expected path %q, got %q", "CLAUDE.md", info.FilePath)
	}
}

// ---------------------------------------------------------------------------
// Detect
// ---------------------------------------------------------------------------

func TestDetect_ClaudeDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), project.DirPerm); err != nil {
		t.Fatal(err)
	}

	detected := link.Detect(dir)
	if !containsIntegration(detected, link.Claude) {
		t.Error("expected Claude detected via .claude/ directory")
	}
}

func TestDetect_ClaudeFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("existing"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	detected := link.Detect(dir)
	if !containsIntegration(detected, link.Claude) {
		t.Error("expected Claude detected via existing CLAUDE.md")
	}
}

func TestDetect_CursorDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), project.DirPerm); err != nil {
		t.Fatal(err)
	}

	detected := link.Detect(dir)
	if !containsIntegration(detected, link.Cursor) {
		t.Error("expected Cursor detected via .cursor/ directory")
	}
}

func TestDetect_CopilotGithubDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github"), project.DirPerm); err != nil {
		t.Fatal(err)
	}

	detected := link.Detect(dir)
	if !containsIntegration(detected, link.Copilot) {
		t.Error("expected Copilot detected via .github/ directory")
	}
}

func TestDetect_AgentsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("existing"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	detected := link.Detect(dir)
	if !containsIntegration(detected, link.Agents) {
		t.Error("expected Agents detected via existing AGENTS.md")
	}
}

func TestDetect_NothingPresent(t *testing.T) {
	dir := t.TempDir()

	detected := link.Detect(dir)
	if len(detected) != 0 {
		t.Errorf("expected 0 detected, got %d", len(detected))
	}
}

func TestDetect_MultiplePresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), project.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".github"), project.DirPerm); err != nil {
		t.Fatal(err)
	}

	detected := link.Detect(dir)
	if !containsIntegration(detected, link.Claude) {
		t.Error("expected Claude detected")
	}
	if !containsIntegration(detected, link.Copilot) {
		t.Error("expected Copilot detected")
	}
}

// ---------------------------------------------------------------------------
// Write
// ---------------------------------------------------------------------------

func TestWrite_SingleIntegration(t *testing.T) {
	dir := t.TempDir()
	contextPath := "docs/.codectx/compiled/context.md"

	results, err := link.Write(dir, contextPath, []link.Integration{link.Claude})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Integration != link.Claude {
		t.Errorf("expected Claude integration, got %v", r.Integration)
	}
	if r.BackedUp {
		t.Error("expected no backup for new file")
	}

	// Verify file content.
	data, readErr := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if readErr != nil {
		t.Fatalf("reading CLAUDE.md: %v", readErr)
	}

	content := string(data)
	if !strings.Contains(content, "Project Instructions") {
		t.Error("expected Project Instructions heading")
	}
	if !strings.Contains(content, contextPath) {
		t.Error("expected context path in content")
	}
	if !strings.Contains(content, "codectx query") {
		t.Error("expected query command reference")
	}
}

func TestWrite_AllIntegrations(t *testing.T) {
	dir := t.TempDir()
	contextPath := "docs/.codectx/compiled/context.md"

	all := []link.Integration{link.Claude, link.Agents, link.Cursor, link.Copilot}
	results, err := link.Write(dir, contextPath, all)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// Verify all files exist.
	for _, info := range link.AllIntegrations() {
		path := filepath.Join(dir, info.FilePath)
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("expected %s to exist: %v", info.FilePath, statErr)
		}
	}
}

func TestWrite_BackupsExistingFile(t *testing.T) {
	dir := t.TempDir()
	contextPath := "docs/.codectx/compiled/context.md"

	// Create an existing CLAUDE.md.
	existingContent := "# My existing instructions\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existingContent), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	results, err := link.Write(dir, contextPath, []link.Integration{link.Claude})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	r := results[0]
	if !r.BackedUp {
		t.Error("expected existing file to be backed up")
	}
	if r.BackupPath == "" {
		t.Error("expected non-empty backup path")
	}
	if !strings.HasSuffix(r.BackupPath, ".bak") {
		t.Errorf("expected .bak suffix, got %q", r.BackupPath)
	}

	// Verify backup contains original content.
	backupData, backupErr := os.ReadFile(filepath.Join(dir, r.BackupPath))
	if backupErr != nil {
		t.Fatalf("reading backup: %v", backupErr)
	}
	if string(backupData) != existingContent {
		t.Errorf("expected backup to contain original content, got %q", string(backupData))
	}

	// Verify new content replaced the file.
	newData, newErr := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if newErr != nil {
		t.Fatalf("reading new CLAUDE.md: %v", newErr)
	}
	if !strings.Contains(string(newData), "Project Instructions") {
		t.Error("expected new content in CLAUDE.md")
	}
}

func TestWrite_CreatesGithubDirectory(t *testing.T) {
	dir := t.TempDir()
	contextPath := "docs/.codectx/compiled/context.md"

	// .github/ should not exist yet.
	results, err := link.Write(dir, contextPath, []link.Integration{link.Copilot})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// .github/copilot-instructions.md should exist.
	path := filepath.Join(dir, ".github", "copilot-instructions.md")
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("expected copilot-instructions.md to exist: %v", statErr)
	}
}

func TestWrite_EmptyList(t *testing.T) {
	dir := t.TempDir()

	results, err := link.Write(dir, "context.md", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// NeedsUpdate
// ---------------------------------------------------------------------------

func TestNeedsUpdate_StaleContextPath(t *testing.T) {
	dir := t.TempDir()

	// Write a codectx file with a completely different context path.
	content := "# Project Instructions\n\n- Context file: documentation/.state/compiled/context.md\n\ncodectx\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(content), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	needsUpdate := link.NeedsUpdate(dir, "docs/.codectx/compiled/context.md")
	if !containsIntegration(needsUpdate, link.Claude) {
		t.Error("expected CLAUDE.md to need update with changed context path")
	}
}

func TestNeedsUpdate_CurrentPath(t *testing.T) {
	dir := t.TempDir()
	contextPath := "docs/.codectx/compiled/context.md"

	// Write with current path.
	results, err := link.Write(dir, contextPath, []link.Integration{link.Claude})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = results

	needsUpdate := link.NeedsUpdate(dir, contextPath)
	if containsIntegration(needsUpdate, link.Claude) {
		t.Error("expected CLAUDE.md to not need update with current context path")
	}
}

func TestNeedsUpdate_NonCodectxFile(t *testing.T) {
	dir := t.TempDir()

	// Write a non-codectx CLAUDE.md.
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Custom instructions\n"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	needsUpdate := link.NeedsUpdate(dir, "docs/.codectx/compiled/context.md")
	if containsIntegration(needsUpdate, link.Claude) {
		t.Error("should not flag non-codectx files for update")
	}
}

func TestNeedsUpdate_NoFiles(t *testing.T) {
	dir := t.TempDir()

	needsUpdate := link.NeedsUpdate(dir, "docs/.codectx/compiled/context.md")
	if len(needsUpdate) != 0 {
		t.Errorf("expected 0 updates needed, got %d", len(needsUpdate))
	}
}

// ---------------------------------------------------------------------------
// NeedsUpdate — nonexistent entry point file
// ---------------------------------------------------------------------------

func TestNeedsUpdate_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	// No entry point files exist. NeedsUpdate should return empty.
	needsUpdate := link.NeedsUpdate(filepath.Join(dir, "nonexistent"), "docs/.codectx/compiled/context.md")
	if len(needsUpdate) != 0 {
		t.Errorf("expected 0 updates needed for nonexistent dir, got %d", len(needsUpdate))
	}
}

// ---------------------------------------------------------------------------
// RenderLinkResults
// ---------------------------------------------------------------------------

func TestRenderLinkResults_Empty(t *testing.T) {
	got := link.RenderLinkResults(nil)
	if got != "" {
		t.Errorf("expected empty string for nil results, got %q", got)
	}
}

func TestRenderLinkResults_SingleResult(t *testing.T) {
	results := []link.WriteResult{
		{Integration: link.Claude, Name: "Claude Code", FilePath: "CLAUDE.md"},
	}
	got := link.RenderLinkResults(results)

	if !strings.Contains(got, "Entry points linked") {
		t.Error("expected success header")
	}
	if !strings.Contains(got, "CLAUDE.md") {
		t.Error("expected file path")
	}
}

func TestRenderLinkResults_WithBackup(t *testing.T) {
	results := []link.WriteResult{
		{
			Integration: link.Claude,
			Name:        "Claude Code",
			FilePath:    "CLAUDE.md",
			BackedUp:    true,
			BackupPath:  "CLAUDE.md.1234567890.bak",
		},
	}
	got := link.RenderLinkResults(results)

	if !strings.Contains(got, "CLAUDE.md") {
		t.Error("expected file path")
	}
	if !strings.Contains(got, "backed up") {
		t.Error("expected backup indication")
	}
	if !strings.Contains(got, ".bak") {
		t.Error("expected backup path")
	}
}

func TestRenderLinkResults_MultipleResults(t *testing.T) {
	results := []link.WriteResult{
		{Integration: link.Claude, Name: "Claude Code", FilePath: "CLAUDE.md"},
		{Integration: link.Agents, Name: "Agents", FilePath: "AGENTS.md"},
		{Integration: link.Cursor, Name: "Cursor", FilePath: ".cursorrules"},
	}
	got := link.RenderLinkResults(results)

	if !strings.Contains(got, "CLAUDE.md") {
		t.Error("expected CLAUDE.md")
	}
	if !strings.Contains(got, "AGENTS.md") {
		t.Error("expected AGENTS.md")
	}
	if !strings.Contains(got, ".cursorrules") {
		t.Error("expected .cursorrules")
	}
}

// ---------------------------------------------------------------------------
// InfoByType — unknown type
// ---------------------------------------------------------------------------

func TestInfoByType_UnknownType(t *testing.T) {
	info := link.InfoByType(link.Integration(999))
	if info.Name != "" {
		t.Errorf("expected empty Name for unknown type, got %q", info.Name)
	}
	if info.FilePath != "" {
		t.Errorf("expected empty FilePath for unknown type, got %q", info.FilePath)
	}
}

// ---------------------------------------------------------------------------
// Write — unknown integration (skipped)
// ---------------------------------------------------------------------------

func TestWrite_UnknownIntegration_Skipped(t *testing.T) {
	dir := t.TempDir()

	// An unknown integration type should be silently skipped.
	results, err := link.Write(dir, "context.md", []link.Integration{link.Integration(999)})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for unknown integration, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Write — error: non-writable parent directory
// ---------------------------------------------------------------------------

func TestWrite_NonWritableParentDir(t *testing.T) {
	dir := t.TempDir()

	// Create a read-only directory to force MkdirAll failure for Copilot
	// (.github/copilot-instructions.md needs .github/ created).
	githubDir := filepath.Join(dir, ".github")
	// Create .github as a file (not a directory) to cause MkdirAll to fail.
	if err := os.WriteFile(githubDir, []byte("not a directory"), project.FilePerm); err != nil {
		t.Fatal(err)
	}

	_, err := link.Write(dir, "context.md", []link.Integration{link.Copilot})
	if err == nil {
		t.Error("expected error when parent directory creation fails")
	}
}

// ---------------------------------------------------------------------------
// Write — error: write to non-writable location
// ---------------------------------------------------------------------------

func TestWrite_WriteFileError(t *testing.T) {
	dir := t.TempDir()

	// Create CLAUDE.md as a directory to cause WriteFile to fail.
	claudeDir := filepath.Join(dir, "CLAUDE.md")
	if err := os.MkdirAll(claudeDir, project.DirPerm); err != nil {
		t.Fatal(err)
	}

	_, err := link.Write(dir, "context.md", []link.Integration{link.Claude})
	if err == nil {
		t.Error("expected error when file write fails")
	}
}

// ---------------------------------------------------------------------------
// Write — backup error: unreadable existing file
// ---------------------------------------------------------------------------

func TestWrite_BackupError_UnreadableFile(t *testing.T) {
	dir := t.TempDir()

	// Create an existing CLAUDE.md that is not readable.
	claudePath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("existing"), project.FilePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(claudePath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(claudePath, project.FilePerm) })

	_, err := link.Write(dir, "context.md", []link.Integration{link.Claude})
	if err == nil {
		t.Error("expected error when backup cannot read existing file")
	}
}

// ---------------------------------------------------------------------------

func containsIntegration(list []link.Integration, target link.Integration) bool {
	for _, i := range list {
		if i == target {
			return true
		}
	}
	return false
}
