package lock

import (
	"os"
	"path/filepath"
	"testing"

	"securacore/codectx/core/config"

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
