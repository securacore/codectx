package llm

import "context"

// Provider streams AI responses for documentation authoring.
type Provider interface {
	// Stream sends a request and returns a channel of streaming events.
	// The channel is closed when the response is complete or an error occurs.
	Stream(ctx context.Context, req *Request) (<-chan Event, error)

	// ID returns the provider identifier (e.g., "claude", "anthropic", "ollama").
	ID() string
}

// Request represents a single exchange with the AI.
type Request struct {
	Prompt       string   // User message for this turn
	SystemPrompt string   // Full system prompt (directive + context)
	SessionID    string   // Provider session ID (empty = new session)
	Model        string   // Model override (empty = provider default)
	Tools        []string // Allowed tools (e.g., ["Read", "Glob", "Grep"])
	WorkDir      string   // Working directory for tool access
}
