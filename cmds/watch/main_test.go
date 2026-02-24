package watch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "watch", Command.Name)
	assert.NotEmpty(t, Command.Usage)
}

func TestRun_missingConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	err = run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestRun_invalidConfig(t *testing.T) {
	dir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	require.NoError(t, os.Chdir(dir))

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, configFile),
		[]byte("{{{{not valid"), 0o644))

	err = run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load config")
}

func TestJoinParts(t *testing.T) {
	tests := []struct {
		name   string
		parts  []string
		expect string
	}{
		{"empty", nil, ""},
		{"single", []string{"3 objects"}, "3 objects"},
		{"two", []string{"3 objects", "1 pruned"}, "3 objects, 1 pruned"},
		{"three", []string{"3 objects", "1 pruned", "2 packages"}, "3 objects, 1 pruned, 2 packages"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, joinParts(tt.parts))
		})
	}
}
