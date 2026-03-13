package version

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// bumpCommand is the CLI definition for `codectx version bump`.
var bumpCommand = &cli.Command{
	Name:      "bump",
	Usage:     "Bump the project/package version",
	ArgsUsage: "[major|minor|patch]",
	Description: `Increments the version in codectx.yml. Defaults to patch if no argument given.

For package authoring projects (type: "package"), also updates the version in
package/codectx.yml to keep both files in sync.

Examples:
  codectx version bump           # 0.1.0 -> 0.1.1
  codectx version bump minor     # 0.1.1 -> 0.2.0
  codectx version bump major     # 0.2.0 -> 1.0.0`,
	Action: runBump,
}

func runBump(_ context.Context, cmd *cli.Command) error {
	part := "patch"
	if cmd.NArg() > 0 {
		part = cmd.Args().First()
	}

	if part != "major" && part != "minor" && part != "patch" {
		fmt.Print(tui.ErrorMsg{
			Title: "Invalid bump type",
			Detail: []string{
				fmt.Sprintf("Got %q, expected one of: major, minor, patch", part),
			},
		}.Render())
		return fmt.Errorf("invalid bump type: %s", part)
	}

	// Discover project.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	oldVersion := cfg.Version
	if oldVersion == "" {
		oldVersion = "0.0.0"
	}

	newVersion, err := bumpVersion(oldVersion, part)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to bump version",
			Detail: []string{err.Error()},
		}.Render())
		return err
	}

	// Update root codectx.yml.
	cfg.Version = newVersion
	cfgPath := projectDir + "/" + project.ConfigFileName
	if err := cfg.WriteToFile(cfgPath); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to update codectx.yml",
			Detail: []string{err.Error()},
		}.Render())
		return err
	}

	fmt.Printf("\n%s Version bumped: %s -> %s\n",
		tui.Success(),
		tui.StyleMuted.Render(oldVersion),
		tui.StyleBold.Render(newVersion),
	)
	fmt.Printf("%s%s\n",
		tui.Indent(1),
		tui.StyleMuted.Render("Updated: codectx.yml"),
	)

	// For package projects, also update package/codectx.yml.
	if cfg.IsPackage() {
		manifestPath := project.PackageConfigPath(projectDir)
		manifest, loadErr := project.LoadPackageManifest(manifestPath)
		if loadErr == nil {
			manifest.Version = newVersion
			if writeErr := manifest.WriteToFile(manifestPath); writeErr != nil {
				fmt.Printf("%s%s Failed to update package manifest: %v\n",
					tui.Indent(1), tui.Warning(), writeErr)
			} else {
				fmt.Printf("%s%s\n",
					tui.Indent(1),
					tui.StyleMuted.Render("Updated: package/codectx.yml"),
				)
			}
		}
	}

	fmt.Println()
	return nil
}

// bumpVersion increments a semver version string by the given part.
// Input is expected without "v" prefix (e.g. "1.2.3").
// Returns the bumped version without "v" prefix.
func bumpVersion(version, part string) (string, error) {
	// Strip "v" prefix if present.
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid semver format %q: expected major.minor.patch", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	switch part {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}
