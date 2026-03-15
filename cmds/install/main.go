// Package install implements the `codectx install` command which resolves
// and downloads documentation packages declared in codectx.yml.
//
// Packages are installed from GitHub Release archives (package.tar.gz).
// Each release contains a tar.gz of the package/ directory with all
// publishable content (foundation/, topics/, plans/, prompts/).
//
// If codectx.lock exists and codectx.yml hasn't changed, packages are
// installed from the lock file (fast, deterministic). If codectx.yml changed
// or no lock exists, dependencies are re-resolved and the lock is updated.
package install

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx install`.
var Command = &cli.Command{
	Name:  "install",
	Usage: "Install packages declared in codectx.yml",
	Description: `Resolves dependencies from codectx.yml, downloads package archives from
GitHub Releases to .codectx/packages/, and generates codectx.lock for
deterministic installs.

If codectx.lock exists and codectx.yml hasn't changed, installs from the
lock file (fast). If dependencies changed, re-resolves and updates the lock.`,
	Action: run,
}

func run(ctx context.Context, _ *cli.Command) error {
	// Step 1: Discover project.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	if len(cfg.Dependencies) == 0 {
		shared.WarnNoDependencies()
		return nil
	}

	paths := shared.ResolveRegistryPaths(projectDir, cfg)

	// Step 2: Check if lock is current.
	lf, lockErr := registry.LoadLock(paths.LockPath)
	if lockErr == nil && registry.LockCurrent(lf, cfg.Dependencies) {
		// Lock is up to date — install from lock.
		return installFromLock(ctx, lf, paths.PackagesDir)
	}

	// Step 3: Resolve and install.
	return shared.ResolveAndInstall(ctx, cfg, paths.Registry, paths.RootDir, paths.PackagesDir, paths.LockPath)
}

// installFromLock installs packages using pinned versions from the lock file.
// Downloads package archives from GitHub Releases.
func installFromLock(ctx context.Context, lf *registry.LockFile, packagesDir string) error {
	fmt.Printf("\n%s Installing from lock file (%d packages)\n\n",
		tui.Arrow(), len(lf.Packages))

	token := registry.GitHubToken()
	ghClient := registry.NewGitHubClient(token)
	httpClient := registry.AuthenticatedHTTPClient(token)
	installer := &registry.ArchiveInstaller{GH: ghClient, HTTP: httpClient}

	refs := lf.SortedPackageRefs()

	for _, ref := range refs {
		pkg := lf.Packages[ref]

		// Parse the ref to get a DepKey.
		name, author, parseErr := registry.ParsePackageRef(ref)
		if parseErr != nil {
			fmt.Printf("%s%s %s: %v\n", tui.Indent(1), tui.ErrorIcon(), tui.StyleAccent.Render(ref), parseErr)
			continue
		}

		dk := registry.DepKey{Name: name, Author: author, Version: pkg.ResolvedVersion}
		destDir := filepath.Join(packagesDir, ref)

		var installErr error
		err := shared.RunWithSpinner(fmt.Sprintf("Installing %s v%s...", ref, pkg.ResolvedVersion), func() {
			_, installErr = installer.Install(ctx, dk, pkg.ResolvedVersion, destDir)
		})
		if err != nil {
			return fmt.Errorf("spinner: %w", err)
		}
		if installErr != nil {
			fmt.Printf("%s%s %s: %v\n", tui.Indent(1), tui.ErrorIcon(), tui.StyleAccent.Render(ref), installErr)
			continue
		}

		source := ""
		if pkg.Source == registry.SourceTransitive {
			source = tui.StyleMuted.Render(" (transitive)")
		}

		fmt.Printf("%s%s %s v%s%s\n", tui.Indent(1), tui.Success(), tui.StyleAccent.Render(ref), pkg.ResolvedVersion, source)
	}

	fmt.Println()
	return nil
}
