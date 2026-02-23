package add

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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

	// Parse package identifier.
	ref, err := resolve.Parse(input)
	if err != nil {
		return fmt.Errorf("parse package: %w", err)
	}

	// Guard: author required for source inference.
	if ref.Author == "" && sourceFlag == "" {
		return fmt.Errorf("author required: use name@author format or provide --source")
	}

	// Guard: check if package already exists.
	for _, pkg := range cfg.Packages {
		if pkg.Name == ref.Name && pkg.Author == ref.Author {
			return fmt.Errorf("package %s@%s already exists in config", ref.Name, ref.Author)
		}
	}

	// Determine source URL.
	source := sourceFlag
	if source == "" {
		source = resolve.InferSource(ref.Name, ref.Author)
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
