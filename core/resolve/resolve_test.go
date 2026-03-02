package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupBareRepo creates a bare git repo with semver tags for testing.
// Returns the path to the bare repo.
func setupBareRepo(t *testing.T, tags []string, includeManifestYml bool) string {
	t.Helper()
	dir := t.TempDir()
	workDir := filepath.Join(dir, "work")
	bareDir := filepath.Join(dir, "bare.git")

	// Initialize a regular (non-bare) repo first.
	repo, err := git.PlainInit(workDir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// Create manifest.yml if requested.
	if includeManifestYml {
		content := "name: test-pkg\nauthor: test-author\nversion: \"1.0.0\"\ndescription: Test\n"
		require.NoError(t, os.WriteFile(filepath.Join(workDir, "manifest.yml"), []byte(content), 0o644))
		_, err = wt.Add("manifest.yml")
		require.NoError(t, err)
	}

	// Create a dummy file to commit.
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Test\n"), 0o644))
	_, err = wt.Add("README.md")
	require.NoError(t, err)

	sig := &object.Signature{
		Name:  "Test",
		Email: "test@test.com",
		When:  time.Now(),
	}

	hash, err := wt.Commit("initial commit", &git.CommitOptions{Author: sig})
	require.NoError(t, err)

	// Create lightweight tags pointing at the commit.
	for _, tag := range tags {
		_, err = repo.CreateTag(tag, hash, nil)
		require.NoError(t, err)
	}

	// Create a bare clone from the working repo.
	_, err = git.PlainClone(bareDir, true, &git.CloneOptions{
		URL:  workDir,
		Tags: git.AllTags,
	})
	require.NoError(t, err)

	return bareDir
}

// --- Resolve ---

func TestResolve_latestVersion(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.0.0", "v1.1.0", "v2.0.0"}, true)

	ref := &PackageRef{Name: "test-pkg", Author: "test-author"}
	resolved, err := Resolve(ref, bareDir)
	require.NoError(t, err)

	assert.Equal(t, "test-pkg", resolved.Name)
	assert.Equal(t, "test-author", resolved.Author)
	assert.Equal(t, "2.0.0", resolved.Version)
	assert.Equal(t, "v2.0.0", resolved.Tag)
	assert.Equal(t, bareDir, resolved.Source)
}

func TestResolve_withConstraint(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.0.0", "v1.1.0", "v2.0.0"}, true)

	ref := &PackageRef{Name: "test-pkg", Author: "test-author", Version: "^1.0.0"}
	resolved, err := Resolve(ref, bareDir)
	require.NoError(t, err)

	assert.Equal(t, "1.1.0", resolved.Version)
	assert.Equal(t, "v1.1.0", resolved.Tag)
}

func TestResolve_noMatchingConstraint(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.0.0", "v1.1.0"}, true)

	ref := &PackageRef{Name: "test-pkg", Author: "test-author", Version: "^3.0.0"}
	_, err := Resolve(ref, bareDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no version matching")
}

func TestResolve_noSemverTags(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"latest", "stable"}, true)

	ref := &PackageRef{Name: "test-pkg", Author: "test-author"}
	_, err := Resolve(ref, bareDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no semver tags")
}

func TestResolve_invalidSource(t *testing.T) {
	ref := &PackageRef{Name: "test-pkg", Author: "test-author"}
	_, err := Resolve(ref, "/nonexistent/repo.git")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list remote")
}

func TestResolve_vPrefixStripped(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.2.3"}, true)

	ref := &PackageRef{Name: "test-pkg", Author: "test-author"}
	resolved, err := Resolve(ref, bareDir)
	require.NoError(t, err)

	// Version should have the v prefix stripped.
	assert.Equal(t, "1.2.3", resolved.Version)
	// Tag should retain the v prefix.
	assert.Equal(t, "v1.2.3", resolved.Tag)
}

func TestResolve_mixedTags(t *testing.T) {
	// Mix of semver and non-semver tags — only semver ones should be considered.
	bareDir := setupBareRepo(t, []string{"v1.0.0", "latest", "v2.0.0-beta.1", "v1.5.0"}, true)

	ref := &PackageRef{Name: "test-pkg", Author: "test-author", Version: "^1.0.0"}
	resolved, err := Resolve(ref, bareDir)
	require.NoError(t, err)

	// Pre-release versions are not matched by ^1.0.0 in semver.
	assert.Equal(t, "1.5.0", resolved.Version)
}

// --- Fetch ---

func TestFetch_success(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.0.0"}, true)
	destDir := filepath.Join(t.TempDir(), "fetched")

	resolved := &ResolvedPackage{
		Name:    "test-pkg",
		Author:  "test-author",
		Version: "1.0.0",
		Source:  bareDir,
		Tag:     "v1.0.0",
	}

	err := Fetch(resolved, destDir)
	require.NoError(t, err)

	// Verify manifest.yml exists.
	_, err = os.Stat(filepath.Join(destDir, "manifest.yml"))
	assert.NoError(t, err)

	// Verify README.md exists.
	_, err = os.Stat(filepath.Join(destDir, "README.md"))
	assert.NoError(t, err)
}

func TestFetch_destAlreadyExists(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.0.0"}, true)
	destDir := t.TempDir() // already exists

	resolved := &ResolvedPackage{
		Name:   "test-pkg",
		Author: "test-author",
		Source: bareDir,
		Tag:    "v1.0.0",
	}

	err := Fetch(resolved, destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestFetch_noManifestYml(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.0.0"}, false)
	destDir := filepath.Join(t.TempDir(), "fetched")

	resolved := &ResolvedPackage{
		Name:   "test-pkg",
		Author: "test-author",
		Source: bareDir,
		Tag:    "v1.0.0",
	}

	err := Fetch(resolved, destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no manifest.yml")

	// Verify cleanup: destDir should not exist.
	_, statErr := os.Stat(destDir)
	assert.True(t, os.IsNotExist(statErr))
}

func TestFetch_invalidTag(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.0.0"}, true)
	destDir := filepath.Join(t.TempDir(), "fetched")

	resolved := &ResolvedPackage{
		Name:   "test-pkg",
		Author: "test-author",
		Source: bareDir,
		Tag:    "v99.99.99", // tag doesn't exist
	}

	err := Fetch(resolved, destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clone")

	// Verify cleanup: destDir should not exist.
	_, statErr := os.Stat(destDir)
	assert.True(t, os.IsNotExist(statErr))
}

func TestFetch_invalidSource(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "fetched")

	resolved := &ResolvedPackage{
		Name:   "test-pkg",
		Author: "test-author",
		Source: "/nonexistent/repo.git",
		Tag:    "v1.0.0",
	}

	err := Fetch(resolved, destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clone")
}

// --- Resolve + Fetch integration ---

func TestResolveAndFetch_endToEnd(t *testing.T) {
	bareDir := setupBareRepo(t, []string{"v1.0.0", "v1.1.0"}, true)

	ref := &PackageRef{Name: "test-pkg", Author: "test-author", Version: "^1.0.0"}
	resolved, err := Resolve(ref, bareDir)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", resolved.Version)

	destDir := filepath.Join(t.TempDir(), fmt.Sprintf("%s@%s", resolved.Name, resolved.Author))
	err = Fetch(resolved, destDir)
	require.NoError(t, err)

	// Verify the fetched repo has manifest.yml.
	_, err = os.Stat(filepath.Join(destDir, "manifest.yml"))
	assert.NoError(t, err)
}
