// Package tokenizer provides real token counting using the o200k_base
// encoding (GPT-4o class). It replaces the previous rough heuristic of
// bytes / 4 with actual BPE tokenization for accurate context-window
// estimation. See docs/product/compilation.md for rationale and
// docs/foundation/ai-authoring/ for the baseline model assumption.
package tokenizer

import (
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

var (
	enc     *tiktoken.Tiktoken
	encOnce sync.Once
	encErr  error
)

// encoder returns the lazily-initialized o200k_base tokenizer.
// The BPE rank data is loaded from an embedded offline file — no
// network access is needed at runtime.
func encoder() (*tiktoken.Tiktoken, error) {
	encOnce.Do(func() {
		tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
		enc, encErr = tiktoken.GetEncoding("o200k_base")
	})
	return enc, encErr
}

// CountTokens returns the number of o200k_base tokens in s.
// If the encoder fails to initialize it falls back to len(s)/4.
func CountTokens(s string) int {
	e, err := encoder()
	if err != nil {
		return len(s) / 4
	}
	return len(e.Encode(s, nil, nil))
}

// CountTokensBytes is a convenience wrapper around CountTokens.
func CountTokensBytes(b []byte) int {
	return CountTokens(string(b))
}
