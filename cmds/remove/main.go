// Package remove implements the `codectx remove` command which removes a
// documentation package dependency from codectx.yml (and optionally from the
// package manifest for package authoring projects).
//
// The command removes the dependency entry from config files, deletes the
// installed package directory, and updates the lock file.
package remove

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// removeTarget describes where the dependency should be removed from.
type removeTarget int

const (
	// rmProject removes the dependency from the root codectx.yml only.
	rmProject removeTarget = iota

	// rmPackage removes the dependency from package/codectx.yml only.
	rmPackage

	// rmBoth removes the dependency from both root codectx.yml and
	// package/codectx.yml.
	rmBoth
)

// Command is the CLI definition for `codectx remove`.
var Command = &cli.Command{
	Name:      "remove",
	Usage:     "Remove a documentation package dependency",
	ArgsUsage: "<name@org[:version]>",
	Description: `Removes a dependency from codectx.yml, deletes the installed package,
and updates the lock file.

The version is optional — the package is matched by name@org.

Examples:
  codectx remove react-patterns@community
  codectx remove react-patterns@community:latest

For package authoring projects (type: "package"), you will be prompted
to choose where the dependency is removed from:
  - Project only: remove from root codectx.yml
  - Package only: remove from package/codectx.yml
  - Both: remove from both config files

Use --project, --package, or --both to skip the prompt.`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "project",
			Usage: "Remove from project only (root codectx.yml)",
		},
		&cli.BoolFlag{
			Name:  "package",
			Usage: "Remove from package manifest only (package/codectx.yml)",
		},
		&cli.BoolFlag{
			Name:  "both",
			Usage: "Remove from both project and package manifest",
		},
	},
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	// --- Step 1: Validate arguments ---
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing dependency argument",
			Detail: []string{
				"Usage: codectx remove <name@org[:version]>",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Example:", Command: "codectx remove react-patterns@community"},
			},
		}.Render())
		return fmt.Errorf("missing dependency argument")
	}

	depStr := cmd.Args().First()
	ref, err := parseRef(depStr)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Invalid dependency format",
			Detail: []string{err.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Expected format:", Command: "name@org or name@org:version"},
			},
		}.Render())
		return err
	}

	// --- Step 2: Discover project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 3: Find the dependency in the project config ---
	depKey, found := findDepByRef(cfg, ref)
	inProject := found

	// Check if it's in the package manifest too.
	inPackageManifest := false
	if cfg.IsPackage() {
		manifestPath := project.PackageConfigPath(projectDir)
		if manifest, loadErr := project.LoadPackageManifest(manifestPath); loadErr == nil {
			if _, ok := manifest.Dependencies[ref]; ok {
				inPackageManifest = true
			}
		}
	}

	if !inProject && !inPackageManifest {
		fmt.Printf("\n%s Dependency %s not found in any config file\n\n",
			tui.Warning(),
			tui.StyleAccent.Render(ref),
		)
		return fmt.Errorf("dependency %s not found", ref)
	}

	// --- Step 4: Determine target ---
	target, err := resolveTarget(cmd, cfg, projectDir, inProject, inPackageManifest)
	if err != nil {
		return err
	}

	// --- Step 5: Remove from config file(s) ---
	if err := removeDependency(projectDir, cfg, ref, depKey, target, inProject, inPackageManifest); err != nil {
		return err
	}

	// --- Step 6: Remove installed package directory ---
	rootDir := project.RootDir(projectDir, cfg)
	packagesDir := project.PackagesPath(rootDir)
	pkgDir := filepath.Join(packagesDir, ref)

	// Only remove from disk if we're removing from the project config
	// (package-only removal shouldn't uninstall since the project still needs it).
	shouldUninstall := target == rmProject || target == rmBoth
	if shouldUninstall {
		if info, statErr := os.Stat(pkgDir); statErr == nil && info.IsDir() {
			if removeErr := os.RemoveAll(pkgDir); removeErr != nil {
				fmt.Printf("%s%s Failed to remove %s: %v\n",
					tui.Indent(1), tui.Warning(),
					tui.StylePath.Render(pkgDir), removeErr,
				)
			}
		}
	}

	// --- Step 7: Update lock file ---
	if shouldUninstall {
		lockPath := filepath.Join(rootDir, registry.LockFileName)
		removeLockEntry(lockPath, ref)
	}

	// --- Step 8: Summary ---
	fmt.Printf("\n%s Removed %s\n", tui.Success(), tui.StyleAccent.Render(ref))
	printTargetInfo(target, shouldUninstall)
	fmt.Println()

	return nil
}

// parseRef parses a dependency argument that can be either "name@org:version"
// or just "name@org". Returns the short ref "name@org".
func parseRef(s string) (string, error) {
	// Try full key format first.
	dk, err := registry.ParseDepKey(s)
	if err == nil {
		return dk.PackageRef(), nil
	}

	// Try short ref format.
	_, _, parseErr := registry.ParsePackageRef(s)
	if parseErr != nil {
		return "", fmt.Errorf("invalid format %q: expected name@org or name@org:version", s)
	}

	return s, nil
}

// findDepByRef searches the project config dependencies for one matching
// the given "name@org" ref. Returns the full key and whether it was found.
func findDepByRef(cfg *project.Config, ref string) (string, bool) {
	for key := range cfg.Dependencies {
		dk, err := registry.ParseDepKey(key)
		if err != nil {
			continue
		}
		if dk.PackageRef() == ref {
			return key, true
		}
	}
	return "", false
}

// resolveTarget determines where the dependency should be removed from based
// on CLI flags, project type, and interactive prompts.
func resolveTarget(
	cmd *cli.Command,
	cfg *project.Config,
	projectDir string,
	inProject, inPackageManifest bool,
) (removeTarget, error) {
	// For standard projects, always target project config.
	if !cfg.IsPackage() {
		return rmProject, nil
	}

	// Check CLI flags.
	flagProject := cmd.Bool("project")
	flagPackage := cmd.Bool("package")
	flagBoth := cmd.Bool("both")

	flagCount := shared.BoolCount(flagProject, flagPackage, flagBoth)
	if flagCount > 1 {
		fmt.Print(tui.ErrorMsg{
			Title:  "Conflicting flags",
			Detail: []string{"Only one of --project, --package, or --both may be specified."},
		}.Render())
		return 0, fmt.Errorf("conflicting target flags")
	}

	if flagProject {
		return rmProject, nil
	}
	if flagPackage {
		return rmPackage, nil
	}
	if flagBoth {
		return rmBoth, nil
	}

	// If it only exists in one place, remove from there.
	if inProject && !inPackageManifest {
		return rmProject, nil
	}
	if !inProject && inPackageManifest {
		return rmPackage, nil
	}

	// Exists in both — prompt.
	interactive := term.IsTerminal(os.Stdin.Fd())
	if !interactive {
		return rmBoth, nil
	}

	// Verify package dir exists.
	pkgDir := project.PackageContentPath(projectDir)
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		return rmProject, nil
	}

	var selected removeTarget
	if err := huh.NewSelect[removeTarget]().
		Title("This dependency exists in both configs. Where should it be removed from?").
		Options(
			huh.NewOption("Both (project + package manifest)", rmBoth),
			huh.NewOption("Project only (keep in package manifest)", rmProject),
			huh.NewOption("Package manifest only (keep in project)", rmPackage),
		).
		Value(&selected).
		Run(); err != nil {
		return 0, err
	}

	return selected, nil
}

// removeDependency removes the dependency from the appropriate config file(s).
func removeDependency(
	projectDir string,
	cfg *project.Config,
	ref, depKey string,
	target removeTarget,
	inProject, inPackageManifest bool,
) error {
	switch target {
	case rmProject:
		if inProject {
			return removeFromProjectConfig(projectDir, cfg, depKey)
		}
		return nil

	case rmPackage:
		if inPackageManifest {
			return removeFromPackageManifest(projectDir, ref)
		}
		return nil

	case rmBoth:
		if inProject {
			if err := removeFromProjectConfig(projectDir, cfg, depKey); err != nil {
				return err
			}
		}
		if inPackageManifest {
			return removeFromPackageManifest(projectDir, ref)
		}
		return nil

	default:
		return fmt.Errorf("unknown target: %d", target)
	}
}

// removeFromProjectConfig removes a dependency from the root codectx.yml.
func removeFromProjectConfig(projectDir string, cfg *project.Config, depKey string) error {
	delete(cfg.Dependencies, depKey)

	configPath := filepath.Join(projectDir, project.ConfigFileName)
	if err := cfg.WriteToFile(configPath); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to update codectx.yml",
			Detail: []string{err.Error()},
		}.Render())
		return fmt.Errorf("writing codectx.yml: %w", err)
	}
	return nil
}

// removeFromPackageManifest removes a dependency from the package/codectx.yml manifest.
func removeFromPackageManifest(projectDir, ref string) error {
	manifestPath := project.PackageConfigPath(projectDir)

	manifest, err := project.LoadPackageManifest(manifestPath)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Failed to load package manifest",
			Detail: []string{
				err.Error(),
				fmt.Sprintf("Expected at: %s", tui.StylePath.Render(manifestPath)),
			},
		}.Render())
		return fmt.Errorf("loading package manifest: %w", err)
	}

	delete(manifest.Dependencies, ref)

	if err := manifest.WriteToFile(manifestPath); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to update package manifest",
			Detail: []string{err.Error()},
		}.Render())
		return fmt.Errorf("writing package manifest: %w", err)
	}

	return nil
}

// removeLockEntry removes a package ref from the lock file, if it exists.
func removeLockEntry(lockPath, ref string) {
	lf, err := registry.LoadLock(lockPath)
	if err != nil {
		return // No lock file or parse error — nothing to clean up.
	}

	if _, ok := lf.Packages[ref]; !ok {
		return // Not in lock file.
	}

	delete(lf.Packages, ref)

	// Also remove any transitive deps that were only required by this package.
	pruneOrphanedTransitive(lf, ref)

	if err := registry.SaveLock(lockPath, lf); err != nil {
		// Non-fatal — lock will be regenerated on next install.
		return
	}
}

// pruneOrphanedTransitive removes transitive packages from the lock file
// that were only required by the removed package.
func pruneOrphanedTransitive(lf *registry.LockFile, removedRef string) {
	for ref, pkg := range lf.Packages {
		if pkg.Source != registry.SourceTransitive {
			continue
		}

		// Filter out the removed ref from RequiredBy.
		filtered := make([]string, 0, len(pkg.RequiredBy))
		for _, rb := range pkg.RequiredBy {
			// RequiredBy entries are "name@org:version", extract ref part.
			name, org, err := registry.ParsePackageRef(extractRef(rb))
			if err != nil {
				filtered = append(filtered, rb)
				continue
			}
			if name+"@"+org != removedRef {
				filtered = append(filtered, rb)
			}
		}

		if len(filtered) == 0 {
			// No remaining requesters — remove this transitive dep too.
			delete(lf.Packages, ref)
		} else {
			pkg.RequiredBy = filtered
		}
	}
}

// extractRef extracts the "name@org" part from a "name@org:version" string.
func extractRef(s string) string {
	dk, err := registry.ParseDepKey(s)
	if err != nil {
		// Fall back to treating the whole string as a ref.
		return s
	}
	return dk.PackageRef()
}

// printTargetInfo prints where the dependency was removed from.
func printTargetInfo(target removeTarget, uninstalled bool) {
	switch target {
	case rmProject:
		msg := "Removed from: codectx.yml (project)"
		if uninstalled {
			msg += " + uninstalled"
		}
		fmt.Printf("%s%s\n", tui.Indent(1), tui.StyleMuted.Render(msg))
	case rmPackage:
		fmt.Printf("%s%s\n",
			tui.Indent(1),
			tui.StyleMuted.Render("Removed from: package/codectx.yml (package manifest)"),
		)
	case rmBoth:
		msg := "Removed from: codectx.yml + package/codectx.yml"
		if uninstalled {
			msg += " + uninstalled"
		}
		fmt.Printf("%s%s\n", tui.Indent(1), tui.StyleMuted.Render(msg))
	}
}
