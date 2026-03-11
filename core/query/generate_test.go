package query_test

import (
	"os"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/query"
)

// cleanupGenerated registers a cleanup function to remove the generated file.
func cleanupGenerated(t *testing.T, path string) {
	t.Helper()
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
}

// ---------------------------------------------------------------------------
// RunGenerate
// ---------------------------------------------------------------------------

func TestRunGenerate_Basic(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunGenerate(compiledDir, "cl100k_base", []string{"obj:abc123.1"})
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}
	cleanupGenerated(t, result.FilePath)

	// Verify result fields.
	if result.FilePath == "" {
		t.Error("FilePath should not be empty")
	}
	if result.TotalTokens <= 0 {
		t.Error("TotalTokens should be positive")
	}
	if len(result.ChunkIDs) != 1 || result.ChunkIDs[0] != "obj:abc123.1" {
		t.Errorf("ChunkIDs = %v, want [obj:abc123.1]", result.ChunkIDs)
	}
	if len(result.Sources) != 1 || result.Sources[0] != "topics/auth.md" {
		t.Errorf("Sources = %v, want [topics/auth.md]", result.Sources)
	}

	// Verify file was written.
	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<!-- codectx:generated") {
		t.Error("missing generated header")
	}
	if !strings.Contains(content, "# Instructions") {
		t.Error("missing Instructions section")
	}
	if !strings.Contains(content, "Login") {
		t.Error("missing chunk content")
	}
}

func TestRunGenerate_MultipleChunks(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunGenerate(compiledDir, "cl100k_base", []string{
		"obj:abc123.1",
		"obj:abc123.2",
		"spec:def456.1",
	})
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}
	cleanupGenerated(t, result.FilePath)

	if len(result.ChunkIDs) != 3 {
		t.Errorf("ChunkIDs count = %d, want 3", len(result.ChunkIDs))
	}

	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	content := string(data)

	// Should have Instructions and Reasoning sections.
	if !strings.Contains(content, "# Instructions") {
		t.Error("missing Instructions section")
	}
	if !strings.Contains(content, "# Reasoning") {
		t.Error("missing Reasoning section")
	}

	// Reasoning section should have the preamble.
	if !strings.Contains(content, "reasoning behind") {
		t.Error("missing Reasoning preamble")
	}
}

func TestRunGenerate_SystemChunks(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunGenerate(compiledDir, "cl100k_base", []string{
		"obj:abc123.1",
		"sys:ghi789.1",
	})
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}
	cleanupGenerated(t, result.FilePath)

	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "# Instructions") {
		t.Error("missing Instructions section")
	}
	if !strings.Contains(content, "# System") {
		t.Error("missing System section")
	}
}

func TestRunGenerate_SortOrder(t *testing.T) {
	compiledDir := testFixture(t)

	// Provide chunks in reverse order -- they should be sorted correctly.
	result, err := query.RunGenerate(compiledDir, "cl100k_base", []string{
		"spec:def456.1",
		"sys:ghi789.1",
		"obj:abc123.2",
		"obj:abc123.1",
	})
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}
	cleanupGenerated(t, result.FilePath)

	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	content := string(data)

	// Instructions should appear before System, which appears before Reasoning.
	instIdx := strings.Index(content, "# Instructions")
	sysIdx := strings.Index(content, "# System")
	reasonIdx := strings.Index(content, "# Reasoning")

	if instIdx < 0 || sysIdx < 0 || reasonIdx < 0 {
		t.Fatalf("missing sections: inst=%d, sys=%d, reason=%d", instIdx, sysIdx, reasonIdx)
	}

	if instIdx > sysIdx {
		t.Error("Instructions should appear before System")
	}
	if sysIdx > reasonIdx {
		t.Error("System should appear before Reasoning")
	}
}

func TestRunGenerate_SourceSeparator(t *testing.T) {
	compiledDir := testFixture(t)

	// Object chunks from the same source should NOT have a separator between them.
	result, err := query.RunGenerate(compiledDir, "cl100k_base", []string{
		"obj:abc123.1",
		"obj:abc123.2",
	})
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}
	cleanupGenerated(t, result.FilePath)

	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	content := string(data)

	// Same source, so no "---" separator between chunks (only at footer).
	// The footer has "---\n<!-- codectx:related".
	footerCount := strings.Count(content, "---\n<!-- codectx:related")
	totalSeparators := strings.Count(content, "\n---\n")
	nonFooterSeparators := totalSeparators - footerCount
	if nonFooterSeparators > 0 {
		t.Errorf("unexpected separator between chunks from same source (found %d non-footer separators)", nonFooterSeparators)
	}
}

func TestRunGenerate_CollectsRelated(t *testing.T) {
	compiledDir := testFixture(t)

	// Generate with only obj:abc123.1 -- should have obj:abc123.2 as related
	// (via adjacency next link).
	result, err := query.RunGenerate(compiledDir, "cl100k_base", []string{"obj:abc123.1"})
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}
	cleanupGenerated(t, result.FilePath)

	// Related should contain adjacent chunks not in the request.
	hasRelated := false
	for _, rel := range result.Related {
		if rel.ChunkID == "obj:abc123.2" {
			hasRelated = true
			break
		}
	}

	if !hasRelated {
		t.Error("expected obj:abc123.2 in related chunks (adjacent next)")
	}
}

func TestRunGenerate_CollectsSources(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunGenerate(compiledDir, "cl100k_base", []string{
		"obj:abc123.1",
		"spec:def456.1",
	})
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}
	cleanupGenerated(t, result.FilePath)

	if len(result.Sources) != 2 {
		t.Errorf("Sources count = %d, want 2", len(result.Sources))
	}
}

func TestRunGenerate_ChunkNotFoundInManifest(t *testing.T) {
	compiledDir := testFixture(t)

	_, err := query.RunGenerate(compiledDir, "cl100k_base", []string{"obj:nonexistent.1"})
	if err == nil {
		t.Error("expected error for nonexistent chunk")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRunGenerate_ChunkFileNotOnDisk(t *testing.T) {
	compiledDir := testFixture(t)

	// Remove a chunk file to simulate missing file.
	chunkPath := strings.Join([]string{compiledDir, "objects", "abc123.1.md"}, string(os.PathSeparator))
	if err := os.Remove(chunkPath); err != nil {
		t.Fatalf("removing chunk file: %v", err)
	}

	_, err := query.RunGenerate(compiledDir, "cl100k_base", []string{"obj:abc123.1"})
	if err == nil {
		t.Error("expected error for missing chunk file")
	}
}

func TestRunGenerate_MissingManifest(t *testing.T) {
	dir := t.TempDir()

	_, err := query.RunGenerate(dir, "cl100k_base", []string{"obj:abc123.1"})
	if err == nil {
		t.Error("expected error for missing manifest")
	}
}

func TestRunGenerate_GeneratedFileHeader(t *testing.T) {
	compiledDir := testFixture(t)

	result, err := query.RunGenerate(compiledDir, "cl100k_base", []string{
		"obj:abc123.1",
		"spec:def456.1",
	})
	if err != nil {
		t.Fatalf("RunGenerate: %v", err)
	}
	cleanupGenerated(t, result.FilePath)

	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	content := string(data)

	// Check generated header includes chunk list.
	if !strings.Contains(content, "chunks: obj:abc123.1, spec:def456.1") {
		t.Error("missing chunk list in generated header")
	}

	// Check generated header includes sources.
	if !strings.Contains(content, "topics/auth.md") {
		t.Error("missing source in generated header")
	}

	// Check footer.
	if !strings.Contains(content, "<!-- codectx:related") {
		t.Error("missing related footer")
	}
}
