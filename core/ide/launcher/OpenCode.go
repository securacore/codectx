package launcher

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// OpenCode implements Launcher for the OpenCode CLI.
type OpenCode struct {
	path string
}

// NewOpenCode creates an OpenCode launcher with the given binary path.
func NewOpenCode(path string) *OpenCode {
	return &OpenCode{path: path}
}

func (o *OpenCode) ID() string     { return "opencode" }
func (o *OpenCode) Binary() string { return o.path }

// NewSessionArgs builds args for a new OpenCode session.
// Uses --prompt to inject the documentation authoring directive.
func (o *OpenCode) NewSessionArgs(_, directive string) []string {
	return []string{
		"--prompt", directive,
	}
}

// ResumeArgs builds args to resume an existing OpenCode session.
// Uses --session to resume by ID and --prompt to re-inject the directive.
func (o *OpenCode) ResumeArgs(sessionID, directive string) []string {
	return []string{
		"--session", sessionID,
		"--prompt", directive,
	}
}

// SupportsSessionID returns false — OpenCode does not accept pre-generated
// session IDs. The session ID must be extracted after the session completes.
func (o *OpenCode) SupportsSessionID() bool { return false }

// FindLatestSession runs `opencode session list` and parses the output to
// extract the most recently updated session ID.
func (o *OpenCode) FindLatestSession(_ string) (string, error) {
	cmd := exec.Command(o.path, "session", "list")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("list opencode sessions: %w", err)
	}

	return parseSessionList(out.String())
}

// parseSessionList extracts the first session ID from `opencode session list`
// output. The expected format is:
//
//	Session ID                      Title                Updated
//	─────────────────────────────────────────────────────────────
//	ses_xxxxx                       Some title           12:43 PM
func parseSessionList(output string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ses_") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0], nil
			}
		}
	}

	return "", fmt.Errorf("no opencode sessions found")
}
