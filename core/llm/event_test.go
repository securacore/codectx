package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventType_String(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventDelta, "delta"},
		{EventToolUse, "tool_use"},
		{EventResult, "result"},
		{EventError, "error"},
		{EventType(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.eventType.String())
	}
}

func TestDeltaEvent(t *testing.T) {
	evt := DeltaEvent("hello")
	assert.Equal(t, EventDelta, evt.Type)
	assert.Equal(t, "hello", evt.Content)
	assert.Empty(t, evt.SessionID)
	assert.Nil(t, evt.Usage)
}

func TestToolUseEvent(t *testing.T) {
	evt := ToolUseEvent("Read")
	assert.Equal(t, EventToolUse, evt.Type)
	assert.Equal(t, "Read", evt.Content)
}

func TestResultEvent(t *testing.T) {
	usage := &Usage{InputTokens: 100, OutputTokens: 50}
	evt := ResultEvent("done", "sess-123", usage)
	assert.Equal(t, EventResult, evt.Type)
	assert.Equal(t, "done", evt.Content)
	assert.Equal(t, "sess-123", evt.SessionID)
	assert.Equal(t, 100, evt.Usage.InputTokens)
	assert.Equal(t, 50, evt.Usage.OutputTokens)
}

func TestErrorEvent(t *testing.T) {
	evt := ErrorEvent("something failed")
	assert.Equal(t, EventError, evt.Type)
	assert.Equal(t, "something failed", evt.Content)
}
