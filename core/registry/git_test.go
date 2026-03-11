package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// initTestRepo creates a temporary git repo with an initial commit and returns
// the repo and its path. The caller should clean up via t.TempDir().
func initTestRepo(t *testing.T) (*git.Repository, string) {
	t.Helper()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	// Create a file and commit it.
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("add: %v", err)
	}

	_, err = wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	return repo, dir
}

// addTagToRepo creates a lightweight tag on the current HEAD.
func addTagToRepo(t *testing.T, repo *git.Repository, tagName string) {
	t.Helper()
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	_, err = repo.CreateTag(tagName, head.Hash(), nil)
	if err != nil {
		t.Fatalf("create tag %q: %v", tagName, err)
	}
}

// addAnnotatedTag creates an annotated tag on the current HEAD.
func addAnnotatedTag(t *testing.T, repo *git.Repository, tagName, message string) {
	t.Helper()
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	_, err = repo.CreateTag(tagName, head.Hash(), &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
		Message: message,
	})
	if err != nil {
		t.Fatalf("create annotated tag %q: %v", tagName, err)
	}
}

// makeCommit creates a new file and commits it, advancing HEAD.
func makeCommit(t *testing.T, repo *git.Repository, dir, filename, content string) plumbing.Hash {
	t.Helper()
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	testFile := filepath.Join(dir, filename)
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := wt.Add(filename); err != nil {
		t.Fatalf("add: %v", err)
	}

	hash, err := wt.Commit("add "+filename, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	return hash
}

func TestListTags_Empty(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	tags, err := gc.ListTags(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected no tags, got %v", tags)
	}
}

func TestListTags_MultipleTags(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	addTagToRepo(t, repo, "v1.0.0")
	addTagToRepo(t, repo, "v1.1.0")
	addTagToRepo(t, repo, "v2.0.0")

	tags, err := gc.ListTags(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(tags), tags)
	}

	// Tags should be present (order may vary).
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}
	for _, expected := range []string{"v1.0.0", "v1.1.0", "v2.0.0"} {
		if !tagSet[expected] {
			t.Errorf("expected tag %q not found in %v", expected, tags)
		}
	}
}

func TestTagCommitSHA_LightweightTag(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	expectedSHA := head.Hash().String()

	addTagToRepo(t, repo, "v1.0.0")

	sha, err := gc.TagCommitSHA(repo, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != expectedSHA {
		t.Errorf("expected SHA %q, got %q", expectedSHA, sha)
	}
}

func TestTagCommitSHA_AnnotatedTag(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	expectedSHA := head.Hash().String()

	addAnnotatedTag(t, repo, "v1.0.0", "Release 1.0.0")

	sha, err := gc.TagCommitSHA(repo, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != expectedSHA {
		t.Errorf("expected SHA %q, got %q", expectedSHA, sha)
	}
}

func TestTagCommitSHA_NotFound(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	_, err := gc.TagCommitSHA(repo, "v99.99.99")
	if err == nil {
		t.Fatal("expected error for nonexistent tag")
	}
}

func TestCheckoutTag(t *testing.T) {
	repo, dir := initTestRepo(t)
	gc := NewGitClient()

	// Create a second commit with a new file.
	makeCommit(t, repo, dir, "second.md", "# Second\n")
	addTagToRepo(t, repo, "v2.0.0")

	// Add a third commit.
	makeCommit(t, repo, dir, "third.md", "# Third\n")
	addTagToRepo(t, repo, "v3.0.0")

	// Checkout v2.0.0 — third.md should not exist.
	if err := gc.CheckoutTag(repo, "v2.0.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "second.md")); os.IsNotExist(err) {
		t.Error("second.md should exist at v2.0.0")
	}
	if _, err := os.Stat(filepath.Join(dir, "third.md")); !os.IsNotExist(err) {
		t.Error("third.md should NOT exist at v2.0.0")
	}

	// Checkout v3.0.0 — third.md should exist.
	if err := gc.CheckoutTag(repo, "v3.0.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "third.md")); os.IsNotExist(err) {
		t.Error("third.md should exist at v3.0.0")
	}
}

func TestCreateLightweightTag(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	err := gc.CreateLightweightTag(repo, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the tag exists.
	tags, err := gc.ListTags(repo)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	found := false
	for _, tag := range tags {
		if tag == "v1.0.0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("tag v1.0.0 not found in %v", tags)
	}
}

func TestCreateLightweightTag_Duplicate(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	if err := gc.CreateLightweightTag(repo, "v1.0.0"); err != nil {
		t.Fatalf("first tag: %v", err)
	}

	err := gc.CreateLightweightTag(repo, "v1.0.0")
	if err == nil {
		t.Fatal("expected error for duplicate tag")
	}
}

func TestTagExists(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	if gc.TagExists(repo, "v1.0.0") {
		t.Error("tag should not exist yet")
	}

	addTagToRepo(t, repo, "v1.0.0")

	if !gc.TagExists(repo, "v1.0.0") {
		t.Error("tag should exist after creation")
	}
}

func TestHeadCommitSHA(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	sha, err := gc.HeadCommitSHA(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("expected 40-char SHA, got %d chars: %q", len(sha), sha)
	}

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if sha != head.Hash().String() {
		t.Errorf("SHA mismatch: got %q, want %q", sha, head.Hash().String())
	}
}

func TestClone_LocalRepo(t *testing.T) {
	// Create a "remote" repo.
	sourceRepo, sourceDir := initTestRepo(t)
	addTagToRepo(t, sourceRepo, "v1.0.0")

	// Clone it to a new directory.
	destDir := filepath.Join(t.TempDir(), "clone")
	gc := NewGitClient()

	repo, err := gc.Clone(context.Background(), sourceDir, destDir)
	if err != nil {
		t.Fatalf("clone: %v", err)
	}

	// Verify the clone has the tag.
	tags, err := gc.ListTags(repo)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	found := false
	for _, tag := range tags {
		if tag == "v1.0.0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("tag v1.0.0 not found in clone: %v", tags)
	}

	// Verify README.md exists.
	if _, err := os.Stat(filepath.Join(destDir, "README.md")); os.IsNotExist(err) {
		t.Error("README.md should exist in clone")
	}
}

func TestClone_ExistingRepo_Fetches(t *testing.T) {
	// Create a "remote" repo.
	sourceRepo, sourceDir := initTestRepo(t)
	addTagToRepo(t, sourceRepo, "v1.0.0")

	// Clone it.
	destDir := filepath.Join(t.TempDir(), "clone")
	gc := NewGitClient()

	_, err := gc.Clone(context.Background(), sourceDir, destDir)
	if err != nil {
		t.Fatalf("clone: %v", err)
	}

	// Add a new tag to the source.
	makeCommit(t, sourceRepo, sourceDir, "extra.md", "# Extra\n")
	addTagToRepo(t, sourceRepo, "v2.0.0")

	// "Clone" again — should fetch.
	repo, err := gc.Clone(context.Background(), sourceDir, destDir)
	if err != nil {
		t.Fatalf("second clone (fetch): %v", err)
	}

	tags, err := gc.ListTags(repo)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}
	if !tagSet["v2.0.0"] {
		t.Errorf("v2.0.0 not found after fetch: %v", tags)
	}
}

func TestReadPackageConfig(t *testing.T) {
	repo, dir := initTestRepo(t)
	gc := NewGitClient()

	// Write a codectx.yml into the repo and commit it.
	configContent := `name: test-package
org: testorg
version: "1.0.0"
description: A test package
dependencies:
  utils@community: ">=1.0.0"
`
	configPath := filepath.Join(dir, "codectx.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if _, err := wt.Add("codectx.yml"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := wt.Commit("add config", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	cfg, err := gc.ReadPackageConfig(repo)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if cfg.Name != "test-package" {
		t.Errorf("name: got %q, want %q", cfg.Name, "test-package")
	}
	if cfg.Org != "testorg" {
		t.Errorf("org: got %q, want %q", cfg.Org, "testorg")
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("version: got %q, want %q", cfg.Version, "1.0.0")
	}
	if cfg.Description != "A test package" {
		t.Errorf("description: got %q, want %q", cfg.Description, "A test package")
	}
	if len(cfg.Dependencies) != 1 {
		t.Fatalf("dependencies: got %d, want 1", len(cfg.Dependencies))
	}
	if cfg.Dependencies["utils@community"] != ">=1.0.0" {
		t.Errorf("dependency constraint: got %q, want %q",
			cfg.Dependencies["utils@community"], ">=1.0.0")
	}
}

func TestReadPackageConfig_NoConfig(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	_, err := gc.ReadPackageConfig(repo)
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestListRemoteTags(t *testing.T) {
	// Create a bare "remote" repo that we can list tags from.
	remoteDir := t.TempDir()
	remoteRepo, err := git.PlainInit(remoteDir, true)
	if err != nil {
		t.Fatalf("init bare repo: %v", err)
	}

	// We need to add a commit and tags via a worktree clone.
	workDir := filepath.Join(t.TempDir(), "work")
	workRepo, err := git.PlainClone(workDir, false, &git.CloneOptions{
		URL: remoteDir,
	})
	// PlainClone on an empty repo may error — init fresh and add remote.
	if err != nil {
		workRepo, err = git.PlainInit(workDir, false)
		if err != nil {
			t.Fatalf("init work repo: %v", err)
		}
		_, err = workRepo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{remoteDir},
		})
		if err != nil {
			t.Fatalf("create remote: %v", err)
		}
	}

	// Make a commit.
	wt, err := workRepo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	testFile := filepath.Join(workDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	_, err = wt.Commit("init", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Create tags and push.
	addTagToRepo(t, workRepo, "v1.0.0")
	addTagToRepo(t, workRepo, "v1.1.0")

	// Push everything to the bare remote.
	err = workRepo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"refs/heads/*:refs/heads/*", "refs/tags/*:refs/tags/*"},
	})
	if err != nil {
		t.Fatalf("push: %v", err)
	}

	// Now list remote tags.
	gc := NewGitClient()
	tags, err := gc.ListRemoteTags(context.Background(), remoteDir)
	if err != nil {
		t.Fatalf("list remote tags: %v", err)
	}

	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tags), tags)
	}

	// Verify both tags exist — bare repos won't have fetched them.
	_ = remoteRepo // used to init the bare repo
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}
	for _, expected := range []string{"v1.0.0", "v1.1.0"} {
		if !tagSet[expected] {
			t.Errorf("expected tag %q not found in %v", expected, tags)
		}
	}
}

func TestPushTag_NoRemote(t *testing.T) {
	// PushTag should error when there is no remote configured.
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	addTagToRepo(t, repo, "v1.0.0")

	err := gc.PushTag(context.Background(), repo, "v1.0.0")
	if err == nil {
		t.Fatal("expected error pushing tag with no remote")
	}
}

func TestPushTag_NonexistentTag(t *testing.T) {
	repo, _ := initTestRepo(t)
	gc := NewGitClient()

	// Add a remote so the error is about the tag, not the remote.
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/nonexistent/repo.git"},
	})
	if err != nil {
		t.Fatalf("create remote: %v", err)
	}

	err = gc.PushTag(context.Background(), repo, "v99.0.0")
	// This will fail because the remote doesn't exist or tag doesn't exist.
	if err == nil {
		t.Fatal("expected error pushing nonexistent tag to invalid remote")
	}
}
