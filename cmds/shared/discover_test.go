package shared

import (
	"os"
	"testing"
)

func TestDiscoverProject_NotFound(t *testing.T) {
	t.Parallel()

	// Run in a temp dir that has no codectx.yml anywhere up the tree.
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	_, _, err := DiscoverProject()
	if err == nil {
		t.Error("expected error when no project exists")
	}
}
