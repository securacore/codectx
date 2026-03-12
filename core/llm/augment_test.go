package llm

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
)

func TestAugment_Disabled(t *testing.T) {
	cfg := AugmentConfig{
		TaxonomyConfig: project.TaxonomyConfig{
			LLMAliasGeneration: false,
		},
	}

	result := Augment(context.Background(), cfg)

	if !result.Skipped {
		t.Fatal("expected Skipped=true")
	}
	if result.SkipReason != "llm_alias_generation disabled" {
		t.Errorf("unexpected skip reason: %q", result.SkipReason)
	}
}

func TestAugment_NoProvider(t *testing.T) {
	// Override LookPath to ensure no CLI is found.
	origLookPath := LookPathFunc
	LookPathFunc = func(_ string) (string, error) {
		return "", &lookPathError{}
	}
	defer func() { LookPathFunc = origLookPath }()

	cfg := AugmentConfig{
		Provider:     "", // auto-detect
		APIKey:       "", // no API key
		ClaudeBinary: "claude",
		TaxonomyConfig: project.TaxonomyConfig{
			LLMAliasGeneration: true,
		},
	}

	result := Augment(context.Background(), cfg)

	if !result.Skipped {
		t.Fatal("expected Skipped=true")
	}
	if result.SkipReason != "no LLM provider available" {
		t.Errorf("unexpected skip reason: %q", result.SkipReason)
	}
}

// lookPathError satisfies the error interface for testing LookPath failures.
type lookPathError struct{}

func (e *lookPathError) Error() string { return "not found" }

func TestAugment_BuildAliasRequests(t *testing.T) {
	tax := &taxonomy.Taxonomy{
		Terms: map[string]*taxonomy.Term{
			"authentication": {
				Canonical: "Authentication",
				Source:    "heading",
				Narrower:  []string{"jwt", "oauth"},
			},
			"jwt": {
				Canonical: "JWT",
				Source:    "code_identifier",
				Broader:   "authentication",
			},
		},
	}

	requests := buildAliasRequests(tax)
	if len(requests) != 2 {
		t.Fatalf("expected 2 alias requests, got %d", len(requests))
	}

	found := map[string]bool{}
	for _, req := range requests {
		found[req.Key] = true
		if req.Key == "authentication" {
			if req.Canonical != "Authentication" {
				t.Errorf("expected canonical 'Authentication', got %q", req.Canonical)
			}
			if len(req.Narrower) != 2 {
				t.Errorf("expected 2 narrower terms, got %d", len(req.Narrower))
			}
		}
	}
	if !found["authentication"] || !found["jwt"] {
		t.Error("expected both 'authentication' and 'jwt' in requests")
	}
}

func TestAugment_InstructionFallback(t *testing.T) {
	// Read from non-existent directory should fall back to embedded.
	instructions := readInstructions("/nonexistent/path", "taxonomy-generation", "defaults/taxonomy-generation.md")

	if instructions == "" {
		t.Fatal("expected non-empty instructions from embedded fallback")
	}
	if len(instructions) < 50 {
		t.Errorf("expected substantial instructions content, got %d bytes", len(instructions))
	}
}

func TestAugment_InstructionFromDisk(t *testing.T) {
	dir := t.TempDir()

	// Write a custom instruction file.
	topicDir := filepath.Join(dir, "taxonomy-generation")
	if err := os.MkdirAll(topicDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "README.md"), []byte("Custom instructions for testing"), 0644); err != nil {
		t.Fatal(err)
	}

	instructions := readInstructions(dir, "taxonomy-generation", "defaults/taxonomy-generation.md")

	if instructions != "Custom instructions for testing" {
		t.Errorf("expected custom instructions, got %q", instructions)
	}
}

func TestAugment_BuildBridgePairsFromChunks(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:a.01", Type: chunk.ChunkObject, Source: "docs/auth.md", Sequence: 1, Content: "Content 1"},
		{ID: "obj:a.02", Type: chunk.ChunkObject, Source: "docs/auth.md", Sequence: 2, Content: "Content 2"},
	}

	pairs := BuildBridgePairs(chunks)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].ChunkID != "obj:a.01" || pairs[0].NextChunkID != "obj:a.02" {
		t.Errorf("unexpected pair: %s -> %s", pairs[0].ChunkID, pairs[0].NextChunkID)
	}
}

func TestAugment_EmptyTaxonomy(t *testing.T) {
	// Even with a sender available, empty taxonomy means no alias requests.
	tax := &taxonomy.Taxonomy{
		Terms: map[string]*taxonomy.Term{},
	}

	requests := buildAliasRequests(tax)
	if len(requests) != 0 {
		t.Errorf("expected 0 alias requests for empty taxonomy, got %d", len(requests))
	}
}

func TestAugment_FullPath_WithMockSender(t *testing.T) {
	// Override LookPath so auto-detect finds "claude".
	origLookPath := LookPathFunc
	LookPathFunc = func(name string) (string, error) {
		if name == "claude" {
			return "/usr/bin/claude", nil
		}
		return "", &lookPathError{}
	}
	defer func() { LookPathFunc = origLookPath }()

	// Override ExecCommandFunc so the CLI sender returns mock data.
	origExec := ExecCommandFunc
	callCount := 0
	ExecCommandFunc = func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		callCount++
		// Check if this is an alias or bridge request by looking at prompt content.
		prompt := args[len(args)-1]
		if strings.Contains(prompt, "Canonical:") {
			// Alias request.
			return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":{"terms":[{"key":"authentication","aliases":["auth","login"]}]}}`)
		}
		// Bridge request.
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":{"bridges":[{"chunk_id":"obj:a.01","summary":"Defined auth flow"}]}}`)
	}
	defer func() { ExecCommandFunc = origExec }()

	tax := &taxonomy.Taxonomy{
		Terms: map[string]*taxonomy.Term{
			"authentication": {
				Canonical: "Authentication",
				Source:    "heading",
			},
		},
	}

	chunks := []chunk.Chunk{
		{ID: "obj:a.01", Type: chunk.ChunkObject, Source: "docs/auth.md", Sequence: 1, Content: "Auth content 1"},
		{ID: "obj:a.02", Type: chunk.ChunkObject, Source: "docs/auth.md", Sequence: 2, Content: "Auth content 2"},
	}

	cfg := AugmentConfig{
		Provider:     "cli",
		Model:        "claude-sonnet-4-20250514",
		ClaudeBinary: "claude",
		Taxonomy:     tax,
		Chunks:       chunks,
		TaxonomyConfig: project.TaxonomyConfig{
			LLMAliasGeneration: true,
			MaxAliasCount:      10,
		},
	}

	result := Augment(context.Background(), cfg)

	if result.Skipped {
		t.Fatalf("expected Skipped=false, got skip reason: %q", result.SkipReason)
	}

	// Verify aliases were generated.
	if result.AliasCount == 0 {
		t.Error("expected non-zero alias count")
	}
	if aliases, ok := result.Aliases["authentication"]; !ok {
		t.Error("expected aliases for 'authentication'")
	} else if len(aliases) < 1 {
		t.Error("expected at least 1 alias for 'authentication'")
	}

	// Verify bridges were generated.
	if result.BridgeCount == 0 {
		t.Error("expected non-zero bridge count")
	}
	if _, ok := result.Bridges["obj:a.01"]; !ok {
		t.Error("expected bridge for 'obj:a.01'")
	}

	if callCount < 2 {
		t.Errorf("expected at least 2 CLI calls (alias + bridge), got %d", callCount)
	}
}

func TestReadInstructions_EmptyDir(t *testing.T) {
	// Empty instructionsDir should fall back to embedded.
	instructions := readInstructions("", "taxonomy-generation", "defaults/taxonomy-generation.md")
	if instructions == "" {
		t.Fatal("expected non-empty instructions from embedded fallback when dir is empty")
	}
}

func TestReadInstructions_EmptyFileOnDisk(t *testing.T) {
	dir := t.TempDir()

	// Write an empty instruction file — should fall back to embedded.
	topicDir := filepath.Join(dir, "taxonomy-generation")
	if err := os.MkdirAll(topicDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topicDir, "README.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	instructions := readInstructions(dir, "taxonomy-generation", "defaults/taxonomy-generation.md")
	if instructions == "" {
		t.Fatal("expected non-empty instructions from embedded fallback when file is empty")
	}
	// Should not be the empty string — should be the embedded content.
	if len(instructions) < 50 {
		t.Errorf("expected substantial embedded fallback content, got %d bytes", len(instructions))
	}
}

func TestAugment_TaxonomyOnly_NoBridges(t *testing.T) {
	// Test Augment with taxonomy terms but no chunks.
	origLookPath := LookPathFunc
	LookPathFunc = func(name string) (string, error) {
		if name == "claude" {
			return "/usr/bin/claude", nil
		}
		return "", &lookPathError{}
	}
	defer func() { LookPathFunc = origLookPath }()

	origExec := ExecCommandFunc
	ExecCommandFunc = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{"type":"result","is_error":false,"structured_output":{"terms":[{"key":"auth","aliases":["authentication"]}]}}`)
	}
	defer func() { ExecCommandFunc = origExec }()

	cfg := AugmentConfig{
		Provider:     "cli",
		Model:        "sonnet",
		ClaudeBinary: "claude",
		Taxonomy: &taxonomy.Taxonomy{
			Terms: map[string]*taxonomy.Term{
				"auth": {Canonical: "Auth", Source: "heading"},
			},
		},
		Chunks: nil, // No chunks
		TaxonomyConfig: project.TaxonomyConfig{
			LLMAliasGeneration: true,
			MaxAliasCount:      5,
		},
	}

	result := Augment(context.Background(), cfg)

	if result.Skipped {
		t.Fatalf("expected not skipped, got reason: %q", result.SkipReason)
	}
	if result.AliasCount == 0 {
		t.Error("expected aliases to be generated")
	}
	if result.BridgeCount != 0 {
		t.Errorf("expected 0 bridges with no chunks, got %d", result.BridgeCount)
	}
}

func TestAugment_NoSenderAvailable(t *testing.T) {
	cfg := AugmentConfig{
		Provider: "api",
		APIKey:   "", // Empty API key means no sender available.
		TaxonomyConfig: project.TaxonomyConfig{
			LLMAliasGeneration: true,
		},
	}

	result := Augment(context.Background(), cfg)

	if !result.Skipped {
		t.Fatal("expected Skipped=true when no sender available")
	}
	if result.SkipReason != "no LLM provider available" {
		t.Errorf("unexpected skip reason: %q", result.SkipReason)
	}
}
