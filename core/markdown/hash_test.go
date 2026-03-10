package markdown

import (
	"testing"
)

func TestHash_Deterministic(t *testing.T) {
	content := []byte("Hello, world!")
	h1 := Hash(content)
	h2 := Hash(content)

	if h1 != h2 {
		t.Errorf("same content should produce same hash: %s != %s", h1, h2)
	}
}

func TestHash_DifferentContent(t *testing.T) {
	h1 := Hash([]byte("Content A"))
	h2 := Hash([]byte("Content B"))

	if h1 == h2 {
		t.Error("different content should produce different hashes")
	}
}

func TestHash_EmptyContent(t *testing.T) {
	h := Hash([]byte(""))
	if h == "" {
		t.Error("hash of empty content should not be empty string")
	}

	// SHA-256 of empty string is well-known.
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if h != expected {
		t.Errorf("hash of empty content: expected %s, got %s", expected, h)
	}
}

func TestHash_Length(t *testing.T) {
	h := Hash([]byte("test content"))

	// SHA-256 hex digest is always 64 characters.
	if len(h) != 64 {
		t.Errorf("expected hash length 64, got %d", len(h))
	}
}

func TestHash_HexCharacters(t *testing.T) {
	h := Hash([]byte("test"))
	for _, c := range h {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("hash should only contain hex characters, found %c", c)
		}
	}
}

func TestHash_WhitespaceMatters(t *testing.T) {
	h1 := Hash([]byte("hello"))
	h2 := Hash([]byte("hello "))
	h3 := Hash([]byte("hello\n"))

	if h1 == h2 {
		t.Error("trailing space should produce different hash")
	}
	if h1 == h3 {
		t.Error("trailing newline should produce different hash")
	}
}
