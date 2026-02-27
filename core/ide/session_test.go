package ide

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func tmpOutputDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// saveRaw writes a session to disk without updating the Updated timestamp,
// so tests can control the exact value.
func saveRaw(t *testing.T, dir string, s *Session) {
	t.Helper()
	sessDir := SessionDir(dir)
	require.NoError(t, os.MkdirAll(sessDir, 0o755))
	data, err := yaml.Marshal(s)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(sessDir, s.ID+".yml"), data, 0o644))
}

func TestNewSession(t *testing.T) {
	s := NewSession("claude")
	assert.Len(t, s.ID, 8) // UUID prefix
	assert.Equal(t, "claude", s.Provider)
	assert.Equal(t, "New document", s.Title)
	assert.Equal(t, PhaseDiscover, s.Phase)
	assert.False(t, s.Created.IsZero())
	assert.False(t, s.Updated.IsZero())
}

func TestSession_saveAndLoad(t *testing.T) {
	dir := tmpOutputDir(t)
	s := NewSession("claude")
	s.Title = "Go Error Handling"
	s.Category = "topic"
	s.Target = "docs/topics/go-error-handling/"
	s.Phase = PhaseDraft

	err := Save(dir, s)
	require.NoError(t, err)

	loaded, err := Load(dir, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, loaded.ID)
	assert.Equal(t, "Go Error Handling", loaded.Title)
	assert.Equal(t, "topic", loaded.Category)
	assert.Equal(t, "docs/topics/go-error-handling/", loaded.Target)
	assert.Equal(t, PhaseDraft, loaded.Phase)
	assert.Equal(t, "claude", loaded.Provider)
}

func TestSession_list(t *testing.T) {
	dir := tmpOutputDir(t)

	s1 := NewSession("claude")
	s1.Title = "First"
	s1.Updated = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, Save(dir, s1))

	s2 := NewSession("claude")
	s2.Title = "Second"
	s2.Updated = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, Save(dir, s2))

	sessions, err := List(dir)
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	// Newest first.
	assert.Equal(t, "Second", sessions[0].Title)
	assert.Equal(t, "First", sessions[1].Title)
}

func TestSession_listEmpty(t *testing.T) {
	dir := tmpOutputDir(t)
	sessions, err := List(dir)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSession_active(t *testing.T) {
	dir := tmpOutputDir(t)

	s1 := NewSession("claude")
	s1.Title = "Active"
	s1.Phase = PhaseDraft
	require.NoError(t, Save(dir, s1))

	s2 := NewSession("claude")
	s2.Title = "Complete"
	s2.Phase = PhaseComplete
	require.NoError(t, Save(dir, s2))

	active, err := Active(dir)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, "Active", active[0].Title)
}

func TestSession_rename(t *testing.T) {
	dir := tmpOutputDir(t)
	s := NewSession("claude")
	oldID := s.ID
	require.NoError(t, Save(dir, s))

	err := Rename(dir, s, "go-error-handling")
	require.NoError(t, err)
	assert.Equal(t, "go-error-handling", s.ID)

	// Old file should not exist.
	_, err = os.Stat(filepath.Join(SessionDir(dir), oldID+".yml"))
	assert.True(t, os.IsNotExist(err))

	// New file should exist and load correctly.
	loaded, err := Load(dir, "go-error-handling")
	require.NoError(t, err)
	assert.Equal(t, "go-error-handling", loaded.ID)
}

func TestSession_renameCollision(t *testing.T) {
	dir := tmpOutputDir(t)

	// Create an existing session with the target name.
	existing := NewSession("claude")
	existing.ID = "go-error-handling"
	require.NoError(t, Save(dir, existing))

	// Create a new session that will be renamed.
	s := NewSession("claude")
	require.NoError(t, Save(dir, s))

	err := Rename(dir, s, "go-error-handling")
	require.NoError(t, err)
	assert.Equal(t, "go-error-handling-2", s.ID)
}

func TestSession_cleanup(t *testing.T) {
	dir := tmpOutputDir(t)

	// Old completed session (60 days ago).
	old := NewSession("claude")
	old.Phase = PhaseComplete
	old.Updated = time.Now().UTC().Add(-60 * 24 * time.Hour)
	saveRaw(t, dir, old)

	// Recent completed session (1 hour ago).
	recent := NewSession("claude")
	recent.Phase = PhaseComplete
	recent.Updated = time.Now().UTC().Add(-1 * time.Hour)
	saveRaw(t, dir, recent)

	// Active session (old but not complete — should never be removed).
	active := NewSession("claude")
	active.Phase = PhaseDraft
	active.Updated = time.Now().UTC().Add(-60 * 24 * time.Hour)
	saveRaw(t, dir, active)

	removed, err := Cleanup(dir, 30*24*time.Hour) // 30-day cutoff
	require.NoError(t, err)
	assert.Equal(t, 1, removed) // Only the old completed session

	remaining, err := List(dir)
	require.NoError(t, err)
	assert.Len(t, remaining, 2) // Recent completed + active
}

func TestSession_cleanupPreservesRecent(t *testing.T) {
	dir := tmpOutputDir(t)

	s := NewSession("claude")
	s.Phase = PhaseComplete
	s.Updated = time.Now().UTC().Add(-5 * 24 * time.Hour) // 5 days ago
	saveRaw(t, dir, s)

	removed, err := Cleanup(dir, 30*24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 0, removed) // Within 30-day window
}

func TestLoad_notFound(t *testing.T) {
	dir := tmpOutputDir(t)
	_, err := Load(dir, "nonexistent")
	assert.Error(t, err)
}
