package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviders_count(t *testing.T) {
	assert.Len(t, Providers, 3)

	ids := make(map[string]bool)
	for _, p := range Providers {
		ids[p.ID] = true
	}

	assert.True(t, ids["claude"], "should contain claude")
	assert.True(t, ids["opencode"], "should contain opencode")
	assert.True(t, ids["ollama"], "should contain ollama")
}

func TestProviders_entries(t *testing.T) {
	expected := []struct {
		ID     string
		Name   string
		Binary string
	}{
		{ID: "claude", Name: "Claude Code", Binary: "claude"},
		{ID: "opencode", Name: "opencode", Binary: "opencode"},
		{ID: "ollama", Name: "Ollama", Binary: "ollama"},
	}

	require.Len(t, Providers, len(expected))
	for i, exp := range expected {
		assert.Equal(t, exp.ID, Providers[i].ID, "provider %d ID", i)
		assert.Equal(t, exp.Name, Providers[i].Name, "provider %d Name", i)
		assert.Equal(t, exp.Binary, Providers[i].Binary, "provider %d Binary", i)
	}
}

func TestProviders_uniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range Providers {
		assert.False(t, seen[p.ID], "duplicate provider ID: %s", p.ID)
		seen[p.ID] = true
	}
}

func TestProviders_uniqueBinaries(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range Providers {
		assert.False(t, seen[p.Binary], "duplicate provider binary: %s", p.Binary)
		seen[p.Binary] = true
	}
}

func TestProviderByID_found(t *testing.T) {
	tests := []struct {
		id       string
		wantName string
	}{
		{id: "claude", wantName: "Claude Code"},
		{id: "opencode", wantName: "opencode"},
		{id: "ollama", wantName: "Ollama"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			p, ok := ProviderByID(tt.id)
			require.True(t, ok)
			assert.Equal(t, tt.wantName, p.Name)
			assert.Equal(t, tt.id, p.ID)
		})
	}
}

func TestProviderByID_notFound(t *testing.T) {
	p, ok := ProviderByID("nonexistent")
	assert.False(t, ok)
	assert.Zero(t, p)
}

func TestProviderByID_emptyString(t *testing.T) {
	p, ok := ProviderByID("")
	assert.False(t, ok)
	assert.Zero(t, p)
}
