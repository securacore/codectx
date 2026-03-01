package tokenizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountTokens_nonZero(t *testing.T) {
	tokens := CountTokens("Hello, world!")
	assert.Greater(t, tokens, 0, "non-empty string should produce tokens")
}

func TestCountTokens_empty(t *testing.T) {
	assert.Equal(t, 0, CountTokens(""))
}

func TestCountTokens_markdown(t *testing.T) {
	md := "# Heading\n\nThis is a paragraph with **bold** and `code`.\n"
	tokens := CountTokens(md)
	assert.Greater(t, tokens, 0)
	// Markdown text should tokenize to fewer tokens than bytes.
	assert.Less(t, tokens, len(md))
}

func TestCountTokensBytes_agreesWithString(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog."
	assert.Equal(t, CountTokens(text), CountTokensBytes([]byte(text)),
		"bytes and string variants should agree")
}

func TestCountTokens_deterministic(t *testing.T) {
	text := "Repeated calls should produce the same token count."
	first := CountTokens(text)
	for i := 0; i < 10; i++ {
		assert.Equal(t, first, CountTokens(text), "call %d should match", i)
	}
}

func TestCountTokens_knownRange(t *testing.T) {
	// "Hello" is a single common token in o200k_base.
	tokens := CountTokens("Hello")
	assert.GreaterOrEqual(t, tokens, 1)
	assert.LessOrEqual(t, tokens, 2)
}

func TestCountTokensBytes_nil(t *testing.T) {
	assert.Equal(t, 0, CountTokensBytes(nil))
}

func BenchmarkCountTokens(b *testing.B) {
	text := "# API Reference\n\nThe function `getUserByID` accepts a user identifier and returns the corresponding user object. Parameters: id (string, required).\n"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CountTokens(text)
	}
}
