// Package add implements the `codectx add` command which adds a documentation
// package dependency to codectx.yml (and optionally to the package manifest
// for package authoring projects).
//
// For standard projects, the dependency is added to the root codectx.yml
// with active: true, then installed to .codectx/packages/.
//
// For package authoring projects (type: "package"), the user is prompted
// to choose where the dependency is added:
//   - Project only: root codectx.yml (for authoring use)
//   - Package only: package/codectx.yml (semver range, published with package)
//     AND root codectx.yml (for install, with active: false by default)
//   - Both: root codectx.yml (active) + package/codectx.yml (semver range)
package add

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// depTarget describes where the dependency should be added.
type depTarget int

const (
	// targetProject adds the dependency to the root codectx.yml only.
	targetProject depTarget = iota

	// targetPackage adds the dependency to package/codectx.yml (semver range)
	// and to root codectx.yml (for install, active: false).
	targetPackage

	// targetBoth adds the dependency to both root codectx.yml (active: true)
	// and package/codectx.yml (semver range).
	targetBoth
)

// Command is the CLI definition for `codectx add`.
var Command = &cli.Command{
	Name:      "add",
	Usage:     "Add a documentation package dependency",
	ArgsUsage: "<name[@author][:version]>",
	Description: `Adds a dependency to codectx.yml and installs it.

Examples:
  codectx add react                           Search and pick from available sources
  codectx add react@community                 Specific author, latest version
  codectx add react@community:2.0.0           Specific author and version
  codectx add react:2.0.0                     Specific version, pick author

When the author is omitted, codectx searches GitHub for packages matching
the name. If exactly one is found, it is used automatically. If multiple
are found, you are prompted to pick one (or an error is shown in
non-interactive mode).

For package authoring projects (type: "package"), you will be prompted
to choose where the dependency is added:
  - Project only: for your authoring workspace
  - Package only: published with your package (as a transitive dep)
  - Both: for authoring and published with your package

Use --project, --package, or --both to skip the prompt.`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "project",
			Usage: "Add to project only (root codectx.yml)",
		},
		&cli.BoolFlag{
			Name:  "package",
			Usage: "Add to package manifest (package/codectx.yml) and root for install",
		},
		&cli.BoolFlag{
			Name:  "both",
			Usage: "Add to both project and package manifest",
		},
		&cli.BoolFlag{
			Name:  "inactive",
			Usage: "Add as inactive (excluded from compiled output)",
		},
		&cli.BoolFlag{
			Name:  "show-uninstallable",
			Usage: "Include packages without a release archive in search results",
		},
	},
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	// --- Step 1: Validate and parse arguments ---
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing dependency argument",
			Detail: []string{
				"Usage: codectx add <name[@author][:version]>",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Examples:"},
				{Text: "Search and pick:", Command: "codectx add react"},
				{Text: "Specific author:", Command: "codectx add react@community"},
				{Text: "Full spec:", Command: "codectx add react@community:2.0.0"},
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
				{Text: "Name only:", Command: "codectx add react"},
				{Text: "Name@author:", Command: "codectx add react@community"},
				{Text: "Full spec:", Command: "codectx add react@community:2.0.0"},
				{Text: "Pinned version:", Command: "codectx add react:2.0.0"},
			},
		}.Render())
		return err
	}

	// --- Step 2: Discover project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 3: Resolve author if missing ---
	if partial.Author == "" {
		showUninstallable := cmd.Bool("show-uninstallable")
		if !showUninstallable {
			// Check project preference.
			if prefs, prefsErr := project.LoadPreferencesConfigForProject(projectDir, cfg); prefsErr == nil {
				showUninstallable = prefs.Search.EffectiveShowUninstallable()
			}
		}
		resolved, resolveErr := resolveAuthor(ctx, partial, showUninstallable)
		if resolveErr != nil {
			return resolveErr
		}
		partial = resolved
	}

	// Convert partial to full DepKey (defaults version to "latest" if empty).
	dk, err := partial.ToDepKey()
	if err != nil {
		return err
	}

	// --- Step 4: Check for duplicate ---
	depKey := dk.String()
	ref := dk.PackageRef()

	if isDuplicate(cfg, ref) {
		fmt.Print(tui.WarnMsg{
			Title: "Dependency already exists",
			Detail: []string{
				fmt.Sprintf("%s is already declared in %s.",
					tui.StyleAccent.Render(ref),
					tui.StylePath.Render("codectx.yml"),
				),
			},
		}.Render())
		return fmt.Errorf("dependency %s already exists", ref)
	}

	// --- Step 5: Determine target ---
	target, err := resolveTarget(cmd, cfg, projectDir)
	if err != nil {
		return err
	}

	inactive := cmd.Bool("inactive")

	// --- Step 6: Update config file(s) ---
	if err := applyDependency(projectDir, cfg, dk, target, inactive); err != nil {
		return err
	}

	// --- Step 7: Install ---
	fmt.Printf("\n%s Added %s\n", tui.Success(), tui.StyleAccent.Render(depKey))
	printTargetInfo(target)

	rootDir := project.RootDir(projectDir, cfg)
	lockPath := filepath.Join(rootDir, registry.LockFileName)
	packagesDir := project.PackagesPath(rootDir)
	reg := cfg.EffectiveRegistry()

	return shared.ResolveAndInstall(ctx, cfg, reg, rootDir, packagesDir, lockPath)
}

// resolveAuthor searches GitHub for packages matching the partial key's name
// and resolves the author component. If exactly one match is found, it is
// selected automatically. If multiple matches exist, an interactive prompt
// is shown (or an error in non-interactive mode).
func resolveAuthor(ctx context.Context, partial registry.PartialDepKey, showUninstallable bool) (registry.PartialDepKey, error) {
	token := registry.GitHubToken()
	gh := registry.NewGitHubClient(token)
	gitClient := registry.NewGitClient(token)

	var results []registry.SearchResult
	var searchErr error

	if err := shared.RunWithSpinner(
		fmt.Sprintf("Searching for %s...", tui.StyleAccent.Render(partial.Name)),
		func() {
			results, searchErr = gh.SearchPackages(ctx, partial.Name, 20)
		},
	); err != nil {
		return partial, fmt.Errorf("spinner: %w", err)
	}
	if searchErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Search failed",
			Detail: []string{searchErr.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Try specifying the author directly:", Command: fmt.Sprintf("codectx add %s@<author>", partial.Name)},
			},
		}.Render())
		return partial, searchErr
	}

	// Filter to exact name matches only.
	results = filterExactName(results, partial.Name)

	// Resolve versions and check release availability.
	if len(results) > 0 {
		_ = shared.RunWithSpinner("Checking releases...", func() {
			for i := range results {
				r := &results[i]
				tags, tagErr := gitClient.ListRemoteTags(ctx, "https://github.com/"+r.FullName)
				if tagErr != nil {
					continue
				}
				resolved, verErr := registry.ResolveVersion(tags, registry.LatestVersion)
				if verErr != nil {
					continue
				}
				r.LatestVersion = registry.VersionFromTag(resolved)

				tag := registry.GitTag(r.LatestVersion)
				repoName := r.FullName[strings.Index(r.FullName, "/")+1:]
				_, releaseErr := gh.ReleaseAssetURL(ctx, r.Author, repoName, tag)
				r.HasRelease = releaseErr == nil
			}
		})
	}

	// Filter out uninstallable packages unless --show-uninstallable is set.
	var hidden int
	if !showUninstallable {
		results, hidden = filterInstallable(results)
	}

	// Handle results.
	switch len(results) {
	case 0:
		detail := []string{
			fmt.Sprintf("No packages found matching %s.", tui.StyleAccent.Render(partial.Name)),
		}
		if hidden > 0 {
			detail = append(detail, fmt.Sprintf(
				"%d %s hidden (no release archive). Use %s to include them.",
				hidden,
				pluralize(hidden, "package", "packages"),
				tui.StyleCommand.Render("--show-uninstallable"),
			))
		}
		fmt.Print(tui.ErrorMsg{
			Title:  "Package not found",
			Detail: detail,
			Suggestions: []tui.Suggestion{
				{Text: "Search for available packages:", Command: fmt.Sprintf("codectx search %s", partial.Name)},
				{Text: "Or specify the author directly:", Command: fmt.Sprintf("codectx add %s@<author>:latest", partial.Name)},
			},
		}.Render())
		return partial, fmt.Errorf("no packages found for %q", partial.Name)

	case 1:
		// Auto-select the single result.
		r := results[0]
		partial.Author = r.Author
		if partial.Version == "" && r.LatestVersion != "" {
			partial.Version = r.LatestVersion
		}

		fmt.Printf("\n%s Found %s\n",
			tui.Arrow(),
			tui.StyleAccent.Render(r.Name+"@"+r.Author),
		)
		if r.LatestVersion != "" {
			fmt.Printf("%s%s\n",
				tui.Indent(1),
				tui.KeyValue("Latest", tui.StyleBold.Render("v"+r.LatestVersion)),
			)
		}
		if r.Description != "" {
			fmt.Printf("%s%s\n", tui.Indent(1), tui.StyleMuted.Render(r.Description))
		}

		return partial, nil

	default:
		// Multiple matches — need selection.
		interactive := term.IsTerminal(os.Stdin.Fd())
		if !interactive {
			fmt.Print(tui.ErrorMsg{
				Title: "Multiple packages found (non-interactive mode)",
				Detail: []string{
					fmt.Sprintf("Found %d packages matching %s:",
						len(results), tui.StyleAccent.Render(partial.Name)),
				},
				Suggestions: authorSuggestions(results, partial),
			}.Render())
			return partial, fmt.Errorf("multiple packages found for %q in non-interactive mode", partial.Name)
		}

		// Interactive prompt.
		options := make([]huh.Option[registry.SearchResult], 0, len(results))
		for _, r := range results {
			label := r.Name + "@" + r.Author
			if r.LatestVersion != "" {
				label += " v" + r.LatestVersion
			}
			if r.Description != "" {
				label += " — " + r.Description
			}
			options = append(options, huh.NewOption(label, r))
		}

		var selected registry.SearchResult
		if err := huh.NewSelect[registry.SearchResult]().
			Title(fmt.Sprintf("Multiple sources found for %q. Which one?", partial.Name)).
			Options(options...).
			Value(&selected).
			Run(); err != nil {
			return partial, err
		}

		partial.Author = selected.Author
		if partial.Version == "" && selected.LatestVersion != "" {
			partial.Version = selected.LatestVersion
		}

		return partial, nil
	}
}

// filterExactName returns only results where the package name exactly matches.
func filterExactName(results []registry.SearchResult, name string) []registry.SearchResult {
	filtered := make([]registry.SearchResult, 0, len(results))
	for _, r := range results {
		if r.Name == name {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// filterInstallable removes results that don't have a release archive.
// Returns the filtered list and the count of hidden results.
func filterInstallable(results []registry.SearchResult) ([]registry.SearchResult, int) {
	return shared.FilterInstallable(results)
}

// authorSuggestions builds suggestion entries listing each available author
// for a multi-match scenario in non-interactive mode.
func authorSuggestions(results []registry.SearchResult, partial registry.PartialDepKey) []tui.Suggestion {
	suggestions := make([]tui.Suggestion, 0, len(results)+1)
	suggestions = append(suggestions, tui.Suggestion{Text: "Specify the author explicitly:"})
	for _, r := range results {
		ver := "latest"
		if partial.Version != "" {
			ver = partial.Version
		}
		suggestions = append(suggestions, tui.Suggestion{
			Command: fmt.Sprintf("codectx add %s@%s:%s", r.Name, r.Author, ver),
		})
	}
	return suggestions
}

// pluralize returns singular when n == 1, plural otherwise.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// isDuplicate checks if a dependency with the same name@author already exists
// in the project config (regardless of version).
func isDuplicate(cfg *project.Config, ref string) bool {
	for key := range cfg.Dependencies {
		dk, err := registry.ParseDepKey(key)
		if err != nil {
			continue
		}
		if dk.PackageRef() == ref {
			return true
		}
	}
	return false
}

// resolveTarget determines where the dependency should be added based on
// CLI flags and interactive prompts.
func resolveTarget(cmd *cli.Command, cfg *project.Config, projectDir string) (depTarget, error) {
	// For standard projects, always target the project config.
	if !cfg.IsPackage() {
		return targetProject, nil
	}

	// Check CLI flags.
	flagProject := cmd.Bool("project")
	flagPackage := cmd.Bool("package")
	flagBoth := cmd.Bool("both")

	flagCount := shared.BoolCount(flagProject, flagPackage, flagBoth)
	if flagCount > 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Conflicting flags",
			Detail: []string{
				fmt.Sprintf("Only one of %s, %s, or %s may be specified.",
					tui.StyleCommand.Render("--project"),
					tui.StyleCommand.Render("--package"),
					tui.StyleCommand.Render("--both"),
				),
			},
		}.Render())
		return 0, fmt.Errorf("conflicting target flags")
	}

	if flagProject {
		return targetProject, nil
	}
	if flagPackage {
		return targetPackage, nil
	}
	if flagBoth {
		return targetBoth, nil
	}

	// Interactive prompt.
	interactive := term.IsTerminal(os.Stdin.Fd())
	if !interactive {
		// Default to project-only in non-interactive mode.
		return targetProject, nil
	}

	// Verify package dir exists.
	pkgDir := project.PackageContentPath(projectDir)
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		fmt.Print(tui.WarnMsg{
			Title: "Package directory not found",
			Detail: []string{
				fmt.Sprintf("Expected %s but it doesn't exist.", tui.StylePath.Render(pkgDir)),
				"Adding to project only.",
			},
		}.Render())
		return targetProject, nil
	}

	var selected depTarget
	if err := huh.NewSelect[depTarget]().
		Title("This is a package project. Where should this dependency be added?").
		Options(
			huh.NewOption("Project only (authoring workspace)", targetProject),
			huh.NewOption("Package only (published as transitive dep)", targetPackage),
			huh.NewOption("Both (authoring + published)", targetBoth),
		).
		Value(&selected).
		Run(); err != nil {
		return 0, err
	}

	return selected, nil
}

// applyDependency writes the dependency to the appropriate config file(s).
func applyDependency(
	projectDir string,
	cfg *project.Config,
	dk registry.DepKey,
	target depTarget,
	inactive bool,
) error {
	depKey := dk.String()

	switch target {
	case targetProject:
		// Add to root codectx.yml with active flag.
		return addToProjectConfig(projectDir, cfg, depKey, !inactive)

	case targetPackage:
		// Add to package/codectx.yml as semver range.
		if err := addToPackageManifest(projectDir, dk); err != nil {
			return err
		}
		// Also add to root codectx.yml for install (inactive by default,
		// unless --inactive=false was explicitly set via the active flag logic).
		return addToProjectConfig(projectDir, cfg, depKey, false)

	case targetBoth:
		// Add to package/codectx.yml as semver range.
		if err := addToPackageManifest(projectDir, dk); err != nil {
			return err
		}
		// Add to root codectx.yml with active flag.
		return addToProjectConfig(projectDir, cfg, depKey, !inactive)

	default:
		return fmt.Errorf("unknown target: %d", target)
	}
}

// addToProjectConfig adds a dependency entry to the root codectx.yml.
func addToProjectConfig(projectDir string, cfg *project.Config, depKey string, active bool) error {
	if cfg.Dependencies == nil {
		cfg.Dependencies = make(map[string]*project.DependencyConfig)
	}
	cfg.Dependencies[depKey] = &project.DependencyConfig{Active: active}

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

// addToPackageManifest adds a dependency to the package/codectx.yml manifest
// using a semver range constraint.
func addToPackageManifest(projectDir string, dk registry.DepKey) error {
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

	if manifest.Dependencies == nil {
		manifest.Dependencies = make(map[string]string)
	}

	// Use semver range format for package dependencies.
	ref := dk.PackageRef()
	constraint := toSemverRange(dk.Version)
	manifest.Dependencies[ref] = constraint

	if err := manifest.WriteToFile(manifestPath); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to update package manifest",
			Detail: []string{err.Error()},
		}.Render())
		return fmt.Errorf("writing package manifest: %w", err)
	}

	return nil
}

// toSemverRange converts a version string to a semver range constraint
// suitable for package/codectx.yml.
//
// Examples:
//
//	"latest" -> "latest"
//	"2.3.1"  -> ">=2.3.1"
//	">=1.0"  -> ">=1.0" (already a range)
func toSemverRange(version string) string {
	if version == registry.LatestVersion {
		return version
	}
	if len(version) > 0 && (version[0] == '>' || version[0] == '<' || version[0] == '=') {
		return version // Already a range expression.
	}
	return ">=" + version
}

// printTargetInfo prints where the dependency was added.
func printTargetInfo(target depTarget) {
	var value string
	switch target {
	case targetProject:
		value = tui.StylePath.Render("codectx.yml") + tui.StyleMuted.Render(" (project)")
	case targetPackage:
		value = tui.StylePath.Render("package/codectx.yml") +
			tui.StyleMuted.Render(" (semver range) + ") +
			tui.StylePath.Render("codectx.yml") +
			tui.StyleMuted.Render(" (inactive)")
	case targetBoth:
		value = tui.StylePath.Render("codectx.yml") +
			tui.StyleMuted.Render(" (project) + ") +
			tui.StylePath.Render("package/codectx.yml") +
			tui.StyleMuted.Render(" (semver range)")
	}
	fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue("Added to", value))
}
