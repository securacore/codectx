package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Session represents a documentation authoring session.
type Session struct {
	ID        string    `yaml:"id"`
	Bin       string    `yaml:"bin"`                  // AI binary used (e.g., "claude", "opencode")
	SessionID string    `yaml:"session_id,omitempty"` // AI binary's own session ID for resume
	Title     string    `yaml:"title"`
	Category  string    `yaml:"category,omitempty"` // foundation/topic/prompt/application
	Target    string    `yaml:"target,omitempty"`   // e.g., docs/topics/go-error-handling/
	Phase     Phase     `yaml:"phase"`
	Created   time.Time `yaml:"created"`
	Updated   time.Time `yaml:"updated"`
}

// NewSession creates a new session with a UUID-based ID.
func NewSession(bin string) *Session {
	id := uuid.New().String()[:8] // Short prefix until classified
	now := time.Now().UTC()
	return &Session{
		ID:      id,
		Bin:     bin,
		Title:   "New document",
		Phase:   PhaseDiscover,
		Created: now,
		Updated: now,
	}
}

// sessionDir returns the sessions directory path inside the output directory.
func sessionDir(outputDir string) string {
	return filepath.Join(outputDir, "sessions")
}

// Save writes the session to a YAML file in the sessions directory.
func Save(outputDir string, s *Session) error {
	dir := sessionDir(outputDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	s.Updated = time.Now().UTC()

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	path := filepath.Join(dir, s.ID+".yml")
	return os.WriteFile(path, data, 0o644)
}

// Load reads a session from the sessions directory by ID.
func Load(outputDir, id string) (*Session, error) {
	path := filepath.Join(sessionDir(outputDir), id+".yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session %q: %w", id, err)
	}

	var s Session
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse session %q: %w", id, err)
	}
	return &s, nil
}

// List returns all sessions sorted by updated time (newest first).
func List(outputDir string) ([]*Session, error) {
	dir := sessionDir(outputDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".yml")
		s, err := Load(outputDir, id)
		if err != nil {
			continue // Skip corrupt sessions
		}
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Updated.After(sessions[j].Updated)
	})

	return sessions, nil
}

// Active returns sessions that are not in PhaseComplete, sorted by updated
// time (newest first).
func Active(outputDir string) ([]*Session, error) {
	all, err := List(outputDir)
	if err != nil {
		return nil, err
	}

	var active []*Session
	for _, s := range all {
		if s.Phase != PhaseComplete {
			active = append(active, s)
		}
	}
	return active, nil
}

// Cleanup removes completed sessions older than maxAge.
func Cleanup(outputDir string, maxAge time.Duration) (int, error) {
	all, err := List(outputDir)
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().UTC().Add(-maxAge)
	removed := 0
	dir := sessionDir(outputDir)

	for _, s := range all {
		if s.Phase == PhaseComplete && s.Updated.Before(cutoff) {
			path := filepath.Join(dir, s.ID+".yml")
			if err := os.Remove(path); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}
