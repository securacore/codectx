package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- ToSet ---

func TestToSet(t *testing.T) {
	s := ToSet([]string{"a", "b", "c"})
	assert.True(t, s["a"])
	assert.True(t, s["b"])
	assert.True(t, s["c"])
	assert.False(t, s["d"])
}

func TestToSet_empty(t *testing.T) {
	s := ToSet([]string{})
	assert.Len(t, s, 0)
}

func TestToSet_nil(t *testing.T) {
	s := ToSet(nil)
	assert.Len(t, s, 0)
}

func TestToSet_duplicates(t *testing.T) {
	s := ToSet([]string{"a", "a", "b"})
	assert.Len(t, s, 2)
	assert.True(t, s["a"])
	assert.True(t, s["b"])
}

// --- SplitKey ---

func TestSplitKey(t *testing.T) {
	section, id := SplitKey("topics:react")
	assert.Equal(t, "topics", section)
	assert.Equal(t, "react", id)
}

func TestSplitKey_noColon(t *testing.T) {
	section, id := SplitKey("noprefix")
	assert.Equal(t, "noprefix", section)
	assert.Equal(t, "", id)
}

func TestSplitKey_emptyString(t *testing.T) {
	section, id := SplitKey("")
	assert.Equal(t, "", section)
	assert.Equal(t, "", id)
}

func TestSplitKey_colonAtStart(t *testing.T) {
	section, id := SplitKey(":foo")
	assert.Equal(t, "", section)
	assert.Equal(t, "foo", id)
}

func TestSplitKey_colonAtEnd(t *testing.T) {
	section, id := SplitKey("foo:")
	assert.Equal(t, "foo", section)
	assert.Equal(t, "", id)
}

func TestSplitKey_multipleColons(t *testing.T) {
	section, id := SplitKey("foundation:a:b")
	assert.Equal(t, "foundation", section)
	assert.Equal(t, "a:b", id)
}

// --- KeyID ---

func TestKeyID(t *testing.T) {
	assert.Equal(t, "philosophy", KeyID("foundation:philosophy"))
	assert.Equal(t, "react", KeyID("topics:react"))
}

func TestKeyID_noColon(t *testing.T) {
	assert.Equal(t, "noprefix", KeyID("noprefix"))
}

func TestKeyID_emptyString(t *testing.T) {
	assert.Equal(t, "", KeyID(""))
}

func TestKeyID_colonAtStart(t *testing.T) {
	assert.Equal(t, "foo", KeyID(":foo"))
}

func TestKeyID_multipleColons(t *testing.T) {
	assert.Equal(t, "a:b", KeyID("foundation:a:b"))
}
