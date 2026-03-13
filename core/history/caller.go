package history

import (
	"os"

	"github.com/shirou/gopsutil/v3/process"
)

// Environment variable names for caller context detection.
// These are codectx-defined contracts. Calling tools may set them to provide
// explicit context. They take highest priority in the resolution chain.
const (
	EnvCaller    = "CODECTX_CALLER"
	EnvSessionID = "CODECTX_SESSION_ID"
	EnvModel     = "CODECTX_MODEL"
)

// CallerContext holds detected caller metadata resolved at invocation time.
type CallerContext struct {
	// Caller is the detected calling program name.
	Caller string

	// SessionID is the detected session identifier.
	SessionID string

	// Model is the detected AI model identifier.
	Model string
}

// ResolveCallerContext detects caller metadata using a priority chain of
// environment variables and parent process detection. Failure at any step
// falls through to the next. Best-effort — detection errors are never
// surfaced to the user.
func ResolveCallerContext() CallerContext {
	return CallerContext{
		Caller: coalesce(
			os.Getenv(EnvCaller),
			os.Getenv("CLAUDE_CODE_ENTRYPOINT"),
			detectParentProcessName(),
		),
		SessionID: coalesce(
			os.Getenv(EnvSessionID),
			os.Getenv("CLAUDE_CODE_SESSION_ID"),
			os.Getenv("CURSOR_SESSION_ID"),
		),
		Model: coalesce(
			os.Getenv(EnvModel),
			os.Getenv("ANTHROPIC_MODEL"),
		),
	}
}

// detectParentProcessName uses gopsutil to get the parent process name.
// Returns empty string on any error, allowing coalesce to fall through.
func detectParentProcessName() string {
	parent, err := process.NewProcess(int32(os.Getppid()))
	if err != nil {
		return ""
	}
	name, err := parent.Name()
	if err != nil {
		return ""
	}
	return name
}

// coalesce returns the first non-empty string from the given values.
// If all values are empty, returns "unknown".
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return "unknown"
}
