// Package update implements the `codectx update` command which re-resolves
// all dependencies to their latest compatible versions.
//
// Unlike install, which uses the lock file when unchanged, update always
// re-resolves all dependencies, updates the lock file, and downloads any
// changed packages. If packages changed, the project is automatically
// recompiled.
package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/compile"
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
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "compile",
			Usage: "Force recompilation after update, even if auto_compile is disabled",
		},
		&cli.BoolFlag{
			Name:  "no-compile",
			Usage: "Skip recompilation after update, even if packages changed",
		},
	},
	Action: run,
}

// changeStatus represents the status of a package after resolution.
type changeStatus string

const (
	statusNew       changeStatus = "new"
	statusUpdated   changeStatus = "updated"
	statusUnchanged changeStatus = "unchanged"
)

// changeEntry holds the classification of a single resolved package
// compared to the previous lock file.
type changeEntry struct {
	Ref        string
	Status     changeStatus
	OldVersion string
	NewVersion string
	Source     string
}

// classifyChanges compares resolved packages against the previous lock file
// to determine which are new, updated, or unchanged.
func classifyChanges(
	result *registry.ResolveResult,
	oldLock *registry.LockFile,
) []changeEntry {
	var entries []changeEntry

	// Sort refs for deterministic output.
	refs := make([]string, 0, len(result.Packages))
	for ref := range result.Packages {
		refs = append(refs, ref)
	}
	sort.Strings(refs)

	for _, ref := range refs {
		pkg := result.Packages[ref]

		entry := changeEntry{
			Ref:        ref,
			NewVersion: pkg.ResolvedVersion,
			Status:     statusNew,
			Source:     pkg.Source,
		}

		if oldLock != nil {
			if oldPkg, ok := oldLock.Packages[ref]; ok {
				entry.OldVersion = oldPkg.ResolvedVersion
				if entry.OldVersion == entry.NewVersion {
					entry.Status = statusUnchanged
				} else {
					entry.Status = statusUpdated
				}
			}
		}

		entries = append(entries, entry)
	}

	return entries
}

// countChanged returns the number of new or updated entries.
func countChanged(entries []changeEntry) int {
	n := 0
	for _, e := range entries {
		if e.Status != statusUnchanged {
			n++
		}
	}
	return n
}

func run(ctx context.Context, cmd *cli.Command) error {
	// Step 1: Discover project.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	if len(cfg.Dependencies) == 0 {
		fmt.Printf("\n%s No dependencies declared in codectx.yml\n\n", tui.Warning())
		return nil
	}

	forceCompile := cmd.IsSet("compile") && cmd.Bool("compile")
	skipCompile := cmd.IsSet("no-compile") && cmd.Bool("no-compile")

	rootDir := project.RootDir(projectDir, cfg)
	lockPath := filepath.Join(rootDir, registry.LockFileName)
	packagesDir := project.PackagesPath(rootDir)

	reg := cfg.EffectiveRegistry()

	// Load existing lock for comparison.
	oldLock, _ := registry.LoadLock(lockPath)

	gc := registry.NewGitClient(registry.GitHubToken())

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

	// Step 3: Compare with old lock and report changes.
	entries := classifyChanges(result, oldLock)
	changed := countChanged(entries)

	fmt.Printf("\n%s Resolving dependencies...\n", tui.Arrow())

	for _, entry := range entries {
		source := ""
		if entry.Source == registry.SourceTransitive {
			source = " (transitive)"
		}

		switch entry.Status {
		case statusUpdated:
			fmt.Printf("%s%s %s: %s -> %s%s\n",
				tui.Indent(1),
				tui.Arrow(),
				tui.StyleAccent.Render(entry.Ref), entry.OldVersion,
				tui.StyleBold.Render(entry.NewVersion),
				tui.StyleMuted.Render(source),
			)
		case statusNew:
			fmt.Printf("%s%s %s: %s%s\n",
				tui.Indent(1),
				tui.Success(),
				tui.StyleAccent.Render(entry.Ref),
				tui.StyleBold.Render(entry.NewVersion),
				tui.StyleMuted.Render(source+" (new)"),
			)
		default:
			fmt.Printf("%s%s %s: %s%s\n",
				tui.Indent(1),
				tui.StyleMuted.Render("-"),
				tui.StyleAccent.Render(entry.Ref),
				entry.NewVersion,
				tui.StyleMuted.Render(source+" (unchanged)"),
			)
		}
	}

	// Step 4: Download changed/new packages.
	commitSHAs := make(map[string]string)

	if changed > 0 {
		fmt.Printf("\n%s Downloading %d changed packages\n\n",
			tui.Arrow(), changed)

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
			fmt.Printf("%s%s Downloaded: %s v%s\n", tui.Indent(1), tui.Success(), tui.StyleAccent.Render(ref), pkg.ResolvedVersion)
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
	if err := shared.SaveLockOrError(lockPath, result, commitSHAs, reg); err != nil {
		return err
	}

	fmt.Printf("\n%s Updated %s\n",
		tui.Success(),
		tui.StylePath.Render(registry.LockFileName),
	)

	// Step 6: Auto-recompile decision.
	if changed == 0 {
		fmt.Printf("\n%s All packages up to date.\n\n", tui.Success())
		return nil
	}

	shouldCompile := shouldAutoCompile(projectDir, cfg, forceCompile, skipCompile)
	if !shouldCompile {
		fmt.Println()
		return nil
	}

	fmt.Printf("\n%s Recompiling (%d packages changed)...\n",
		tui.Arrow(), changed)

	aiCfg, aiErr := project.LoadAIConfigForProject(projectDir, cfg)
	if aiErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to load AI configuration",
			Detail: []string{aiErr.Error()},
		}.Render())
		return aiErr
	}

	prefsCfg, prefsErr := project.LoadPreferencesConfigForProject(projectDir, cfg)
	if prefsErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to load preferences",
			Detail: []string{prefsErr.Error()},
		}.Render())
		return prefsErr
	}

	compileCfg := compile.BuildConfig(projectDir, rootDir, cfg, aiCfg, prefsCfg)

	var compileResult *compile.Result
	var compileErr error

	if err = shared.RunWithSpinner("Compiling...", func() {
		compileResult, compileErr = compile.Run(compileCfg, nil)
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if compileErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Recompilation failed",
			Detail: []string{compileErr.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Run compilation manually:", Command: "codectx compile"},
			},
		}.Render())
		return compileErr
	}

	fmt.Print(shared.RenderCompactCompileSummary(compileResult))

	return nil
}

// shouldAutoCompile determines whether auto-recompilation should run
// based on CLI flags and the auto_compile preference.
func shouldAutoCompile(
	projectDir string,
	cfg *project.Config,
	forceCompile, skipCompile bool,
) bool {
	// Load preferences to check auto_compile setting.
	prefsCfg, err := project.LoadPreferencesConfigForProject(projectDir, cfg)
	if err != nil {
		// If preferences can't be loaded, default to compiling.
		// The actual compile step will report the error with details.
		prefsCfg = &project.PreferencesConfig{}
	}

	return shared.ShouldAutoCompile(prefsCfg, forceCompile, skipCompile, "recompile")
}
