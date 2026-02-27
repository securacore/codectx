package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Claude wraps the Claude CLI binary for streaming AI responses.
// It uses the --output-format stream-json protocol to parse NDJSON events.
type Claude struct {
	binary string // Path to the claude binary
}

// NewClaude creates a Claude provider using the given binary path.
func NewClaude(binary string) *Claude {
	return &Claude{binary: binary}
}

// ID returns "claude".
func (c *Claude) ID() string { return "claude" }

// Stream sends a prompt to the Claude CLI and returns streaming events.
// Each event is sent on the returned channel. The channel is closed
// when the response is complete or an error occurs.
func (c *Claude) Stream(ctx context.Context, req *Request) (<-chan Event, error) {
	args := buildClaudeArgs(req)

	// Pass prompt via stdin rather than as a positional argument.
	// The --tools flag is variadic (<tools...>) and would consume
	// a trailing positional prompt as another tool name.
	cmd := exec.CommandContext(ctx, c.binary, args...)
	cmd.Stdin = strings.NewReader(req.Prompt)
	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("claude stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("claude start: %w", err)
	}

	ch := make(chan Event, 16)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(stdout)
		// Increase buffer for large responses.
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			evt, ok := parseClaudeEvent(line)
			if !ok {
				continue
			}

			select {
			case ch <- evt:
			case <-ctx.Done():
				_ = cmd.Process.Kill()
				return
			}
		}

		// Wait for the process to finish.
		waitErr := cmd.Wait()

		// If we got a scan error, report it.
		if scanErr := scanner.Err(); scanErr != nil {
			ch <- ErrorEvent(fmt.Sprintf("claude output read error: %v", scanErr))
			return
		}

		// If the process exited with an error and we haven't sent a result,
		// report the stderr content.
		if waitErr != nil {
			errMsg := strings.TrimSpace(stderr.String())
			if errMsg == "" {
				errMsg = waitErr.Error()
			}
			ch <- ErrorEvent(fmt.Sprintf("claude process error: %s", errMsg))
		}
	}()

	return ch, nil
}

// buildClaudeArgs constructs the CLI arguments for a claude invocation.
// The prompt is NOT included — it is piped via stdin to avoid the
// --tools variadic flag consuming it as a tool name.
func buildClaudeArgs(req *Request) []string {
	args := []string{
		"-p",
		"--verbose",
		"--output-format", "stream-json",
	}

	if req.SystemPrompt != "" {
		args = append(args, "--system-prompt", req.SystemPrompt)
	}

	if req.SessionID != "" {
		args = append(args, "--session-id", req.SessionID)
	}

	if len(req.Tools) > 0 {
		args = append(args, "--tools", strings.Join(req.Tools, ","))
	}

	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	return args
}

// claudeJSON represents the common fields in Claude CLI stream-json output.
// The Claude CLI emits different JSON shapes per event type; we parse
// the union of all fields and dispatch on "type".
type claudeJSON struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`

	// For assistant messages.
	Message *claudeMessage `json:"message,omitempty"`

	// For content_block_delta events.
	Delta *claudeDelta `json:"delta,omitempty"`

	// For result events.
	Result    string      `json:"result,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
	Usage     *claudeUsge `json:"usage,omitempty"`
}

type claudeMessage struct {
	Content []claudeContent `json:"content,omitempty"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Name string `json:"name,omitempty"` // tool name for tool_use
}

type claudeDelta struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type claudeUsge struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// parseClaudeEvent parses a single line of Claude CLI stream-json output
// and returns the corresponding Event. Returns false if the line should
// be skipped (unrecognized or irrelevant event type).
func parseClaudeEvent(line []byte) (Event, bool) {
	var raw claudeJSON
	if err := json.Unmarshal(line, &raw); err != nil {
		return Event{}, false
	}

	switch raw.Type {
	case "assistant":
		// Full assistant message. Extract text content.
		if raw.Message != nil {
			for _, c := range raw.Message.Content {
				switch c.Type {
				case "text":
					return DeltaEvent(c.Text), true
				case "tool_use":
					return ToolUseEvent(c.Name), true
				}
			}
		}
		return Event{}, false

	case "content_block_delta":
		if raw.Delta != nil && raw.Delta.Text != "" {
			return DeltaEvent(raw.Delta.Text), true
		}
		return Event{}, false

	case "result":
		if raw.IsError {
			msg := raw.Result
			if strings.Contains(msg, "authentication_error") || strings.Contains(msg, "expired") {
				msg = "Claude auth expired. Run: claude /login"
			} else if strings.Contains(msg, "rate_limit") {
				msg = "Claude rate limit reached. Wait a moment and try again."
			}
			return ErrorEvent(msg), true
		}

		var usage *Usage
		if raw.Usage != nil {
			usage = &Usage{
				InputTokens:  raw.Usage.InputTokens,
				OutputTokens: raw.Usage.OutputTokens,
			}
		}
		return ResultEvent(raw.Result, raw.SessionID, usage), true

	default:
		// Ignore unrecognized event types (system, tool_result, etc.).
		return Event{}, false
	}
}
