// Package tokens provides token counting primitives for the codectx
// compilation pipeline. Token counting is a first-class concern — every
// size-related decision (chunk boundaries, context budgets, session
// assembly) is measured in tokens, not words or characters.
//
// The package wraps tiktoken-go/tokenizer to provide a simple API:
// create a Counter for an encoding, then count tokens in strings or
// annotate markdown.Document blocks with their token counts.
//
// Supported encodings:
//   - cl100k_base: GPT-4, GPT-4 Turbo, GPT-3.5 Turbo (default)
//   - o200k_base: GPT-4o, O-series models
//   - p50k_base: Codex models
//   - r50k_base: GPT-3 legacy models
//
// The Counter is safe for concurrent use.
package tokens

import (
	"fmt"

	"github.com/securacore/codectx/core/markdown"
	"github.com/tiktoken-go/tokenizer"
)

// Encoding constants for the supported tokenizer encodings.
// These are the string names used in ai.yml configuration.
const (
	Cl100kBase = "cl100k_base"
	O200kBase  = "o200k_base"
	P50kBase   = "p50k_base"
	R50kBase   = "r50k_base"
)

// Counter wraps a tiktoken codec for counting tokens in text.
// Create one via New() and reuse it — the codec is safe for concurrent use
// and expensive to initialize.
type Counter struct {
	codec    tokenizer.Codec
	encoding string
}

// New creates a Counter for the given encoding name.
// Supported values: "cl100k_base", "o200k_base", "p50k_base", "r50k_base".
// Returns an error for unsupported or empty encoding names.
func New(encoding string) (*Counter, error) {
	enc, err := resolveEncoding(encoding)
	if err != nil {
		return nil, err
	}

	codec, err := tokenizer.Get(enc)
	if err != nil {
		return nil, fmt.Errorf("initializing tokenizer for %s: %w", encoding, err)
	}

	return &Counter{
		codec:    codec,
		encoding: encoding,
	}, nil
}

// Count returns the number of tokens in the given text.
func (c *Counter) Count(text string) (int, error) {
	return c.codec.Count(text)
}

// Encoding returns the encoding name this counter was created with.
func (c *Counter) Encoding() string {
	return c.encoding
}

// CountBlocks annotates each block in the document with its token count
// and sets doc.TotalTokens to the sum. Any previously set token counts
// are overwritten.
//
// Returns the first error encountered during counting. On error, blocks
// processed before the error will have their Tokens set; later blocks
// and TotalTokens may be incomplete.
func CountBlocks(doc *markdown.Document, c *Counter) error {
	if doc == nil {
		return markdown.ErrNilDocument
	}
	total := 0
	for i := range doc.Blocks {
		n, err := c.Count(doc.Blocks[i].Content)
		if err != nil {
			return fmt.Errorf("counting tokens for block %d (%s): %w",
				i, doc.Blocks[i].Type, err)
		}
		doc.Blocks[i].Tokens = n
		total += n
	}
	doc.TotalTokens = total
	return nil
}

// resolveEncoding maps a string encoding name to the tiktoken-go constant.
func resolveEncoding(name string) (tokenizer.Encoding, error) {
	switch name {
	case Cl100kBase:
		return tokenizer.Cl100kBase, nil
	case O200kBase:
		return tokenizer.O200kBase, nil
	case P50kBase:
		return tokenizer.P50kBase, nil
	case R50kBase:
		return tokenizer.R50kBase, nil
	default:
		return "", fmt.Errorf("unsupported encoding: %q (supported: cl100k_base, o200k_base, p50k_base, r50k_base)", name)
	}
}
