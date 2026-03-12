package query

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/core/manifest"
)

// ---------------------------------------------------------------------------
// typeRankFor
// ---------------------------------------------------------------------------

func TestTypeRankFor_AllTypes(t *testing.T) {
	tests := []struct {
		chunkType string
		want      int
	}{
		{"object", 0},
		{"system", 1},
		{"spec", 2},
		{"unknown", 0}, // default
	}
	for _, tt := range tests {
		got := typeRankFor(tt.chunkType)
		if got != tt.want {
			t.Errorf("typeRankFor(%q) = %d, want %d", tt.chunkType, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// sectionTitle
// ---------------------------------------------------------------------------

func TestSectionTitle_AllRanks(t *testing.T) {
	tests := []struct {
		rank int
		want string
	}{
		{0, "Instructions"},
		{1, "System"},
		{2, "Reasoning"},
		{99, "Instructions"}, // default
	}
	for _, tt := range tests {
		got := sectionTitle(tt.rank)
		if got != tt.want {
			t.Errorf("sectionTitle(%d) = %q, want %q", tt.rank, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// sectionPreamble
// ---------------------------------------------------------------------------

func TestSectionPreamble_Reasoning(t *testing.T) {
	got := sectionPreamble(2)
	if got == "" {
		t.Error("expected non-empty preamble for reasoning section")
	}
}

func TestSectionPreamble_NonReasoning(t *testing.T) {
	for _, rank := range []int{0, 1, 99} {
		got := sectionPreamble(rank)
		if got != "" {
			t.Errorf("sectionPreamble(%d) = %q, want empty", rank, got)
		}
	}
}

// ---------------------------------------------------------------------------
// ParseChunkIDs
// ---------------------------------------------------------------------------

func TestParseChunkIDs_CommaSeparated(t *testing.T) {
	got := ParseChunkIDs("obj:a1b2c3.03,spec:f7g8h9.02,sys:d4e5f6.01")
	want := []string{"obj:a1b2c3.03", "spec:f7g8h9.02", "sys:d4e5f6.01"}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, id := range got {
		if id != want[i] {
			t.Errorf("ParseChunkIDs[%d] = %q, want %q", i, id, want[i])
		}
	}
}

func TestParseChunkIDs_TrimsWhitespace(t *testing.T) {
	got := ParseChunkIDs(" obj:a1b2c3.03 , spec:f7g8h9.02 ")

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != "obj:a1b2c3.03" {
		t.Errorf("[0] = %q, want %q", got[0], "obj:a1b2c3.03")
	}
	if got[1] != "spec:f7g8h9.02" {
		t.Errorf("[1] = %q, want %q", got[1], "spec:f7g8h9.02")
	}
}

func TestParseChunkIDs_FiltersEmpty(t *testing.T) {
	got := ParseChunkIDs("obj:a1b2c3.03,,, ,spec:f7g8h9.02,")
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %v", len(got), got)
	}
}

func TestParseChunkIDs_EmptyString(t *testing.T) {
	got := ParseChunkIDs("")
	if len(got) != 0 {
		t.Errorf("expected empty result for empty input, got %v", got)
	}
}

func TestParseChunkIDs_SingleID(t *testing.T) {
	got := ParseChunkIDs("obj:a1b2c3.03")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0] != "obj:a1b2c3.03" {
		t.Errorf("[0] = %q, want %q", got[0], "obj:a1b2c3.03")
	}
}

func TestParseChunkIDs_OnlyWhitespace(t *testing.T) {
	got := ParseChunkIDs("  ,  ,  ")
	if len(got) != 0 {
		t.Errorf("expected empty result for whitespace-only input, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// assembleDocument — bridge insertion
// ---------------------------------------------------------------------------

func TestAssembleDocument_AdjacentChunks_NoBridge(t *testing.T) {
	bridgeText := "Some bridge text"
	resolved := []resolvedChunk{
		{
			id: "obj:abc123.01",
			entry: &manifest.ManifestEntry{
				Type: "object", Source: "auth.md", Heading: "Auth",
				Sequence: 1, BridgeToNext: &bridgeText,
			},
			content:  "# Auth\n\nFirst chunk content.",
			typeRank: 0,
		},
		{
			id: "obj:abc123.02",
			entry: &manifest.ManifestEntry{
				Type: "object", Source: "auth.md", Heading: "Auth",
				Sequence: 2,
			},
			content:  "Second chunk content.",
			typeRank: 0,
		},
	}

	got := assembleDocument(resolved, []string{"obj:abc123.01", "obj:abc123.02"})

	// Adjacent chunks (seq 1 -> 2) should NOT have a bridge inserted.
	if strings.Contains(got, "Context bridge") {
		t.Error("should not insert bridge between adjacent chunks")
	}
}

func TestAssembleDocument_NonAdjacentChunks_InsertsBridge(t *testing.T) {
	bridgeText := "Established JWT token structure and signing requirements"
	resolved := []resolvedChunk{
		{
			id: "obj:abc123.01",
			entry: &manifest.ManifestEntry{
				Type: "object", Source: "auth.md", Heading: "Auth > JWT",
				Sequence: 1, BridgeToNext: &bridgeText,
			},
			content:  "# JWT Tokens\n\nFirst chunk.",
			typeRank: 0,
		},
		{
			id: "obj:abc123.03",
			entry: &manifest.ManifestEntry{
				Type: "object", Source: "auth.md", Heading: "Auth > Validation",
				Sequence: 3,
			},
			content:  "# Validation\n\nThird chunk.",
			typeRank: 0,
		},
	}

	got := assembleDocument(resolved, []string{"obj:abc123.01", "obj:abc123.03"})

	if !strings.Contains(got, "Context bridge") {
		t.Error("expected bridge between non-adjacent chunks")
	}
	if !strings.Contains(got, "Established JWT token structure") {
		t.Error("expected bridge text content")
	}
}

func TestAssembleDocument_NonAdjacentChunks_NilBridge(t *testing.T) {
	resolved := []resolvedChunk{
		{
			id: "obj:abc123.01",
			entry: &manifest.ManifestEntry{
				Type: "object", Source: "auth.md", Heading: "Auth",
				Sequence: 1, BridgeToNext: nil,
			},
			content:  "First chunk.",
			typeRank: 0,
		},
		{
			id: "obj:abc123.03",
			entry: &manifest.ManifestEntry{
				Type: "object", Source: "auth.md", Heading: "Auth",
				Sequence: 3,
			},
			content:  "Third chunk.",
			typeRank: 0,
		},
	}

	got := assembleDocument(resolved, []string{"obj:abc123.01", "obj:abc123.03"})

	// No bridge text available — should not insert bridge marker.
	if strings.Contains(got, "Context bridge") {
		t.Error("should not insert bridge when BridgeToNext is nil")
	}
}

func TestAssembleDocument_DifferentSources_NoBridge(t *testing.T) {
	bridgeText := "Some bridge text"
	resolved := []resolvedChunk{
		{
			id: "obj:abc123.01",
			entry: &manifest.ManifestEntry{
				Type: "object", Source: "auth.md", Heading: "Auth",
				Sequence: 1, BridgeToNext: &bridgeText,
			},
			content:  "Auth chunk.",
			typeRank: 0,
		},
		{
			id: "obj:def456.01",
			entry: &manifest.ManifestEntry{
				Type: "object", Source: "db.md", Heading: "Database",
				Sequence: 1,
			},
			content:  "Database chunk.",
			typeRank: 0,
		},
	}

	got := assembleDocument(resolved, []string{"obj:abc123.01", "obj:def456.01"})

	// Different source files — should use separator, not bridge.
	if strings.Contains(got, "Context bridge") {
		t.Error("should not insert bridge between chunks from different sources")
	}
	if !strings.Contains(got, "---") {
		t.Error("expected separator between different source files")
	}
}
