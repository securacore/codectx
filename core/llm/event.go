package llm

// EventType classifies streaming events from a provider.
type EventType int

const (
	EventDelta   EventType = iota // Incremental text content
	EventToolUse                  // AI is invoking a tool (Read, Glob, Grep)
	EventResult                   // Final result with session ID and usage
	EventError                    // Error from the provider
)

// String returns the event type name for logging and display.
func (t EventType) String() string {
	switch t {
	case EventDelta:
		return "delta"
	case EventToolUse:
		return "tool_use"
	case EventResult:
		return "result"
	case EventError:
		return "error"
	default:
		return "unknown"
	}
}

// Event represents a single streaming event from the provider.
type Event struct {
	Type      EventType // Delta, ToolUse, Result, Error
	Content   string    // Text delta, tool name, result text, or error message
	SessionID string    // Populated on Result events (Claude CLI)
	Usage     *Usage    // Populated on Result events
}

// Usage tracks token consumption for a single exchange.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// DeltaEvent creates a text delta event.
func DeltaEvent(content string) Event {
	return Event{Type: EventDelta, Content: content}
}

// ToolUseEvent creates a tool invocation event.
func ToolUseEvent(toolName string) Event {
	return Event{Type: EventToolUse, Content: toolName}
}

// ResultEvent creates a final result event.
func ResultEvent(content, sessionID string, usage *Usage) Event {
	return Event{
		Type:      EventResult,
		Content:   content,
		SessionID: sessionID,
		Usage:     usage,
	}
}

// ErrorEvent creates an error event.
func ErrorEvent(msg string) Event {
	return Event{Type: EventError, Content: msg}
}
