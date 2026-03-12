package embed_test

import (
	"strings"
	"testing"

	"github.com/securacore/codectx/embed"
)

func TestSystemFiles_ReturnsAllExpectedFiles(t *testing.T) {
	files := embed.SystemFiles()

	if len(files) != 9 {
		t.Fatalf("expected 9 system files, got %d", len(files))
	}

	// Verify all dest paths start with "system/".
	for _, f := range files {
		if !strings.HasPrefix(f.DestPath, "system/") {
			t.Errorf("expected dest path to start with system/, got %q", f.DestPath)
		}
	}

	// Verify all embed paths start with "defaults/".
	for _, f := range files {
		if !strings.HasPrefix(f.EmbedPath, "defaults/") {
			t.Errorf("expected embed path to start with defaults/, got %q", f.EmbedPath)
		}
	}
}

func TestReadFile_ReadsEmbeddedContent(t *testing.T) {
	files := embed.SystemFiles()

	for _, f := range files {
		data, err := embed.ReadFile(f.EmbedPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", f.EmbedPath, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("expected %s to have content", f.EmbedPath)
		}
		// All default files should start with a markdown heading.
		if !strings.HasPrefix(string(data), "# ") {
			t.Errorf("expected %s to start with markdown heading, got %q", f.EmbedPath, string(data[:20]))
		}
	}
}

func TestReadFile_ErrorsOnMissingFile(t *testing.T) {
	_, err := embed.ReadFile("defaults/nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSystemFiles_DestPathsAreUnique(t *testing.T) {
	files := embed.SystemFiles()
	seen := make(map[string]bool)

	for _, f := range files {
		if seen[f.DestPath] {
			t.Errorf("duplicate dest path: %s", f.DestPath)
		}
		seen[f.DestPath] = true
	}
}

func TestSystemFiles_CoversExpectedTopics(t *testing.T) {
	files := embed.SystemFiles()

	expectedPaths := map[string]bool{
		"system/foundation/compiler-philosophy/README.md":      false,
		"system/foundation/compiler-philosophy/README.spec.md": false,
		"system/foundation/cli-usage/README.md":                false,
		"system/foundation/history/README.md":                  false,
		"system/topics/taxonomy-generation/README.md":          false,
		"system/topics/taxonomy-generation/README.spec.md":     false,
		"system/topics/bridge-summaries/README.md":             false,
		"system/topics/bridge-summaries/README.spec.md":        false,
		"system/topics/context-assembly/README.md":             false,
	}

	for _, f := range files {
		if _, ok := expectedPaths[f.DestPath]; ok {
			expectedPaths[f.DestPath] = true
		}
	}

	for path, found := range expectedPaths {
		if !found {
			t.Errorf("expected system file not found: %s", path)
		}
	}
}
