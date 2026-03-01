// Package launcher provides AI binary launchers for the codectx ide command.
// Each supported AI tool (Claude, OpenCode) has its own Launcher implementation
// that knows how to build CLI arguments for new and resumed sessions.
package launcher

import (
	"fmt"
	"os/exec"

	"github.com/securacore/codectx/core/ai"
	"github.com/securacore/codectx/core/preferences"
)

// Launcher abstracts how an AI binary is invoked for documentation authoring.
// Each AI tool has different CLI flags for system prompts, session management,
// and interactive mode.
type Launcher interface {
	// ID returns the launcher identifier (e.g., "claude", "opencode").
	ID() string

	// Binary returns the resolved path to the executable.
	Binary() string

	// NewSessionArgs builds CLI arguments for starting a new session.
	// sessionID is a pre-generated ID if SupportsSessionID() is true.
	NewSessionArgs(sessionID, directive string) []string

	// ResumeArgs builds CLI arguments for resuming an existing session.
	ResumeArgs(sessionID, directive string) []string

	// SupportsSessionID returns true if the binary accepts a pre-generated
	// session ID at creation time (e.g., Claude's --session-id flag).
	SupportsSessionID() bool

	// FindLatestSession attempts to discover the session ID created by the
	// most recent interactive session. Used when SupportsSessionID() is false.
	FindLatestSession(projectDir string) (string, error)
}

// Resolve finds the appropriate Launcher based on user preferences and
// available binaries. It checks ai.bin from preferences first, then
// falls back to auto-detection on PATH.
func Resolve(prefs *preferences.Preferences) (Launcher, error) {
	// If ai.bin is configured, use it.
	if prefs != nil && prefs.AI != nil && prefs.AI.Bin != "" {
		return resolveByID(prefs.AI.Bin)
	}

	// Auto-detect: try known binaries in priority order.
	for _, id := range []string{"claude", "opencode"} {
		l, err := resolveByID(id)
		if err == nil {
			return l, nil
		}
	}

	return nil, fmt.Errorf("no supported ai binary found — install claude or opencode, or run: codectx set ai.bin=<binary>")
}

// resolveByID resolves a launcher by provider ID, verifying the binary exists.
func resolveByID(id string) (Launcher, error) {
	provider, ok := ai.ProviderByID(id)
	if !ok {
		return nil, fmt.Errorf("unknown ai binary %q", id)
	}

	path, err := exec.LookPath(provider.Binary)
	if err != nil {
		return nil, fmt.Errorf("launch binary %q not found on PATH", provider.Binary)
	}

	switch id {
	case "claude":
		return NewClaude(path), nil
	case "opencode":
		return NewOpenCode(path), nil
	default:
		return nil, fmt.Errorf("ai binary %q is not supported for interactive sessions", id)
	}
}
