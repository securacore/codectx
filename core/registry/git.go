package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"gopkg.in/yaml.v3"
)

// GitClient wraps go-git operations for cloning, fetching, and inspecting
// package repositories. When created with a token, all operations authenticate
// with it (required for pushing to remotes and accessing private repos).
type GitClient struct {
	auth transport.AuthMethod
}

// NewGitClient creates a new GitClient. If token is non-empty, all git
// operations (clone, fetch, push, list remote) will authenticate with it.
// If token is empty, operations are unauthenticated (public repos only).
func NewGitClient(token string) *GitClient {
	gc := &GitClient{}
	if token != "" {
		gc.auth = &githttp.BasicAuth{
			Username: "x-access-token",
			Password: token,
		}
	}
	return gc
}

// Clone clones a repository to a local path. If the directory already contains
// a git repository, it fetches updates instead.
//
// Returns the opened or cloned go-git Repository.
func (gc *GitClient) Clone(ctx context.Context, url, destPath string) (*git.Repository, error) {
	// Try opening an existing repo first.
	repo, err := git.PlainOpen(destPath)
	if err == nil {
		// Repo exists, fetch updates.
		if fetchErr := gc.fetch(ctx, repo); fetchErr != nil {
			return repo, fetchErr
		}
		return repo, nil
	}

	repo, err = git.PlainCloneContext(ctx, destPath, false, &git.CloneOptions{
		URL:  url,
		Tags: git.AllTags,
		Auth: gc.auth,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning %s: %w", url, err)
	}

	return repo, nil
}

// fetch fetches all refs and tags from the remote.
// Returns nil if already up to date.
func (gc *GitClient) fetch(ctx context.Context, repo *git.Repository) error {
	err := repo.FetchContext(ctx, &git.FetchOptions{
		Tags:  git.AllTags,
		Force: true,
		Auth:  gc.auth,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetching: %w", err)
	}
	return nil
}

// listTags returns all tag names from the repository. Tags are returned
// with the "v" prefix if present (e.g. "v1.0.0").
// Used internally by tests to verify tag operations on local repos.
func (gc *GitClient) listTags(repo *git.Repository) ([]string, error) {
	tagIter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	var tags []string
	err = tagIter.ForEach(func(ref *plumbing.Reference) error {
		// Tag names come as "refs/tags/v1.0.0" — extract short name.
		name := ref.Name().Short()
		tags = append(tags, name)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	return tags, nil
}

// TagCommitSHA resolves a tag name to the commit SHA it points to.
// Handles both lightweight tags (direct ref to commit) and annotated tags
// (ref to tag object which contains a target commit).
func (gc *GitClient) TagCommitSHA(repo *git.Repository, tagName string) (string, error) {
	refName := plumbing.NewTagReferenceName(tagName)
	ref, err := repo.Reference(refName, true)
	if err != nil {
		return "", fmt.Errorf("resolving tag %q: %w", tagName, err)
	}

	hash := ref.Hash()

	// Try to resolve as annotated tag (tag object -> commit).
	tagObj, err := repo.TagObject(hash)
	if err == nil {
		// Annotated tag — follow to the commit.
		commit, commitErr := tagObj.Commit()
		if commitErr != nil {
			return "", fmt.Errorf("resolving annotated tag %q to commit: %w", tagName, commitErr)
		}
		return commit.Hash.String(), nil
	}

	// Not an annotated tag — verify the hash points to a commit.
	_, err = repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("tag %q does not point to a commit: %w", tagName, err)
	}

	return hash.String(), nil
}

// CheckoutTag checks out a specific tag in the repository's worktree.
// The worktree will be in detached HEAD state at the tag's commit.
func (gc *GitClient) CheckoutTag(repo *git.Repository, tagName string) error {
	sha, err := gc.TagCommitSHA(repo, tagName)
	if err != nil {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	hash := plumbing.NewHash(sha)
	err = wt.Checkout(&git.CheckoutOptions{
		Hash:  hash,
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("checking out tag %q: %w", tagName, err)
	}

	return nil
}

// CreateLightweightTag creates a lightweight tag on the current HEAD commit.
// Used during codectx publish to tag the version.
func (gc *GitClient) CreateLightweightTag(repo *git.Repository, tagName string) error {
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("getting HEAD: %w", err)
	}

	_, err = repo.CreateTag(tagName, head.Hash(), nil)
	if err != nil {
		return fmt.Errorf("creating tag %q: %w", tagName, err)
	}

	return nil
}

// PushTag pushes a specific tag to the remote repository.
// Used during codectx publish after tagging.
func (gc *GitClient) PushTag(ctx context.Context, repo *git.Repository, tagName string) error {
	refSpec := config.RefSpec(fmt.Sprintf(
		"refs/tags/%s:refs/tags/%s", tagName, tagName,
	))

	err := repo.PushContext(ctx, &git.PushOptions{
		RefSpecs: []config.RefSpec{refSpec},
		Auth:     gc.auth,
	})
	if err != nil {
		return fmt.Errorf("pushing tag %q: %w", tagName, err)
	}

	return nil
}

// HeadCommitSHA returns the SHA of the current HEAD commit.
func (gc *GitClient) HeadCommitSHA(repo *git.Repository) (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}

	return head.Hash().String(), nil
}

// ReadPackageConfig reads and parses the codectx.yml (or codectx.yaml) from
// a checked-out repository worktree. Returns the raw YAML content as a
// PackageConfig suitable for transitive dependency resolution.
func (gc *GitClient) ReadPackageConfig(repo *git.Repository) (*PackageConfig, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree: %w", err)
	}

	// Try codectx.yml first, then codectx.yaml.
	for _, name := range []string{"codectx.yml", "codectx.yaml"} {
		f, openErr := wt.Filesystem.Open(name)
		if openErr != nil {
			continue
		}

		cfg, parseErr := parsePackageConfig(f)
		_ = f.Close()
		if parseErr != nil {
			return nil, parseErr
		}
		return cfg, nil
	}

	return nil, fmt.Errorf("no codectx.yml or codectx.yaml found in package")
}

// ListRemoteTags lists all tags from a remote repository without cloning.
// This is used by codectx search and version resolution when we don't need
// the full repo content.
func (gc *GitClient) ListRemoteTags(ctx context.Context, url string) ([]string, error) {
	rem := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
	})

	refs, err := rem.ListContext(ctx, &git.ListOptions{
		Auth: gc.auth,
	})
	if err != nil {
		return nil, fmt.Errorf("listing remote %s: %w", url, err)
	}

	var tags []string
	for _, ref := range refs {
		name := ref.Name().String()
		if strings.HasPrefix(name, "refs/tags/") {
			// Skip ^{} dereferenced tag refs.
			if strings.HasSuffix(name, "^{}") {
				continue
			}
			tags = append(tags, ref.Name().Short())
		}
	}

	return tags, nil
}

// PackageConfig represents the minimal fields from a package's codectx.yml
// needed for transitive dependency resolution.
type PackageConfig struct {
	Name         string            `yaml:"name"`
	Org          string            `yaml:"org"`
	Version      string            `yaml:"version"`
	Description  string            `yaml:"description"`
	Dependencies map[string]string `yaml:"dependencies"`
}

// parsePackageConfig reads and parses a PackageConfig from an io.Reader.
func parsePackageConfig(r io.Reader) (*PackageConfig, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading package config: %w", err)
	}

	var cfg PackageConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing package config: %w", err)
	}

	return &cfg, nil
}

// TagExists checks whether a tag already exists in the repository.
func (gc *GitClient) TagExists(repo *git.Repository, tagName string) bool {
	refName := plumbing.NewTagReferenceName(tagName)
	_, err := repo.Reference(refName, true)
	return err == nil
}
