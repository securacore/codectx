package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAnthropic_ID(t *testing.T) {
	p := NewAnthropic()
	assert.Equal(t, "anthropic", p.ID())
}

func TestNewAnthropic_implementsProvider(t *testing.T) {
	var _ Provider = NewAnthropic()
}

func TestNewAnthropic_emptyHistory(t *testing.T) {
	p := NewAnthropic()
	assert.Empty(t, p.history)
}
