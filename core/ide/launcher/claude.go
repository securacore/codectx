package launcher

import "github.com/google/uuid"

// Claude implements Launcher for the Claude Code CLI.
type Claude struct {
	path string
}

// NewClaude creates a Claude launcher with the given binary path.
func NewClaude(path string) *Claude {
	return &Claude{path: path}
}

func (c *Claude) ID() string     { return "claude" }
func (c *Claude) Binary() string { return c.path }

// NewSessionArgs builds args for a new Claude session.
// Uses --session-id to supply a pre-generated UUID and --append-system-prompt
// to layer the documentation authoring directive on top of Claude's built-in
// system prompt.
func (c *Claude) NewSessionArgs(sessionID, directive string) []string {
	return []string{
		"--session-id", sessionID,
		"--append-system-prompt", directive,
	}
}

// ResumeArgs builds args to resume an existing Claude session.
// Uses --resume with the session UUID and re-injects the directive
// (it may have changed due to documentation updates since last session).
func (c *Claude) ResumeArgs(sessionID, directive string) []string {
	return []string{
		"--resume", sessionID,
		"--append-system-prompt", directive,
	}
}

// SupportsSessionID returns true — Claude accepts --session-id <uuid>.
func (c *Claude) SupportsSessionID() bool { return true }

// GenerateSessionID creates a new UUID suitable for --session-id.
func (c *Claude) GenerateSessionID() string {
	return uuid.New().String()
}

// FindLatestSession is not needed for Claude (SupportsSessionID is true).
func (c *Claude) FindLatestSession(_ string) (string, error) {
	return "", nil
}
