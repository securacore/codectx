package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_returnsResultForEachProvider(t *testing.T) {
	results := Detect()
	assert.Len(t, results, len(Providers))

	for i, r := range results {
		assert.Equal(t, Providers[i].ID, r.Provider.ID)
		assert.Equal(t, Providers[i].Name, r.Provider.Name)
		assert.Equal(t, Providers[i].Binary, r.Provider.Binary)
	}
}

func TestDetect_foundResultsHavePath(t *testing.T) {
	results := Detect()
	for _, r := range results {
		if r.Found {
			assert.NotEmpty(t, r.Path, "found provider %s should have a path", r.Provider.ID)
		} else {
			assert.Empty(t, r.Path, "not-found provider %s should have empty path", r.Provider.ID)
		}
	}
}

func TestDetectProvider_knownBinary(t *testing.T) {
	// "ls" exists on every Unix-like system; use it to validate the
	// detection path without depending on AI tools being installed.
	p := Provider{ID: "test", Name: "Test", Binary: "ls"}
	result := DetectProvider(p)

	assert.True(t, result.Found)
	assert.NotEmpty(t, result.Path)
	assert.Equal(t, "test", result.Provider.ID)
}

func TestDetectProvider_nonexistentBinary(t *testing.T) {
	p := Provider{ID: "fake", Name: "Fake", Binary: "codectx-nonexistent-binary-xyz"}
	result := DetectProvider(p)

	assert.False(t, result.Found)
	assert.Empty(t, result.Path)
	assert.Equal(t, "fake", result.Provider.ID)
}

func TestFound_filtersToFoundOnly(t *testing.T) {
	results := []DetectionResult{
		{Provider: Provider{ID: "a"}, Found: true, Path: "/usr/bin/a"},
		{Provider: Provider{ID: "b"}, Found: false},
		{Provider: Provider{ID: "c"}, Found: true, Path: "/usr/bin/c"},
		{Provider: Provider{ID: "d"}, Found: false},
	}

	found := Found(results)
	require.Len(t, found, 2)
	assert.Equal(t, "a", found[0].Provider.ID)
	assert.Equal(t, "c", found[1].Provider.ID)
}

func TestFound_allFound(t *testing.T) {
	results := []DetectionResult{
		{Provider: Provider{ID: "a"}, Found: true, Path: "/usr/bin/a"},
		{Provider: Provider{ID: "b"}, Found: true, Path: "/usr/bin/b"},
	}

	found := Found(results)
	assert.Len(t, found, 2)
}

func TestFound_noneFound(t *testing.T) {
	results := []DetectionResult{
		{Provider: Provider{ID: "a"}, Found: false},
		{Provider: Provider{ID: "b"}, Found: false},
	}

	found := Found(results)
	assert.Empty(t, found)
}

func TestFound_emptyInput(t *testing.T) {
	found := Found(nil)
	assert.Empty(t, found)
}
