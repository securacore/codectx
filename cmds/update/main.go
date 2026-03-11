// Package update implements the `codectx update` command which re-resolves
// all dependencies to their latest compatible versions.
//
// Unlike install, which uses the lock file when unchanged, update always
// re-resolves all dependencies, updates the lock file, and downloads any
// changed packages.
package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2/spinner"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx update`.
var Command = &cli.Command{
	Name:  "update",
	Usage: "Update all dependencies to latest compatible versions",
	Description: `Re-resolves all dependencies to their latest compatible versions,
updates codectx.lock, and downloads any changed packages.

Use this command to pull in newer versions of your dependencies.`,
	Action: run,
}

func run(ctx context.Context, _ *cli.Command) error {
	// Step 1: Discover project.
	projectDir, cfg, err := project.DiscoverAndLoad(".")
	if err != nil {
		fmt.Print(tui.ProjectNotFoundError())
		return fmt.Errorf("project not found: %w", err)
	}

	if len(cfg.Dependencies) == 0 {
		fmt.Printf("\n%s No dependencies declared in codectx.yml\n\n", tui.Warning())
		return nil
	}

	rootDir := project.RootDir(projectDir, cfg)
	lockPath := filepath.Join(rootDir, registry.LockFileName)
	packagesDir := project.PackagesPath(rootDir)

	reg := cfg.Registry
	if reg == "" {
		reg = project.DefaultRegistry
	}

	// Load existing lock for comparison.
	oldLock, _ := registry.LoadLock(lockPath)

	gc := registry.NewGitClient()

	// Create temp dir for transitive dep resolution cache.
	cacheDir, err := os.MkdirTemp("", "codectx-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()

	tags := &registry.GitTagLister{GC: gc}
	configs := &registry.GitConfigReader{GC: gc, CacheDir: cacheDir}

	// Step 2: Resolve all dependencies.
	var result *registry.ResolveResult
	var resolveErr error

	err = spinner.New().
		Title("Resolving dependencies...").
		Action(func() {
			result, resolveErr = registry.Resolve(ctx, cfg.Dependencies, reg, tags, configs)
		}).
		Run()
	if err != nil {
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
	for _, conflict := range result.Conflicts {
		fmt.Print(tui.WarnMsg{
			Title: fmt.Sprintf("Version conflict: %s", conflict.PackageRef),
			Detail: func() []string {
				var lines []string
				for requester, version := range conflict.Versions {
					lines = append(lines, fmt.Sprintf("  %s requires %s", requester, version))
				}
				return lines
			}(),
		}.Render())
	}

	// Step 3: Compare with old lock and report changes.
	fmt.Printf("\n%s Resolving dependencies...\n", tui.StyleAccent.Render("->"))

	changed := 0
	commitSHAs := make(map[string]string)

	for ref, pkg := range result.Packages {
		oldVersion := ""
		status := "new"
		if oldLock != nil {
			if oldPkg, ok := oldLock.Packages[ref]; ok {
				oldVersion = oldPkg.ResolvedVersion
				if oldVersion == pkg.ResolvedVersion {
					status = "unchanged"
				} else {
					status = "updated"
				}
			}
		}

		source := ""
		if pkg.Source == registry.SourceTransitive {
			source = " (transitive)"
		}

		switch status {
		case "updated":
			changed++
			fmt.Printf("  %s %s: %s -> %s%s\n",
				tui.StyleAccent.Render("->"),
				ref, oldVersion,
				tui.StyleBold.Render(pkg.ResolvedVersion),
				tui.StyleMuted.Render(source),
			)
		case "new":
			changed++
			fmt.Printf("  %s %s: %s%s\n",
				tui.Success(),
				ref,
				tui.StyleBold.Render(pkg.ResolvedVersion),
				tui.StyleMuted.Render(source+" (new)"),
			)
		default:
			fmt.Printf("  %s %s: %s%s\n",
				tui.StyleMuted.Render("-"),
				ref,
				pkg.ResolvedVersion,
				tui.StyleMuted.Render(source+" (unchanged)"),
			)
		}
	}

	// Step 4: Download changed/new packages.
	if changed > 0 {
		fmt.Printf("\n%s Downloading %d changed packages\n\n",
			tui.StyleAccent.Render("->"), changed)

		for ref, pkg := range result.Packages {
			url := pkg.Key.RepoURL(reg)
			destDir := filepath.Join(packagesDir, ref)

			var installErr error
			var sha string

			err = spinner.New().
				Title(fmt.Sprintf("Installing %s v%s...", ref, pkg.ResolvedVersion)).
				Action(func() {
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
				}).
				Run()
			if err != nil {
				return fmt.Errorf("spinner: %w", err)
			}
			if installErr != nil {
				fmt.Printf("  %s %s: %v\n", tui.ErrorIcon(), ref, installErr)
				continue
			}

			commitSHAs[ref] = sha
			fmt.Printf("  %s Downloaded: %s v%s\n", tui.Success(), ref, pkg.ResolvedVersion)
		}
	} else {
		// Still need commit SHAs for unchanged packages.
		for ref, pkg := range result.Packages {
			if oldLock != nil {
				if oldPkg, ok := oldLock.Packages[ref]; ok {
					commitSHAs[ref] = oldPkg.Commit
					continue
				}
			}
			// Need to get SHA for new packages.
			destDir := filepath.Join(packagesDir, ref)
			url := pkg.Key.RepoURL(reg)

			repo, cloneErr := gc.Clone(ctx, url, destDir)
			if cloneErr != nil {
				continue
			}
			if checkoutErr := gc.CheckoutTag(repo, pkg.ResolvedTag); checkoutErr != nil {
				continue
			}
			sha, shaErr := gc.TagCommitSHA(repo, pkg.ResolvedTag)
			if shaErr != nil {
				continue
			}
			commitSHAs[ref] = sha
		}
	}

	// Step 5: Write lock file.
	lf := registry.ToLockFile(result, commitSHAs, reg)
	if err := registry.SaveLock(lockPath, lf); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to write lock file",
			Detail: []string{err.Error()},
		}.Render())
		return err
	}

	fmt.Printf("\n%s Updated %s\n",
		tui.Success(),
		tui.StylePath.Render(registry.LockFileName),
	)

	if changed > 0 {
		fmt.Printf("\nRecompile to update compiled output: %s\n\n",
			tui.StyleCommand.Render("codectx compile"))
	} else {
		fmt.Printf("All packages up to date.\n\n")
	}

	return nil
}
