package query

import (
	"testing"

	"github.com/securacore/codectx/core/manifest"
)

// ---------------------------------------------------------------------------
// topicSlug
// ---------------------------------------------------------------------------

func TestTopicSlug_BasicHeading(t *testing.T) {
	resolved := []resolvedChunk{
		{entry: &manifest.ManifestEntry{Heading: "Auth > JWT Tokens"}},
	}
	got := topicSlug(resolved)
	if got != "auth-jwt-tokens" {
		t.Errorf("got %q, want %q", got, "auth-jwt-tokens")
	}
}

func TestTopicSlug_EmptyResolved(t *testing.T) {
	got := topicSlug(nil)
	if got != "generated" {
		t.Errorf("got %q, want %q", got, "generated")
	}
}

func TestTopicSlug_SpecialCharacters(t *testing.T) {
	resolved := []resolvedChunk{
		{entry: &manifest.ManifestEntry{Heading: "Auth (v2) & Sessions!"}},
	}
	got := topicSlug(resolved)
	if got != "auth-v2-sessions" {
		t.Errorf("got %q, want %q", got, "auth-v2-sessions")
	}
}

func TestTopicSlug_LeadingHyphens(t *testing.T) {
	resolved := []resolvedChunk{
		{entry: &manifest.ManifestEntry{Heading: "> Leading Arrow"}},
	}
	got := topicSlug(resolved)
	if got != "leading-arrow" {
		t.Errorf("got %q, want %q", got, "leading-arrow")
	}
}

func TestTopicSlug_ConsecutiveHyphens(t *testing.T) {
	resolved := []resolvedChunk{
		{entry: &manifest.ManifestEntry{Heading: "Auth  >  JWT"}},
	}
	got := topicSlug(resolved)
	if got != "auth-jwt" {
		t.Errorf("got %q, want %q", got, "auth-jwt")
	}
}

func TestTopicSlug_OnlySpecialChars(t *testing.T) {
	resolved := []resolvedChunk{
		{entry: &manifest.ManifestEntry{Heading: "@#$%^&*()"}},
	}
	got := topicSlug(resolved)
	if got != "generated" {
		t.Errorf("got %q, want %q", got, "generated")
	}
}

func TestTopicSlug_Truncation(t *testing.T) {
	// Create a heading that produces a slug > 60 chars.
	resolved := []resolvedChunk{
		{entry: &manifest.ManifestEntry{Heading: "Authentication > JWT Tokens > Refresh Flow > Token Rotation > Expiry Strategy > Fallback"}},
	}
	got := topicSlug(resolved)
	if len(got) > 60 {
		t.Errorf("slug length %d exceeds 60: %q", len(got), got)
	}
	// Should not end with a hyphen after truncation.
	if got[len(got)-1] == '-' {
		t.Errorf("slug ends with hyphen: %q", got)
	}
}

func TestTopicSlug_Numbers(t *testing.T) {
	resolved := []resolvedChunk{
		{entry: &manifest.ManifestEntry{Heading: "API v2 > Endpoint 42"}},
	}
	got := topicSlug(resolved)
	if got != "api-v2-endpoint-42" {
		t.Errorf("got %q, want %q", got, "api-v2-endpoint-42")
	}
}

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
