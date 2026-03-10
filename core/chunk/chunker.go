package chunk

import (
	"errors"
	"fmt"

	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/project"
)

// Options configures the chunking algorithm parameters.
type Options struct {
	// TargetTokens is the target chunk size in tokens.
	TargetTokens int

	// MinTokens is the minimum chunk size to avoid tiny fragments.
	MinTokens int

	// MaxTokens is the maximum chunk size (hard ceiling for non-atomic blocks).
	MaxTokens int

	// FlexibilityWindow is the fraction of TargetTokens at which to break
	// if the next block would exceed the target. E.g., 0.8 means break
	// after reaching 80% of target.
	FlexibilityWindow float64

	// HashLength is the number of hex characters from SHA-256 for chunk IDs.
	HashLength int
}

// DefaultOptions returns chunking options derived from the default
// preferences configuration. This ensures a single source of truth
// for default chunking parameters.
func DefaultOptions() Options {
	return OptionsFromConfig(project.DefaultPreferencesConfig().Chunking)
}

// OptionsFromConfig converts a project.ChunkingConfig to Options.
func OptionsFromConfig(cfg project.ChunkingConfig) Options {
	return Options{
		TargetTokens:      cfg.TargetTokens,
		MinTokens:         cfg.MinTokens,
		MaxTokens:         cfg.MaxTokens,
		FlexibilityWindow: cfg.FlexibilityWindow,
		HashLength:        project.ClampHashLength(cfg.HashLength),
	}
}

// ChunkDocument splits a token-annotated Document into chunks using semantic
// block accumulation with token-based windows.
//
// The document should have been processed by tokens.CountBlocks() first
// so that each block has a meaningful Tokens value. Blocks with zero tokens
// are accumulated normally (they don't affect token-based decisions).
//
// sourcePath is the relative path of the source file (used for metadata).
// chunkType determines the ID prefix and output routing.
func ChunkDocument(doc *markdown.Document, sourcePath string, chunkType ChunkType, opts Options) ([]Chunk, error) {
	if doc == nil {
		return nil, errors.New("document is nil")
	}
	if len(doc.Blocks) == 0 {
		return nil, nil
	}
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid chunking options: %w", err)
	}

	flexThreshold := int(float64(opts.TargetTokens) * opts.FlexibilityWindow)

	var chunks []Chunk
	var currentBlocks []markdown.Block
	currentTokens := 0

	for _, block := range doc.Blocks {
		// If this single block exceeds max_tokens on its own, it becomes
		// its own oversized chunk.
		if block.Tokens > opts.MaxTokens {
			// Flush current accumulator first.
			if len(currentBlocks) > 0 {
				chunks = append(chunks, buildChunk(currentBlocks, currentTokens, sourcePath, chunkType, opts, false))
				currentBlocks = nil
				currentTokens = 0
			}
			chunks = append(chunks, buildChunk([]markdown.Block{block}, block.Tokens, sourcePath, chunkType, opts, true))
			continue
		}

		wouldExceed := currentTokens+block.Tokens > opts.TargetTokens

		if wouldExceed && len(currentBlocks) > 0 {
			// (a) Heading → always break before it.
			if block.Type == markdown.BlockHeading {
				chunks = append(chunks, buildChunk(currentBlocks, currentTokens, sourcePath, chunkType, opts, false))
				currentBlocks = nil
				currentTokens = 0
			} else if currentTokens >= flexThreshold {
				// (b) Current chunk >= flexibility_window of target → break.
				chunks = append(chunks, buildChunk(currentBlocks, currentTokens, sourcePath, chunkType, opts, false))
				currentBlocks = nil
				currentTokens = 0
			}
			// (c) Current < flexibility_window → include the block (go over target).
		} else if block.Type == markdown.BlockHeading && len(currentBlocks) > 0 {
			// Even if we wouldn't exceed, headings should start new chunks
			// when there's existing content.
			chunks = append(chunks, buildChunk(currentBlocks, currentTokens, sourcePath, chunkType, opts, false))
			currentBlocks = nil
			currentTokens = 0
		}

		currentBlocks = append(currentBlocks, block)
		currentTokens += block.Tokens
	}

	// Flush remaining blocks.
	if len(currentBlocks) > 0 {
		chunks = append(chunks, buildChunk(currentBlocks, currentTokens, sourcePath, chunkType, opts, false))
	}

	// Rebalance: if the last chunk is below MinTokens, rebalance with previous.
	chunks = rebalanceTail(chunks, opts)

	// Assign sequence numbers and TotalInFile.
	for i := range chunks {
		chunks[i].Sequence = i + 1
		chunks[i].TotalInFile = len(chunks)
	}

	// Compute content, hash, and ID for each chunk.
	for i := range chunks {
		chunks[i].Content = JoinContent(chunks[i].Blocks)
		hash := ContentHash(chunks[i].Content, opts.HashLength)
		chunks[i].ID = FormatID(chunkType, hash, chunks[i].Sequence)
	}

	return chunks, nil
}

// buildChunk creates a Chunk from accumulated blocks.
func buildChunk(blocks []markdown.Block, tokens int, source string, ct ChunkType, opts Options, oversized bool) Chunk {
	// The heading for the chunk is taken from the first block's heading hierarchy.
	var heading string
	if len(blocks) > 0 {
		heading = FormatHeading(blocks[0].Heading)
	}

	return Chunk{
		Type:      ct,
		Source:    source,
		Heading:   heading,
		Tokens:    tokens,
		Blocks:    append([]markdown.Block(nil), blocks...),
		Oversized: oversized,
	}
}

// rebalanceTail handles the min_tokens constraint for the last chunk.
//
// If the last chunk is below MinTokens and there's a previous chunk:
//   - If merging both chunks would stay within TargetTokens, merge them.
//   - Otherwise, pool all blocks from the last two chunks and re-split
//     them as evenly as possible. This prevents BM25 length normalization
//     from penalizing one large chunk while creating a tiny orphan.
func rebalanceTail(chunks []Chunk, opts Options) []Chunk {
	if len(chunks) < 2 {
		return chunks
	}

	last := &chunks[len(chunks)-1]
	if last.Tokens >= opts.MinTokens || last.Oversized {
		return chunks
	}

	// Don't merge across heading boundaries — if the last chunk starts
	// with a heading, it's a semantic boundary that should be preserved.
	if len(last.Blocks) > 0 && last.Blocks[0].Type == markdown.BlockHeading {
		return chunks
	}

	prev := &chunks[len(chunks)-2]
	if prev.Oversized {
		return chunks
	}

	combinedTokens := prev.Tokens + last.Tokens

	// If combined fits within target, just merge.
	if combinedTokens <= opts.TargetTokens {
		prev.Blocks = append(prev.Blocks, last.Blocks...)
		prev.Tokens = combinedTokens
		return chunks[:len(chunks)-1]
	}

	// Rebalance: pool blocks and split as evenly as possible.
	pooled := make([]markdown.Block, 0, len(prev.Blocks)+len(last.Blocks))
	pooled = append(pooled, prev.Blocks...)
	pooled = append(pooled, last.Blocks...)

	targetPerChunk := combinedTokens / 2
	splitIdx := findBalancedSplit(pooled, targetPerChunk)

	if splitIdx <= 0 || splitIdx >= len(pooled) {
		// Can't split meaningfully — just merge.
		prev.Blocks = pooled
		prev.Tokens = combinedTokens
		return chunks[:len(chunks)-1]
	}

	// Rebuild the two chunks.
	firstBlocks := pooled[:splitIdx]
	secondBlocks := pooled[splitIdx:]

	firstTokens := sumTokens(firstBlocks)
	secondTokens := sumTokens(secondBlocks)

	prev.Blocks = firstBlocks
	prev.Tokens = firstTokens
	prev.Heading = FormatHeading(firstBlocks[0].Heading)

	last.Blocks = secondBlocks
	last.Tokens = secondTokens
	last.Heading = FormatHeading(secondBlocks[0].Heading)

	return chunks
}

// findBalancedSplit finds the split index that produces two groups
// closest to the target token count for the first group.
func findBalancedSplit(blocks []markdown.Block, target int) int {
	if len(blocks) <= 1 {
		return 0
	}

	bestIdx := 1
	bestDiff := abs(sumTokens(blocks[:1]) - target)
	running := blocks[0].Tokens

	for i := 1; i < len(blocks); i++ {
		diff := abs(running - target)
		if diff < bestDiff {
			bestDiff = diff
			bestIdx = i
		}
		running += blocks[i].Tokens
	}

	// Also check after adding the last block.
	diff := abs(running - target)
	if diff < bestDiff {
		bestIdx = len(blocks)
	}

	return bestIdx
}

// sumTokens returns the total token count across blocks.
func sumTokens(blocks []markdown.Block) int {
	total := 0
	for _, b := range blocks {
		total += b.Tokens
	}
	return total
}

// abs returns the absolute value of an integer.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// validateOptions checks that chunking options are internally consistent.
func validateOptions(opts Options) error {
	if opts.TargetTokens <= 0 {
		return errors.New("target_tokens must be positive")
	}
	if opts.MaxTokens <= 0 {
		return errors.New("max_tokens must be positive")
	}
	if opts.MinTokens < 0 {
		return errors.New("min_tokens must be non-negative")
	}
	if opts.FlexibilityWindow < 0 || opts.FlexibilityWindow > 1 {
		return errors.New("flexibility_window must be between 0 and 1")
	}
	return nil
}

// CheckCollisions verifies that no two chunks have the same ID with different
// content. Returns an error describing the collision if found.
func CheckCollisions(chunks []Chunk) error {
	seen := make(map[string]string, len(chunks)) // ID → content hash (full)
	for _, c := range chunks {
		full := ContentHash(c.Content, project.MaxHashLength)
		if prev, ok := seen[c.ID]; ok {
			if prev != full {
				return fmt.Errorf("hash collision detected: chunk ID %q maps to different content", c.ID)
			}
		}
		seen[c.ID] = full
	}
	return nil
}
