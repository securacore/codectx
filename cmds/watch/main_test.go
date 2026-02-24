package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/securacore/codectx/core/compile"
	corewatch "github.com/securacore/codectx/core/watch"
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

func TestPrintResult_error(t *testing.T) {
	r := corewatch.Result{
		Error:     fmt.Errorf("something went wrong"),
		Timestamp: time.Now(),
	}
	assert.NotPanics(t, func() { printResult(r) })
}

func TestPrintResult_nilCompiled(t *testing.T) {
	r := corewatch.Result{
		Compiled:  nil,
		Timestamp: time.Now(),
	}
	assert.NotPanics(t, func() { printResult(r) })
}

func TestPrintResult_upToDate(t *testing.T) {
	r := corewatch.Result{
		Compiled:  &compile.Result{UpToDate: true},
		Timestamp: time.Now(),
	}
	assert.NotPanics(t, func() { printResult(r) })
}

func TestPrintResult_normalCompile(t *testing.T) {
	r := corewatch.Result{
		Compiled:  &compile.Result{ObjectsStored: 5},
		Timestamp: time.Now(),
	}
	assert.NotPanics(t, func() { printResult(r) })
}

func TestPrintResult_withPrunedAndPackages(t *testing.T) {
	r := corewatch.Result{
		Compiled: &compile.Result{
			ObjectsStored: 3,
			ObjectsPruned: 2,
			Packages:      1,
		},
		Timestamp: time.Now(),
	}
	assert.NotPanics(t, func() { printResult(r) })
}

func TestPrintResult_withConflicts(t *testing.T) {
	r := corewatch.Result{
		Compiled: &compile.Result{
			ObjectsStored: 4,
			Dedup: compile.DeduplicationReport{
				Conflicts: []compile.ConflictEntry{
					{
						Section:    "foundation",
						ID:         "overview",
						WinnerPkg:  "local",
						SkippedPkg: "react@org",
						Reason:     "conflict",
					},
				},
			},
		},
		Timestamp: time.Now(),
	}
	assert.NotPanics(t, func() { printResult(r) })
}
