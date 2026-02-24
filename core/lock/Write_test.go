package lock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWrite_createsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")

	lck := &Lock{
		CompiledAt: "2025-01-15",
		Packages: []LockedPackage{
			{
				Name:    "react",
				Author:  "facebook",
				Version: "1.2.0",
				Source:  "https://github.com/facebook/codectx-react.git",
				Active:  config.Activation{Mode: "all"},
			},
		},
	}

	err := Write(path, lck)
	require.NoError(t, err)

	// Verify file exists and has content.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)

	// Verify it's valid YAML that round-trips.
	var parsed Lock
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2025-01-15", parsed.CompiledAt)
	require.Len(t, parsed.Packages, 1)
	assert.Equal(t, "react", parsed.Packages[0].Name)
	assert.Equal(t, "1.2.0", parsed.Packages[0].Version)
}

func TestWrite_emptyPackages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")

	lck := &Lock{
		CompiledAt: "2025-01-15",
	}

	err := Write(path, lck)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed Lock
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Empty(t, parsed.Packages)
}

func TestWrite_roundTripAllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")

	lck := &Lock{
		CompiledAt: "2025-06-20",
		Packages: []LockedPackage{
			{
				Name:    "react",
				Author:  "facebook",
				Version: "1.2.0",
				Source:  "https://github.com/facebook/codectx-react.git",
				Active:  config.Activation{Mode: "all"},
			},
			{
				Name:    "go",
				Author:  "google",
				Version: "3.0.0",
				Source:  "https://github.com/google/codectx-go.git",
				Active:  config.Activation{Mode: "none"},
			},
			{
				Name:    "custom",
				Author:  "acme",
				Version: "0.5.0",
				Active: config.Activation{
					Map: &config.ActivationMap{
						Foundation: []string{"philosophy"},
						Topics:     []string{"react", "go"},
						Prompts:    []string{"review"},
						Plans:      []string{"migration"},
					},
				},
			},
		},
	}

	err := Write(path, lck)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed Lock
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "2025-06-20", parsed.CompiledAt)
	require.Len(t, parsed.Packages, 3)

	// First package: mode "all".
	assert.Equal(t, "react", parsed.Packages[0].Name)
	assert.Equal(t, "facebook", parsed.Packages[0].Author)
	assert.Equal(t, "1.2.0", parsed.Packages[0].Version)
	assert.Equal(t, "https://github.com/facebook/codectx-react.git", parsed.Packages[0].Source)
	assert.True(t, parsed.Packages[0].Active.IsAll())

	// Second package: mode "none".
	assert.Equal(t, "go", parsed.Packages[1].Name)
	assert.Equal(t, "google", parsed.Packages[1].Author)
	assert.Equal(t, "3.0.0", parsed.Packages[1].Version)
	assert.True(t, parsed.Packages[1].Active.IsNone())

	// Third package: granular activation.
	assert.Equal(t, "custom", parsed.Packages[2].Name)
	assert.Equal(t, "acme", parsed.Packages[2].Author)
	assert.True(t, parsed.Packages[2].Active.IsGranular())
	require.NotNil(t, parsed.Packages[2].Active.Map)
	assert.Equal(t, []string{"philosophy"}, parsed.Packages[2].Active.Map.Foundation)
	assert.Equal(t, []string{"react", "go"}, parsed.Packages[2].Active.Map.Topics)
	assert.Equal(t, []string{"review"}, parsed.Packages[2].Active.Map.Prompts)
	assert.Equal(t, []string{"migration"}, parsed.Packages[2].Active.Map.Plans)
}

func TestWrite_invalidPath(t *testing.T) {
	lck := &Lock{
		CompiledAt: "2025-01-15",
		Packages: []LockedPackage{
			{
				Name:    "test",
				Author:  "test",
				Version: "1.0.0",
				Active:  config.Activation{Mode: "all"},
			},
		},
	}

	err := Write("/nonexistent/path/codectx.lock", lck)
	assert.Error(t, err)
}

func TestWrite_overwriteExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")

	first := &Lock{
		CompiledAt: "2025-01-01T00:00:00Z",
		Packages: []LockedPackage{
			{Name: "old", Author: "org", Version: "1.0.0"},
		},
	}
	require.NoError(t, Write(path, first))

	second := &Lock{
		CompiledAt: "2025-06-01T00:00:00Z",
		Packages: []LockedPackage{
			{Name: "new", Author: "org", Version: "2.0.0"},
		},
	}
	require.NoError(t, Write(path, second))

	// Read back and verify second write replaced first.
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var loaded Lock
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	require.Len(t, loaded.Packages, 1)
	assert.Equal(t, "new", loaded.Packages[0].Name)
	assert.Equal(t, "2.0.0", loaded.Packages[0].Version)
}

func TestWrite_nilLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")

	err := Write(path, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestWrite_sourceOmitempty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codectx.lock")

	l := &Lock{
		CompiledAt: "2025-01-01T00:00:00Z",
		Packages: []LockedPackage{
			{Name: "nosource", Author: "org", Version: "1.0.0"},
		},
	}
	require.NoError(t, Write(path, l))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "source:")
}
