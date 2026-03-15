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
	ArgsUsage: "<name[@author][:version]>",
	Description: `Removes a dependency from codectx.yml, deletes the installed package,
and updates the lock file.

The author and version are optional — when omitted, codectx searches
your project dependencies for matches by name.

Examples:
  codectx remove react                         Search deps by name, auto-select if unique
  codectx remove react@community               Specific author
  codectx remove react@community:latest         Full key format

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
	// --- Step 1: Validate and parse arguments ---
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing dependency argument",
			Detail: []string{
				"Usage: codectx remove <name[@author][:version]>",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Examples:"},
				{Text: "By name:", Command: "codectx remove react"},
				{Text: "By name@author:", Command: "codectx remove react@community"},
			},
		}.Render())
		return fmt.Errorf("missing dependency argument")
	}

	depStr := cmd.Args().First()
	partial, err := registry.ParsePartialDepKey(depStr)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Invalid dependency format",
			Detail: []string{err.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Name only:", Command: "codectx remove react"},
				{Text: "Name@author:", Command: "codectx remove react@community"},
				{Text: "Full key:", Command: "codectx remove react@community:latest"},
			},
		}.Render())
		return err
	}

	// --- Step 2: Discover project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 3: Resolve the dependency ref from project deps ---
	ref, err := resolveDepRef(cfg, partial)
	if err != nil {
		return err
	}

	// --- Step 4: Find the dependency in the project config ---
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
		fmt.Print(tui.WarnMsg{
			Title: "Dependency not found",
			Detail: []string{
				fmt.Sprintf("%s is not declared in any config file.",
					tui.StyleAccent.Render(ref),
				),
			},
		}.Render())
		return fmt.Errorf("dependency %s not found", ref)
	}

	// --- Step 5: Determine target ---
	target, err := resolveTarget(cmd, cfg, projectDir, inProject, inPackageManifest)
	if err != nil {
		return err
	}

	// --- Step 6: Remove from config file(s) ---
	if err := removeDependency(projectDir, cfg, ref, depKey, target, inProject, inPackageManifest); err != nil {
		return err
	}

	// --- Step 7: Remove installed package directory ---
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

	// --- Step 8: Update lock file ---
	if shouldUninstall {
		lockPath := filepath.Join(rootDir, registry.LockFileName)
		removeLockEntry(lockPath, ref)
	}

	// --- Step 9: Summary ---
	fmt.Printf("\n%s Removed %s\n", tui.Success(), tui.StyleAccent.Render(ref))
	printTargetInfo(target, shouldUninstall)
	fmt.Println()

	return nil
}

// resolveDepRef takes a partial dependency key and resolves it to a full
// "name@author" ref by searching the project's existing dependencies.
//
// When the author is already specified, the ref is returned directly.
// When only a name is given, it searches deps for matches:
//   - 0 matches → error with available deps hint
//   - 1 match → auto-select with info message
//   - multiple → interactive prompt or error in non-interactive mode
func resolveDepRef(cfg *project.Config, partial registry.PartialDepKey) (string, error) {
	// If author is known, return the ref directly.
	if partial.Author != "" {
		return partial.Name + "@" + partial.Author, nil
	}

	// Search project deps by name.
	matches := findDepsByName(cfg, partial.Name)

	switch len(matches) {
	case 0:
		fmt.Print(tui.ErrorMsg{
			Title: "Dependency not found",
			Detail: []string{
				fmt.Sprintf("No dependency matching %s in codectx.yml.",
					tui.StyleAccent.Render(partial.Name)),
			},
			Suggestions: depListSuggestions(cfg),
		}.Render())
		return "", fmt.Errorf("no dependency matching %q", partial.Name)

	case 1:
		ref := matches[0]
		fmt.Printf("\n%s Matched %s\n",
			tui.Arrow(),
			tui.StyleAccent.Render(ref),
		)
		return ref, nil

	default:
		// Multiple matches — need disambiguation.
		interactive := term.IsTerminal(os.Stdin.Fd())
		if !interactive {
			suggestions := make([]tui.Suggestion, 0, len(matches)+1)
			suggestions = append(suggestions, tui.Suggestion{Text: "Specify the full reference:"})
			for _, ref := range matches {
				suggestions = append(suggestions, tui.Suggestion{
					Command: fmt.Sprintf("codectx remove %s", ref),
				})
			}
			fmt.Print(tui.ErrorMsg{
				Title: "Multiple dependencies match (non-interactive mode)",
				Detail: []string{
					fmt.Sprintf("Found %d dependencies matching %s.",
						len(matches), tui.StyleAccent.Render(partial.Name)),
				},
				Suggestions: suggestions,
			}.Render())
			return "", fmt.Errorf("multiple dependencies match %q in non-interactive mode", partial.Name)
		}

		// Interactive select.
		options := make([]huh.Option[string], 0, len(matches))
		for _, ref := range matches {
			options = append(options, huh.NewOption(ref, ref))
		}

		var selected string
		if err := huh.NewSelect[string]().
			Title(fmt.Sprintf("Multiple dependencies match %q. Which one?", partial.Name)).
			Options(options...).
			Value(&selected).
			Run(); err != nil {
			return "", err
		}

		return selected, nil
	}
}

// findDepsByName returns all "name@author" refs from the project config
// where the package name matches.
func findDepsByName(cfg *project.Config, name string) []string {
	var matches []string
	for key := range cfg.Dependencies {
		dk, err := registry.ParseDepKey(key)
		if err != nil {
			continue
		}
		if dk.Name == name {
			matches = append(matches, dk.PackageRef())
		}
	}
	return matches
}

// depListSuggestions returns suggestions listing the current project dependencies.
func depListSuggestions(cfg *project.Config) []tui.Suggestion {
	if len(cfg.Dependencies) == 0 {
		return []tui.Suggestion{
			{Text: "No dependencies are currently configured."},
		}
	}

	suggestions := []tui.Suggestion{
		{Text: "Current dependencies:"},
	}
	for key := range cfg.Dependencies {
		dk, err := registry.ParseDepKey(key)
		if err != nil {
			continue
		}
		suggestions = append(suggestions, tui.Suggestion{
			Command: fmt.Sprintf("codectx remove %s", dk.PackageRef()),
		})
	}
	return suggestions
}

// findDepByRef searches the project config dependencies for one matching
// the given "name@author" ref. Returns the full key and whether it was found.
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

	if errMsg := shared.ValidateExclusiveTargetFlags(flagProject, flagPackage, flagBoth); errMsg != "" {
		fmt.Print(errMsg)
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
			// RequiredBy entries are "name@author:version", extract ref part.
			name, author, err := registry.ParsePackageRef(extractRef(rb))
			if err != nil {
				filtered = append(filtered, rb)
				continue
			}
			if name+"@"+author != removedRef {
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

// extractRef extracts the "name@author" part from a "name@author:version" string.
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
	var value string
	suffix := ""
	if uninstalled {
		suffix = tui.StyleMuted.Render(" + uninstalled")
	}

	switch target {
	case rmProject:
		value = tui.StylePath.Render("codectx.yml") +
			tui.StyleMuted.Render(" (project)") + suffix
	case rmPackage:
		value = tui.StylePath.Render("package/codectx.yml") +
			tui.StyleMuted.Render(" (package manifest)")
	case rmBoth:
		value = tui.StylePath.Render("codectx.yml") +
			tui.StyleMuted.Render(" + ") +
			tui.StylePath.Render("package/codectx.yml") + suffix
	}
	fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue("Removed from", value))
}
