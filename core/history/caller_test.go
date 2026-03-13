package history

import (
	"os"
	"testing"
)

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name string
		vals []string
		want string
	}{
		{"first non-empty wins", []string{"a", "b", "c"}, "a"},
		{"skips empty", []string{"", "b", "c"}, "b"},
		{"all empty returns unknown", []string{"", "", ""}, "unknown"},
		{"single value", []string{"hello"}, "hello"},
		{"no values returns unknown", []string{}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coalesce(tt.vals...)
			if got != tt.want {
				t.Errorf("coalesce(%v) = %q, want %q", tt.vals, got, tt.want)
			}
		})
	}
}

func TestResolveCallerContext_EnvVars(t *testing.T) {
	// Save and restore env vars using t.Setenv (which auto-restores).
	envVars := []string{
		EnvCaller, EnvSessionID, EnvModel,
		"CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_SESSION_ID",
		"CURSOR_SESSION_ID", "ANTHROPIC_MODEL",
	}

	// Clear all relevant env vars (save originals via t.Setenv's restore).
	for _, k := range envVars {
		// t.Setenv will save the current value and restore it after the test.
		if v, ok := os.LookupEnv(k); ok {
			t.Setenv(k, v)
		}
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unsetting %s: %v", k, err)
		}
	}

	t.Run("explicit env vars take priority", func(t *testing.T) {
		t.Setenv(EnvCaller, "my-tool")
		t.Setenv(EnvSessionID, "sess_explicit")
		t.Setenv(EnvModel, "gpt-4o")

		// Also set fallback vars — they should be ignored.
		t.Setenv("CLAUDE_CODE_ENTRYPOINT", "should-not-use")
		t.Setenv("CLAUDE_CODE_SESSION_ID", "should-not-use")
		t.Setenv("ANTHROPIC_MODEL", "should-not-use")

		ctx := ResolveCallerContext()

		if ctx.Caller != "my-tool" {
			t.Errorf("Caller = %q, want %q", ctx.Caller, "my-tool")
		}
		if ctx.SessionID != "sess_explicit" {
			t.Errorf("SessionID = %q, want %q", ctx.SessionID, "sess_explicit")
		}
		if ctx.Model != "gpt-4o" {
			t.Errorf("Model = %q, want %q", ctx.Model, "gpt-4o")
		}
	})

	t.Run("fallback env vars used when primary absent", func(t *testing.T) {
		t.Setenv("CLAUDE_CODE_ENTRYPOINT", "claude-code")
		t.Setenv("CLAUDE_CODE_SESSION_ID", "sess_claude")
		t.Setenv("ANTHROPIC_MODEL", "claude-sonnet")

		ctx := ResolveCallerContext()

		if ctx.Caller != "claude-code" {
			t.Errorf("Caller = %q, want %q", ctx.Caller, "claude-code")
		}
		if ctx.SessionID != "sess_claude" {
			t.Errorf("SessionID = %q, want %q", ctx.SessionID, "sess_claude")
		}
		if ctx.Model != "claude-sonnet" {
			t.Errorf("Model = %q, want %q", ctx.Model, "claude-sonnet")
		}
	})

	t.Run("cursor session ID fallback", func(t *testing.T) {
		t.Setenv("CURSOR_SESSION_ID", "cursor-sess-42")

		ctx := ResolveCallerContext()

		if ctx.SessionID != "cursor-sess-42" {
			t.Errorf("SessionID = %q, want %q", ctx.SessionID, "cursor-sess-42")
		}
	})
}

func TestResolveCallerContext_DefaultsToUnknown(t *testing.T) {
	// Save and restore env vars.
	envVars := []string{
		EnvCaller, EnvSessionID, EnvModel,
		"CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_SESSION_ID",
		"CURSOR_SESSION_ID", "ANTHROPIC_MODEL",
	}

	// Clear all relevant env vars.
	for _, k := range envVars {
		if v, ok := os.LookupEnv(k); ok {
			t.Setenv(k, v)
		}
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unsetting %s: %v", k, err)
		}
	}

	ctx := ResolveCallerContext()

	// SessionID and Model should be "unknown" when no env vars are set.
	if ctx.SessionID != "unknown" {
		t.Errorf("SessionID = %q, want %q", ctx.SessionID, "unknown")
	}
	if ctx.Model != "unknown" {
		t.Errorf("Model = %q, want %q", ctx.Model, "unknown")
	}

	// Caller may be "unknown" or a parent process name (non-deterministic).
	// Just verify it's not empty.
	if ctx.Caller == "" {
		t.Error("Caller should not be empty")
	}
}

func TestDetectParentProcessName(t *testing.T) {
	// This is a smoke test — the actual process name is non-deterministic
	// in CI vs local, but it should never panic.
	name := detectParentProcessName()
	// We can't assert the exact name, but it should be a non-panicking call.
	_ = name
}

func TestCallerContext_ZeroValue(t *testing.T) {
	var ctx CallerContext
	if ctx.Caller != "" {
		t.Error("zero-value Caller should be empty")
	}
	if ctx.SessionID != "" {
		t.Error("zero-value SessionID should be empty")
	}
	if ctx.Model != "" {
		t.Error("zero-value Model should be empty")
	}
}

func TestEnvVarConstants(t *testing.T) {
	// Verify the constants match the expected contract.
	if EnvCaller != "CODECTX_CALLER" {
		t.Errorf("EnvCaller = %q, want CODECTX_CALLER", EnvCaller)
	}
	if EnvSessionID != "CODECTX_SESSION_ID" {
		t.Errorf("EnvSessionID = %q, want CODECTX_SESSION_ID", EnvSessionID)
	}
	if EnvModel != "CODECTX_MODEL" {
		t.Errorf("EnvModel = %q, want CODECTX_MODEL", EnvModel)
	}
}
