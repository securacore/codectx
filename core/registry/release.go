package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ReleaseAssetURL finds the download URL for "package.tar.gz" attached to
// a GitHub Release at the given tag. Returns the browser_download_url
// which can be used to download the archive without authentication.
//
// Parameters:
//   - owner: GitHub owner/author (e.g. "community")
//   - repo: GitHub repo name without the codectx- prefix will be added (e.g. "codectx-react-patterns")
//   - tag: git tag including "v" prefix (e.g. "v2.0.0")
//
// Returns the asset download URL, or an error if the release or asset is not found.
func (gh *GitHubClient) ReleaseAssetURL(ctx context.Context, owner, repo, tag string) (string, error) {
	release, _, err := gh.client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return "", fmt.Errorf("getting release %s for %s/%s: %w", tag, owner, repo, err)
	}

	for _, asset := range release.Assets {
		if asset.GetName() == PackageArchiveName {
			return asset.GetBrowserDownloadURL(), nil
		}
	}

	return "", fmt.Errorf("asset %q not found in release %s for %s/%s", PackageArchiveName, tag, owner, repo)
}

// ReleaseAssetURLForDep is a convenience wrapper that takes a DepKey and
// constructs the owner/repo/tag from it.
func (gh *GitHubClient) ReleaseAssetURLForDep(ctx context.Context, dk DepKey, version string) (string, error) {
	owner := dk.Author
	repo := dk.RepoName()
	tag := GitTag(version)
	return gh.ReleaseAssetURL(ctx, owner, repo, tag)
}

// PackageInstaller abstracts the installation of a single resolved package.
// The archive-based implementation downloads the release asset and extracts it.
type PackageInstaller interface {
	// Install downloads and installs a package to the given destination directory.
	// Returns the release asset URL used (for logging) and any error.
	Install(ctx context.Context, dk DepKey, version, destDir string) (assetURL string, err error)
}

// ArchiveInstaller implements PackageInstaller using GitHub Release archives.
type ArchiveInstaller struct {
	GH   *GitHubClient
	HTTP HTTPClient
}

// Install downloads the package.tar.gz from the GitHub Release and extracts
// it to destDir.
func (ai *ArchiveInstaller) Install(ctx context.Context, dk DepKey, version, destDir string) (string, error) {
	url, err := ai.GH.ReleaseAssetURLForDep(ctx, dk, version)
	if err != nil {
		return "", err
	}

	if err := InstallPackageFromArchive(ctx, ai.HTTP, url, destDir); err != nil {
		return url, err
	}

	return url, nil
}

// ReadDepsFromRelease downloads a package archive to a temp directory and
// reads its codectx.yml for transitive dependency information.
// This replaces the git-clone-based PackageConfigReader for archive installs.
type ArchiveConfigReader struct {
	GH       *GitHubClient
	HTTP     HTTPClient
	CacheDir string
}

// ReadDeps downloads the package archive, extracts it, and reads the
// dependency map from its codectx.yml.
func (ar *ArchiveConfigReader) ReadDeps(ctx context.Context, dk DepKey, version string, reg string) (map[string]string, error) {
	destDir := filepath.Join(ar.CacheDir, dk.Name+"@"+dk.Author+"-"+version)

	// Check if already cached.
	cfgPath := filepath.Join(destDir, "codectx.yml")
	if _, err := os.Stat(cfgPath); err == nil {
		cfg, parseErr := loadPackageConfigFromFile(cfgPath)
		if parseErr != nil {
			return nil, parseErr
		}
		return cfg.Dependencies, nil
	}

	// Download and extract.
	url, err := ar.GH.ReleaseAssetURLForDep(ctx, dk, version)
	if err != nil {
		return nil, err
	}

	if err := InstallPackageFromArchive(ctx, ar.HTTP, url, destDir); err != nil {
		return nil, fmt.Errorf("installing package for dep reading: %w", err)
	}

	cfg, err := loadPackageConfigFromFile(cfgPath)
	if err != nil {
		return nil, err
	}

	return cfg.Dependencies, nil
}

// loadPackageConfigFromFile reads a codectx.yml from disk and returns
// the PackageConfig.
func loadPackageConfigFromFile(path string) (*PackageConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	return parsePackageConfig(f)
}
