package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClaudeEvent_delta(t *testing.T) {
	line := []byte(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}`)
	evt, ok := parseClaudeEvent(line)
	require.True(t, ok)
	assert.Equal(t, EventDelta, evt.Type)
	assert.Equal(t, "Hello", evt.Content)
}

func TestParseClaudeEvent_assistantText(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Full response here"}]}}`)
	evt, ok := parseClaudeEvent(line)
	require.True(t, ok)
	assert.Equal(t, EventDelta, evt.Type)
	assert.Equal(t, "Full response here", evt.Content)
}

func TestParseClaudeEvent_assistantToolUse(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read"}]}}`)
	evt, ok := parseClaudeEvent(line)
	require.True(t, ok)
	assert.Equal(t, EventToolUse, evt.Type)
	assert.Equal(t, "Read", evt.Content)
}

func TestParseClaudeEvent_result(t *testing.T) {
	line := []byte(`{
		"type":"result",
		"subtype":"success",
		"is_error":false,
		"result":"The answer is 42",
		"session_id":"abc-123",
		"usage":{"input_tokens":100,"output_tokens":50}
	}`)
	evt, ok := parseClaudeEvent(line)
	require.True(t, ok)
	assert.Equal(t, EventResult, evt.Type)
	assert.Equal(t, "The answer is 42", evt.Content)
	assert.Equal(t, "abc-123", evt.SessionID)
	require.NotNil(t, evt.Usage)
	assert.Equal(t, 100, evt.Usage.InputTokens)
	assert.Equal(t, 50, evt.Usage.OutputTokens)
}

func TestParseClaudeEvent_resultError(t *testing.T) {
	line := []byte(`{
		"type":"result",
		"is_error":true,
		"result":"API Error: 401 authentication_error expired token"
	}`)
	evt, ok := parseClaudeEvent(line)
	require.True(t, ok)
	assert.Equal(t, EventError, evt.Type)
	assert.Contains(t, evt.Content, "claude /login")
}

func TestParseClaudeEvent_resultRateLimit(t *testing.T) {
	line := []byte(`{
		"type":"result",
		"is_error":true,
		"result":"API Error: 429 rate_limit exceeded"
	}`)
	evt, ok := parseClaudeEvent(line)
	require.True(t, ok)
	assert.Equal(t, EventError, evt.Type)
	assert.Contains(t, evt.Content, "rate limit")
}

func TestParseClaudeEvent_unknownType(t *testing.T) {
	line := []byte(`{"type":"system","subtype":"init","session_id":"xyz"}`)
	_, ok := parseClaudeEvent(line)
	assert.False(t, ok)
}

func TestParseClaudeEvent_invalidJSON(t *testing.T) {
	line := []byte(`not json at all`)
	_, ok := parseClaudeEvent(line)
	assert.False(t, ok)
}

func TestParseClaudeEvent_emptyDelta(t *testing.T) {
	line := []byte(`{"type":"content_block_delta","delta":{"type":"text_delta","text":""}}`)
	_, ok := parseClaudeEvent(line)
	assert.False(t, ok)
}

func TestParseClaudeEvent_resultNoUsage(t *testing.T) {
	line := []byte(`{"type":"result","is_error":false,"result":"done","session_id":"s1"}`)
	evt, ok := parseClaudeEvent(line)
	require.True(t, ok)
	assert.Equal(t, EventResult, evt.Type)
	assert.Equal(t, "s1", evt.SessionID)
	assert.Nil(t, evt.Usage)
}

func TestNewClaude_ID(t *testing.T) {
	c := NewClaude("/usr/bin/claude")
	assert.Equal(t, "claude", c.ID())
}

// --- buildClaudeArgs ---

func TestBuildClaudeArgs_minimal(t *testing.T) {
	req := &Request{Prompt: "Hello"}
	args := buildClaudeArgs(req)

	assert.Equal(t, []string{"-p", "--verbose", "--output-format", "stream-json"}, args)
	// Prompt must NOT be in the args — it goes via stdin.
	assert.NotContains(t, args, "Hello")
}

func TestBuildClaudeArgs_allFields(t *testing.T) {
	req := &Request{
		Prompt:       "Explain this code",
		SystemPrompt: "You are a documentation expert.",
		SessionID:    "sess-42",
		Tools:        []string{"Read", "Glob", "Grep"},
		Model:        "claude-sonnet-4-20250514",
	}
	args := buildClaudeArgs(req)

	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "--verbose")
	assert.Contains(t, args, "--system-prompt")
	assert.Contains(t, args, "You are a documentation expert.")
	assert.Contains(t, args, "--session-id")
	assert.Contains(t, args, "sess-42")
	assert.Contains(t, args, "--tools")
	assert.Contains(t, args, "Read,Glob,Grep")
	assert.Contains(t, args, "--model")
	assert.Contains(t, args, "claude-sonnet-4-20250514")
	// Prompt must NOT appear as a positional arg.
	assert.NotContains(t, args, "Explain this code")
}

func TestBuildClaudeArgs_noToolsNoPromptLeak(t *testing.T) {
	req := &Request{
		Prompt: "What is codectx?",
	}
	args := buildClaudeArgs(req)

	// Even without tools, prompt must not be in args.
	for _, a := range args {
		assert.NotEqual(t, "What is codectx?", a)
	}
}

func TestBuildClaudeArgs_optionalFieldsOmitted(t *testing.T) {
	req := &Request{Prompt: "test"}
	args := buildClaudeArgs(req)

	assert.NotContains(t, args, "--system-prompt")
	assert.NotContains(t, args, "--session-id")
	assert.NotContains(t, args, "--tools")
	assert.NotContains(t, args, "--model")
}
