package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectFilePaths_allSections(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md", Spec: "application/arch/spec/README.md", Files: []string{"application/arch/decisions.md"}},
		},
		Topics: []manifest.TopicEntry{
			{ID: "b", Path: "topics/b/README.md", Spec: "topics/b/spec/README.md", Files: []string{"topics/b/extra.md"}},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "c", Path: "prompts/c/README.md"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "d", Path: "plans/d/README.md", State: "plans/d/state.yml"},
		},
	}

	paths := collectFilePaths(m)

	expected := []string{
		"foundation/a.md",
		"application/arch/README.md",
		"application/arch/spec/README.md",
		"application/arch/decisions.md",
		"topics/b/README.md",
		"topics/b/spec/README.md",
		"topics/b/extra.md",
		"prompts/c/README.md",
		"plans/d/README.md",
		"plans/d/state.yml",
	}
	assert.Equal(t, expected, paths)
}

func TestCollectFilePaths_topicWithoutSpec(t *testing.T) {
	m := &manifest.Manifest{
		Topics: []manifest.TopicEntry{
			{ID: "b", Path: "topics/b/README.md"},
		},
	}

	paths := collectFilePaths(m)

	assert.Equal(t, []string{"topics/b/README.md"}, paths)
}

func TestCollectFilePaths_applicationWithoutSpec(t *testing.T) {
	m := &manifest.Manifest{
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md"},
		},
	}

	paths := collectFilePaths(m)

	assert.Equal(t, []string{"application/arch/README.md"}, paths)
}

func TestCollectFilePaths_planWithoutState(t *testing.T) {
	m := &manifest.Manifest{
		Plans: []manifest.PlanEntry{
			{ID: "d", Path: "plans/d/README.md"},
		},
	}

	paths := collectFilePaths(m)

	assert.Equal(t, []string{"plans/d/README.md"}, paths)
}

func TestCollectFilePaths_empty(t *testing.T) {
	m := &manifest.Manifest{}
	paths := collectFilePaths(m)
	assert.Nil(t, paths)
}

// --- copyFile ---

func TestCopyFile_success(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.md")
	dst := filepath.Join(dir, "out", "dst.md")

	require.NoError(t, os.WriteFile(src, []byte("hello"), 0o644))

	err := copyFile(src, dst)
	require.NoError(t, err)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), data)
}

func TestCopyFile_missingSource(t *testing.T) {
	dir := t.TempDir()
	err := copyFile(filepath.Join(dir, "missing.md"), filepath.Join(dir, "dst.md"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestCopyFile_createsParentDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.md")
	dst := filepath.Join(dir, "a", "b", "c", "dst.md")

	require.NoError(t, os.WriteFile(src, []byte("deep"), 0o644))

	err := copyFile(src, dst)
	require.NoError(t, err)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte("deep"), data)
}

func TestCopyFile_failsMkdirAll(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.md")
	require.NoError(t, os.WriteFile(src, []byte("content"), 0o644))

	// Place a regular file where MkdirAll needs to create a directory.
	blocker := filepath.Join(dir, "out")
	require.NoError(t, os.WriteFile(blocker, []byte("not a dir"), 0o644))

	dst := filepath.Join(blocker, "sub", "dst.md")
	err := copyFile(src, dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create directory")
}

// --- copyManifestFiles ---

func TestCopyManifestFiles_copiesAll(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	// Create source files matching manifest paths.
	require.NoError(t, os.MkdirAll(filepath.Join(srcRoot, "foundation"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(srcRoot, "topics", "go"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "foundation", "a.md"), []byte("found"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "topics", "go", "README.md"), []byte("topic"), 0o644))

	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md"},
		},
	}

	copied, err := copyManifestFiles(m, srcRoot, dstRoot)
	require.NoError(t, err)
	assert.Equal(t, 2, copied)

	// Verify files exist at destination.
	data, err := os.ReadFile(filepath.Join(dstRoot, "foundation", "a.md"))
	require.NoError(t, err)
	assert.Equal(t, []byte("found"), data)

	data, err = os.ReadFile(filepath.Join(dstRoot, "topics", "go", "README.md"))
	require.NoError(t, err)
	assert.Equal(t, []byte("topic"), data)
}

func TestCopyManifestFiles_skipsMissing(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	// Create only one of two referenced files.
	require.NoError(t, os.MkdirAll(filepath.Join(srcRoot, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "foundation", "a.md"), []byte("exists"), 0o644))

	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md"},
			{ID: "b", Path: "foundation/b.md"}, // does not exist on disk
		},
	}

	copied, err := copyManifestFiles(m, srcRoot, dstRoot)
	require.NoError(t, err)
	assert.Equal(t, 1, copied)
}

func TestCopyManifestFiles_emptyManifest(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	m := &manifest.Manifest{}

	copied, err := copyManifestFiles(m, srcRoot, dstRoot)
	require.NoError(t, err)
	assert.Equal(t, 0, copied)
}

func TestCopyManifestFiles_allSections(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	// Create files for all sections.
	for _, dir := range []string{"foundation", "application/arch", "application/arch/spec", "topics/go", "topics/go/spec", "prompts/lint", "plans/migrate"} {
		require.NoError(t, os.MkdirAll(filepath.Join(srcRoot, dir), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "foundation", "a.md"), []byte("f"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "application", "arch", "README.md"), []byte("app"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "application", "arch", "spec", "README.md"), []byte("appspec"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "application", "arch", "decisions.md"), []byte("dec"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "topics", "go", "README.md"), []byte("t"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "topics", "go", "spec", "README.md"), []byte("s"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "topics", "go", "extra.md"), []byte("e"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "prompts", "lint", "README.md"), []byte("p"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "plans", "migrate", "README.md"), []byte("pl"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, "plans", "migrate", "state.yml"), []byte("st"), 0o644))

	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md"},
		},
		Application: []manifest.ApplicationEntry{
			{ID: "arch", Path: "application/arch/README.md", Spec: "application/arch/spec/README.md", Files: []string{"application/arch/decisions.md"}},
		},
		Topics: []manifest.TopicEntry{
			{ID: "go", Path: "topics/go/README.md", Spec: "topics/go/spec/README.md", Files: []string{"topics/go/extra.md"}},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "lint", Path: "prompts/lint/README.md"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "migrate", Path: "plans/migrate/README.md", State: "plans/migrate/state.yml"},
		},
	}

	copied, err := copyManifestFiles(m, srcRoot, dstRoot)
	require.NoError(t, err)
	assert.Equal(t, 10, copied)
}

func TestCopyManifestFiles_errorDuringCopy(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	// Create a source file.
	require.NoError(t, os.MkdirAll(filepath.Join(srcRoot, "foundation"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(srcRoot, "foundation", "a.md"), []byte("content"), 0o644))

	// Place a regular file at dstRoot/foundation so MkdirAll fails
	// when copyFile tries to create the parent directory.
	require.NoError(t, os.WriteFile(
		filepath.Join(dstRoot, "foundation"), []byte("blocker"), 0o644))

	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md"},
		},
	}

	_, err := copyManifestFiles(m, srcRoot, dstRoot)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "copy")
}
