package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupWatchProject creates a minimal project in a temp directory,
// changes cwd into it, and returns the project root.
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
	require.NoError(t, config.Write(filepath.Join(dir, "codectx.yml"), cfg))

	m := &manifest.Manifest{
		Name:    "watch-test",
		Version: "1.0.0",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	return dir
}

func TestNew_defaults(t *testing.T) {
	w := New("codectx.yml")
	assert.Equal(t, "codectx.yml", w.configFile)
	assert.Equal(t, defaultDebounce, w.debounce)
	assert.Equal(t, defaultPollInterval, w.pollInterval)
	assert.NotNil(t, w.results)
}

func TestNew_withDebounce(t *testing.T) {
	w := New("codectx.yml", WithDebounce(500*time.Millisecond))
	assert.Equal(t, 500*time.Millisecond, w.debounce)
}

func TestNew_withPollInterval(t *testing.T) {
	w := New("codectx.yml", WithPollInterval(60*time.Second))
	assert.Equal(t, 60*time.Second, w.pollInterval)
}

func TestNew_withPollIntervalDisabled(t *testing.T) {
	w := New("codectx.yml", WithPollInterval(0))
	assert.Equal(t, time.Duration(0), w.pollInterval)
}

func TestRun_initialCompile(t *testing.T) {
	setupWatchProject(t)

	ctx, cancel := context.WithCancel(context.Background())
	w := New("codectx.yml", WithDebounce(50*time.Millisecond))

	go func() {
		_ = w.Run(ctx)
	}()

	// Wait for initial compile result.
	select {
	case result := <-w.Results():
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Compiled)
		assert.False(t, result.Compiled.UpToDate)
		assert.False(t, result.Timestamp.IsZero())
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile result")
	}

	cancel()
}

func TestRun_detectsFileChange(t *testing.T) {
	dir := setupWatchProject(t)
	docsDir := filepath.Join(dir, "docs")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml", WithDebounce(100*time.Millisecond))

	go func() {
		_ = w.Run(ctx)
	}()

	// Consume initial compile result.
	select {
	case <-w.Results():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// Brief pause to ensure watcher is fully set up.
	time.Sleep(100 * time.Millisecond)

	// Modify the package manifest (a file that changes the fingerprint).
	m := &manifest.Manifest{
		Name:        "watch-test",
		Version:     "1.0.1",
		Description: "Updated",
	}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Wait for the change-triggered compile.
	select {
	case result := <-w.Results():
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Compiled)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for file change compile")
	}
}

func TestRun_debounce(t *testing.T) {
	dir := setupWatchProject(t)
	docsDir := filepath.Join(dir, "docs")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml",
		WithDebounce(200*time.Millisecond),
		WithPollInterval(0), // disable polling to avoid interference
	)

	go func() {
		_ = w.Run(ctx)
	}()

	// Consume initial compile.
	select {
	case <-w.Results():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// Write to manifest.yml multiple times rapidly -- should coalesce.
	for i := range 5 {
		m := &manifest.Manifest{
			Name:    "watch-test",
			Version: fmt.Sprintf("1.0.%d", i+1),
		}
		require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for the single debounced compile.
	select {
	case result := <-w.Results():
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Compiled)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for debounced compile")
	}

	// Verify no additional compile result arrives (debouncing worked).
	// The conditional manifest.Write prevents Compile's sync write-back
	// from generating a self-triggering filesystem event.
	select {
	case <-w.Results():
		t.Fatal("received unexpected second compile result")
	case <-time.After(500 * time.Millisecond):
		// Good: no extra compile.
	}
}

func TestRun_contextCancellation(t *testing.T) {
	setupWatchProject(t)

	ctx, cancel := context.WithCancel(context.Background())
	w := New("codectx.yml", WithDebounce(50*time.Millisecond))

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(ctx)
	}()

	// Consume initial compile.
	select {
	case <-w.Results():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// Cancel context.
	cancel()

	// Run should exit cleanly.
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for watcher to stop")
	}
}

func TestRun_compileErrorContinues(t *testing.T) {
	dir := setupWatchProject(t)
	docsDir := filepath.Join(dir, "docs")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml", WithDebounce(100*time.Millisecond))

	go func() {
		_ = w.Run(ctx)
	}()

	// Consume initial compile.
	select {
	case <-w.Results():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// Brief pause to ensure watcher is fully set up.
	time.Sleep(100 * time.Millisecond)

	// Corrupt the package manifest to cause a compile error.
	// This changes the watched file content, triggering an event.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "manifest.yml"),
		[]byte("{{{{not valid yaml"), 0o644))

	// The error result should be received but the watcher continues.
	select {
	case result := <-w.Results():
		assert.Error(t, result.Error)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for error result")
	}

	// Fix the manifest to trigger another change.
	m := &manifest.Manifest{Name: "watch-test", Version: "1.0.1"}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Should get a successful compile.
	select {
	case result := <-w.Results():
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Compiled)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for recovery compile")
	}
}

func TestRun_newDirectoryWatched(t *testing.T) {
	dir := setupWatchProject(t)
	docsDir := filepath.Join(dir, "docs")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml", WithDebounce(100*time.Millisecond))

	go func() {
		_ = w.Run(ctx)
	}()

	// Consume initial compile.
	select {
	case <-w.Results():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// Create a new subdirectory -- this itself triggers a Create event
	// which the debounce coalesces into a compile. The compile will
	// result in UpToDate (new dir doesn't change the fingerprint),
	// but we verify the watcher doesn't crash and continues.
	newDir := filepath.Join(docsDir, "topics", "newlang")
	require.NoError(t, os.MkdirAll(newDir, 0o755))

	// Now modify manifest.yml so the fingerprint changes, to verify
	// the watcher is still functional after adding the new directory.
	time.Sleep(200 * time.Millisecond) // let dir creation event settle
	m := &manifest.Manifest{Name: "watch-test", Version: "2.0.0"}
	require.NoError(t, manifest.Write(filepath.Join(docsDir, "manifest.yml"), m))

	// Should trigger a compile (may get UpToDate from dir create first).
	deadline := time.After(5 * time.Second)
	for {
		select {
		case result := <-w.Results():
			if result.Error == nil && result.Compiled != nil && !result.Compiled.UpToDate {
				return // success: got a real compile after dir creation
			}
			// Got an UpToDate result from the dir creation -- keep waiting.
		case <-deadline:
			t.Fatal("timed out waiting for compile after new directory")
		}
	}
}

func TestRun_missingConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml", WithDebounce(50*time.Millisecond))

	// Run should fail because the initial compile sends an error,
	// then config.Load fails for watcher setup.
	err = w.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestIsUnderDir(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		dir    string
		expect bool
	}{
		{"exact match", "/a/b", "/a/b", true},
		{"child path", "/a/b/c", "/a/b", true},
		{"not under", "/a/x", "/a/b", false},
		{"prefix but not dir", "/a/bc", "/a/b", false},
		{"empty dir", "/a/b", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, isUnderDir(tt.path, tt.dir))
		})
	}
}

func TestResults_channel(t *testing.T) {
	w := New("codectx.yml")
	ch := w.Results()
	assert.NotNil(t, ch)
}

func TestRun_pollHeartbeat(t *testing.T) {
	setupWatchProject(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a short poll interval and disable debounce-triggered events
	// by using a very long debounce so only the heartbeat fires.
	w := New("codectx.yml",
		WithDebounce(50*time.Millisecond),
		WithPollInterval(200*time.Millisecond),
	)

	go func() {
		_ = w.Run(ctx)
	}()

	// Consume initial compile.
	select {
	case result := <-w.Results():
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Compiled)
		assert.False(t, result.Compiled.UpToDate)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// Without any filesystem changes, the poll heartbeat should fire
	// and produce an UpToDate result.
	select {
	case result := <-w.Results():
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Compiled)
		assert.True(t, result.Compiled.UpToDate)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for poll heartbeat result")
	}
}

func TestRun_pollDisabled(t *testing.T) {
	setupWatchProject(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml",
		WithDebounce(50*time.Millisecond),
		WithPollInterval(0),
	)

	go func() {
		_ = w.Run(ctx)
	}()

	// Consume initial compile.
	select {
	case <-w.Results():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// With polling disabled and no filesystem changes, no result
	// should arrive within a reasonable window.
	select {
	case <-w.Results():
		t.Fatal("received unexpected result with polling disabled")
	case <-time.After(500 * time.Millisecond):
		// Good: no heartbeat fired.
	}
}

func TestRun_docsDirChange(t *testing.T) {
	dir := setupWatchProject(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml",
		WithDebounce(100*time.Millisecond),
		WithPollInterval(0), // disable polling to avoid spurious compiles
	)

	go func() {
		_ = w.Run(ctx)
	}()

	// Consume initial compile.
	select {
	case <-w.Results():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// Brief pause to ensure watcher is fully set up.
	time.Sleep(100 * time.Millisecond)

	// Create the alternative docs directory with its own manifest.yml.
	newDocsDir := filepath.Join(dir, "documentation")
	for _, sub := range []string{"foundation", "topics", "prompts", "plans"} {
		require.NoError(t, os.MkdirAll(filepath.Join(newDocsDir, sub), 0o755))
	}
	m := &manifest.Manifest{
		Name:    "watch-test",
		Version: "1.0.0",
	}
	require.NoError(t, manifest.Write(filepath.Join(newDocsDir, "manifest.yml"), m))

	// Modify codectx.yml to point docs_dir to "documentation".
	cfg := &config.Config{
		Name:     "watch-test",
		Packages: []config.PackageDep{},
		Config:   &config.BuildConfig{DocsDir: "documentation"},
	}
	require.NoError(t, config.Write(filepath.Join(dir, "codectx.yml"), cfg))

	// Wait for the config-change compile (triggered by writing codectx.yml).
	select {
	case result := <-w.Results():
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Compiled)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for config-change compile")
	}

	// Brief pause to let the watcher rewire to the new docs dir.
	time.Sleep(200 * time.Millisecond)

	// Modify a file in the NEW documentation/ directory.
	m2 := &manifest.Manifest{
		Name:        "watch-test",
		Version:     "2.0.0",
		Description: "Updated in new docs dir",
	}
	require.NoError(t, manifest.Write(filepath.Join(newDocsDir, "manifest.yml"), m2))

	// Verify a compile is triggered from the new docs directory.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case result := <-w.Results():
			if result.Error == nil && result.Compiled != nil && !result.Compiled.UpToDate {
				return // success: got a real compile from the new docs dir
			}
			// May get UpToDate results; keep waiting.
		case <-deadline:
			t.Fatal("timed out waiting for compile from new docs directory")
		}
	}
}

// --- Sync integration tests ---

func TestRun_initialCompileIncludesSyncResult(t *testing.T) {
	setupWatchProject(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml", WithDebounce(50*time.Millisecond), WithPollInterval(0))

	go func() {
		_ = w.Run(ctx)
	}()

	select {
	case result := <-w.Results():
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.Sync, "initial compile should include sync result")
		assert.Equal(t, 0, result.Sync.Entries)
		assert.Equal(t, 0, result.Sync.Discovered)
		assert.Equal(t, 0, result.Sync.Removed)
		assert.Equal(t, 0, result.Sync.Relationships)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}
}

func TestRun_syncReportsDiscoveredEntry(t *testing.T) {
	dir := setupWatchProject(t)
	docsDir := filepath.Join(dir, "docs")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml", WithDebounce(100*time.Millisecond), WithPollInterval(0))

	go func() {
		_ = w.Run(ctx)
	}()

	// Consume initial compile (empty project).
	select {
	case result := <-w.Results():
		require.NoError(t, result.Error)
		require.NotNil(t, result.Sync)
		assert.Equal(t, 0, result.Sync.Entries)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}

	// Brief pause to ensure watcher is set up.
	time.Sleep(100 * time.Millisecond)

	// Create a new foundation file that Sync will discover.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "new-doc.md"),
		[]byte("# New Document\n"), 0o644))

	// The file creation triggers a compile. Sync should discover the entry.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case result := <-w.Results():
			if result.Error != nil {
				continue
			}
			if result.Sync != nil && result.Sync.Discovered > 0 {
				assert.Equal(t, 1, result.Sync.Discovered)
				assert.Equal(t, 1, result.Sync.Entries)
				return // success
			}
			// May get UpToDate results; keep waiting.
		case <-deadline:
			t.Fatal("timed out waiting for sync to discover new entry")
		}
	}
}

func TestRun_syncReportsRelationships(t *testing.T) {
	dir := setupWatchProject(t)
	docsDir := filepath.Join(dir, "docs")

	// Pre-create two foundation files that link to each other.
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "alpha.md"),
		[]byte("# Alpha\nSee [beta](beta.md).\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(docsDir, "foundation", "beta.md"),
		[]byte("# Beta\nSee [alpha](alpha.md).\n"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New("codectx.yml", WithDebounce(50*time.Millisecond), WithPollInterval(0))

	go func() {
		_ = w.Run(ctx)
	}()

	// Initial compile should discover both entries and infer relationships.
	select {
	case result := <-w.Results():
		require.NoError(t, result.Error)
		require.NotNil(t, result.Sync)
		assert.Equal(t, 2, result.Sync.Entries)
		assert.Equal(t, 2, result.Sync.Discovered)
		// alpha→beta and beta→alpha = 2 depends_on relationships.
		assert.Equal(t, 2, result.Sync.Relationships)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial compile")
	}
}

// --- Helper unit tests ---

func TestCountManifestEntries(t *testing.T) {
	m := &manifest.Manifest{
		Foundation:  []manifest.FoundationEntry{{ID: "a"}, {ID: "b"}},
		Application: []manifest.ApplicationEntry{{ID: "c"}},
		Topics:      []manifest.TopicEntry{{ID: "d"}, {ID: "e"}, {ID: "f"}},
		Prompts:     []manifest.PromptEntry{{ID: "g"}},
		Plans:       []manifest.PlanEntry{{ID: "h"}},
	}
	assert.Equal(t, 8, countManifestEntries(m))
}

func TestCountManifestEntries_empty(t *testing.T) {
	m := &manifest.Manifest{}
	assert.Equal(t, 0, countManifestEntries(m))
}

func TestCountRelationships_watch(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", DependsOn: []string{"b", "c"}},
			{ID: "b", DependsOn: []string{"a"}},
		},
		Topics: []manifest.TopicEntry{
			{ID: "d", DependsOn: []string{"a"}},
		},
	}
	assert.Equal(t, 4, countRelationships(m))
}

func TestCountRelationships_empty(t *testing.T) {
	m := &manifest.Manifest{}
	assert.Equal(t, 0, countRelationships(m))
}

func TestCountRelationships_allSections(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", DependsOn: []string{"b"}},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "b", DependsOn: []string{"c", "d"}},
		},
		Topics: []manifest.TopicEntry{
			{ID: "c", DependsOn: []string{"a"}},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "d", DependsOn: []string{"a"}},
		},
		Plans: []manifest.PlanEntry{
			{ID: "e", DependsOn: []string{"a", "b"}},
		},
	}
	assert.Equal(t, 7, countRelationships(m))
}
