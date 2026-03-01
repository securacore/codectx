// Package activate implements the codectx activate command. It provides
// both a full-screen interactive TUI for managing package activation and
// a direct CLI for activating specific packages by name.
package activate

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

var Command = &cli.Command{
	Name:      "activate",
	Usage:     "Manage package activation for compilation",
	Category:  "Package Management",
	ArgsUsage: "[package@author ...]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "activate",
			Usage: "Activation mode: all, none, or section:id,... (e.g., topics:react,foundation:philosophy)",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() == 0 {
			return runInteractive()
		}
		return runCLI(c.Args().Slice(), c.String("activate"))
	},
}

// runCLI activates one or more packages by name via the command line.
func runCLI(identifiers []string, activateFlag string) error {
	cfg, err := config.Load(shared.ConfigFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Default to "all" when no --activate flag is given.
	var activation config.Activation
	if activateFlag != "" {
		activation, err = shared.ParseActivateFlag(activateFlag)
		if err != nil {
			return fmt.Errorf("parse --activate: %w", err)
		}
	} else {
		activation = config.Activation{Mode: "all"}
	}

	modified := false
	for _, ident := range identifiers {
		idx := findPackage(cfg, ident)
		if idx < 0 {
			ui.Fail(fmt.Sprintf("package %s not found in config", ident))
			continue
		}

		// Check for collisions when activating.
		if !activation.IsNone() {
			docsDir := cfg.DocsDir()
			pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", cfg.Packages[idx].Name, cfg.Packages[idx].Author))
			pkgManifestPath := filepath.Join(pkgDir, "manifest.yml")
			if pkgManifest, err := manifest.Load(pkgManifestPath); err == nil {
				pkgManifest = manifest.Discover(pkgDir, pkgManifest)
				collisions := shared.DetectCollisions(cfg, idx, pkgManifest, activation)
				if len(collisions) > 0 {
					ui.Warn(fmt.Sprintf("%d collision(s) for %s:", len(collisions), ident))
					for _, c := range collisions {
						ui.Item(fmt.Sprintf("[%s] %s already active from %s", c.Section, c.ID, c.Pkg))
					}
				}
			}
		}

		cfg.Packages[idx].Active = activation
		modified = true
		ui.Done(fmt.Sprintf("Activated %s@%s: %s", cfg.Packages[idx].Name, cfg.Packages[idx].Author, activationLabel(activation)))
	}

	if !modified {
		return fmt.Errorf("no packages were modified")
	}

	if err := config.Write(shared.ConfigFile, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if err := shared.MaybeAutoCompile(cfg); err != nil {
		return err
	}

	return nil
}

// findPackage looks up a package in the config by "name@author" identifier.
// Returns the index or -1 if not found.
func findPackage(cfg *config.Config, ident string) int {
	parts := strings.SplitN(ident, "@", 2)
	if len(parts) != 2 {
		return -1
	}
	name, author := parts[0], parts[1]
	for i, pkg := range cfg.Packages {
		if pkg.Name == name && pkg.Author == author {
			return i
		}
	}
	return -1
}

// activationLabel returns a human-readable label for an activation state.
func activationLabel(a config.Activation) string {
	if a.IsAll() {
		return "all"
	}
	if a.IsNone() {
		return "none"
	}
	var parts []string
	if len(a.Map.Foundation) > 0 {
		parts = append(parts, fmt.Sprintf("foundation: %s", strings.Join(a.Map.Foundation, ", ")))
	}
	if len(a.Map.Application) > 0 {
		parts = append(parts, fmt.Sprintf("application: %s", strings.Join(a.Map.Application, ", ")))
	}
	if len(a.Map.Topics) > 0 {
		parts = append(parts, fmt.Sprintf("topics: %s", strings.Join(a.Map.Topics, ", ")))
	}
	if len(a.Map.Prompts) > 0 {
		parts = append(parts, fmt.Sprintf("prompts: %s", strings.Join(a.Map.Prompts, ", ")))
	}
	if len(a.Map.Plans) > 0 {
		parts = append(parts, fmt.Sprintf("plans: %s", strings.Join(a.Map.Plans, ", ")))
	}
	return strings.Join(parts, "; ")
}

// activationEntryCount returns the total number of activated entries.
func activationEntryCount(a config.Activation) int {
	if a.IsNone() || a.Map == nil {
		return 0
	}
	return len(a.Map.Foundation) + len(a.Map.Application) + len(a.Map.Topics) + len(a.Map.Prompts) + len(a.Map.Plans)
}
