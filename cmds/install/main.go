// Package install implements the `codectx install` command which resolves
// and downloads documentation packages declared in codectx.yml.
//
// If codectx.lock exists and codectx.yml hasn't changed, packages are
// installed from the lock file (fast, deterministic). If codectx.yml changed
// or no lock exists, dependencies are re-resolved and the lock is updated.
package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx install`.
var Command = &cli.Command{
	Name:  "install",
	Usage: "Install packages declared in codectx.yml",
	Description: `Resolves dependencies from codectx.yml, downloads packages to
.codectx/packages/, and generates codectx.lock for deterministic installs.

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
		fmt.Printf("\n%s No dependencies declared in codectx.yml\n\n", tui.Warning())
		return nil
	}

	rootDir := project.RootDir(projectDir, cfg)
	lockPath := filepath.Join(rootDir, registry.LockFileName)
	packagesDir := project.PackagesPath(rootDir)

	reg := cfg.EffectiveRegistry()

	// Step 2: Check if lock is current.
	lf, lockErr := registry.LoadLock(lockPath)
	if lockErr == nil && registry.LockCurrent(lf, cfg.Dependencies) {
		// Lock is up to date — install from lock.
		return installFromLock(ctx, lf, packagesDir)
	}

	// Step 3: Resolve dependencies.
	return resolveAndInstall(ctx, cfg, reg, rootDir, packagesDir, lockPath)
}

// installFromLock installs packages using pinned versions from the lock file.
func installFromLock(ctx context.Context, lf *registry.LockFile, packagesDir string) error {
	fmt.Printf("\n%s Installing from lock file (%d packages)\n\n",
		tui.Arrow(), len(lf.Packages))

	gc := registry.NewGitClient(registry.GitHubToken())
	refs := lf.SortedPackageRefs()

	for _, ref := range refs {
		pkg := lf.Packages[ref]
		url := "https://" + pkg.Repo
		destDir := filepath.Join(packagesDir, ref)

		var installErr error
		err := shared.RunWithSpinner(fmt.Sprintf("Installing %s v%s...", ref, pkg.ResolvedVersion), func() {
			repo, cloneErr := gc.Clone(ctx, url, destDir)
			if cloneErr != nil {
				installErr = cloneErr
				return
			}

			tag := registry.GitTag(pkg.ResolvedVersion)
			installErr = gc.CheckoutTag(repo, tag)
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

// resolveAndInstall runs the full dependency resolution, downloads packages,
// and writes the lock file.
func resolveAndInstall(
	ctx context.Context,
	cfg *project.Config,
	reg, rootDir, packagesDir, lockPath string,
) error {
	gc := registry.NewGitClient(registry.GitHubToken())

	// Create temp dir for transitive dep resolution cache.
	cacheDir, err := os.MkdirTemp("", "codectx-resolve-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()

	tags := &registry.GitTagLister{GC: gc}
	configs := &registry.GitConfigReader{GC: gc, CacheDir: cacheDir}

	var result *registry.ResolveResult
	var resolveErr error

	if err = shared.RunWithSpinner("Resolving dependencies...", func() {
		result, resolveErr = registry.Resolve(ctx, cfg.Dependencies, reg, tags, configs)
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

	// Report conflicts.
	shared.PrintConflicts(result.Conflicts)

	// Step 4: Download resolved packages.
	fmt.Printf("\n%s Installing %d packages\n\n",
		tui.Arrow(), len(result.Packages))

	commitSHAs := make(map[string]string)
	installed := 0

	for ref, pkg := range result.Packages {
		url := pkg.Key.RepoURL(reg)
		destDir := filepath.Join(packagesDir, ref)

		var installErr error
		var sha string

		if err = shared.RunWithSpinner(fmt.Sprintf("Installing %s v%s...", ref, pkg.ResolvedVersion), func() {
			repo, cloneErr := gc.Clone(ctx, url, destDir)
			if cloneErr != nil {
				installErr = cloneErr
				return
			}

			if checkoutErr := gc.CheckoutTag(repo, pkg.ResolvedTag); checkoutErr != nil {
				installErr = checkoutErr
				return
			}

			sha, installErr = gc.TagCommitSHA(repo, pkg.ResolvedTag)
		}); err != nil {
			return fmt.Errorf("spinner: %w", err)
		}
		if installErr != nil {
			fmt.Printf("%s%s %s: %v\n", tui.Indent(1), tui.ErrorIcon(), tui.StyleAccent.Render(ref), installErr)
			continue
		}

		commitSHAs[ref] = sha
		installed++

		source := ""
		if pkg.Source == registry.SourceTransitive {
			source = tui.StyleMuted.Render(" (transitive)")
		}

		fmt.Printf("%s%s %s v%s%s\n", tui.Indent(1), tui.Success(), tui.StyleAccent.Render(ref), pkg.ResolvedVersion, source)
		fmt.Printf("%s%s %s\n", tui.Indent(2), tui.StyleMuted.Render(tui.IconArrow), tui.StyleMuted.Render(url+"@"+pkg.ResolvedTag))
	}

	// Step 5: Write lock file.
	if err := shared.SaveLockOrError(lockPath, result, commitSHAs, reg); err != nil {
		return err
	}

	fmt.Printf("\n%s Installed %d packages, updated %s\n\n",
		tui.Success(),
		installed,
		tui.StylePath.Render(registry.LockFileName),
	)

	return nil
}
