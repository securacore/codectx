package add

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/resolve"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

var Command = &cli.Command{
	Name:      "add",
	Usage:     "Add one or more documentation packages",
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
	cfg, err := config.Load(configFile)
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
		activation, err = parseActivateFlag(activateFlag)
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
			collisions := detectCollisions(cfg, t.manifest, activation)
			if len(collisions) > 0 {
				ui.Blank()
				ui.Warn(fmt.Sprintf("%d collision(s) for %s@%s:", len(collisions), t.ref.Name, t.ref.Author))
				for _, c := range collisions {
					ui.Item(fmt.Sprintf("[%s] %s already active from %s", c.section, c.id, c.pkg))
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

	if err := config.Write(configFile, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Sync local manifest: discover new entries, remove stale, infer relationships.
	manifestPath := filepath.Join(docsDir, "manifest.yml")
	if localManifest, loadErr := manifest.Load(manifestPath); loadErr == nil {
		synced := manifest.Sync(docsDir, localManifest)
		_ = manifest.Write(manifestPath, synced)
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

// parseActivateFlag parses the --activate flag value into an Activation.
// Accepted values: "all", "none", or "section:id,section:id,..."
func parseActivateFlag(value string) (config.Activation, error) {
	if value == "all" {
		return config.Activation{Mode: "all"}, nil
	}
	if value == "none" {
		return config.Activation{Mode: "none"}, nil
	}

	// Parse granular: "topics:react,foundation:philosophy"
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

// collision represents a single entry ID that collides with an already-active entry.
type collision struct {
	section string
	id      string
	pkg     string // "local" or "name@author"
}

// detectCollisions checks if any entries in the new package would collide
// with entries already active in the local manifest or other installed packages.
func detectCollisions(cfg *config.Config, newManifest *manifest.Manifest, activation config.Activation) []collision {
	// Collect all currently active entry IDs.
	activeIDs := make(map[string]string) // "section:id" -> source package label

	// Load and sync local manifest.
	docsDir := cfg.DocsDir()
	localManifestPath := filepath.Join(docsDir, "manifest.yml")
	if localManifest, err := manifest.Load(localManifestPath); err == nil {
		localManifest = manifest.Sync(docsDir, localManifest)
		for key := range compile.CollectActiveIDs(localManifest) {
			activeIDs[key] = "local"
		}
	}

	// Load each active installed package manifest.
	for _, pkg := range cfg.Packages {
		if pkg.Active.IsNone() {
			continue
		}
		pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", pkg.Name, pkg.Author))
		pkgManifestPath := filepath.Join(pkgDir, "manifest.yml")
		pkgManifest, err := manifest.Load(pkgManifestPath)
		if err != nil {
			continue
		}
		pkgManifest = manifest.Discover(pkgDir, pkgManifest)
		filtered := filterManifestForIDs(pkgManifest, pkg.Active)
		pkgLabel := fmt.Sprintf("%s@%s", pkg.Name, pkg.Author)
		for key := range compile.CollectActiveIDs(filtered) {
			activeIDs[key] = pkgLabel
		}
	}

	// Filter the new package manifest by its activation and check for collisions.
	filtered := filterManifestForIDs(newManifest, activation)
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
// This mirrors compile.filterManifest but operates locally to avoid circular deps.
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
		ids := toSetLocal(am.Foundation)
		for _, e := range m.Foundation {
			if ids[e.ID] {
				filtered.Foundation = append(filtered.Foundation, e)
			}
		}
	}
	if am.Application != nil {
		ids := toSetLocal(am.Application)
		for _, e := range m.Application {
			if ids[e.ID] {
				filtered.Application = append(filtered.Application, e)
			}
		}
	}
	if am.Topics != nil {
		ids := toSetLocal(am.Topics)
		for _, e := range m.Topics {
			if ids[e.ID] {
				filtered.Topics = append(filtered.Topics, e)
			}
		}
	}
	if am.Prompts != nil {
		ids := toSetLocal(am.Prompts)
		for _, e := range m.Prompts {
			if ids[e.ID] {
				filtered.Prompts = append(filtered.Prompts, e)
			}
		}
	}
	if am.Plans != nil {
		ids := toSetLocal(am.Plans)
		for _, e := range m.Plans {
			if ids[e.ID] {
				filtered.Plans = append(filtered.Plans, e)
			}
		}
	}

	return filtered
}

func toSetLocal(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

// splitKey splits "section:id" into its parts.
func splitKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
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
