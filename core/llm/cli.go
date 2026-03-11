package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// cliSender implements Sender by invoking the local Claude CLI binary in
// headless mode. Structured output uses the --json-schema flag for
// schema-validated responses.
type cliSender struct {
	binary string // absolute path to the claude binary
	model  string // model name to pass via --model
}

// newCLISender creates a CLI sender from a binary path and model name.
func newCLISender(binary, model string) *cliSender {
	return &cliSender{
		binary: binary,
		model:  model,
	}
}

// ExecCommandFunc creates exec.Cmd instances. Defaults to exec.CommandContext.
// Override in tests to mock CLI invocation.
var ExecCommandFunc = exec.CommandContext

// SendAliases invokes the claude CLI with the alias JSON schema.
func (s *cliSender) SendAliases(ctx context.Context, system, prompt string) (*AliasResponse, error) {
	raw, err := s.send(ctx, system, prompt, aliasJSONSchema)
	if err != nil {
		return nil, err
	}

	var resp AliasResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing alias response: %w", err)
	}
	return &resp, nil
}

// SendBridges invokes the claude CLI with the bridge JSON schema.
func (s *cliSender) SendBridges(ctx context.Context, system, prompt string) (*BridgeResponse, error) {
	raw, err := s.send(ctx, system, prompt, bridgeJSONSchema)
	if err != nil {
		return nil, err
	}

	var resp BridgeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parsing bridge response: %w", err)
	}
	return &resp, nil
}

// cliEnvelope is the outer JSON structure returned by claude --output-format json.
type cliEnvelope struct {
	Type             string          `json:"type"`
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
}

// send invokes the claude CLI in headless mode and returns the structured output.
func (s *cliSender) send(ctx context.Context, system, prompt, jsonSchema string) (json.RawMessage, error) {
	//nolint:gosec // Arguments are constructed internally, not from user input.
	args := []string{
		"-p",
		"--tools", "",
		"--no-session-persistence",
		"--output-format", "json",
		"--model", s.model,
		"--system-prompt", system,
		"--json-schema", jsonSchema,
		prompt,
	}

	cmd := ExecCommandFunc(ctx, s.binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude CLI failed: %w (stderr: %s)", err, stderr.String())
	}

	var envelope cliEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		return nil, fmt.Errorf("parsing claude CLI output: %w", err)
	}

	if envelope.IsError {
		return nil, fmt.Errorf("claude CLI returned error: %s", envelope.Result)
	}

	if len(envelope.StructuredOutput) == 0 || string(envelope.StructuredOutput) == "null" {
		return nil, fmt.Errorf("claude CLI returned no structured output")
	}

	return envelope.StructuredOutput, nil
}
