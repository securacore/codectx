package md

import "github.com/securacore/codectx/core/tokenizer"

// Analyze computes compression statistics for a Markdown document.
func Analyze(markdown []byte) (*Stats, error) {
	encoded, err := Encode(markdown)
	if err != nil {
		return nil, err
	}

	originalBytes := len(markdown)
	compressedBytes := len(encoded)

	var byteSavings float64
	if originalBytes > 0 {
		byteSavings = float64(originalBytes-compressedBytes) / float64(originalBytes) * 100
	}

	// Token counting via o200k_base (GPT-4o class).
	estTokensBefore := tokenizer.CountTokensBytes(markdown)
	estTokensAfter := tokenizer.CountTokensBytes(encoded)

	var tokenSavings float64
	if estTokensBefore > 0 {
		tokenSavings = float64(estTokensBefore-estTokensAfter) / float64(estTokensBefore) * 100
	}

	return &Stats{
		OriginalBytes:   originalBytes,
		CompressedBytes: compressedBytes,
		ByteSavings:     byteSavings,
		EstTokensBefore: estTokensBefore,
		EstTokensAfter:  estTokensAfter,
		TokenSavings:    tokenSavings,
	}, nil
}
