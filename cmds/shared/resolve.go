package shared

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
)

// ResolveContext holds the clients and temp state needed for dependency
// resolution and archive-based installation. Create via NewResolveContext
// and defer Cleanup to release resources.
type ResolveContext struct {
	GH        *registry.GitHubClient
	HTTP      registry.HTTPClient
	Tags      *registry.GitTagLister
	Configs   *registry.ArchiveConfigReader
	Installer *registry.ArchiveInstaller
	cacheDir  string
}

// NewResolveContext creates all clients needed for dependency resolution
// and archive installation. The caller must defer Cleanup().
func NewResolveContext() (*ResolveContext, error) {
	token := registry.GitHubToken()
	gc := registry.NewGitClient(token)
	ghClient := registry.NewGitHubClient(token)
	httpClient := registry.AuthenticatedHTTPClient(token)

	cacheDir, err := os.MkdirTemp("", "codectx-resolve-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	return &ResolveContext{
		GH:   ghClient,
		HTTP: httpClient,
		Tags: &registry.GitTagLister{GC: gc},
		Configs: &registry.ArchiveConfigReader{
			GH:       ghClient,
			HTTP:     httpClient,
			CacheDir: cacheDir,
		},
		Installer: &registry.ArchiveInstaller{GH: ghClient, HTTP: httpClient},
		cacheDir:  cacheDir,
	}, nil
}

// Cleanup removes temporary files created during resolution.
func (rc *ResolveContext) Cleanup() {
	if rc.cacheDir != "" {
		_ = os.RemoveAll(rc.cacheDir)
	}
}

// InstallResult holds counts from a package installation run.
type InstallResult struct {
	Installed int
	Failed    int
}

// InstallPackages downloads and extracts resolved packages from GitHub Release
// archives to the packages directory. Prints per-package progress with spinners.
func InstallPackages(
	ctx context.Context,
	installer *registry.ArchiveInstaller,
	packages map[string]*registry.ResolvedPackage,
	packagesDir string,
) (*InstallResult, error) {
	result := &InstallResult{}

	for ref, pkg := range packages {
		destDir := filepath.Join(packagesDir, ref)

		var installErr error
		var assetURL string

		if err := RunWithSpinner(fmt.Sprintf("Installing %s v%s...", ref, pkg.ResolvedVersion), func() {
			assetURL, installErr = installer.Install(ctx, pkg.Key, pkg.ResolvedVersion, destDir)
		}); err != nil {
			return result, fmt.Errorf("spinner: %w", err)
		}
		if installErr != nil {
			fmt.Printf("%s%s %s: %v\n", tui.Indent(1), tui.ErrorIcon(), tui.StyleAccent.Render(ref), installErr)
			result.Failed++
			continue
		}

		result.Installed++

		source := ""
		if pkg.Source == registry.SourceTransitive {
			source = tui.StyleMuted.Render(" (transitive)")
		}

		fmt.Printf("%s%s %s v%s%s\n", tui.Indent(1), tui.Success(), tui.StyleAccent.Render(ref), pkg.ResolvedVersion, source)
		if assetURL != "" {
			fmt.Printf("%s%s %s\n", tui.Indent(2), tui.StyleMuted.Render(tui.IconArrow), tui.StyleMuted.Render(assetURL))
		}
	}

	return result, nil
}

// ResolveAndInstall runs the full dependency resolution, downloads packages
// from GitHub Release archives, and writes the lock file. This is the common
// implementation shared by `codectx add` and `codectx install`.
func ResolveAndInstall(
	ctx context.Context,
	cfg *project.Config,
	reg, rootDir, packagesDir, lockPath string,
) error {
	if len(cfg.Dependencies) == 0 {
		return nil
	}

	rc, err := NewResolveContext()
	if err != nil {
		return err
	}
	defer rc.Cleanup()

	var result *registry.ResolveResult
	var resolveErr error

	if err := RunWithSpinner("Resolving dependencies...", func() {
		result, resolveErr = registry.Resolve(ctx, cfg.Dependencies, reg, rc.Tags, rc.Configs)
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if resolveErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Dependency resolution failed",
			Detail: []string{resolveErr.Error()},
		}.Render())
		return resolveErr
	}

	PrintConflicts(result.Conflicts)

	fmt.Printf("\n%s Installing %d packages\n\n",
		tui.Arrow(), len(result.Packages))

	installResult, err := InstallPackages(ctx, rc.Installer, result.Packages, packagesDir)
	if err != nil {
		return err
	}

	// Write lock file.
	commitSHAs := make(map[string]string)
	if err := SaveLockOrError(lockPath, result, commitSHAs, reg); err != nil {
		return err
	}

	fmt.Printf("\n%s Installed %d packages\n\n",
		tui.Success(),
		installResult.Installed,
	)

	return nil
}
