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
	ArgsUsage: "<name@org:version>",
	Description: `Adds a dependency to codectx.yml and installs it.

Examples:
  codectx add react-patterns@community:latest
  codectx add company-standards@acme:2.0.0

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
	},
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	// --- Step 1: Validate arguments ---
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing dependency argument",
			Detail: []string{
				"Usage: codectx add <name@org:version>",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Example:", Command: "codectx add react-patterns@community:latest"},
			},
		}.Render())
		return fmt.Errorf("missing dependency argument")
	}

	depStr := cmd.Args().First()
	dk, err := registry.ParseDepKey(depStr)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Invalid dependency format",
			Detail: []string{err.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Expected format:", Command: "name@org:version"},
				{Text: "Example:", Command: "react-patterns@community:latest"},
			},
		}.Render())
		return err
	}

	// --- Step 2: Discover project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 3: Check for duplicate ---
	depKey := dk.String()
	ref := dk.PackageRef()

	if isDuplicate(cfg, ref) {
		fmt.Printf("\n%s Dependency %s is already declared in codectx.yml\n\n",
			tui.Warning(),
			tui.StyleAccent.Render(ref),
		)
		return fmt.Errorf("dependency %s already exists", ref)
	}

	// --- Step 4: Determine target ---
	target, err := resolveTarget(cmd, cfg, projectDir)
	if err != nil {
		return err
	}

	inactive := cmd.Bool("inactive")

	// --- Step 5: Update config file(s) ---
	if err := applyDependency(projectDir, cfg, dk, target, inactive); err != nil {
		return err
	}

	// --- Step 6: Install ---
	fmt.Printf("\n%s Added %s\n", tui.Success(), tui.StyleAccent.Render(depKey))
	printTargetInfo(target)

	rootDir := project.RootDir(projectDir, cfg)
	lockPath := filepath.Join(rootDir, registry.LockFileName)
	packagesDir := project.PackagesPath(rootDir)
	reg := cfg.EffectiveRegistry()

	return shared.ResolveAndInstall(ctx, cfg, reg, rootDir, packagesDir, lockPath)
}

// isDuplicate checks if a dependency with the same name@org already exists
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
			Title:  "Conflicting flags",
			Detail: []string{"Only one of --project, --package, or --both may be specified."},
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
	switch target {
	case targetProject:
		fmt.Printf("%s%s\n",
			tui.Indent(1),
			tui.StyleMuted.Render("Added to: codectx.yml (project)"),
		)
	case targetPackage:
		fmt.Printf("%s%s\n",
			tui.Indent(1),
			tui.StyleMuted.Render("Added to: package/codectx.yml (semver range) + codectx.yml (inactive, for install)"),
		)
	case targetBoth:
		fmt.Printf("%s%s\n",
			tui.Indent(1),
			tui.StyleMuted.Render("Added to: codectx.yml (project) + package/codectx.yml (semver range)"),
		)
	}
}
