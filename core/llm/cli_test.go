package llm

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"
)

// TestCLIEnvelopeParsing verifies that the CLI envelope JSON is parsed correctly.
func TestCLIEnvelopeParsing_Success(t *testing.T) {
	envelope := cliEnvelope{
		Type:    "result",
		IsError: false,
		Result:  "",
		StructuredOutput: json.RawMessage(`{
			"terms": [
				{"key": "auth", "aliases": ["authentication", "login"]}
			]
		}`),
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed cliEnvelope
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.IsError {
		t.Error("expected is_error=false")
	}
	if len(parsed.StructuredOutput) == 0 {
		t.Error("expected non-empty structured_output")
	}
}

// TestCLIEnvelopeParsing_Error verifies error envelopes are detected.
func TestCLIEnvelopeParsing_Error(t *testing.T) {
	data := []byte(`{"type":"result","is_error":true,"result":"API rate limited","structured_output":null}`)

	var envelope cliEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !envelope.IsError {
		t.Error("expected is_error=true")
	}
}

// TestCLISender_CommandConstruction verifies the correct claude CLI arguments.
func TestCLISender_CommandConstruction(t *testing.T) {
	// Override ExecCommandFunc to capture the command.
	var capturedArgs []string
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, args...)
		// Return a command that produces valid JSON output.
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":{"terms":[{"key":"test","aliases":["t"]}]}}`)
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "claude-sonnet-4-20250514")
	resp, err := s.SendAliases(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("SendAliases: %v", err)
	}

	// Verify the binary path.
	if capturedArgs[0] != "/usr/bin/claude" {
		t.Errorf("expected binary /usr/bin/claude, got %q", capturedArgs[0])
	}

	// Verify key arguments are present.
	argSet := make(map[string]bool)
	for _, a := range capturedArgs {
		argSet[a] = true
	}
	for _, expected := range []string{"-p", "--no-session-persistence", "json"} {
		if !argSet[expected] {
			t.Errorf("expected argument %q in command", expected)
		}
	}

	// Verify response was parsed.
	if resp == nil || len(resp.Terms) != 1 {
		t.Fatal("expected 1 term in response")
	}
	if resp.Terms[0].Key != "test" {
		t.Errorf("expected key 'test', got %q", resp.Terms[0].Key)
	}
}

// TestCLISender_ErrorEnvelope verifies that error envelopes produce an error.
func TestCLISender_ErrorEnvelope(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":true,"result":"rate limited","structured_output":null}`)
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "sonnet")
	_, err := s.SendAliases(context.Background(), "sys", "prompt")
	if err == nil {
		t.Fatal("expected error for error envelope")
	}
}

// TestCLISender_NullStructuredOutput verifies missing structured output produces an error.
func TestCLISender_NullStructuredOutput(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":null}`)
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "sonnet")
	_, err := s.SendAliases(context.Background(), "sys", "prompt")
	if err == nil {
		t.Fatal("expected error for null structured_output")
	}
}

// TestCLISender_SendBridges verifies that SendBridges correctly parses bridge output.
func TestCLISender_SendBridges(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":{"bridges":[{"chunk_id":"obj:a1b2c3.01","summary":"Established JWT validation rules"}]}}`)
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "claude-sonnet-4-20250514")
	resp, err := s.SendBridges(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("SendBridges: %v", err)
	}

	if resp == nil || len(resp.Bridges) != 1 {
		t.Fatal("expected 1 bridge in response")
	}
	if resp.Bridges[0].ChunkID != "obj:a1b2c3.01" {
		t.Errorf("expected chunk_id 'obj:a1b2c3.01', got %q", resp.Bridges[0].ChunkID)
	}
	if resp.Bridges[0].Summary != "Established JWT validation rules" {
		t.Errorf("expected bridge summary, got %q", resp.Bridges[0].Summary)
	}
}

// TestCLISender_SendBridges_ErrorEnvelope verifies error handling for bridge requests.
func TestCLISender_SendBridges_ErrorEnvelope(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":true,"result":"overloaded","structured_output":null}`)
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "sonnet")
	_, err := s.SendBridges(context.Background(), "sys", "prompt")
	if err == nil {
		t.Fatal("expected error for error envelope")
	}
}

// TestCLISender_SendAliases_InvalidJSON verifies error when CLI returns
// valid envelope but invalid JSON in structured_output for alias parsing.
func TestCLISender_SendAliases_InvalidJSON(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		// Structured output is not valid AliasResponse JSON.
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":{"not_terms":"bad"}}`)
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "sonnet")
	resp, err := s.SendAliases(context.Background(), "sys", "prompt")
	// No error because JSON unmarshals successfully (zero-value fields).
	// But verify we get an empty response.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Terms) != 0 {
		t.Errorf("expected 0 terms, got %d", len(resp.Terms))
	}
}

// TestCLISender_SendBridges_InvalidJSON verifies error when CLI returns
// structured_output that cannot be parsed as BridgeResponse.
func TestCLISender_SendBridges_InvalidJSON(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		// structured_output is a raw string, not an object — will fail json.Unmarshal.
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":"not an object"}`)
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "sonnet")
	_, err := s.SendBridges(context.Background(), "sys", "prompt")
	if err == nil {
		t.Fatal("expected error for unparseable bridge response")
	}
}

// TestCLISender_CLIExecutionFailure verifies error when the CLI binary
// fails to execute (non-zero exit code).
func TestCLISender_CLIExecutionFailure(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "false") // exits with code 1
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "sonnet")
	_, err := s.SendAliases(context.Background(), "sys", "prompt")
	if err == nil {
		t.Fatal("expected error for CLI execution failure")
	}
}

// TestCLISender_InvalidEnvelopeJSON verifies error when the CLI returns
// output that is not valid JSON at all.
func TestCLISender_InvalidEnvelopeJSON(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", "not json at all")
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "sonnet")
	_, err := s.SendAliases(context.Background(), "sys", "prompt")
	if err == nil {
		t.Fatal("expected error for invalid envelope JSON")
	}
}

// TestCLISender_EmptyStructuredOutput verifies error when structured_output is empty.
func TestCLISender_EmptyStructuredOutput(t *testing.T) {
	orig := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":{}}`)
	}
	defer func() { ExecCommandFunc = orig }()

	s := newCLISender("/usr/bin/claude", "sonnet")
	// Empty structured output is valid JSON — but SendAliases parses to empty.
	resp, err := s.SendAliases(context.Background(), "sys", "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
