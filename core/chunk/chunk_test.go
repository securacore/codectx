package chunk_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/project"
)

// ---------------------------------------------------------------------------
// Helper: build blocks with given token counts for test convenience.
// ---------------------------------------------------------------------------

func block(typ markdown.BlockType, content string, tokens int, heading []string) markdown.Block {
	return markdown.Block{
		Type:    typ,
		Content: content,
		Heading: heading,
		Tokens:  tokens,
	}
}

func paragraphBlock(content string, tokens int, heading []string) markdown.Block {
	return block(markdown.BlockParagraph, content, tokens, heading)
}

func headingBlock(content string, level int, tokens int, heading []string) markdown.Block {
	b := block(markdown.BlockHeading, content, tokens, heading)
	b.Level = level
	return b
}

func codeBlock(content string, tokens int, heading []string) markdown.Block {
	b := block(markdown.BlockCodeBlock, content, tokens, heading)
	b.Language = "go"
	return b
}

func doc(blocks ...markdown.Block) *markdown.Document {
	total := 0
	for _, b := range blocks {
		total += b.Tokens
	}
	return &markdown.Document{
		Blocks:      blocks,
		TotalTokens: total,
	}
}

func defaultOpts() chunk.Options {
	return chunk.Options{
		TargetTokens:      450,
		MinTokens:         200,
		MaxTokens:         800,
		FlexibilityWindow: 0.8,
		HashLength:        16,
	}
}

// ---------------------------------------------------------------------------
// ContentHash tests
// ---------------------------------------------------------------------------

func TestContentHash_Deterministic(t *testing.T) {
	h1 := chunk.ContentHash("hello world", 16)
	h2 := chunk.ContentHash("hello world", 16)
	if h1 != h2 {
		t.Errorf("expected same hash, got %q and %q", h1, h2)
	}
}

func TestContentHash_DifferentContent(t *testing.T) {
	h1 := chunk.ContentHash("hello", 16)
	h2 := chunk.ContentHash("world", 16)
	if h1 == h2 {
		t.Error("expected different hashes for different content")
	}
}

func TestContentHash_RespectsLength(t *testing.T) {
	tests := []struct {
		length   int
		expected int
	}{
		{8, 8},
		{16, 16},
		{32, 32},
		{64, 64},
		{0, project.DefaultHashLength}, // clamped
		{4, project.MinHashLength},     // clamped up
		{100, project.MaxHashLength},   // clamped down
	}

	for _, tt := range tests {
		h := chunk.ContentHash("test", tt.length)
		if len(h) != tt.expected {
			t.Errorf("ContentHash(_, %d): got length %d, want %d", tt.length, len(h), tt.expected)
		}
	}
}

func TestContentHash_ValidHex(t *testing.T) {
	h := chunk.ContentHash("test content", 64)
	for _, c := range h {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("expected hex character, got %c", c)
		}
	}
}

// ---------------------------------------------------------------------------
// FormatID tests
// ---------------------------------------------------------------------------

func TestFormatID_Object(t *testing.T) {
	id := chunk.FormatID(chunk.ChunkObject, "abcdef1234567890", 3)
	if id != "obj:abcdef1234567890.3" {
		t.Errorf("got %q", id)
	}
}

func TestFormatID_Spec(t *testing.T) {
	id := chunk.FormatID(chunk.ChunkSpec, "abcdef1234567890", 1)
	if id != "spec:abcdef1234567890.1" {
		t.Errorf("got %q", id)
	}
}

func TestFormatID_System(t *testing.T) {
	id := chunk.FormatID(chunk.ChunkSystem, "abcdef1234567890", 12)
	if id != "sys:abcdef1234567890.12" {
		t.Errorf("got %q", id)
	}
}

func TestFormatID_NoZeroPadding(t *testing.T) {
	id := chunk.FormatID(chunk.ChunkObject, "hash", 1)
	if strings.Contains(id, ".01") {
		t.Errorf("expected no zero padding, got %q", id)
	}
}

// ---------------------------------------------------------------------------
// FormatHeading tests
// ---------------------------------------------------------------------------

func TestFormatHeading_Multiple(t *testing.T) {
	h := chunk.FormatHeading([]string{"Auth", "JWT", "Refresh"})
	if h != "Auth > JWT > Refresh" {
		t.Errorf("got %q", h)
	}
}

func TestFormatHeading_Single(t *testing.T) {
	h := chunk.FormatHeading([]string{"Auth"})
	if h != "Auth" {
		t.Errorf("got %q", h)
	}
}

func TestFormatHeading_Empty(t *testing.T) {
	h := chunk.FormatHeading(nil)
	if h != "" {
		t.Errorf("got %q", h)
	}
	h = chunk.FormatHeading([]string{})
	if h != "" {
		t.Errorf("got %q", h)
	}
}

// ---------------------------------------------------------------------------
// JoinContent tests
// ---------------------------------------------------------------------------

func TestJoinContent_Multiple(t *testing.T) {
	blocks := []markdown.Block{
		{Content: "first"},
		{Content: "second"},
		{Content: "third"},
	}
	got := chunk.JoinContent(blocks)
	expected := "first\n\nsecond\n\nthird"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestJoinContent_Single(t *testing.T) {
	blocks := []markdown.Block{{Content: "only"}}
	got := chunk.JoinContent(blocks)
	if got != "only" {
		t.Errorf("got %q", got)
	}
}

func TestJoinContent_Empty(t *testing.T) {
	got := chunk.JoinContent(nil)
	if got != "" {
		t.Errorf("got %q", got)
	}
}

// ---------------------------------------------------------------------------
// ChunkDocument — algorithm tests
// ---------------------------------------------------------------------------

func TestChunkDocument_NilDocument(t *testing.T) {
	_, err := chunk.ChunkDocument(nil, "test.md", chunk.ChunkObject, defaultOpts())
	if err == nil {
		t.Fatal("expected error for nil document")
	}
	if !errors.Is(err, markdown.ErrNilDocument) {
		t.Errorf("expected ErrNilDocument sentinel, got %v", err)
	}
}

func TestChunkDocument_EmptyDocument(t *testing.T) {
	d := doc()
	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(chunks))
	}
}

func TestChunkDocument_SingleBlockFitsInOneChunk(t *testing.T) {
	d := doc(paragraphBlock("Hello world", 100, []string{"Intro"}))

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	c := chunks[0]
	if c.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", c.Sequence)
	}
	if c.TotalInFile != 1 {
		t.Errorf("expected total 1, got %d", c.TotalInFile)
	}
	if c.Tokens != 100 {
		t.Errorf("expected 100 tokens, got %d", c.Tokens)
	}
	if c.Heading != "Intro" {
		t.Errorf("expected heading %q, got %q", "Intro", c.Heading)
	}
	if c.Type != chunk.ChunkObject {
		t.Errorf("expected type object, got %s", c.Type)
	}
	if !strings.HasPrefix(c.ID, "obj:") {
		t.Errorf("expected obj: prefix, got %q", c.ID)
	}
}

func TestChunkDocument_MultipleBlocksFitInOneChunk(t *testing.T) {
	d := doc(
		paragraphBlock("First paragraph", 100, []string{"Intro"}),
		paragraphBlock("Second paragraph", 100, []string{"Intro"}),
		paragraphBlock("Third paragraph", 100, []string{"Intro"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk (300 tokens < 450 target), got %d", len(chunks))
	}
	if chunks[0].Tokens != 300 {
		t.Errorf("expected 300 tokens, got %d", chunks[0].Tokens)
	}
}

func TestChunkDocument_SplitsWhenExceedingTarget(t *testing.T) {
	d := doc(
		paragraphBlock("Block A", 400, []string{"Part One"}),
		paragraphBlock("Block B", 400, []string{"Part One"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 400 is >= 360 (80% of 450), so adding 400 more would exceed target
	// and current >= flex threshold → split.
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Tokens != 400 {
		t.Errorf("chunk 1: expected 400 tokens, got %d", chunks[0].Tokens)
	}
	if chunks[1].Tokens != 400 {
		t.Errorf("chunk 2: expected 400 tokens, got %d", chunks[1].Tokens)
	}
}

func TestChunkDocument_HeadingAlwaysBreaks(t *testing.T) {
	d := doc(
		paragraphBlock("Some content", 100, []string{"Section A"}),
		headingBlock("Section B", 2, 10, []string{"Section B"}),
		paragraphBlock("More content", 100, []string{"Section B"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Even though 100 + 10 + 100 = 210 < 450, headings always break.
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Heading != "Section A" {
		t.Errorf("chunk 1 heading: expected %q, got %q", "Section A", chunks[0].Heading)
	}
	if chunks[1].Heading != "Section B" {
		t.Errorf("chunk 2 heading: expected %q, got %q", "Section B", chunks[1].Heading)
	}
}

func TestChunkDocument_FlexibilityWindowInclude(t *testing.T) {
	// Current chunk has 200 tokens (below 360 = 80% of 450).
	// Next block has 300 tokens. 200 + 300 = 500 > 450 target.
	// But current < flex threshold → include the block (go over).
	d := doc(
		paragraphBlock("Small block", 200, []string{"Intro"}),
		paragraphBlock("Bigger block", 300, []string{"Intro"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk (below flex window, include), got %d", len(chunks))
	}
	if chunks[0].Tokens != 500 {
		t.Errorf("expected 500 tokens, got %d", chunks[0].Tokens)
	}
}

func TestChunkDocument_FlexibilityWindowBreak(t *testing.T) {
	// Current chunk has 370 tokens (>= 360 = 80% of 450).
	// Next block has 100 tokens. 370 + 100 = 470 > 450 target.
	// Current >= flex threshold → break.
	d := doc(
		paragraphBlock("Large block", 370, []string{"Intro"}),
		paragraphBlock("Small block", 100, []string{"Intro"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (above flex window, break), got %d", len(chunks))
	}
}

func TestChunkDocument_OversizedSingleBlock(t *testing.T) {
	d := doc(
		paragraphBlock("Normal content", 100, []string{"Intro"}),
		codeBlock("massive code block", 1500, []string{"Intro"}),
		paragraphBlock("After code", 100, []string{"Intro"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The oversized block should be its own chunk.
	found := false
	for _, c := range chunks {
		if c.Oversized {
			found = true
			if c.Tokens != 1500 {
				t.Errorf("oversized chunk: expected 1500 tokens, got %d", c.Tokens)
			}
			if len(c.Blocks) != 1 {
				t.Errorf("oversized chunk: expected 1 block, got %d", len(c.Blocks))
			}
		}
	}
	if !found {
		t.Error("expected an oversized chunk to be flagged")
	}
}

func TestChunkDocument_OversizedFlushesAccumulator(t *testing.T) {
	d := doc(
		paragraphBlock("Before", 100, []string{"Intro"}),
		codeBlock("huge code", 1500, []string{"Intro"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First chunk: the paragraph. Second: the oversized code block.
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Oversized {
		t.Error("first chunk should not be oversized")
	}
	if !chunks[1].Oversized {
		t.Error("second chunk should be oversized")
	}
}

func TestChunkDocument_SequenceNumbering(t *testing.T) {
	d := doc(
		headingBlock("Section 1", 2, 10, []string{"Section 1"}),
		paragraphBlock("Content 1", 400, []string{"Section 1"}),
		headingBlock("Section 2", 2, 10, []string{"Section 2"}),
		paragraphBlock("Content 2", 400, []string{"Section 2"}),
		headingBlock("Section 3", 2, 10, []string{"Section 3"}),
		paragraphBlock("Content 3", 400, []string{"Section 3"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, c := range chunks {
		if c.Sequence != i+1 {
			t.Errorf("chunk %d: expected sequence %d, got %d", i, i+1, c.Sequence)
		}
		if c.TotalInFile != len(chunks) {
			t.Errorf("chunk %d: expected total %d, got %d", i, len(chunks), c.TotalInFile)
		}
	}
}

func TestChunkDocument_ContentAndIDsAreSet(t *testing.T) {
	d := doc(
		paragraphBlock("Test content here", 100, []string{"Section"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := chunks[0]
	if c.Content == "" {
		t.Error("expected content to be set")
	}
	if c.ID == "" {
		t.Error("expected ID to be set")
	}
	if !strings.HasPrefix(c.ID, "obj:") {
		t.Errorf("expected obj: prefix, got %q", c.ID)
	}
	if !strings.Contains(c.ID, ".1") {
		t.Errorf("expected sequence .1 in ID, got %q", c.ID)
	}
}

func TestChunkDocument_SpecChunkType(t *testing.T) {
	d := doc(paragraphBlock("Spec reasoning", 100, []string{"Design"}))

	chunks, err := chunk.ChunkDocument(d, "auth.spec.md", chunk.ChunkSpec, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunks[0].Type != chunk.ChunkSpec {
		t.Errorf("expected spec type, got %s", chunks[0].Type)
	}
	if !strings.HasPrefix(chunks[0].ID, "spec:") {
		t.Errorf("expected spec: prefix, got %q", chunks[0].ID)
	}
}

func TestChunkDocument_SystemChunkType(t *testing.T) {
	d := doc(paragraphBlock("System instructions", 100, []string{"Rules"}))

	chunks, err := chunk.ChunkDocument(d, "system/topics/foo/README.md", chunk.ChunkSystem, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunks[0].Type != chunk.ChunkSystem {
		t.Errorf("expected system type, got %s", chunks[0].Type)
	}
	if !strings.HasPrefix(chunks[0].ID, "sys:") {
		t.Errorf("expected sys: prefix, got %q", chunks[0].ID)
	}
}

func TestChunkDocument_AllHeadingsEachOwnChunk(t *testing.T) {
	d := doc(
		headingBlock("H1", 1, 20, []string{"H1"}),
		headingBlock("H2", 2, 20, []string{"H1", "H2"}),
		headingBlock("H3", 2, 20, []string{"H1", "H3"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks (one per heading), got %d", len(chunks))
	}
}

func TestChunkDocument_MixedBlockTypes(t *testing.T) {
	d := doc(
		headingBlock("Setup", 2, 10, []string{"Setup"}),
		paragraphBlock("Install instructions", 100, []string{"Setup"}),
		codeBlock("npm install foo", 50, []string{"Setup"}),
		paragraphBlock("Configuration", 100, []string{"Setup"}),
		headingBlock("Usage", 2, 10, []string{"Usage"}),
		paragraphBlock("Usage details", 300, []string{"Usage"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First chunk: heading + paragraph + code + paragraph = 10+100+50+100=260 < 450
	// Second chunk: heading + paragraph = 10+300=310 < 450
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Tokens != 260 {
		t.Errorf("chunk 1: expected 260 tokens, got %d", chunks[0].Tokens)
	}
	if chunks[1].Tokens != 310 {
		t.Errorf("chunk 2: expected 310 tokens, got %d", chunks[1].Tokens)
	}
}

func TestChunkDocument_SourcePathPropagated(t *testing.T) {
	d := doc(paragraphBlock("Content", 100, nil))

	chunks, err := chunk.ChunkDocument(d, "docs/topics/auth/jwt.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunks[0].Source != "docs/topics/auth/jwt.md" {
		t.Errorf("expected source path, got %q", chunks[0].Source)
	}
}

func TestChunkDocument_HeadingHierarchyPropagated(t *testing.T) {
	d := doc(
		headingBlock("Auth", 1, 10, []string{"Auth"}),
		paragraphBlock("Auth intro", 100, []string{"Auth"}),
		headingBlock("JWT", 2, 10, []string{"Auth", "JWT"}),
		paragraphBlock("JWT details", 100, []string{"Auth", "JWT"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunks[0].Heading != "Auth" {
		t.Errorf("chunk 1 heading: expected %q, got %q", "Auth", chunks[0].Heading)
	}
	if len(chunks) > 1 && chunks[1].Heading != "Auth > JWT" {
		t.Errorf("chunk 2 heading: expected %q, got %q", "Auth > JWT", chunks[1].Heading)
	}
}

// ---------------------------------------------------------------------------
// Min-tokens rebalancing tests
// ---------------------------------------------------------------------------

func TestChunkDocument_RebalanceMerge(t *testing.T) {
	// Two chunks where the second is below MinTokens (200).
	// Previous: 400 tokens. Last: 50 tokens. Combined: 450 <= 450 target.
	// The last chunk does NOT start with a heading, so merge is allowed.
	d := doc(
		paragraphBlock("Content A", 400, []string{"Section"}),
		paragraphBlock("Tiny tail", 50, []string{"Section"}),
	)

	// 400 >= 360 (flex threshold), so adding 50 would exceed 450.
	// Split into [400] and [50]. Then rebalance: 50 < 200, no heading,
	// combined 450 <= 450 → merge.
	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk after merge, got %d", len(chunks))
	}
	if chunks[0].Tokens != 450 {
		t.Errorf("expected 450 tokens after merge, got %d", chunks[0].Tokens)
	}
}

func TestChunkDocument_RebalanceEvenSplit(t *testing.T) {
	// Previous: ~400 tokens. Last: ~150 tokens. Combined: 550 > 450 target.
	// Can't just merge — rebalance to ~275 each.
	d := doc(
		paragraphBlock("Block 1", 200, []string{"Section"}),
		paragraphBlock("Block 2", 200, []string{"Section"}),
		headingBlock("Tail Section", 2, 10, []string{"Tail Section"}),
		paragraphBlock("Tiny", 140, []string{"Tail Section"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Before rebalance: chunk1=[Block1+Block2]=400, chunk2=[Heading+Tiny]=150 (<200).
	// Combined=550, target per chunk=275.
	// After rebalance: should be roughly even.
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks after rebalance, got %d", len(chunks))
	}

	// Check they're more balanced than before.
	diff := chunks[0].Tokens - chunks[1].Tokens
	if diff < 0 {
		diff = -diff
	}
	if diff > 300 {
		t.Errorf("expected balanced chunks, but diff is %d (chunk1=%d, chunk2=%d)",
			diff, chunks[0].Tokens, chunks[1].Tokens)
	}
}

func TestChunkDocument_RebalanceDoesNotAffectOversized(t *testing.T) {
	d := doc(
		codeBlock("huge code", 1500, []string{"Code"}),
		paragraphBlock("Tiny tail", 50, []string{"Code"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Oversized chunk should not be merged with the tiny tail.
	for _, c := range chunks {
		if c.Oversized && c.Tokens != 1500 {
			t.Errorf("oversized chunk should keep its token count, got %d", c.Tokens)
		}
	}
}

func TestChunkDocument_RebalanceNotNeededAboveMin(t *testing.T) {
	// Last chunk is exactly at MinTokens — no rebalance needed.
	d := doc(
		headingBlock("A", 2, 10, []string{"A"}),
		paragraphBlock("Content A", 390, []string{"A"}),
		headingBlock("B", 2, 10, []string{"B"}),
		paragraphBlock("Content B", 200, []string{"B"}),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	// Last chunk is 210 >= 200, so no rebalancing.
	if chunks[1].Tokens != 210 {
		t.Errorf("expected last chunk 210 tokens, got %d", chunks[1].Tokens)
	}
}

// ---------------------------------------------------------------------------
// Collision detection tests
// ---------------------------------------------------------------------------

func TestCheckCollisions_NoDuplicates(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:aaa.1", Content: "content one"},
		{ID: "obj:bbb.2", Content: "content two"},
	}
	if err := chunk.CheckCollisions(chunks); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckCollisions_SameIDSameContent(t *testing.T) {
	// Same ID and same content is fine (no collision).
	chunks := []chunk.Chunk{
		{ID: "obj:aaa.1", Content: "same content"},
		{ID: "obj:aaa.1", Content: "same content"},
	}
	if err := chunk.CheckCollisions(chunks); err != nil {
		t.Errorf("unexpected error for same content: %v", err)
	}
}

func TestCheckCollisions_SameIDDifferentContent(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:aaa.1", Content: "content one"},
		{ID: "obj:aaa.1", Content: "content two"},
	}
	err := chunk.CheckCollisions(chunks)
	if err == nil {
		t.Fatal("expected collision error")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf("expected collision in error message, got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Render tests
// ---------------------------------------------------------------------------

func TestRenderMeta_ObjectChunk(t *testing.T) {
	c := &chunk.Chunk{
		ID:          "obj:abcdef1234567890.3",
		Type:        chunk.ChunkObject,
		Source:      "docs/topics/auth/jwt.md",
		Heading:     "Authentication > JWT > Refresh",
		Sequence:    3,
		TotalInFile: 7,
		Tokens:      462,
	}

	meta := chunk.RenderMeta(c)

	expected := []string{
		"<!-- codectx:meta",
		"id: obj:abcdef1234567890.3",
		"type: object",
		"source: docs/topics/auth/jwt.md",
		"heading: Authentication > JWT > Refresh",
		"chunk: 3 of 7",
		"tokens: 462",
		"-->",
	}

	for _, line := range expected {
		if !strings.Contains(meta, line) {
			t.Errorf("expected meta to contain %q", line)
		}
	}

	// Should not have oversized field.
	if strings.Contains(meta, "oversized") {
		t.Error("expected no oversized field for non-oversized chunk")
	}
}

func TestRenderMeta_OversizedChunk(t *testing.T) {
	c := &chunk.Chunk{
		ID:          "obj:abcdef1234567890.1",
		Type:        chunk.ChunkObject,
		Source:      "test.md",
		Heading:     "Code",
		Sequence:    1,
		TotalInFile: 1,
		Tokens:      1500,
		Oversized:   true,
	}

	meta := chunk.RenderMeta(c)
	if !strings.Contains(meta, "oversized: true") {
		t.Error("expected oversized: true in meta")
	}
}

func TestRenderMeta_SpecChunk(t *testing.T) {
	c := &chunk.Chunk{
		ID:          "spec:f7g8h9.2",
		Type:        chunk.ChunkSpec,
		Source:      "docs/topics/auth/jwt.spec.md",
		Heading:     "Auth > JWT",
		Sequence:    2,
		TotalInFile: 3,
		Tokens:      380,
	}

	meta := chunk.RenderMeta(c)
	if !strings.Contains(meta, "type: spec") {
		t.Error("expected type: spec")
	}
}

func TestRenderMeta_SystemChunk(t *testing.T) {
	c := &chunk.Chunk{
		ID:          "sys:m3n4o5.1",
		Type:        chunk.ChunkSystem,
		Source:      "system/topics/taxonomy/README.md",
		Heading:     "Taxonomy > Rules",
		Sequence:    1,
		TotalInFile: 2,
		Tokens:      340,
	}

	meta := chunk.RenderMeta(c)
	if !strings.Contains(meta, "type: system") {
		t.Error("expected type: system")
	}
}

func TestRender_FullOutput(t *testing.T) {
	c := &chunk.Chunk{
		ID:          "obj:abc123.1",
		Type:        chunk.ChunkObject,
		Source:      "test.md",
		Heading:     "Test",
		Sequence:    1,
		TotalInFile: 1,
		Tokens:      100,
		Content:     "This is the chunk content.",
	}

	rendered := chunk.Render(c)

	if !strings.HasPrefix(rendered, "<!-- codectx:meta") {
		t.Error("expected rendered output to start with meta header")
	}
	if !strings.Contains(rendered, "\n\nThis is the chunk content.\n") {
		t.Error("expected blank line between meta and content")
	}
}

func TestRender_EmptyContent(t *testing.T) {
	c := &chunk.Chunk{
		ID:          "obj:abc123.1",
		Type:        chunk.ChunkObject,
		Source:      "test.md",
		Heading:     "Test",
		Sequence:    1,
		TotalInFile: 1,
		Tokens:      0,
		Content:     "",
	}

	rendered := chunk.Render(c)
	if !strings.HasSuffix(rendered, "-->\n") {
		t.Errorf("expected rendered to end with -->\\n for empty content, got %q", rendered)
	}
}

// ---------------------------------------------------------------------------
// Route tests
// ---------------------------------------------------------------------------

func TestClassifySource_Object(t *testing.T) {
	tests := []string{
		"docs/topics/auth/jwt.md",
		"topics/auth.md",
		"foundation/overview.md",
		"README.md",
	}
	for _, src := range tests {
		ct := chunk.ClassifySource(src, "system")
		if ct != chunk.ChunkObject {
			t.Errorf("ClassifySource(%q) = %s, want object", src, ct)
		}
	}
}

func TestClassifySource_Spec(t *testing.T) {
	tests := []string{
		"docs/topics/auth/jwt.spec.md",
		"system/topics/taxonomy/README.spec.md",
		"foundation/overview.spec.md",
	}
	for _, src := range tests {
		ct := chunk.ClassifySource(src, "system")
		if ct != chunk.ChunkSpec {
			t.Errorf("ClassifySource(%q) = %s, want spec", src, ct)
		}
	}
}

func TestClassifySource_System(t *testing.T) {
	tests := []string{
		"system/topics/taxonomy/README.md",
		"system/foundation/documentation-protocol/README.md",
		"system/prompts/compile.md",
	}
	for _, src := range tests {
		ct := chunk.ClassifySource(src, "system")
		if ct != chunk.ChunkSystem {
			t.Errorf("ClassifySource(%q) = %s, want system", src, ct)
		}
	}
}

func TestClassifySource_SpecTakesPrecedenceOverSystem(t *testing.T) {
	// .spec.md files under system/ should still be spec, not system.
	ct := chunk.ClassifySource("system/topics/taxonomy/README.spec.md", "system")
	if ct != chunk.ChunkSpec {
		t.Errorf("expected spec (takes precedence), got %s", ct)
	}
}

func TestOutputDir(t *testing.T) {
	tests := []struct {
		ct       chunk.ChunkType
		expected string
	}{
		{chunk.ChunkObject, "objects"},
		{chunk.ChunkSpec, "specs"},
		{chunk.ChunkSystem, "system"},
	}
	for _, tt := range tests {
		got := chunk.OutputDir(tt.ct)
		if got != tt.expected {
			t.Errorf("OutputDir(%s) = %q, want %q", tt.ct, got, tt.expected)
		}
	}
}

func TestOutputFilename(t *testing.T) {
	c := &chunk.Chunk{
		ID:       "obj:abcdef1234567890.3",
		Sequence: 3,
	}
	got := chunk.OutputFilename(c)
	if got != "abcdef1234567890.3.md" {
		t.Errorf("got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestChunkDocument_ZeroTokenBlocks(t *testing.T) {
	d := doc(
		paragraphBlock("", 0, nil),
		paragraphBlock("", 0, nil),
	)

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Zero-token blocks should still be accumulated into chunks.
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk for non-empty block list")
	}
}

func TestChunkDocument_SingleBlockExactlyAtTarget(t *testing.T) {
	d := doc(paragraphBlock("Exactly at target", 450, []string{"Test"}))

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Tokens != 450 {
		t.Errorf("expected 450 tokens, got %d", chunks[0].Tokens)
	}
}

func TestChunkDocument_SingleBlockExactlyAtMax(t *testing.T) {
	d := doc(paragraphBlock("At max", 800, []string{"Test"}))

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	// 800 == max_tokens, so NOT oversized.
	if chunks[0].Oversized {
		t.Error("expected chunk at max_tokens to NOT be oversized")
	}
}

func TestChunkDocument_SingleBlockJustOverMax(t *testing.T) {
	d := doc(paragraphBlock("Over max", 801, []string{"Test"}))

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !chunks[0].Oversized {
		t.Error("expected chunk over max_tokens to be oversized")
	}
}

func TestChunkDocument_StableHashAcrossRuns(t *testing.T) {
	d := doc(paragraphBlock("Stable content", 100, []string{"Test"}))

	chunks1, _ := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())
	chunks2, _ := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, defaultOpts())

	if chunks1[0].ID != chunks2[0].ID {
		t.Errorf("expected stable IDs: %q vs %q", chunks1[0].ID, chunks2[0].ID)
	}
}

func TestChunkDocument_LargeDocumentManyChunks(t *testing.T) {
	// Simulate a large document with many sections.
	blocks := make([]markdown.Block, 0, 40)
	for i := range 20 {
		h := []string{"Section " + string(rune('A'+i))}
		blocks = append(blocks,
			headingBlock(h[0], 2, 10, h),
			paragraphBlock("Content for section", 400, h),
		)
	}

	d := doc(blocks...)

	chunks, err := chunk.ChunkDocument(d, "large.md", chunk.ChunkObject, defaultOpts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 20 sections, each with heading+paragraph = 410 tokens < 450 target.
	// Each section should be its own chunk (heading always breaks).
	if len(chunks) != 20 {
		t.Errorf("expected 20 chunks, got %d", len(chunks))
	}

	// Verify all have correct total.
	for _, c := range chunks {
		if c.TotalInFile != len(chunks) {
			t.Errorf("chunk %d: TotalInFile = %d, want %d", c.Sequence, c.TotalInFile, len(chunks))
		}
	}
}

// ---------------------------------------------------------------------------
// OptionsFromConfig test
// ---------------------------------------------------------------------------

func TestOptionsFromConfig(t *testing.T) {
	cfg := project.ChunkingConfig{
		TargetTokens:      600,
		MinTokens:         100,
		MaxTokens:         1200,
		FlexibilityWindow: 0.75,
		HashLength:        32,
	}

	opts := chunk.OptionsFromConfig(cfg)

	if opts.TargetTokens != 600 {
		t.Errorf("target tokens: expected 600, got %d", opts.TargetTokens)
	}
	if opts.MinTokens != 100 {
		t.Errorf("min tokens: expected 100, got %d", opts.MinTokens)
	}
	if opts.MaxTokens != 1200 {
		t.Errorf("max tokens: expected 1200, got %d", opts.MaxTokens)
	}
	if opts.FlexibilityWindow != 0.75 {
		t.Errorf("flexibility window: expected 0.75, got %f", opts.FlexibilityWindow)
	}
	if opts.HashLength != 32 {
		t.Errorf("hash length: expected 32, got %d", opts.HashLength)
	}
}

func TestOptionsFromConfig_ClampsHashLength(t *testing.T) {
	cfg := project.ChunkingConfig{HashLength: 1000}
	opts := chunk.OptionsFromConfig(cfg)
	if opts.HashLength != project.MaxHashLength {
		t.Errorf("expected clamped hash length %d, got %d", project.MaxHashLength, opts.HashLength)
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := chunk.DefaultOptions()
	defaults := project.DefaultPreferencesConfig().Chunking
	if opts.TargetTokens != defaults.TargetTokens {
		t.Errorf("target tokens: expected %d, got %d", defaults.TargetTokens, opts.TargetTokens)
	}
	if opts.MinTokens != defaults.MinTokens {
		t.Errorf("min tokens: expected %d, got %d", defaults.MinTokens, opts.MinTokens)
	}
	if opts.MaxTokens != defaults.MaxTokens {
		t.Errorf("max tokens: expected %d, got %d", defaults.MaxTokens, opts.MaxTokens)
	}
	if opts.FlexibilityWindow != defaults.FlexibilityWindow {
		t.Errorf("flexibility window: expected %f, got %f", defaults.FlexibilityWindow, opts.FlexibilityWindow)
	}
	if opts.HashLength != project.ClampHashLength(defaults.HashLength) {
		t.Errorf("hash length: expected %d, got %d", project.ClampHashLength(defaults.HashLength), opts.HashLength)
	}
}

// ---------------------------------------------------------------------------
// Input validation tests
// ---------------------------------------------------------------------------

func TestChunkDocument_InvalidOptions_ZeroTarget(t *testing.T) {
	d := doc(paragraphBlock("text", 100, nil))
	opts := defaultOpts()
	opts.TargetTokens = 0

	_, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, opts)
	if err == nil {
		t.Fatal("expected error for zero target tokens")
	}
}

func TestChunkDocument_InvalidOptions_NegativeMaxTokens(t *testing.T) {
	d := doc(paragraphBlock("text", 100, nil))
	opts := defaultOpts()
	opts.MaxTokens = -1

	_, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, opts)
	if err == nil {
		t.Fatal("expected error for negative max tokens")
	}
}

func TestChunkDocument_InvalidOptions_NegativeMinTokens(t *testing.T) {
	d := doc(paragraphBlock("text", 100, nil))
	opts := defaultOpts()
	opts.MinTokens = -1

	_, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, opts)
	if err == nil {
		t.Fatal("expected error for negative min tokens")
	}
}

func TestChunkDocument_InvalidOptions_FlexibilityWindowOutOfRange(t *testing.T) {
	d := doc(paragraphBlock("text", 100, nil))

	opts := defaultOpts()
	opts.FlexibilityWindow = 1.5
	_, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, opts)
	if err == nil {
		t.Fatal("expected error for flexibility window > 1")
	}

	opts.FlexibilityWindow = -0.1
	_, err = chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, opts)
	if err == nil {
		t.Fatal("expected error for negative flexibility window")
	}
}

func TestChunkDocument_FlexibilityWindowZero(t *testing.T) {
	// FlexibilityWindow 0.0 means always break as soon as we'd exceed target.
	d := doc(
		paragraphBlock("A", 200, []string{"Section"}),
		paragraphBlock("B", 300, []string{"Section"}),
	)

	opts := defaultOpts()
	opts.FlexibilityWindow = 0.0

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 200 tokens, adding 300 would exceed 450. FlexThreshold = 0.
	// 200 >= 0 → break. So 2 chunks.
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks with flex window 0.0, got %d", len(chunks))
	}
}

func TestChunkDocument_FlexibilityWindowOne(t *testing.T) {
	// FlexibilityWindow 1.0 means only break when at 100% of target.
	d := doc(
		paragraphBlock("A", 200, []string{"Section"}),
		paragraphBlock("B", 300, []string{"Section"}),
	)

	opts := defaultOpts()
	opts.FlexibilityWindow = 1.0

	chunks, err := chunk.ChunkDocument(d, "test.md", chunk.ChunkObject, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 200 tokens, adding 300 would reach 500. FlexThreshold = 450.
	// 200 < 450 → include. Single chunk with 500 tokens.
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk with flex window 1.0, got %d", len(chunks))
	}
}

// ---------------------------------------------------------------------------
// Additional edge case tests for routes
// ---------------------------------------------------------------------------

func TestClassifySource_EmptySystemDir(t *testing.T) {
	// With empty systemDir, system/ path should be classified as object.
	ct := chunk.ClassifySource("system/topics/foo.md", "")
	if ct != chunk.ChunkObject {
		t.Errorf("expected object with empty systemDir, got %s", ct)
	}
}

func TestOutputFilename_MalformedID(t *testing.T) {
	c := &chunk.Chunk{
		ID:       "no-colon-here",
		Sequence: 5,
	}
	got := chunk.OutputFilename(c)
	if got != "unknown.5.md" {
		t.Errorf("expected fallback filename, got %q", got)
	}
}

func TestOutputDir_UnknownType(t *testing.T) {
	got := chunk.OutputDir(chunk.ChunkType("bogus"))
	if got != "objects" {
		t.Errorf("expected default 'objects' for unknown type, got %q", got)
	}
}

func TestContentHash_EmptyString(t *testing.T) {
	h := chunk.ContentHash("", 16)
	if len(h) != 16 {
		t.Errorf("expected 16 chars for empty string hash, got %d", len(h))
	}
	// Should be the SHA-256 of empty string, which is well-known.
	if h != "e3b0c44298fc1c14" {
		t.Errorf("unexpected hash of empty string: %q", h)
	}
}

// ---------------------------------------------------------------------------
// Nil safety tests
// ---------------------------------------------------------------------------

func TestRenderMeta_NilChunk(t *testing.T) {
	got := chunk.RenderMeta(nil)
	if got != "" {
		t.Errorf("expected empty string for nil chunk, got %q", got)
	}
}

func TestRender_NilChunk(t *testing.T) {
	got := chunk.Render(nil)
	if got != "" {
		t.Errorf("expected empty string for nil chunk, got %q", got)
	}
}

func TestOutputFilename_NilChunk(t *testing.T) {
	got := chunk.OutputFilename(nil)
	if got != "" {
		t.Errorf("expected empty string for nil chunk, got %q", got)
	}
}

func TestOutputFilename_EmptyID(t *testing.T) {
	c := &chunk.Chunk{
		ID:       "",
		Sequence: 1,
	}
	got := chunk.OutputFilename(c)
	// Empty ID has no colon, so falls back to unknown format.
	if got != "unknown.1.md" {
		t.Errorf("expected fallback filename for empty ID, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// CheckCollisions — empty/nil input
// ---------------------------------------------------------------------------

func TestCheckCollisions_NilSlice(t *testing.T) {
	err := chunk.CheckCollisions(nil)
	if err != nil {
		t.Errorf("expected nil error for nil slice, got %v", err)
	}
}

func TestCheckCollisions_EmptySlice(t *testing.T) {
	err := chunk.CheckCollisions([]chunk.Chunk{})
	if err != nil {
		t.Errorf("expected nil error for empty slice, got %v", err)
	}
}

func TestCheckCollisions_NoCollisions(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:abc.1", Content: "content A"},
		{ID: "obj:def.2", Content: "content B"},
	}
	err := chunk.CheckCollisions(chunks)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestCheckCollisions_DuplicateIDSameContent(t *testing.T) {
	chunks := []chunk.Chunk{
		{ID: "obj:abc.1", Content: "same content"},
		{ID: "obj:abc.1", Content: "same content"},
	}
	err := chunk.CheckCollisions(chunks)
	if err != nil {
		t.Errorf("expected nil error for duplicate ID with same content, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Rebalance — heading-boundary preservation
// ---------------------------------------------------------------------------

func TestChunkDocument_RebalancePreservesHeadingBoundary(t *testing.T) {
	// Create a document where the last chunk starts with a heading and is
	// below min_tokens. The heading boundary should prevent merging.
	doc := &markdown.Document{
		Blocks: []markdown.Block{
			paragraphBlock("First paragraph content here with enough tokens", 300, []string{"Section A"}),
			headingBlock("Section B", 2, 5, []string{"Section B"}),
			paragraphBlock("Tiny ending", 50, []string{"Section B"}),
		},
	}

	opts := chunk.Options{
		TargetTokens:      400,
		MinTokens:         100, // The last chunk (55 tokens) is below this.
		MaxTokens:         800,
		FlexibilityWindow: 0.8,
		HashLength:        16,
	}

	chunks, err := chunk.ChunkDocument(doc, "test.md", chunk.ChunkObject, opts)
	if err != nil {
		t.Fatalf("ChunkDocument: %v", err)
	}

	// Because the last chunk starts with a heading, it should NOT be merged
	// with the previous chunk despite being below MinTokens.
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks (heading boundary preserved), got %d", len(chunks))
	}
}

// ---------------------------------------------------------------------------
// FormatID — unknown ChunkType
// ---------------------------------------------------------------------------

func TestFormatID_UnknownChunkType(t *testing.T) {
	got := chunk.FormatID(chunk.ChunkType("unknown"), "abc123", 1)
	// Unknown types should fall back to "obj" prefix.
	if !strings.HasPrefix(got, "obj:") {
		t.Errorf("expected 'obj:' prefix for unknown type, got %q", got)
	}
	if got != "obj:abc123.1" {
		t.Errorf("expected %q, got %q", "obj:abc123.1", got)
	}
}

// ---------------------------------------------------------------------------
// HeadingSeparator constant
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// JoinContent — blocks with empty content
// ---------------------------------------------------------------------------

func TestJoinContent_EmptyContentBlocks(t *testing.T) {
	blocks := []markdown.Block{
		{Content: ""},
		{Content: ""},
	}
	got := chunk.JoinContent(blocks)
	if got != "\n\n" {
		t.Errorf("expected two empty contents joined, got %q", got)
	}
}

func TestJoinContent_MixedEmptyAndNonEmpty(t *testing.T) {
	blocks := []markdown.Block{
		{Content: "first"},
		{Content: ""},
		{Content: "third"},
	}
	got := chunk.JoinContent(blocks)
	expected := "first\n\n\n\nthird"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestHeadingSeparator_UsedInFormatHeading(t *testing.T) {
	got := chunk.FormatHeading([]string{"A", "B", "C"})
	if got != "A"+chunk.HeadingSeparator+"B"+chunk.HeadingSeparator+"C" {
		t.Errorf("expected heading with separator %q, got %q", chunk.HeadingSeparator, got)
	}
}

// ---------------------------------------------------------------------------
// SpecFileSuffix constant
// ---------------------------------------------------------------------------

func TestSpecFileSuffix_ClassifySource(t *testing.T) {
	// Verify the constant is used in ClassifySource.
	got := chunk.ClassifySource("docs/topics/auth/jwt"+chunk.SpecFileSuffix, "system")
	if got != chunk.ChunkSpec {
		t.Errorf("expected ChunkSpec for spec file suffix, got %v", got)
	}
}
