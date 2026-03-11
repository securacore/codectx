package generate

import (
	"testing"
)

func TestParseChunkIDs_CommaSeparated(t *testing.T) {
	got := parseChunkIDs("obj:a1b2c3.03,spec:f7g8h9.02,sys:d4e5f6.01")
	want := []string{"obj:a1b2c3.03", "spec:f7g8h9.02", "sys:d4e5f6.01"}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, id := range got {
		if id != want[i] {
			t.Errorf("parseChunkIDs[%d] = %q, want %q", i, id, want[i])
		}
	}
}

func TestParseChunkIDs_TrimsWhitespace(t *testing.T) {
	got := parseChunkIDs(" obj:a1b2c3.03 , spec:f7g8h9.02 ")

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
	got := parseChunkIDs("obj:a1b2c3.03,,, ,spec:f7g8h9.02,")

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %v", len(got), got)
	}
}

func TestParseChunkIDs_EmptyString(t *testing.T) {
	got := parseChunkIDs("")
	if len(got) != 0 {
		t.Errorf("expected empty result for empty input, got %v", got)
	}
}

func TestParseChunkIDs_SingleID(t *testing.T) {
	got := parseChunkIDs("obj:a1b2c3.03")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0] != "obj:a1b2c3.03" {
		t.Errorf("[0] = %q, want %q", got[0], "obj:a1b2c3.03")
	}
}

func TestParseChunkIDs_OnlyWhitespace(t *testing.T) {
	got := parseChunkIDs("  ,  ,  ")
	if len(got) != 0 {
		t.Errorf("expected empty result for whitespace-only input, got %v", got)
	}
}
