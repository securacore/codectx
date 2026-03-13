package usage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/history"
	"github.com/securacore/codectx/core/project"
)

func TestInitMetrics(t *testing.T) {
	m := initMetrics()
	if m.TokensByCaller == nil {
		t.Error("TokensByCaller should be initialized")
	}
	if m.TokensByModel == nil {
		t.Error("TokensByModel should be initialized")
	}
	if m.FirstSeen == 0 {
		t.Error("FirstSeen should be set")
	}
}

func TestInitLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.yml")

	if err := InitLocalFile(path); err != nil {
		t.Fatalf("InitLocalFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "total_tokens: 0") {
		t.Error("expected total_tokens: 0")
	}
	if !strings.Contains(content, "local machine") {
		t.Error("expected local machine header comment")
	}
}

func TestInitGlobalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "global_usage.yml")

	if err := InitGlobalFile(path, "my-project"); err != nil {
		t.Fatalf("InitGlobalFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "project: my-project") {
		t.Error("expected project name in global file")
	}
	if !strings.Contains(content, "project lifetime") {
		t.Error("expected project lifetime header comment")
	}
}

func TestInitFile_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.yml")

	// Create initial file.
	if err := InitLocalFile(path); err != nil {
		t.Fatal(err)
	}

	// Modify the file to verify it's not overwritten.
	m := readOrInit(path)
	m.QueryInvocations = 42
	if err := writeMetrics(path, m, localHeader); err != nil {
		t.Fatal(err)
	}

	// Second init should be a no-op.
	if err := InitLocalFile(path); err != nil {
		t.Fatalf("second InitLocalFile: %v", err)
	}

	// Verify the modification survived.
	m2 := readOrInit(path)
	if m2.QueryInvocations != 42 {
		t.Errorf("expected QueryInvocations 42, got %d", m2.QueryInvocations)
	}
}

func TestUpdateQuery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.yml")

	if err := UpdateQuery(path); err != nil {
		t.Fatalf("UpdateQuery: %v", err)
	}

	m := readOrInit(path)
	if m.QueryInvocations != 1 {
		t.Errorf("QueryInvocations = %d, want 1", m.QueryInvocations)
	}
	if m.LastUpdated == 0 {
		t.Error("LastUpdated should be set")
	}

	// Second call should increment.
	if err := UpdateQuery(path); err != nil {
		t.Fatalf("second UpdateQuery: %v", err)
	}
	m2 := readOrInit(path)
	if m2.QueryInvocations != 2 {
		t.Errorf("QueryInvocations = %d, want 2", m2.QueryInvocations)
	}
}

func TestUpdateGenerate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.yml")

	caller := history.CallerContext{
		Caller: "claude",
		Model:  "claude-sonnet",
	}

	if err := UpdateGenerate(path, 1500, false, caller); err != nil {
		t.Fatalf("UpdateGenerate: %v", err)
	}

	m := readOrInit(path)
	if m.GenerateInvocations != 1 {
		t.Errorf("GenerateInvocations = %d, want 1", m.GenerateInvocations)
	}
	if m.TotalTokens != 1500 {
		t.Errorf("TotalTokens = %d, want 1500", m.TotalTokens)
	}
	if m.CacheHits != 0 {
		t.Errorf("CacheHits = %d, want 0", m.CacheHits)
	}
	if m.TokensByCaller["claude"] != 1500 {
		t.Errorf("TokensByCaller[claude] = %d, want 1500", m.TokensByCaller["claude"])
	}
	if m.TokensByModel["claude-sonnet"] != 1500 {
		t.Errorf("TokensByModel[claude-sonnet] = %d, want 1500", m.TokensByModel["claude-sonnet"])
	}
}

func TestUpdateGenerate_CacheHit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.yml")

	caller := history.CallerContext{Caller: "test", Model: "test-model"}

	if err := UpdateGenerate(path, 500, true, caller); err != nil {
		t.Fatal(err)
	}

	m := readOrInit(path)
	if m.CacheHits != 1 {
		t.Errorf("CacheHits = %d, want 1", m.CacheHits)
	}
}

func TestUpdateGenerate_Accumulates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.yml")

	caller1 := history.CallerContext{Caller: "claude", Model: "claude-sonnet"}
	caller2 := history.CallerContext{Caller: "cursor", Model: "gpt-4o"}

	_ = UpdateGenerate(path, 1000, false, caller1)
	_ = UpdateGenerate(path, 2000, true, caller2)
	_ = UpdateGenerate(path, 500, false, caller1)

	m := readOrInit(path)
	if m.GenerateInvocations != 3 {
		t.Errorf("GenerateInvocations = %d, want 3", m.GenerateInvocations)
	}
	if m.TotalTokens != 3500 {
		t.Errorf("TotalTokens = %d, want 3500", m.TotalTokens)
	}
	if m.CacheHits != 1 {
		t.Errorf("CacheHits = %d, want 1", m.CacheHits)
	}
	if m.TokensByCaller["claude"] != 1500 {
		t.Errorf("TokensByCaller[claude] = %d, want 1500", m.TokensByCaller["claude"])
	}
	if m.TokensByCaller["cursor"] != 2000 {
		t.Errorf("TokensByCaller[cursor] = %d, want 2000", m.TokensByCaller["cursor"])
	}
	if m.TokensByModel["claude-sonnet"] != 1500 {
		t.Errorf("TokensByModel[claude-sonnet] = %d, want 1500", m.TokensByModel["claude-sonnet"])
	}
	if m.TokensByModel["gpt-4o"] != 2000 {
		t.Errorf("TokensByModel[gpt-4o] = %d, want 2000", m.TokensByModel["gpt-4o"])
	}
}

func TestSyncGlobal(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "usage.yml")
	globalPath := filepath.Join(dir, "global_usage.yml")

	// Populate local with some data.
	caller := history.CallerContext{Caller: "claude", Model: "claude-sonnet"}
	_ = UpdateQuery(localPath)
	_ = UpdateQuery(localPath)
	_ = UpdateGenerate(localPath, 1500, false, caller)
	_ = UpdateGenerate(localPath, 500, true, caller)

	if err := SyncGlobal(localPath, globalPath, "test-project"); err != nil {
		t.Fatalf("SyncGlobal: %v", err)
	}

	// Verify global has the merged data.
	global := ReadGlobal(globalPath)
	if global.TotalTokens != 2000 {
		t.Errorf("global.TotalTokens = %d, want 2000", global.TotalTokens)
	}
	if global.QueryInvocations != 2 {
		t.Errorf("global.QueryInvocations = %d, want 2", global.QueryInvocations)
	}
	if global.GenerateInvocations != 2 {
		t.Errorf("global.GenerateInvocations = %d, want 2", global.GenerateInvocations)
	}
	if global.CacheHits != 1 {
		t.Errorf("global.CacheHits = %d, want 1", global.CacheHits)
	}
	if global.Project != "test-project" {
		t.Errorf("global.Project = %q, want %q", global.Project, "test-project")
	}
	if global.LastCompile == 0 {
		t.Error("global.LastCompile should be set")
	}

	// Verify local was reset.
	local := ReadLocal(localPath)
	if local.TotalTokens != 0 {
		t.Errorf("local.TotalTokens = %d, want 0 after sync", local.TotalTokens)
	}
	if local.QueryInvocations != 0 {
		t.Errorf("local.QueryInvocations = %d, want 0 after sync", local.QueryInvocations)
	}
}

func TestSyncGlobal_MergesIntoExisting(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "usage.yml")
	globalPath := filepath.Join(dir, "global_usage.yml")

	// First sync.
	caller := history.CallerContext{Caller: "claude", Model: "claude-sonnet"}
	_ = UpdateGenerate(localPath, 1000, false, caller)
	_ = SyncGlobal(localPath, globalPath, "test-project")

	// Second sync with new local data.
	_ = UpdateGenerate(localPath, 2000, false, caller)
	_ = SyncGlobal(localPath, globalPath, "test-project")

	global := ReadGlobal(globalPath)
	if global.TotalTokens != 3000 {
		t.Errorf("global.TotalTokens = %d, want 3000 (merged)", global.TotalTokens)
	}
	if global.GenerateInvocations != 2 {
		t.Errorf("global.GenerateInvocations = %d, want 2 (merged)", global.GenerateInvocations)
	}
}

func TestSyncGlobal_NoopWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "usage.yml")
	globalPath := filepath.Join(dir, "global_usage.yml")

	// Create empty local file.
	_ = InitLocalFile(localPath)

	if err := SyncGlobal(localPath, globalPath, "test-project"); err != nil {
		t.Fatalf("SyncGlobal: %v", err)
	}

	// Global should not exist since there was nothing to sync.
	if _, err := os.Stat(globalPath); err == nil {
		t.Error("global file should not be created when local is empty")
	}
}

func TestReadLocal_MissingFile(t *testing.T) {
	m := ReadLocal("/nonexistent/path/usage.yml")
	if m.TokensByCaller == nil {
		t.Error("TokensByCaller should be initialized even for missing file")
	}
	if m.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0 for missing file", m.TotalTokens)
	}
}

func TestReadGlobal_MissingFile(t *testing.T) {
	m := ReadGlobal("/nonexistent/path/global_usage.yml")
	if m.TokensByModel == nil {
		t.Error("TokensByModel should be initialized even for missing file")
	}
}

func TestReadOrInit_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.yml")

	// Write invalid YAML.
	if err := os.WriteFile(path, []byte(":::not yaml:::"), 0644); err != nil {
		t.Fatal(err)
	}

	m := readOrInit(path)
	// Should return zero-initialized metrics, not panic.
	if m.TokensByCaller == nil {
		t.Error("TokensByCaller should be initialized for corrupted file")
	}
}

func TestLocalPath(t *testing.T) {
	cfg := &project.Config{Root: "docs"}
	path := LocalPath("/project", cfg)
	if !strings.HasSuffix(path, "usage.yml") {
		t.Errorf("expected path to end with usage.yml, got %q", path)
	}
	if !strings.Contains(path, ".codectx") {
		t.Errorf("expected path to contain .codectx, got %q", path)
	}
}

func TestGlobalPath(t *testing.T) {
	cfg := &project.Config{Root: "docs"}
	path := GlobalPath("/project", cfg)
	if !strings.HasSuffix(path, "global_usage.yml") {
		t.Errorf("expected path to end with global_usage.yml, got %q", path)
	}
}
