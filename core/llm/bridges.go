package llm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/securacore/codectx/core/chunk"
)

// defaultBridgeBatchSize is the number of chunk pairs per API call.
// Lower than alias batch size because each pair includes content excerpts.
const defaultBridgeBatchSize = 20

// bridgeContentMaxLen is the maximum character length for content excerpts
// in bridge prompts. Content is truncated at word boundaries.
const bridgeContentMaxLen = 500

// BridgePair represents a pair of adjacent chunks needing a bridge summary.
type BridgePair struct {
	// ChunkID is the ID of the "from" chunk.
	ChunkID string

	// NextChunkID is the ID of the "to" chunk.
	NextChunkID string

	// Source is the source file path.
	Source string

	// Heading is the heading breadcrumb of the "from" chunk.
	Heading string

	// NextHeading is the heading breadcrumb of the "to" chunk.
	NextHeading string

	// Content is the truncated content of the "from" chunk.
	Content string

	// NextContent is the truncated content of the "to" chunk.
	NextContent string
}

// BridgeResult holds the output of batched bridge generation.
type BridgeResult struct {
	// Bridges maps chunk IDs to their generated bridge summary text.
	Bridges map[string]string

	// Errors is the number of batch requests that failed (non-fatal).
	Errors int
}

// GenerateBridges sends batched bridge summary requests to the LLM.
//
// Each batch contains up to batchSize pairs (default 20 if <= 0).
// The instructions parameter is the content of the bridge-summaries
// README.md file, used as the system prompt.
//
// Returns partial results if some batches fail (graceful degradation).
func GenerateBridges(ctx context.Context, sender Sender, pairs []*BridgePair, instructions string, batchSize int) *BridgeResult {
	if batchSize <= 0 {
		batchSize = defaultBridgeBatchSize
	}

	result := &BridgeResult{
		Bridges: make(map[string]string),
	}

	if len(pairs) == 0 {
		return result
	}

	for i := 0; i < len(pairs); i += batchSize {
		end := i + batchSize
		if end > len(pairs) {
			end = len(pairs)
		}
		batch := pairs[i:end]

		prompt := buildBridgeBatchPrompt(batch)

		resp, err := sender.SendBridges(ctx, instructions, prompt)
		if err != nil {
			result.Errors++
			continue
		}

		// Build a set of valid chunk IDs for this batch.
		validIDs := make(map[string]bool, len(batch))
		for _, p := range batch {
			validIDs[p.ChunkID] = true
		}

		for _, bridge := range resp.Bridges {
			if !validIDs[bridge.ChunkID] {
				continue // Ignore unexpected IDs from the LLM.
			}
			if bridge.Summary != "" {
				result.Bridges[bridge.ChunkID] = bridge.Summary
			}
		}
	}

	return result
}

// buildBridgeBatchPrompt constructs the user message for a batch of chunk pairs.
func buildBridgeBatchPrompt(batch []*BridgePair) string {
	var b strings.Builder

	b.WriteString("Generate a one-line bridge summary for each chunk boundary below. " +
		"Each bridge should summarize what the previous chunk established that the " +
		"next chunk assumes the reader already knows.\n\n")
	b.WriteString("Boundaries:\n")

	for _, p := range batch {
		b.WriteString("\n---\n")
		fmt.Fprintf(&b, "From: %s (%s)\n", p.ChunkID, p.Heading)
		fmt.Fprintf(&b, "Content: %s\n", p.Content)
		fmt.Fprintf(&b, "To: %s (%s)\n", p.NextChunkID, p.NextHeading)
		fmt.Fprintf(&b, "Content: %s\n", p.NextContent)
	}

	return b.String()
}

// BuildBridgePairs constructs BridgePair list from chunks.
//
// For each pair of adjacent non-spec chunks from the same source file,
// a BridgePair is created. Spec chunks are excluded because they use
// parent_object linking instead of adjacency.
//
// Content is truncated to bridgeContentMaxLen characters at word boundaries.
func BuildBridgePairs(chunks []chunk.Chunk) []*BridgePair {
	// Group non-spec chunks by source file.
	bySource := make(map[string][]chunk.Chunk)
	for _, c := range chunks {
		if c.Type == chunk.ChunkSpec {
			continue
		}
		bySource[c.Source] = append(bySource[c.Source], c)
	}

	var pairs []*BridgePair

	for _, group := range bySource {
		// Sort by sequence within each file.
		sort.Slice(group, func(i, j int) bool {
			return group[i].Sequence < group[j].Sequence
		})

		// Create pairs from consecutive chunks.
		for i := 0; i < len(group)-1; i++ {
			curr := &group[i]
			next := &group[i+1]

			pairs = append(pairs, &BridgePair{
				ChunkID:     curr.ID,
				NextChunkID: next.ID,
				Source:      curr.Source,
				Heading:     curr.Heading,
				NextHeading: next.Heading,
				Content:     truncateContent(curr.Content, bridgeContentMaxLen),
				NextContent: truncateContent(next.Content, bridgeContentMaxLen),
			})
		}
	}

	return pairs
}

// truncateContent returns content truncated to approximately maxLen characters,
// breaking at word boundaries. If the content is shorter than maxLen, it is
// returned unchanged.
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}

	// Find the last space before maxLen.
	truncated := content[:maxLen]
	lastSpace := strings.LastIndexByte(truncated, ' ')
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
