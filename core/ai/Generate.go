package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// generateTimeout is the maximum time to wait for an AI response.
const generateTimeout = 30 * time.Second

// Generate invokes the configured AI binary to produce text from a prompt.
// It verifies the binary still exists on PATH before invocation and enforces
// a 30-second timeout.
//
// Supported bins: "claude" (claude -p) and "opencode" (opencode run).
// Returns the trimmed response text or an error.
func Generate(bin, prompt string) (string, error) {
	// Verify the binary is still available.
	path, err := exec.LookPath(bin)
	if err != nil {
		return "", fmt.Errorf("AI binary %q not found on PATH: %w", bin, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), generateTimeout)
	defer cancel()

	var cmd *exec.Cmd
	switch bin {
	case "claude":
		cmd = exec.CommandContext(ctx, path, "-p", prompt)
	case "opencode":
		cmd = exec.CommandContext(ctx, path, "run", prompt)
	default:
		return "", fmt.Errorf("unsupported AI binary for text generation: %q", bin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("AI generation timed out after %s", generateTimeout)
		}
		return "", fmt.Errorf("AI generation failed: %w", err)
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", fmt.Errorf("AI returned an empty response")
	}

	// Guard against excessively long responses: take the first sentence
	// or truncate at 300 characters, whichever comes first.
	result = truncateResponse(result)

	return result, nil
}

// truncateResponse limits a response to the first sentence or 300 characters.
func truncateResponse(s string) string {
	// Try to find the first sentence boundary.
	for i, ch := range s {
		if ch == '.' && i > 10 && i < 300 {
			return s[:i+1]
		}
	}

	// No sentence boundary found; truncate at 300 chars.
	if len(s) > 300 {
		return s[:300]
	}
	return s
}
