package llm

// Message represents a single message in conversation history.
type Message struct {
	Role    Role   `yaml:"role" json:"role"`
	Content string `yaml:"content" json:"content"`
}

// Role identifies the author of a message.
type Role int

const (
	RoleSystem    Role = iota // System prompt / directive
	RoleUser                  // User input
	RoleAssistant             // AI response
)

// String returns the role name used in YAML serialization and display.
func (r Role) String() string {
	switch r {
	case RoleSystem:
		return "system"
	case RoleUser:
		return "user"
	case RoleAssistant:
		return "assistant"
	default:
		return "unknown"
	}
}

// SystemMessage creates a system message.
func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

// UserMessage creates a user message.
func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

// AssistantMessage creates an assistant message.
func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}
