package add

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"securacore/codectx/core/compile"
	"securacore/codectx/core/config"
	"securacore/codectx/core/manifest"
	"securacore/codectx/core/resolve"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

var Command = &cli.Command{
	Name:      "add",
	Usage:     "Add a documentation package",
	ArgsUsage: "<package>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "source",
			Usage: "Explicit Git repository URL (overrides inference)",
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
		return run(c.Args().First(), c.String("source"), c.String("activate"))
	},
}

func run(input, sourceFlag, activateFlag string) error {
	// Load config.
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Parse package identifier (URL or shorthand).
	var ref *resolve.PackageRef
	var source string

	if resolve.IsURL(input) {
		// Input is a GitHub URL: extract name, author, and source.
		var urlSource string
		ref, urlSource, err = resolve.ParseURL(input)
		if err != nil {
			return fmt.Errorf("parse URL: %w", err)
		}
		source = urlSource
	} else {
		// Input is shorthand (name@author[:version]).
		ref, err = resolve.Parse(input)
		if err != nil {
			return fmt.Errorf("parse package: %w", err)
		}

		// Guard: author required for source inference.
		if ref.Author == "" && sourceFlag == "" {
			return fmt.Errorf("author required: use name@author format or provide --source")
		}

		source = sourceFlag
		if source == "" {
			source = resolve.InferSource(ref.Name, ref.Author)
		}
	}

	// Guard: check if package already exists.
	for _, pkg := range cfg.Packages {
		if pkg.Name == ref.Name && pkg.Author == ref.Author {
			return fmt.Errorf("package %s@%s already exists in config", ref.Name, ref.Author)
		}
	}

	// Resolve version from Git tags.
	fmt.Printf("Resolving %s from %s...\n", input, source)
	resolved, err := resolve.Resolve(ref, source)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}
	fmt.Printf("Resolved version: %s (tag: %s)\n", resolved.Version, resolved.Tag)

	// Fetch (clone) into docs/packages/name@author/.
	docsDir := cfg.DocsDir()
	pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", resolved.Name, resolved.Author))

	fmt.Printf("Fetching to %s...\n", pkgDir)
	if err := resolve.Fetch(resolved, pkgDir); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	// Load the fetched package's manifest.
	pkgManifestPath := filepath.Join(pkgDir, "package.yml")
	pkgManifest, err := manifest.Load(pkgManifestPath)
	if err != nil {
		return fmt.Errorf("load package manifest: %w", err)
	}

	// Determine activation.
	var activation config.Activation
	if activateFlag != "" {
		activation, err = parseActivateFlag(activateFlag)
		if err != nil {
			return fmt.Errorf("parse --activate: %w", err)
		}
	} else {
		activation, err = promptActivation(pkgManifest)
		if err != nil {
			return fmt.Errorf("activation prompt: %w", err)
		}
	}

	// Check for entry collisions against currently active entries.
	if !activation.IsNone() {
		collisions := detectCollisions(cfg, pkgManifest, activation)
		if len(collisions) > 0 {
			fmt.Printf("\nWarning: %d entry collision(s) detected:\n", len(collisions))
			for _, c := range collisions {
				fmt.Printf("  [%s] %s already active from %s\n", c.section, c.id, c.pkg)
			}
			fmt.Println("\nDuring compilation, existing entries take precedence (deduplication).")

			if activateFlag == "" {
				// Interactive mode: ask for confirmation.
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
				)
				if err := confirmForm.Run(); err != nil {
					return fmt.Errorf("confirmation prompt: %w", err)
				}
				if !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}
		}
	}

	// Append to config and write.
	dep := config.PackageDep{
		Name:    resolved.Name,
		Author:  resolved.Author,
		Version: ref.Version,
		Source:  sourceFlag, // only record explicit source
		Active:  activation,
	}

	// Use the constraint from input, or pin to caret range of resolved version.
	if dep.Version == "" {
		dep.Version = fmt.Sprintf("^%s", resolved.Version)
	}

	cfg.Packages = append(cfg.Packages, dep)
	if err := config.Write(configFile, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("\nAdded %s@%s v%s\n", resolved.Name, resolved.Author, resolved.Version)
	printActivation(activation)

	return nil
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

// promptActivation uses huh to interactively select which entries to activate.
func promptActivation(m *manifest.Manifest) (config.Activation, error) {
	// Collect all entry IDs with descriptive labels.
	type entry struct {
		section string
		id      string
		label   string
	}

	var entries []entry
	for _, e := range m.Foundation {
		entries = append(entries, entry{"foundation", e.ID, fmt.Sprintf("[foundation] %s - %s", e.ID, e.Description)})
	}
	for _, e := range m.Topics {
		entries = append(entries, entry{"topics", e.ID, fmt.Sprintf("[topics] %s - %s", e.ID, e.Description)})
	}
	for _, e := range m.Prompts {
		entries = append(entries, entry{"prompts", e.ID, fmt.Sprintf("[prompts] %s - %s", e.ID, e.Description)})
	}
	for _, e := range m.Plans {
		entries = append(entries, entry{"plans", e.ID, fmt.Sprintf("[plans] %s - %s", e.ID, e.Description)})
	}

	if len(entries) == 0 {
		fmt.Println("Package has no entries to activate.")
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
				Description(fmt.Sprintf("Package: %s@%s v%s", m.Name, m.Author, m.Version)).
				Options(options...).
				Height(min(len(entries)+4, 20)).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return config.Activation{}, err
	}

	// If all selected, use "all" mode.
	if len(selected) == len(entries) {
		return config.Activation{Mode: "all"}, nil
	}

	// If none selected, use "none" mode.
	if len(selected) == 0 {
		return config.Activation{Mode: "none"}, nil
	}

	// Build granular activation map.
	am := &config.ActivationMap{}
	for _, idx := range selected {
		e := entries[idx]
		switch e.section {
		case "foundation":
			am.Foundation = append(am.Foundation, e.id)
		case "topics":
			am.Topics = append(am.Topics, e.id)
		case "prompts":
			am.Prompts = append(am.Prompts, e.id)
		case "plans":
			am.Plans = append(am.Plans, e.id)
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

	// Load local manifest.
	docsDir := cfg.DocsDir()
	localManifestPath := filepath.Join(docsDir, "package.yml")
	if localManifest, err := manifest.Load(localManifestPath); err == nil {
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
		pkgManifestPath := filepath.Join(pkgDir, "package.yml")
		pkgManifest, err := manifest.Load(pkgManifestPath)
		if err != nil {
			continue
		}
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
		fmt.Println("Activation: all entries")
		return
	}
	if a.IsNone() {
		fmt.Println("Activation: none (installed but not active)")
		return
	}
	fmt.Println("Activation:")
	if len(a.Map.Foundation) > 0 {
		fmt.Printf("  foundation: %s\n", strings.Join(a.Map.Foundation, ", "))
	}
	if len(a.Map.Topics) > 0 {
		fmt.Printf("  topics:     %s\n", strings.Join(a.Map.Topics, ", "))
	}
	if len(a.Map.Prompts) > 0 {
		fmt.Printf("  prompts:    %s\n", strings.Join(a.Map.Prompts, ", "))
	}
	if len(a.Map.Plans) > 0 {
		fmt.Printf("  plans:      %s\n", strings.Join(a.Map.Plans, ", "))
	}
}
