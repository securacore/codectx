package add

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/resolve"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"
)

var Command = &cli.Command{
	Name:      "add",
	Aliases:   []string{"a"},
	Usage:     "Add one or more documentation packages",
	Category:  "Core Workflow",
	ArgsUsage: "<package> [package...]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "source",
			Usage: "Explicit Git repository URL (overrides inference, single package only)",
		},
		&cli.StringFlag{
			Name:  "activate",
			Usage: "Non-interactive activation: all, none, or section:id,... (e.g., topics:react,foundation:philosophy)",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() < 1 {
			return fmt.Errorf("package argument required (e.g., name@author, name@author:version)")
		}
		inputs := c.Args().Slice()
		sourceFlag := c.String("source")
		if sourceFlag != "" && len(inputs) > 1 {
			return fmt.Errorf("--source can only be used with a single package")
		}
		return Run(inputs, sourceFlag, c.String("activate"))
	},
}

// addTarget holds the parsed and resolved data for one package being added.
type addTarget struct {
	input    string
	ref      *resolve.PackageRef
	source   string
	resolved *resolve.ResolvedPackage
	pkgDir   string
	manifest *manifest.Manifest
}

// Run adds one or more packages to the project. It resolves, fetches, prompts
// for activation, writes config, and optionally auto-compiles.
// Exported so other commands (e.g., search) can trigger the add flow.
func Run(inputs []string, sourceFlag, activateFlag string) error {
	// Load config.
	cfg, err := config.Load(shared.ConfigFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	docsDir := cfg.DocsDir()
	var targets []*addTarget
	var failures []string

	for _, input := range inputs {
		t, err := parseAndResolve(input, sourceFlag, cfg, docsDir)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", input, err))
			continue
		}
		targets = append(targets, t)
	}

	if len(targets) == 0 {
		for _, f := range failures {
			ui.Fail(f)
		}
		return fmt.Errorf("no packages were added")
	}

	// Determine activation.
	var activation config.Activation
	if activateFlag != "" {
		activation, err = shared.ParseActivateFlag(activateFlag)
		if err != nil {
			return fmt.Errorf("parse --activate: %w", err)
		}
	} else {
		activation, err = promptCombinedActivation(targets)
		if err != nil {
			return fmt.Errorf("activation prompt: %w", err)
		}
	}

	// Check for entry collisions against currently active entries.
	if !activation.IsNone() {
		for _, t := range targets {
			collisions := shared.DetectCollisions(cfg, -1, t.manifest, activation)
			if len(collisions) > 0 {
				ui.Blank()
				ui.Warn(fmt.Sprintf("%d collision(s) for %s@%s:", len(collisions), t.ref.Name, t.ref.Author))
				for _, c := range collisions {
					ui.Item(fmt.Sprintf("[%s] %s already active from %s", c.Section, c.ID, c.Pkg))
				}

				if activateFlag == "" {
					var confirm bool
					confirmForm := huh.NewForm(
						huh.NewGroup(
							huh.NewConfirm().
								Title("Continue with these collisions?").
								Description("Colliding entries will be deduplicated at compile time.").
								Affirmative("Yes, continue").
								Negative("Cancel").
								Value(&confirm),
						),
					).WithTheme(ui.Theme())
					if err := confirmForm.Run(); err != nil {
						return fmt.Errorf("confirmation prompt: %w", err)
					}
					if !confirm {
						ui.Canceled()
						return nil
					}
				}
			}
		}
	}

	// Append all successful targets to config and write once.
	for _, t := range targets {
		dep := config.PackageDep{
			Name:    t.resolved.Name,
			Author:  t.resolved.Author,
			Version: t.ref.Version,
			Source:  sourceFlag,
			Active:  activation,
		}

		if dep.Version == "" {
			dep.Version = fmt.Sprintf("^%s", t.resolved.Version)
		}

		cfg.Packages = append(cfg.Packages, dep)
	}

	if err := config.Write(shared.ConfigFile, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Sync local manifest: discover new entries, remove stale, infer relationships.
	manifestPath := filepath.Join(docsDir, "manifest.yml")
	if localManifest, loadErr := manifest.Load(manifestPath); loadErr == nil {
		synced := manifest.Sync(docsDir, localManifest)
		if writeErr := manifest.Write(manifestPath, synced); writeErr != nil {
			ui.Warn(fmt.Sprintf("Failed to sync manifest: %s", writeErr))
		}
	}

	// Print results.
	ui.Blank()
	for _, t := range targets {
		ui.Done(fmt.Sprintf("Added %s@%s v%s", t.resolved.Name, t.resolved.Author, t.resolved.Version))
	}
	if len(failures) > 0 {
		ui.Blank()
		for _, f := range failures {
			ui.Fail(f)
		}
	}
	printActivation(activation)

	// Auto-compile if preferences say so, or prompt if unset.
	if err := shared.MaybeAutoCompile(cfg); err != nil {
		return err
	}

	return nil
}

// parseAndResolve handles parsing, duplicate checking, resolving, and fetching
// for a single package input string. Returns an addTarget on success.
func parseAndResolve(input, sourceFlag string, cfg *config.Config, docsDir string) (*addTarget, error) {
	var ref *resolve.PackageRef
	var source string
	var err error

	if resolve.IsURL(input) {
		var urlSource string
		ref, urlSource, err = resolve.ParseURL(input)
		if err != nil {
			return nil, fmt.Errorf("parse URL: %w", err)
		}
		source = urlSource
	} else {
		ref, err = resolve.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("parse package: %w", err)
		}
		if ref.Author == "" && sourceFlag == "" {
			return nil, fmt.Errorf("author required: use name@author format or provide --source")
		}
		source = sourceFlag
		if source == "" {
			source = resolve.InferSource(ref.Name, ref.Author)
		}
	}

	// Guard: check if package already exists.
	for _, pkg := range cfg.Packages {
		if pkg.Name == ref.Name && pkg.Author == ref.Author {
			return nil, fmt.Errorf("package %s@%s already exists in config", ref.Name, ref.Author)
		}
	}

	// Resolve version.
	var resolved *resolve.ResolvedPackage
	ui.Spin(fmt.Sprintf("Resolving %s...", input), func() {
		resolved, err = resolve.Resolve(ref, source)
	})
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}

	// Fetch into docs/packages/name@author/.
	pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", resolved.Name, resolved.Author))

	err = ui.SpinErr(fmt.Sprintf("Fetching %s@%s v%s...", resolved.Name, resolved.Author, resolved.Version), func() error {
		return resolve.Fetch(resolved, pkgDir)
	})
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	// Load the fetched package's manifest and discover entries from disk.
	pkgManifestPath := filepath.Join(pkgDir, "manifest.yml")
	pkgManifest, err := manifest.Load(pkgManifestPath)
	if err != nil {
		return nil, fmt.Errorf("load package manifest: %w", err)
	}
	pkgManifest = manifest.Discover(pkgDir, pkgManifest)

	return &addTarget{
		input:    input,
		ref:      ref,
		source:   source,
		resolved: resolved,
		pkgDir:   pkgDir,
		manifest: pkgManifest,
	}, nil
}

// activationEntry represents a selectable manifest entry for activation prompts.
type activationEntry struct {
	section string
	id      string
	label   string
}

// buildCombinedEntries collects all selectable manifest entries from the
// given add targets, grouped by package label.
func buildCombinedEntries(targets []*addTarget) []activationEntry {
	var entries []activationEntry
	for _, t := range targets {
		pkgLabel := fmt.Sprintf("%s@%s", t.resolved.Name, t.resolved.Author)
		for _, e := range t.manifest.Foundation {
			entries = append(entries, activationEntry{"foundation", e.ID, fmt.Sprintf("[%s / foundation] %s - %s", pkgLabel, e.ID, e.Description)})
		}
		for _, e := range t.manifest.Application {
			entries = append(entries, activationEntry{"application", e.ID, fmt.Sprintf("[%s / application] %s - %s", pkgLabel, e.ID, e.Description)})
		}
		for _, e := range t.manifest.Topics {
			entries = append(entries, activationEntry{"topics", e.ID, fmt.Sprintf("[%s / topics] %s - %s", pkgLabel, e.ID, e.Description)})
		}
		for _, e := range t.manifest.Prompts {
			entries = append(entries, activationEntry{"prompts", e.ID, fmt.Sprintf("[%s / prompts] %s - %s", pkgLabel, e.ID, e.Description)})
		}
		for _, e := range t.manifest.Plans {
			entries = append(entries, activationEntry{"plans", e.ID, fmt.Sprintf("[%s / plans] %s - %s", pkgLabel, e.ID, e.Description)})
		}
	}
	return entries
}

// resolveActivation converts a user's entry selection into an Activation.
// If all entries are selected, it returns "all" mode.
// If none are selected, it returns "none" mode.
// Otherwise, it builds a granular ActivationMap.
func resolveActivation(entries []activationEntry, selected []int) config.Activation {
	if len(selected) == len(entries) {
		return config.Activation{Mode: "all"}
	}
	if len(selected) == 0 {
		return config.Activation{Mode: "none"}
	}

	am := &config.ActivationMap{}
	for _, idx := range selected {
		e := entries[idx]
		switch e.section {
		case "foundation":
			am.Foundation = append(am.Foundation, e.id)
		case "application":
			am.Application = append(am.Application, e.id)
		case "topics":
			am.Topics = append(am.Topics, e.id)
		case "prompts":
			am.Prompts = append(am.Prompts, e.id)
		case "plans":
			am.Plans = append(am.Plans, e.id)
		}
	}
	return config.Activation{Map: am}
}

// promptCombinedActivation shows a single multi-select with entries from all
// packages grouped by package label. Each entry is prefixed with the package
// it belongs to.
func promptCombinedActivation(targets []*addTarget) (config.Activation, error) {
	entries := buildCombinedEntries(targets)

	if len(entries) == 0 {
		ui.Done("Packages have no entries to activate.")
		return config.Activation{Mode: "none"}, nil
	}

	// Build huh options (all selected by default).
	options := make([]huh.Option[int], len(entries))
	for i, e := range entries {
		options[i] = huh.NewOption(e.label, i).Selected(true)
	}

	var selected []int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select entries to activate").
				Description("Entries from all packages are shown below").
				Options(options...).
				Height(min(len(entries)+4, 20)).
				Value(&selected),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return config.Activation{}, err
	}

	return resolveActivation(entries, selected), nil
}

// printActivation prints a human-readable summary of the activation state.
func printActivation(a config.Activation) {
	if a.IsAll() {
		ui.KV("Activation", "all entries", 14)
		return
	}
	if a.IsNone() {
		ui.KV("Activation", "none (installed but not active)", 14)
		return
	}
	ui.Header("Activation:")
	if len(a.Map.Foundation) > 0 {
		ui.KV("foundation", strings.Join(a.Map.Foundation, ", "), 14)
	}
	if len(a.Map.Application) > 0 {
		ui.KV("application", strings.Join(a.Map.Application, ", "), 14)
	}
	if len(a.Map.Topics) > 0 {
		ui.KV("topics", strings.Join(a.Map.Topics, ", "), 14)
	}
	if len(a.Map.Prompts) > 0 {
		ui.KV("prompts", strings.Join(a.Map.Prompts, ", "), 14)
	}
	if len(a.Map.Plans) > 0 {
		ui.KV("plans", strings.Join(a.Map.Plans, ", "), 14)
	}
}
