package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOllama_ID(t *testing.T) {
	p := NewOllama("")
	assert.Equal(t, "ollama", p.ID())
}

func TestNewOllama_implementsProvider(t *testing.T) {
	var _ Provider = NewOllama("")
}

func TestNewOllama_defaultModel(t *testing.T) {
	p := NewOllama("")
	assert.Equal(t, ollamaDefaultModel, p.model)
}

func TestNewOllama_customModel(t *testing.T) {
	p := NewOllama("codellama:7b")
	assert.Equal(t, "codellama:7b", p.model)
}

func TestNewOllama_emptyHistory(t *testing.T) {
	p := NewOllama("")
	assert.Empty(t, p.history)
}

func TestOllama_streamWithMockServer(t *testing.T) {
	// Create a mock Ollama server that returns a streaming chat response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		// Decode request to verify structure.
		var req ollamaChatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "test-model", req.Model)
		assert.True(t, req.Stream)
		assert.NotEmpty(t, req.Messages)

		// Send streaming response.
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		// Delta 1
		resp1, _ := json.Marshal(ollamaChatResponse{
			Message: ollamaChatMessage{Role: "assistant", Content: "Hello "},
			Done:    false,
		})
		_, _ = w.Write(resp1)
		_, _ = w.Write([]byte("\n"))
		if flusher != nil {
			flusher.Flush()
		}

		// Delta 2
		resp2, _ := json.Marshal(ollamaChatResponse{
			Message: ollamaChatMessage{Role: "assistant", Content: "world"},
			Done:    false,
		})
		_, _ = w.Write(resp2)
		_, _ = w.Write([]byte("\n"))
		if flusher != nil {
			flusher.Flush()
		}

		// Done
		resp3, _ := json.Marshal(ollamaChatResponse{
			Message: ollamaChatMessage{Role: "assistant", Content: ""},
			Done:    true,
		})
		_, _ = w.Write(resp3)
		_, _ = w.Write([]byte("\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := NewOllama("test-model")
	p.addr = server.URL

	ch, err := p.Stream(context.Background(), &Request{
		Prompt:       "Hello",
		SystemPrompt: "Be helpful",
	})
	require.NoError(t, err)

	var events []Event
	for evt := range ch {
		events = append(events, evt)
	}

	// Should have 2 deltas + 1 result.
	require.Len(t, events, 3)
	assert.Equal(t, EventDelta, events[0].Type)
	assert.Equal(t, "Hello ", events[0].Content)
	assert.Equal(t, EventDelta, events[1].Type)
	assert.Equal(t, "world", events[1].Content)
	assert.Equal(t, EventResult, events[2].Type)
	assert.Equal(t, "Hello world", events[2].Content)

	// History should be updated.
	assert.Len(t, p.history, 2)
	assert.Equal(t, "user", p.history[0].Role)
	assert.Equal(t, "assistant", p.history[1].Role)
}

func TestOllama_streamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("model not found"))
	}))
	defer server.Close()

	p := NewOllama("nonexistent")
	p.addr = server.URL

	_, err := p.Stream(context.Background(), &Request{Prompt: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model not found")
}
