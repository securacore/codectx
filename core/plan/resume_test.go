package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/index"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/testutil"
)

// buildCompiledFixture creates a minimal compiled directory with manifest,
// hashes, BM25 indexes, and chunk files for testing resume operations.
func buildCompiledFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	chunks := []chunk.Chunk{
		{
			ID: "obj:abc123.01", Type: chunk.ChunkObject,
			Source: "topics/auth.md", Heading: "Auth > Login",
			Sequence: 1, TotalInFile: 2, Tokens: 400,
			Content: "JWT authentication login flow",
		},
		{
			ID: "obj:abc123.02", Type: chunk.ChunkObject,
			Source: "topics/auth.md", Heading: "Auth > Tokens",
			Sequence: 2, TotalInFile: 2, Tokens: 350,
			Content: "Token validation and rotation",
		},
		{
			ID: "spec:def456.01", Type: chunk.ChunkSpec,
			Source: "topics/auth.spec.md", Heading: "Auth > Login",
			Sequence: 1, TotalInFile: 1, Tokens: 250,
			Content: "Reasoning behind JWT choice",
		},
	}

	// BM25 indexes.
	idx := index.New(1.2, 0.75)
	idx.BuildFromChunks(chunks)
	if err := idx.Save(dir); err != nil {
		t.Fatalf("saving indexes: %v", err)
	}

	// Manifest.
	mfst := manifest.BuildManifest(chunks, "cl100k_base", nil, nil)
	if err := mfst.WriteTo(filepath.Join(dir, "manifest.yml")); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}

	// Hashes — simulate compiled hashes with known values.
	hashes := manifest.BuildHashes(map[string]string{
		"foundation/overview.md": "sha256:hash-foundation",
		"topics/auth.md":         "sha256:hash-auth",
		"topics/auth.spec.md":    "sha256:hash-auth-spec",
	}, nil)
	if err := hashes.WriteTo(filepath.Join(dir, "hashes.yml")); err != nil {
		t.Fatalf("writing hashes: %v", err)
	}

	// Chunk files.
	writeChunkFile(t, dir, "objects", "abc123.01.md",
		"<!-- codectx:meta\nid: obj:abc123.01\n-->\n\n## Login\n\nJWT auth login.")
	writeChunkFile(t, dir, "objects", "abc123.02.md",
		"<!-- codectx:meta\nid: obj:abc123.02\n-->\n\n## Tokens\n\nToken rotation.")
	writeChunkFile(t, dir, "specs", "def456.01.md",
		"<!-- codectx:meta\nid: spec:def456.01\n-->\n\n## Login Reasoning\n\nWhy JWT.")

	return dir
}

func writeChunkFile(t *testing.T, compiledDir, subdir, filename, content string) {
	t.Helper()
	dir := filepath.Join(compiledDir, subdir)
	if err := os.MkdirAll(dir, project.DirPerm); err != nil {
		t.Fatalf("creating chunk dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), project.FilePerm); err != nil {
		t.Fatalf("writing chunk file: %v", err)
	}
}

func TestResume_NoCurrentStep(t *testing.T) {
	t.Parallel()

	compiledDir := buildCompiledFixture(t)

	planDir := t.TempDir()
	planPath := filepath.Join(planDir, PlanFile)
	testutil.MustWriteFile(t, planPath, `name: "Draft Plan"
status: "draft"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 0
steps:
  - id: 1
    title: "Step one"
    status: "pending"
`)

	result, err := Resume(planPath, compiledDir, "cl100k_base")
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	if result.Plan == nil {
		t.Fatal("Plan should not be nil")
	}
	if result.Check != nil {
		t.Error("Check should be nil when no current step")
	}
	// Output should be the status format (fallback).
	output := stripANSI(result.Output)
	if !strings.Contains(output, "Draft Plan") {
		t.Errorf("output should contain plan name: %s", output)
	}
}

func TestResume_AllHashesMatch(t *testing.T) {
	t.Parallel()

	compiledDir := buildCompiledFixture(t)

	planDir := t.TempDir()
	planPath := filepath.Join(planDir, PlanFile)
	testutil.MustWriteFile(t, planPath, `name: "Auth Migration"
status: "in-progress"
created: "2025-03-01T00:00:00Z"
updated: "2025-03-09T00:00:00Z"
dependencies:
  - path: "foundation/overview"
    hash: "sha256:hash-foundation"
  - path: "topics/auth"
    hash: "sha256:hash-auth"
current_step: 1
steps:
  - id: 1
    title: "Audit JWT"
    status: "in-progress"
    started_at: "2025-03-07T09:00:00Z"
    notes: "In progress."
    queries:
      - "jwt audit"
    chunks:
      - "obj:abc123.01,obj:abc123.02"
`)

	result, err := Resume(planPath, compiledDir, "cl100k_base")
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	if result.Check == nil {
		t.Fatal("Check should not be nil")
	}
	if !result.Check.AllMatch {
		t.Error("AllMatch should be true")
	}
	if len(result.GenerateResults) == 0 {
		t.Error("GenerateResults should be populated on match")
	}

	// Output should include "all unchanged" and "Replaying".
	output := stripANSI(result.Output)
	if !strings.Contains(output, "all unchanged") {
		t.Errorf("output missing 'all unchanged': %s", output)
	}
	if !strings.Contains(output, "Replaying context") {
		t.Errorf("output missing 'Replaying context': %s", output)
	}
	if !strings.Contains(output, "Audit JWT") {
		t.Errorf("output missing current step title: %s", output)
	}

	// Clean up generated files.
	for _, gr := range result.GenerateResults {
		_ = os.Remove(gr.FilePath)
	}
}

func TestResume_HashesDrifted(t *testing.T) {
	t.Parallel()

	compiledDir := buildCompiledFixture(t)

	planDir := t.TempDir()
	planPath := filepath.Join(planDir, PlanFile)
	testutil.MustWriteFile(t, planPath, `name: "Auth Migration"
status: "in-progress"
created: "2025-03-01T00:00:00Z"
updated: "2025-03-09T00:00:00Z"
dependencies:
  - path: "foundation/overview"
    hash: "sha256:hash-foundation"
  - path: "topics/auth"
    hash: "sha256:old-hash-that-doesnt-match"
current_step: 1
steps:
  - id: 1
    title: "Audit JWT"
    status: "in-progress"
    queries:
      - "jwt audit query"
      - "token validation"
    chunks:
      - "obj:abc123.01"
`)

	result, err := Resume(planPath, compiledDir, "cl100k_base")
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	if result.Check.AllMatch {
		t.Error("AllMatch should be false")
	}
	if result.Check.ChangedCount != 1 {
		t.Errorf("ChangedCount = %d, want 1", result.Check.ChangedCount)
	}
	if len(result.GenerateResults) != 0 {
		t.Error("GenerateResults should be empty on drift")
	}

	// Output should include drift report and stored queries.
	output := stripANSI(result.Output)
	if !strings.Contains(output, "Documentation changes") {
		t.Errorf("output missing drift report: %s", output)
	}
	if !strings.Contains(output, "topics/auth") {
		t.Errorf("output missing changed dependency: %s", output)
	}
	if !strings.Contains(output, "jwt audit query") {
		t.Errorf("output missing stored query: %s", output)
	}
	if !strings.Contains(output, "Recommendation") {
		t.Errorf("output missing recommendation: %s", output)
	}
}

func TestResume_NoDependencies(t *testing.T) {
	t.Parallel()

	compiledDir := buildCompiledFixture(t)

	planDir := t.TempDir()
	planPath := filepath.Join(planDir, PlanFile)
	testutil.MustWriteFile(t, planPath, `name: "Simple Plan"
status: "in-progress"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 1
steps:
  - id: 1
    title: "Only step"
    status: "in-progress"
    chunks:
      - "obj:abc123.01"
`)

	result, err := Resume(planPath, compiledDir, "cl100k_base")
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	// No dependencies means AllMatch is true (vacuously).
	if !result.Check.AllMatch {
		t.Error("AllMatch should be true when no dependencies")
	}

	// Clean up generated files.
	for _, gr := range result.GenerateResults {
		_ = os.Remove(gr.FilePath)
	}
}

func TestResume_StepWithNoChunks(t *testing.T) {
	t.Parallel()

	compiledDir := buildCompiledFixture(t)

	planDir := t.TempDir()
	planPath := filepath.Join(planDir, PlanFile)
	testutil.MustWriteFile(t, planPath, `name: "New Plan"
status: "in-progress"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
dependencies:
  - path: "foundation/overview"
    hash: "sha256:hash-foundation"
current_step: 1
steps:
  - id: 1
    title: "Just started"
    status: "in-progress"
    notes: "Just started, no chunks yet."
`)

	result, err := Resume(planPath, compiledDir, "cl100k_base")
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	// Hashes match, but no chunks to replay.
	if !result.Check.AllMatch {
		t.Error("AllMatch should be true")
	}
	if len(result.GenerateResults) != 0 {
		t.Error("GenerateResults should be empty when no chunks")
	}
	output := stripANSI(result.Output)
	if !strings.Contains(output, "all unchanged") {
		t.Errorf("output should show deps unchanged: %s", output)
	}
}

func TestResume_InvalidPlanPath(t *testing.T) {
	t.Parallel()

	_, err := Resume("/nonexistent/plan.yml", "/tmp", "cl100k_base")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestResume_MissingHashesFile(t *testing.T) {
	t.Parallel()

	planDir := t.TempDir()
	planPath := filepath.Join(planDir, PlanFile)
	testutil.MustWriteFile(t, planPath, `name: "Plan"
status: "in-progress"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
dependencies:
  - path: "some/dep"
    hash: "sha256:abc"
current_step: 1
steps:
  - id: 1
    title: "Step"
    status: "in-progress"
`)

	// compiledDir points to empty temp dir — no hashes.yml.
	emptyDir := t.TempDir()
	_, err := Resume(planPath, emptyDir, "cl100k_base")
	if err == nil {
		t.Error("expected error when hashes.yml is missing")
	}
}

func TestResume_MultipleChunkEntries(t *testing.T) {
	t.Parallel()

	compiledDir := buildCompiledFixture(t)

	planDir := t.TempDir()
	planPath := filepath.Join(planDir, PlanFile)
	testutil.MustWriteFile(t, planPath, `name: "Multi-Chunk Plan"
status: "in-progress"
created: "2025-01-01T00:00:00Z"
updated: "2025-01-01T00:00:00Z"
current_step: 1
steps:
  - id: 1
    title: "Multi-generate step"
    status: "in-progress"
    chunks:
      - "obj:abc123.01"
      - "obj:abc123.02,spec:def456.01"
`)

	result, err := Resume(planPath, compiledDir, "cl100k_base")
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	// Should have two generate results (one per chunk entry).
	if len(result.GenerateResults) != 2 {
		t.Errorf("GenerateResults count = %d, want 2", len(result.GenerateResults))
	}

	// Clean up generated files.
	for _, gr := range result.GenerateResults {
		_ = os.Remove(gr.FilePath)
	}
}
