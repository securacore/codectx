package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWrite_createsNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	m := &Manifest{
		Name:    "test",
		Version: "1.0.0",
	}

	err := Write(path, m)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "name: test")
	assert.Contains(t, string(data), "version: 1.0.0")
}

func TestWrite_updatesChangedContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	m1 := &Manifest{Name: "v1", Version: "1.0.0"}
	require.NoError(t, Write(path, m1))

	m2 := &Manifest{Name: "v2", Version: "2.0.0"}
	require.NoError(t, Write(path, m2))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "name: v2")
	assert.Contains(t, string(data), "version: 2.0.0")
}

func TestWrite_skipsIdenticalContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	m := &Manifest{Name: "stable", Version: "1.0.0"}

	// First write creates the file.
	require.NoError(t, Write(path, m))

	info1, err := os.Stat(path)
	require.NoError(t, err)
	modTime1 := info1.ModTime()

	// Ensure enough time passes for mtime to differ if file were rewritten.
	time.Sleep(50 * time.Millisecond)

	// Second write with identical content should be skipped.
	require.NoError(t, Write(path, m))

	info2, err := os.Stat(path)
	require.NoError(t, err)
	modTime2 := info2.ModTime()

	assert.Equal(t, modTime1, modTime2, "mtime should not change when content is identical")
}

func TestWrite_roundTripStable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yml")

	// Write a manifest with entries across multiple sections.
	m := &Manifest{
		Name:        "roundtrip",
		Author:      "tester",
		Version:     "1.0.0",
		Description: "Test round-trip stability",
		Foundation: []FoundationEntry{
			{ID: "alpha", Path: "foundation/alpha.md", Description: "Alpha", DependsOn: []string{"beta"}},
			{ID: "beta", Path: "foundation/beta.md", Description: "Beta", RequiredBy: []string{"alpha"}},
		},
		Topics: []TopicEntry{
			{ID: "go", Path: "topics/go/README.md", Description: "Go", Spec: "topics/go/spec/README.md"},
		},
	}

	// First write.
	require.NoError(t, Write(path, m))
	data1, err := os.ReadFile(path)
	require.NoError(t, err)

	// Load and write again — should produce identical bytes.
	loaded, err := loadRaw(path)
	require.NoError(t, err)
	require.NoError(t, Write(path, loaded))
	data2, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, string(data1), string(data2), "round-trip should produce identical YAML")
}

// loadRaw reads a manifest without schema validation (for round-trip testing).
func loadRaw(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
