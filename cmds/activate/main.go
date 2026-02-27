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
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

var Command = &cli.Command{
	Name:      "activate",
	Usage:     "Manage package activation for compilation",
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
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Default to "all" when no --activate flag is given.
	var activation config.Activation
	if activateFlag != "" {
		activation, err = parseActivateFlag(activateFlag)
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
				collisions := detectCollisions(cfg, idx, pkgManifest, activation)
				if len(collisions) > 0 {
					ui.Warn(fmt.Sprintf("%d collision(s) for %s:", len(collisions), ident))
					for _, c := range collisions {
						ui.Item(fmt.Sprintf("[%s] %s already active from %s", c.section, c.id, c.pkg))
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

	if err := config.Write(configFile, cfg); err != nil {
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

// collision represents a single entry ID that collides with an already-active entry.
type collision struct {
	section string
	id      string
	pkg     string
}

// detectCollisions checks if activating the package at skipIdx would collide
// with other currently active entries.
func detectCollisions(cfg *config.Config, skipIdx int, pkgManifest *manifest.Manifest, activation config.Activation) []collision {
	activeIDs := make(map[string]string)

	docsDir := cfg.DocsDir()
	localManifestPath := filepath.Join(docsDir, "manifest.yml")
	if localManifest, err := manifest.Load(localManifestPath); err == nil {
		localManifest = manifest.Sync(docsDir, localManifest)
		for key := range compile.CollectActiveIDs(localManifest) {
			activeIDs[key] = "local"
		}
	}

	for i, pkg := range cfg.Packages {
		if i == skipIdx || pkg.Active.IsNone() {
			continue
		}
		pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", pkg.Name, pkg.Author))
		pkgPath := filepath.Join(pkgDir, "manifest.yml")
		m, err := manifest.Load(pkgPath)
		if err != nil {
			continue
		}
		m = manifest.Discover(pkgDir, m)
		filtered := filterManifestForIDs(m, pkg.Active)
		pkgLabel := fmt.Sprintf("%s@%s", pkg.Name, pkg.Author)
		for key := range compile.CollectActiveIDs(filtered) {
			activeIDs[key] = pkgLabel
		}
	}

	filtered := filterManifestForIDs(pkgManifest, activation)
	newIDs := compile.CollectActiveIDs(filtered)

	var collisions []collision
	for key := range newIDs {
		if pkg, exists := activeIDs[key]; exists {
			section, id := splitKey(key)
			collisions = append(collisions, collision{section: section, id: id, pkg: pkg})
		}
	}

	return collisions
}

// filterManifestForIDs applies activation filtering to a manifest for ID collection.
func filterManifestForIDs(m *manifest.Manifest, activation config.Activation) *manifest.Manifest {
	if activation.IsAll() {
		return m
	}
	if activation.IsNone() {
		return &manifest.Manifest{}
	}

	am := activation.Map
	filtered := &manifest.Manifest{}

	if am.Foundation != nil {
		ids := toSet(am.Foundation)
		for _, e := range m.Foundation {
			if ids[e.ID] {
				filtered.Foundation = append(filtered.Foundation, e)
			}
		}
	}
	if am.Application != nil {
		ids := toSet(am.Application)
		for _, e := range m.Application {
			if ids[e.ID] {
				filtered.Application = append(filtered.Application, e)
			}
		}
	}
	if am.Topics != nil {
		ids := toSet(am.Topics)
		for _, e := range m.Topics {
			if ids[e.ID] {
				filtered.Topics = append(filtered.Topics, e)
			}
		}
	}
	if am.Prompts != nil {
		ids := toSet(am.Prompts)
		for _, e := range m.Prompts {
			if ids[e.ID] {
				filtered.Prompts = append(filtered.Prompts, e)
			}
		}
	}
	if am.Plans != nil {
		ids := toSet(am.Plans)
		for _, e := range m.Plans {
			if ids[e.ID] {
				filtered.Plans = append(filtered.Plans, e)
			}
		}
	}

	return filtered
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

func splitKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

// parseActivateFlag parses the --activate flag into an Activation.
func parseActivateFlag(value string) (config.Activation, error) {
	if value == "all" {
		return config.Activation{Mode: "all"}, nil
	}
	if value == "none" {
		return config.Activation{Mode: "none"}, nil
	}

	am := &config.ActivationMap{}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			return config.Activation{}, fmt.Errorf("invalid format %q: expected section:id", part)
		}
		section := part[:colonIdx]
		id := part[colonIdx+1:]
		if id == "" {
			return config.Activation{}, fmt.Errorf("empty id in %q", part)
		}

		switch section {
		case "foundation":
			am.Foundation = append(am.Foundation, id)
		case "application":
			am.Application = append(am.Application, id)
		case "topics":
			am.Topics = append(am.Topics, id)
		case "prompts":
			am.Prompts = append(am.Prompts, id)
		case "plans":
			am.Plans = append(am.Plans, id)
		default:
			return config.Activation{}, fmt.Errorf("unknown section %q in %q", section, part)
		}
	}

	return config.Activation{Map: am}, nil
}
