package registry

import (
	"context"
	"path/filepath"
)

// GitTagLister implements TagLister using GitClient to list remote tags
// without cloning.
type GitTagLister struct {
	GC *GitClient
}

// AvailableTags lists semver tags from the remote repository.
func (g *GitTagLister) AvailableTags(ctx context.Context, dk DepKey, reg string) ([]string, error) {
	url := dk.RepoURL(reg)
	return g.GC.ListRemoteTags(ctx, url)
}

// GitConfigReader implements PackageConfigReader by cloning the package
// (to a cache directory) and reading its codectx.yml.
type GitConfigReader struct {
	GC       *GitClient
	CacheDir string
}

// ReadDeps clones the package, checks out the given version, and returns the
// dependency map from its codectx.yml.
func (g *GitConfigReader) ReadDeps(ctx context.Context, dk DepKey, version string, reg string) (map[string]string, error) {
	url := dk.RepoURL(reg)
	destDir := filepath.Join(g.CacheDir, dk.PackageRef())

	repo, err := g.GC.Clone(ctx, url, destDir)
	if err != nil {
		return nil, err
	}

	tag := GitTag(version)
	if err := g.GC.CheckoutTag(repo, tag); err != nil {
		return nil, err
	}

	cfg, err := g.GC.ReadPackageConfig(repo)
	if err != nil {
		return nil, err
	}

	return cfg.Dependencies, nil
}
