package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	ollamaDefaultAddr  = "http://localhost:11434"
	ollamaDefaultModel = "llama3.1"
)

// Ollama implements Provider using the Ollama HTTP API (/api/chat).
// It manages conversation history as a slice of ollamaChatMessage,
// replayed on each call since the API is stateless.
type Ollama struct {
	addr    string
	model   string
	history []ollamaChatMessage
	client  *http.Client
}

// ollamaChatMessage matches the Ollama /api/chat message format.
type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatRequest is the request body for POST /api/chat.
type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

// ollamaChatResponse is a single line in the streaming response.
type ollamaChatResponse struct {
	Model     string            `json:"model"`
	Message   ollamaChatMessage `json:"message"`
	Done      bool              `json:"done"`
	DoneReson string            `json:"done_reason,omitempty"`
}

// NewOllama creates a provider backed by the Ollama HTTP API.
func NewOllama(model string) *Ollama {
	if model == "" {
		model = ollamaDefaultModel
	}
	return &Ollama{
		addr:   ollamaDefaultAddr,
		model:  model,
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

func (o *Ollama) ID() string { return "ollama" }

// Stream sends a chat request to Ollama and returns streaming events.
// Conversation history is replayed on each call for multi-turn continuity.
func (o *Ollama) Stream(ctx context.Context, req *Request) (<-chan Event, error) {
	// Build messages with history.
	var messages []ollamaChatMessage

	// System prompt as the first message.
	if req.SystemPrompt != "" {
		messages = append(messages, ollamaChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Append history.
	messages = append(messages, o.history...)

	// Append user message.
	userMsg := ollamaChatMessage{Role: "user", Content: req.Prompt}
	messages = append(messages, userMsg)

	// Build request body.
	body := ollamaChatRequest{
		Model:    o.model,
		Messages: messages,
		Stream:   true,
	}

	if req.Model != "" {
		body.Model = req.Model
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.addr+"/api/chat", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(errBody))
	}

	ch := make(chan Event, 32)

	go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()

		scanner := bufio.NewScanner(resp.Body)
		var fullContent string

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chatResp ollamaChatResponse
			if err := json.Unmarshal(line, &chatResp); err != nil {
				ch <- ErrorEvent("parse ollama response: " + err.Error())
				return
			}

			if chatResp.Message.Content != "" {
				ch <- DeltaEvent(chatResp.Message.Content)
				fullContent += chatResp.Message.Content
			}

			if chatResp.Done {
				// Save to history.
				o.history = append(o.history, userMsg)
				o.history = append(o.history, ollamaChatMessage{
					Role:    "assistant",
					Content: fullContent,
				})

				ch <- ResultEvent(fullContent, "", nil)
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- ErrorEvent("read ollama stream: " + err.Error())
		}
	}()

	return ch, nil
}
