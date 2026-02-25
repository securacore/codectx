package watch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	corewatch "github.com/securacore/codectx/core/watch"
)

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

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "watch", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}

func setupWatchProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	docsDir := filepath.Join(dir, "docs")
	for _, sub := range []string{"foundation", "topics", "prompts", "plans"} {
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, sub), 0o755))
	}

	cfg := &config.Config{
		Name:     "watch-test",
		Packages: []config.PackageDep{},
	}
	require.NoError(t, config.Write(filepath.Join(dir, configFile), cfg))

	m := &manifest.Manifest{
		Name:    "watch-test",
		Version: "1.0.0",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	return dir
}

func TestRun_contextCancellation(t *testing.T) {
	setupWatchProject(t)

	// Pre-cancel the context so run() exits via the ctx.Done() branch.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	out := captureStdout(t, func() {
		err := run(ctx)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Watching")
}

func TestRun_missingConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_invalidConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, configFile),
		[]byte("{{{{not valid"), 0o644))

	err = run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestJoinParts(t *testing.T) {
	tests := []struct {
		name   string
		parts  []string
		expect string
	}{
		{"empty", nil, ""},
		{"single", []string{"3 objects"}, "3 objects"},
		{"two", []string{"3 objects", "1 pruned"}, "3 objects, 1 pruned"},
		{"three", []string{"3 objects", "1 pruned", "2 packages"}, "3 objects, 1 pruned, 2 packages"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, joinParts(tt.parts))
		})
	}
}

func TestPrintResult_error(t *testing.T) {
	r := corewatch.Result{
		Error:     fmt.Errorf("something went wrong"),
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	assert.Contains(t, out, "something went wrong")
}

func TestPrintResult_nilCompiled(t *testing.T) {
	r := corewatch.Result{
		Compiled:  nil,
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	assert.Empty(t, out)
}

func TestPrintResult_upToDate(t *testing.T) {
	r := corewatch.Result{
		Compiled:  &compile.Result{UpToDate: true},
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	// Up-to-date produces no output (silent skip).
	assert.Empty(t, out)
}

func TestPrintResult_normalCompile(t *testing.T) {
	r := corewatch.Result{
		Compiled:  &compile.Result{ObjectsStored: 5},
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	assert.Contains(t, out, "Compiled")
	assert.Contains(t, out, "5 objects")
}

func TestPrintResult_withPrunedAndPackages(t *testing.T) {
	r := corewatch.Result{
		Compiled: &compile.Result{
			ObjectsStored: 3,
			ObjectsPruned: 2,
			Packages:      1,
		},
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	assert.Contains(t, out, "Compiled")
	assert.Contains(t, out, "3 objects")
	assert.Contains(t, out, "2 pruned")
	assert.Contains(t, out, "1 packages")
}

func TestPrintResult_withConflicts(t *testing.T) {
	r := corewatch.Result{
		Compiled: &compile.Result{
			ObjectsStored: 4,
			Dedup: compile.DeduplicationReport{
				Conflicts: []compile.ConflictEntry{
					{
						Section:    "foundation",
						ID:         "overview",
						WinnerPkg:  "local",
						SkippedPkg: "react@org",
						Reason:     "conflict",
					},
				},
			},
		},
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	assert.Contains(t, out, "Compiled")
	assert.Contains(t, out, "4 objects")
	assert.Contains(t, out, "1 conflict")
}

// --- Sync output tests ---

func TestPrintResult_withSyncDiscovered(t *testing.T) {
	r := corewatch.Result{
		Compiled: &compile.Result{ObjectsStored: 3},
		Sync: &corewatch.SyncResult{
			Entries:       5,
			Discovered:    2,
			Relationships: 3,
		},
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	assert.Contains(t, out, "Synced")
	assert.Contains(t, out, "5 entries")
	assert.Contains(t, out, "+2 discovered")
	assert.Contains(t, out, "3 relationships")
	assert.Contains(t, out, "Compiled")
}

func TestPrintResult_withSyncRemoved(t *testing.T) {
	r := corewatch.Result{
		Compiled: &compile.Result{ObjectsStored: 1},
		Sync: &corewatch.SyncResult{
			Entries: 3,
			Removed: 1,
		},
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	assert.Contains(t, out, "Synced")
	assert.Contains(t, out, "3 entries")
	assert.Contains(t, out, "-1 removed")
}

func TestPrintResult_withSyncNoChanges(t *testing.T) {
	r := corewatch.Result{
		Compiled: &compile.Result{ObjectsStored: 2},
		Sync: &corewatch.SyncResult{
			Entries:       4,
			Discovered:    0,
			Removed:       0,
			Relationships: 1,
		},
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	// No sync output when there are no discovered/removed entries.
	assert.NotContains(t, out, "Synced")
	// Compile output should still appear.
	assert.Contains(t, out, "Compiled")
}

func TestPrintResult_withSyncNil(t *testing.T) {
	r := corewatch.Result{
		Compiled:  &compile.Result{ObjectsStored: 1},
		Sync:      nil,
		Timestamp: time.Now(),
	}
	out := captureStdout(t, func() { printResult(r) })
	assert.NotContains(t, out, "Synced")
	assert.Contains(t, out, "Compiled")
}

func TestPrintSyncResult_allFields(t *testing.T) {
	s := &corewatch.SyncResult{
		Entries:       10,
		Discovered:    3,
		Relationships: 5,
	}
	out := captureStdout(t, func() { printSyncResult(s, "10:30:00") })
	assert.Contains(t, out, "10 entries")
	assert.Contains(t, out, "+3 discovered")
	assert.Contains(t, out, "5 relationships")
	assert.Contains(t, out, "10:30:00")
}

func TestPrintSyncResult_removedOnly(t *testing.T) {
	s := &corewatch.SyncResult{
		Entries: 2,
		Removed: 1,
	}
	out := captureStdout(t, func() { printSyncResult(s, "12:00:00") })
	assert.Contains(t, out, "2 entries")
	assert.Contains(t, out, "-1 removed")
	assert.NotContains(t, out, "discovered")
	assert.NotContains(t, out, "relationships")
}
