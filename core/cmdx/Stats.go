package cmdx

import "github.com/securacore/codectx/core/tokenizer"

// Analyze computes compression statistics for a Markdown document.
// It encodes the input and measures the difference in size.
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

	// Parse the encoded CMDX to get dictionary info.
	doc, err := Parse(encoded)
	if err != nil {
		return nil, err
	}

	dictEntries := 0
	dictSavings := 0
	if doc.Dict != nil {
		dictEntries = len(doc.Dict.Entries)
		// Estimate dictionary savings: encode without dictionary to compare.
		noDictOpts := DefaultEncoderOptions()
		noDictOpts.MaxDictEntries = 0
		noDictEncoded, err := Encode(markdown, noDictOpts)
		if err == nil {
			dictSavings = len(noDictEncoded) - compressedBytes
		}
	}

	// Estimate domain block savings: encode without domain blocks to compare.
	domainSavings := 0
	noDomainOpts := DefaultEncoderOptions()
	noDomainOpts.EnableDomainBlocks = false
	noDomainEncoded, err := Encode(markdown, noDomainOpts)
	if err == nil {
		domainSavings = len(noDomainEncoded) - compressedBytes
	}

	// Real token counting via o200k_base (GPT-4o class).
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
		DictEntries:     dictEntries,
		DictSavings:     dictSavings,
		DomainSavings:   domainSavings,
		EstTokensBefore: estTokensBefore,
		EstTokensAfter:  estTokensAfter,
		TokenSavings:    tokenSavings,
	}, nil
}
