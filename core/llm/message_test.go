package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRole_String(t *testing.T) {
	assert.Equal(t, "system", RoleSystem.String())
	assert.Equal(t, "user", RoleUser.String())
	assert.Equal(t, "assistant", RoleAssistant.String())
	assert.Equal(t, "unknown", Role(99).String())
}

func TestSystemMessage(t *testing.T) {
	msg := SystemMessage("you are helpful")
	assert.Equal(t, RoleSystem, msg.Role)
	assert.Equal(t, "you are helpful", msg.Content)
}

func TestUserMessage(t *testing.T) {
	msg := UserMessage("hello")
	assert.Equal(t, RoleUser, msg.Role)
	assert.Equal(t, "hello", msg.Content)
}

func TestAssistantMessage(t *testing.T) {
	msg := AssistantMessage("world")
	assert.Equal(t, RoleAssistant, msg.Role)
	assert.Equal(t, "world", msg.Content)
}
