package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClasses_count(t *testing.T) {
	assert.Len(t, Classes, 3)

	ids := make(map[string]bool)
	for _, c := range Classes {
		ids[c.ID] = true
	}

	assert.True(t, ids["gpt-4o-class"], "should contain gpt-4o-class")
	assert.True(t, ids["claude-sonnet-class"], "should contain claude-sonnet-class")
	assert.True(t, ids["o1-class"], "should contain o1-class")
}

func TestClasses_entries(t *testing.T) {
	expected := []struct {
		ID   string
		Name string
	}{
		{ID: "gpt-4o-class", Name: "GPT-4o class"},
		{ID: "claude-sonnet-class", Name: "Claude Sonnet class"},
		{ID: "o1-class", Name: "o1 class"},
	}

	require.Len(t, Classes, len(expected))
	for i, exp := range expected {
		assert.Equal(t, exp.ID, Classes[i].ID, "class %d ID", i)
		assert.Equal(t, exp.Name, Classes[i].Name, "class %d Name", i)
		assert.NotEmpty(t, Classes[i].Description, "class %d Description", i)
	}
}

func TestClasses_uniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, c := range Classes {
		assert.False(t, seen[c.ID], "duplicate class ID: %s", c.ID)
		seen[c.ID] = true
	}
}

func TestClassByID_found(t *testing.T) {
	tests := []struct {
		id       string
		wantName string
	}{
		{id: "gpt-4o-class", wantName: "GPT-4o class"},
		{id: "claude-sonnet-class", wantName: "Claude Sonnet class"},
		{id: "o1-class", wantName: "o1 class"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			c, ok := ClassByID(tt.id)
			require.True(t, ok)
			assert.Equal(t, tt.wantName, c.Name)
			assert.Equal(t, tt.id, c.ID)
		})
	}
}

func TestClassByID_notFound(t *testing.T) {
	c, ok := ClassByID("nonexistent")
	assert.False(t, ok)
	assert.Zero(t, c)
}

func TestClassByID_emptyString(t *testing.T) {
	c, ok := ClassByID("")
	assert.False(t, ok)
	assert.Zero(t, c)
}
