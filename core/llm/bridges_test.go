package llm

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
)

func TestBuildBridgeBatchPrompt(t *testing.T) {
	pairs := []*bridgePair{
		{
			ChunkID:     "obj:a1b2c3.01",
			NextChunkID: "obj:a1b2c3.02",
			Source:      "docs/topics/auth.md",
			Heading:     "Authentication > JWT Tokens",
			NextHeading: "Authentication > JWT Tokens > Validation",
			Content:     "JWT tokens use RS256 signing...",
			NextContent: "Validation requires checking...",
		},
		{
			ChunkID:     "obj:a1b2c3.02",
			NextChunkID: "obj:a1b2c3.03",
			Source:      "docs/topics/auth.md",
			Heading:     "Authentication > JWT Tokens > Validation",
			NextHeading: "Authentication > JWT Tokens > Refresh",
			Content:     "Signature verification ensures...",
			NextContent: "Refresh tokens expire after...",
		},
	}

	prompt := buildBridgeBatchPrompt(pairs)

	if !strings.Contains(prompt, "From: obj:a1b2c3.01") {
		t.Error("expected first chunk ID in prompt")
	}
	if !strings.Contains(prompt, "To: obj:a1b2c3.02") {
		t.Error("expected second chunk ID in prompt")
	}
	if !strings.Contains(prompt, "Authentication > JWT Tokens") {
		t.Error("expected heading in prompt")
	}
	if !strings.Contains(prompt, "JWT tokens use RS256") {
		t.Error("expected content excerpt in prompt")
	}
}

func TestGenerateBridges_MockSender(t *testing.T) {
	sender := &mockSender{
		bridgeResponses: []*BridgeResponse{
			{
				Bridges: []BridgeEntryResponse{
					{ChunkID: "obj:a1b2c3.01", Summary: "Established JWT token structure and signing requirements"},
					{ChunkID: "obj:a1b2c3.02", Summary: "Covered validation rules and signature verification"},
				},
			},
		},
	}

	pairs := []*bridgePair{
		{ChunkID: "obj:a1b2c3.01", NextChunkID: "obj:a1b2c3.02"},
		{ChunkID: "obj:a1b2c3.02", NextChunkID: "obj:a1b2c3.03"},
	}

	result := generateBridges(context.Background(), bridgeGenConfig{
		sender: sender, pairs: pairs, instructions: "instructions", batchSize: 20,
	})

	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}
	if len(result.Bridges) != 2 {
		t.Errorf("expected 2 bridges, got %d", len(result.Bridges))
	}
	if result.Bridges["obj:a1b2c3.01"] == "" {
		t.Error("expected bridge for obj:a1b2c3.01")
	}
}

func TestGenerateBridges_EmptyPairs(t *testing.T) {
	sender := &mockSender{}
	result := generateBridges(context.Background(), bridgeGenConfig{
		sender: sender, instructions: "instructions", batchSize: 20,
	})

	if len(result.Bridges) != 0 {
		t.Errorf("expected 0 bridges, got %d", len(result.Bridges))
	}
	if sender.bridgeCalls != 0 {
		t.Errorf("expected 0 sender calls, got %d", sender.bridgeCalls)
	}
}

func TestGenerateBridges_UnknownIDFiltered(t *testing.T) {
	sender := &mockSender{
		bridgeResponses: []*BridgeResponse{
			{
				Bridges: []BridgeEntryResponse{
					{ChunkID: "obj:valid.01", Summary: "Valid bridge"},
					{ChunkID: "obj:unknown.99", Summary: "Should be ignored"},
				},
			},
		},
	}

	pairs := []*bridgePair{
		{ChunkID: "obj:valid.01", NextChunkID: "obj:valid.02"},
	}

	result := generateBridges(context.Background(), bridgeGenConfig{
		sender: sender, pairs: pairs, instructions: "instructions", batchSize: 20,
	})

	if _, ok := result.Bridges["obj:unknown.99"]; ok {
		t.Error("expected unknown ID to be filtered out")
	}
	if len(result.Bridges) != 1 {
		t.Errorf("expected 1 bridge, got %d", len(result.Bridges))
	}
}

func TestGenerateBridges_BatchError(t *testing.T) {
	sender := &mockSender{
		bridgeResponses: []*BridgeResponse{
			{Bridges: []BridgeEntryResponse{{ChunkID: "a", Summary: "bridge a"}}},
			nil, // error batch
			{Bridges: []BridgeEntryResponse{{ChunkID: "c", Summary: "bridge c"}}},
		},
		bridgeErrors: []error{
			nil,
			fmt.Errorf("API error"),
			nil,
		},
	}

	pairs := []*bridgePair{
		{ChunkID: "a"},
		{ChunkID: "b"},
		{ChunkID: "c"},
	}

	result := generateBridges(context.Background(), bridgeGenConfig{
		sender: sender, pairs: pairs, instructions: "instructions", batchSize: 1,
	})

	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
	if len(result.Bridges) != 2 {
		t.Errorf("expected 2 bridges (a + c), got %d", len(result.Bridges))
	}
}

func TestBuildBridgePairs(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:abc.01", Type: chunk.ChunkObject, Source: "docs/auth.md", Sequence: 1, Content: "First chunk content"},
		{ID: "obj:abc.02", Type: chunk.ChunkObject, Source: "docs/auth.md", Sequence: 2, Content: "Second chunk content"},
		{ID: "obj:abc.03", Type: chunk.ChunkObject, Source: "docs/auth.md", Sequence: 3, Content: "Third chunk content"},
		{ID: "spec:def.01", Type: chunk.ChunkSpec, Source: "docs/auth.spec.md", Sequence: 1, Content: "Spec content"},
		{ID: "sys:ghi.01", Type: chunk.ChunkSystem, Source: "system/topics/tax.md", Sequence: 1, Content: "System 1"},
		{ID: "sys:ghi.02", Type: chunk.ChunkSystem, Source: "system/topics/tax.md", Sequence: 2, Content: "System 2"},
	}

	pairs := buildBridgePairs(chunks)

	// 3 object chunks in auth.md -> 2 pairs.
	// 2 system chunks in tax.md -> 1 pair.
	// 1 spec chunk -> 0 pairs (specs excluded).
	// Total: 3 pairs.
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	// Verify a pair from auth.md.
	found := false
	for _, p := range pairs {
		if p.ChunkID == "obj:abc.01" && p.NextChunkID == "obj:abc.02" {
			found = true
			if p.Source != "docs/auth.md" {
				t.Errorf("expected source 'docs/auth.md', got %q", p.Source)
			}
		}
	}
	if !found {
		t.Error("expected pair obj:abc.01 -> obj:abc.02")
	}
}

func TestBuildBridgePairs_ExcludesSpecs(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "spec:a.01", Type: chunk.ChunkSpec, Source: "docs/auth.spec.md", Sequence: 1, Content: "Spec 1"},
		{ID: "spec:a.02", Type: chunk.ChunkSpec, Source: "docs/auth.spec.md", Sequence: 2, Content: "Spec 2"},
	}

	pairs := buildBridgePairs(chunks)
	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs for spec-only chunks, got %d", len(pairs))
	}
}

func TestTruncateContent_Long(t *testing.T) {
	// Create a string longer than 500 chars.
	words := make([]string, 200)
	for i := range words {
		words[i] = "word"
	}
	content := strings.Join(words, " ")

	truncated := truncateContent(content, 500)

	if len(truncated) > 510 { // Allow for "..." suffix.
		t.Errorf("expected truncated content under ~510 chars, got %d", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Error("expected '...' suffix on truncated content")
	}
}

func TestTruncateContent_Short(t *testing.T) {
	content := "Short content that fits."
	truncated := truncateContent(content, 500)

	if truncated != content {
		t.Errorf("expected unchanged content, got %q", truncated)
	}
}

func TestTruncateContent_ExactLength(t *testing.T) {
	content := strings.Repeat("a", 500)
	truncated := truncateContent(content, 500)

	if truncated != content {
		t.Error("expected unchanged content at exact max length")
	}
}
