package query

import (
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/testutil"
)

func TestResolveTopN_FlagValueOverrides(t *testing.T) {
	// When flag value is positive, it should be returned directly.
	got := resolveTopN(5, "/nonexistent", nil)
	if got != 5 {
		t.Errorf("resolveTopN(5, ...) = %d, want 5", got)
	}
}

func TestResolveTopN_FlagValueZeroUsesDefault(t *testing.T) {
	// When flag is 0 and no AI config exists, should use DefaultResultsCount.
	got := resolveTopN(0, "/nonexistent", nil)
	if got != project.DefaultResultsCount {
		t.Errorf("resolveTopN(0, ...) = %d, want %d", got, project.DefaultResultsCount)
	}
}

func TestResolveTopN_NegativeUsesDefault(t *testing.T) {
	got := resolveTopN(-1, "/nonexistent", nil)
	if got != project.DefaultResultsCount {
		t.Errorf("resolveTopN(-1, ...) = %d, want %d", got, project.DefaultResultsCount)
	}
}

func TestResolveTopN_AIConfigOverridesDefault(t *testing.T) {
	// Set up a project with an AI config that has results_count.
	dir := t.TempDir()
	root := "docs"

	// Write codectx.yml.
	testutil.MustWriteFile(t, filepath.Join(dir, root, "codectx.yml"), `name: test
org: test
version: "0.1.0"
root: docs
`)

	// Write ai.yml with custom results_count.
	testutil.MustWriteFile(t, filepath.Join(dir, root, project.CodectxDir, "ai.yml"), `consumption:
  results_count: 25
`)

	cfg := &project.Config{Root: root}
	got := resolveTopN(0, dir, cfg)

	if got != 25 {
		t.Errorf("resolveTopN(0, ...) with AI config = %d, want 25", got)
	}
}

func TestResolveTopN_FlagOverridesAIConfig(t *testing.T) {
	// Even with AI config, flag should take precedence.
	dir := t.TempDir()
	root := "docs"

	testutil.MustWriteFile(t, filepath.Join(dir, root, "codectx.yml"), `name: test
org: test
version: "0.1.0"
root: docs
`)
	testutil.MustWriteFile(t, filepath.Join(dir, root, project.CodectxDir, "ai.yml"), `consumption:
  results_count: 25
`)

	cfg := &project.Config{Root: root}
	got := resolveTopN(3, dir, cfg)

	if got != 3 {
		t.Errorf("resolveTopN(3, ...) with AI config = %d, want 3", got)
	}
}
