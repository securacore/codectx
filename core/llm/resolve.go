package llm

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// Resolve detects the best available provider and returns it.
// Priority order: claude CLI > ANTHROPIC_API_KEY > ollama.
func Resolve() (Provider, error) {
	// 1. Check for claude binary on PATH.
	if path, err := exec.LookPath("claude"); err == nil {
		return NewClaude(path), nil
	}

	// 2. Check for ANTHROPIC_API_KEY environment variable.
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		_ = key // SDK reads ANTHROPIC_API_KEY automatically.
		return NewAnthropic(), nil
	}

	// 3. Check for ollama binary and running service.
	if _, err := exec.LookPath("ollama"); err == nil {
		if ollamaReady() {
			return NewOllama(""), nil
		}
	}

	return nil, fmt.Errorf(
		"no AI provider available\n\n" +
			"  Install Claude Code:  https://claude.ai/download\n" +
			"  Or set:               ANTHROPIC_API_KEY=<your-key>\n" +
			"  Or install Ollama:    https://ollama.com",
	)
}

// ollamaReady checks whether the ollama HTTP service is responding.
func ollamaReady() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
