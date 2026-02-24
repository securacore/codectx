package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/securacore/codectx/core/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDedupDirs creates two package directories with test files.
func setupDedupDirs(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()

	pkgA := filepath.Join(dir, "pkgA")
	pkgB := filepath.Join(dir, "pkgB")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgA, "foundation"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgB, "foundation"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgA, "topics"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgB, "topics"), 0o755))

	return pkgA, pkgB
}

func TestMergeManifestDedup_noConflict(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)

	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "foundation/a.md"), []byte("doc A"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/b.md"), []byte("doc B"), 0o644))

	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md", Description: "A"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "b", Path: "foundation/b.md", Description: "B"},
		},
	}

	seen := map[string]seenEntry{
		"foundation:a": {pkg: "local", hash: fileHash(filepath.Join(pkgA, "foundation/a.md"))},
	}

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)

	assert.Empty(t, events)
	require.Len(t, dst.Foundation, 2)
	assert.Equal(t, "a", dst.Foundation[0].ID)
	assert.Equal(t, "b", dst.Foundation[1].ID)
}

func TestMergeManifestDedup_sameContentDedup(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)

	sameContent := []byte("shared philosophy document")
	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "foundation/philosophy.md"), sameContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/philosophy.md"), sameContent, 0o644))

	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Philosophy"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "philosophy", Path: "foundation/philosophy.md", Description: "Philosophy from B"},
		},
	}

	seen := map[string]seenEntry{
		"foundation:philosophy": {pkg: "local", hash: fileHash(filepath.Join(pkgA, "foundation/philosophy.md"))},
	}

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)

	require.Len(t, events, 1)
	assert.Equal(t, "duplicate", events[0].Reason)
	assert.Equal(t, "philosophy", events[0].ID)
	assert.Equal(t, "local", events[0].WinnerPkg)
	assert.Equal(t, "pkgB@org", events[0].SkippedPkg)

	// dst should NOT have the duplicate appended.
	require.Len(t, dst.Foundation, 1)
}

func TestMergeManifestDedup_differentContentConflict(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)

	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "foundation/conventions.md"), []byte("version A"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/conventions.md"), []byte("version B"), 0o644))

	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "conventions", Path: "foundation/conventions.md", Description: "Conventions A"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "conventions", Path: "foundation/conventions.md", Description: "Conventions B"},
		},
	}

	seen := map[string]seenEntry{
		"foundation:conventions": {pkg: "local", hash: fileHash(filepath.Join(pkgA, "foundation/conventions.md"))},
	}

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)

	require.Len(t, events, 1)
	assert.Equal(t, "conflict", events[0].Reason)
	assert.Equal(t, "conventions", events[0].ID)
	assert.Equal(t, "local", events[0].WinnerPkg)
	assert.Equal(t, "pkgB@org", events[0].SkippedPkg)

	// dst should NOT have the conflicting entry.
	require.Len(t, dst.Foundation, 1)
	assert.Equal(t, "Conventions A", dst.Foundation[0].Description)
}

func TestMergeManifestDedup_mixedSections(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)

	sameContent := []byte("shared")
	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "foundation/shared.md"), sameContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/shared.md"), sameContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "topics/new.md"), []byte("new topic"), 0o644))

	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "shared", Path: "foundation/shared.md"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "shared", Path: "foundation/shared.md"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "new", Path: "topics/new.md", Description: "New topic"},
		},
	}

	seen := map[string]seenEntry{
		"foundation:shared": {pkg: "local", hash: fileHash(filepath.Join(pkgA, "foundation/shared.md"))},
	}

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)

	// Foundation:shared should be deduped, topics:new should be added.
	require.Len(t, events, 1)
	assert.Equal(t, "duplicate", events[0].Reason)
	require.Len(t, dst.Foundation, 1)
	require.Len(t, dst.Topics, 1)
	assert.Equal(t, "new", dst.Topics[0].ID)
}

func TestMergeManifestDedup_precedenceOrder(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)

	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "foundation/doc.md"), []byte("first"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/doc.md"), []byte("second"), 0o644))

	// Simulate: pkgA was merged first (it's in seen), now pkgB tries to merge same ID.
	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "doc", Path: "foundation/doc.md", Description: "From A"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "doc", Path: "foundation/doc.md", Description: "From B"},
		},
	}

	seen := map[string]seenEntry{
		"foundation:doc": {pkg: "pkgA@org", hash: fileHash(filepath.Join(pkgA, "foundation/doc.md"))},
	}

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)

	require.Len(t, events, 1)
	assert.Equal(t, "conflict", events[0].Reason)
	assert.Equal(t, "pkgA@org", events[0].WinnerPkg)
	assert.Equal(t, "pkgB@org", events[0].SkippedPkg)

	// dst still has only the original entry.
	require.Len(t, dst.Foundation, 1)
	assert.Equal(t, "From A", dst.Foundation[0].Description)
}

func TestMergeManifestDedup_emptySource(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)
	dst := &manifest.Manifest{}
	src := &manifest.Manifest{}
	seen := make(map[string]seenEntry)

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "empty@org", seen)

	assert.Empty(t, events)
	assert.Empty(t, dst.Foundation)
	assert.Empty(t, dst.Topics)
}

func TestMergeManifestDedup_allSections(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)

	require.NoError(t, os.MkdirAll(filepath.Join(pkgB, "prompts"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgB, "plans"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/f.md"), []byte("f"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "topics/t.md"), []byte("t"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "prompts/p.md"), []byte("p"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "plans/pl.md"), []byte("pl"), 0o644))

	dst := &manifest.Manifest{}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "f", Path: "foundation/f.md"}},
		Topics:     []manifest.TopicEntry{{ID: "t", Path: "topics/t.md"}},
		Prompts:    []manifest.PromptEntry{{ID: "p", Path: "prompts/p.md"}},
		Plans:      []manifest.PlanEntry{{ID: "pl", Path: "plans/pl.md"}},
	}
	seen := make(map[string]seenEntry)

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)

	assert.Empty(t, events)
	assert.Len(t, dst.Foundation, 1)
	assert.Len(t, dst.Topics, 1)
	assert.Len(t, dst.Prompts, 1)
	assert.Len(t, dst.Plans, 1)

	// Verify seen map was populated.
	assert.Contains(t, seen, "foundation:f")
	assert.Contains(t, seen, "topics:t")
	assert.Contains(t, seen, "prompts:p")
	assert.Contains(t, seen, "plans:pl")
}

// --- DeduplicationReport ---

func TestDeduplicationReport_hasConflicts(t *testing.T) {
	r := &DeduplicationReport{
		Conflicts: []ConflictEntry{{Reason: "conflict"}},
	}
	assert.True(t, r.HasConflicts())
}

func TestDeduplicationReport_noConflicts(t *testing.T) {
	r := &DeduplicationReport{
		Duplicates: []ConflictEntry{{Reason: "duplicate"}},
	}
	assert.False(t, r.HasConflicts())
}

func TestDeduplicationReport_total(t *testing.T) {
	r := &DeduplicationReport{
		Duplicates: []ConflictEntry{{}, {}},
		Conflicts:  []ConflictEntry{{}},
	}
	assert.Equal(t, 3, r.Total())
}

// --- CollectActiveIDs ---

func TestCollectActiveIDs(t *testing.T) {
	m := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a"}, {ID: "b"},
		},
		Topics: []manifest.TopicEntry{
			{ID: "c"},
		},
		Prompts: []manifest.PromptEntry{
			{ID: "d"},
		},
		Plans: []manifest.PlanEntry{
			{ID: "e"},
		},
	}

	ids := CollectActiveIDs(m)

	assert.True(t, ids["foundation:a"])
	assert.True(t, ids["foundation:b"])
	assert.True(t, ids["topics:c"])
	assert.True(t, ids["prompts:d"])
	assert.True(t, ids["plans:e"])
	assert.False(t, ids["foundation:z"])
}

func TestCollectActiveIDs_empty(t *testing.T) {
	m := &manifest.Manifest{}
	ids := CollectActiveIDs(m)
	assert.Empty(t, ids)
}

func TestMergeManifestDedup_topicConflict(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)

	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "topics/react.md"), []byte("v1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "topics/react.md"), []byte("v2"), 0o644))

	dst := &manifest.Manifest{
		Topics: []manifest.TopicEntry{{ID: "react", Path: "topics/react.md"}},
	}
	src := &manifest.Manifest{
		Topics: []manifest.TopicEntry{{ID: "react", Path: "topics/react.md"}},
	}
	seen := map[string]seenEntry{
		"topics:react": {pkg: "local", hash: fileHash(filepath.Join(pkgA, "topics/react.md"))},
	}

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)
	require.Len(t, events, 1)
	assert.Equal(t, "conflict", events[0].Reason)
	assert.Equal(t, "topics", events[0].Section)
}

func TestMergeManifestDedup_bothHashesEmpty(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)
	// Both files missing — hashes are both empty.
	// "" == "" is true but "" != "" is false, so this hits the CONFLICT path.
	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "missing", Path: "foundation/missing.md"}},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{{ID: "missing", Path: "foundation/missing.md"}},
	}
	seen := map[string]seenEntry{
		"foundation:missing": {pkg: "local", hash: ""},
	}

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)
	require.Len(t, events, 1)
	assert.Equal(t, "conflict", events[0].Reason)
}

func TestMergeManifestDedup_mixedWithinSection(t *testing.T) {
	pkgA, pkgB := setupDedupDirs(t)

	sameContent := []byte("shared")
	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "foundation/a.md"), sameContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/a.md"), sameContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgA, "foundation/b.md"), []byte("v1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/b.md"), []byte("v2"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgB, "foundation/c.md"), []byte("new"), 0o644))

	dst := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md"},
			{ID: "b", Path: "foundation/b.md"},
		},
	}
	src := &manifest.Manifest{
		Foundation: []manifest.FoundationEntry{
			{ID: "a", Path: "foundation/a.md"}, // dedup
			{ID: "b", Path: "foundation/b.md"}, // conflict
			{ID: "c", Path: "foundation/c.md"}, // new
		},
	}
	seen := map[string]seenEntry{
		"foundation:a": {pkg: "local", hash: fileHash(filepath.Join(pkgA, "foundation/a.md"))},
		"foundation:b": {pkg: "local", hash: fileHash(filepath.Join(pkgA, "foundation/b.md"))},
	}

	events := mergeManifestDedup(dst, src, pkgA, pkgB, "pkgB@org", seen)
	require.Len(t, events, 2)
	assert.Equal(t, "duplicate", events[0].Reason)
	assert.Equal(t, "conflict", events[1].Reason)
	// dst should have original 2 + new 1 = 3
	require.Len(t, dst.Foundation, 3)
	assert.Equal(t, "c", dst.Foundation[2].ID)
}

func TestFileHash_emptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))
	h := fileHash(path)
	assert.NotEmpty(t, h)
	assert.Len(t, h, 64)
}

// --- keyID ---

func TestKeyID(t *testing.T) {
	assert.Equal(t, "philosophy", keyID("foundation:philosophy"))
	assert.Equal(t, "react", keyID("topics:react"))
	assert.Equal(t, "noprefix", keyID("noprefix"))
}

// --- fileHash ---

func TestFileHash_validFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))

	h := fileHash(path)
	assert.NotEmpty(t, h)
	assert.Len(t, h, 64) // SHA256 hex is 64 chars
}

func TestFileHash_missingFile(t *testing.T) {
	h := fileHash("/nonexistent/file.txt")
	assert.Empty(t, h)
}

func TestFileHash_sameContentSameHash(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "a.txt")
	path2 := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(path1, []byte("identical"), 0o644))
	require.NoError(t, os.WriteFile(path2, []byte("identical"), 0o644))

	assert.Equal(t, fileHash(path1), fileHash(path2))
}

func TestFileHash_differentContentDifferentHash(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "a.txt")
	path2 := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(path1, []byte("version A"), 0o644))
	require.NoError(t, os.WriteFile(path2, []byte("version B"), 0o644))

	assert.NotEqual(t, fileHash(path1), fileHash(path2))
}

func TestCheckDedup_notFound(t *testing.T) {
	seen := map[string]seenEntry{
		"foundation:existing": {pkg: "local", hash: "abc123"},
	}

	ev, skip := checkDedup("foundation:missing", "foundation/missing.md", "foundation", "/src", "/dst", "pkg@org", seen)
	assert.False(t, skip)
	assert.Equal(t, ConflictEntry{}, ev)
}

func TestKeyID_multipleColons(t *testing.T) {
	// Everything after the first colon should be returned.
	assert.Equal(t, "a:b", keyID("foundation:a:b"))
}

func TestKeyID_emptyString(t *testing.T) {
	assert.Equal(t, "", keyID(""))
}

func TestKeyID_colonAtStart(t *testing.T) {
	assert.Equal(t, "foo", keyID(":foo"))
}
