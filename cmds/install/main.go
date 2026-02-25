// Package install implements the codectx install command. It sets up a
// project from an existing codectx.yml by installing all declared packages,
// prompting for activation, and running the initial compilation.
package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/lock"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/core/resolve"
	"github.com/securacore/codectx/core/schema"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"
const lockFile = "codectx.lock"
const manifestFile = "manifest.yml"
const defaultBackupDir = ".codectx-docs"

var Command = &cli.Command{
	Name:  "install",
	Usage: "Install packages from codectx.yml and set up the project",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "activate",
			Usage: "Non-interactive activation: all, none, or section:id,...",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		return run(c.String("activate"))
	},
}

// installedPkg holds the result of a successful package installation.
type installedPkg struct {
	idx      int
	pkg      config.PackageDep
	manifest *manifest.Manifest
}

func run(activateFlag string) error {
	// Load config.
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Packages) == 0 {
		ui.Done("No packages declared in codectx.yml.")
		return nil
	}

	// Set up docs directory.
	docsDir := cfg.DocsDir()
	if err := setupDocsDir(cfg, &docsDir); err != nil {
		return err
	}

	// Ensure output directory and gitignore.
	outputDir := cfg.OutputDir()
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := ensureGitignore(outputDir); err != nil {
		return err
	}

	// Load lock file for pinned versions.
	lck, err := lock.Load(lockFile)
	if err != nil {
		return fmt.Errorf("load lock file: %w", err)
	}

	// Install each package.
	var successes []installedPkg
	var failures []string

	for i, pkg := range cfg.Packages {
		pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", pkg.Name, pkg.Author))

		// Skip if already fetched.
		if _, err := os.Stat(filepath.Join(pkgDir, manifestFile)); err == nil {
			m, loadErr := manifest.Load(filepath.Join(pkgDir, manifestFile))
			if loadErr == nil {
				m = manifest.Discover(pkgDir, m)
				successes = append(successes, installedPkg{idx: i, pkg: pkg, manifest: m})
				ui.Done(fmt.Sprintf("Already installed: %s@%s", pkg.Name, pkg.Author))
				continue
			}
		}

		// Resolve version.
		ref := &resolve.PackageRef{
			Name:    pkg.Name,
			Author:  pkg.Author,
			Version: pkg.Version,
		}

		// Use lock file version if available.
		if lck != nil {
			for _, lp := range lck.Packages {
				if lp.Name == pkg.Name && lp.Author == pkg.Author {
					ref.Version = lp.Version
					break
				}
			}
		}

		source := pkg.Source
		if source == "" {
			source = resolve.InferSource(pkg.Name, pkg.Author)
		}

		var resolved *resolve.ResolvedPackage
		err = ui.SpinErr(fmt.Sprintf("Resolving %s@%s...", pkg.Name, pkg.Author), func() error {
			var resolveErr error
			resolved, resolveErr = resolve.Resolve(ref, source)
			return resolveErr
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s@%s: %s", pkg.Name, pkg.Author, err))
			continue
		}

		err = ui.SpinErr(fmt.Sprintf("Fetching %s@%s v%s...", pkg.Name, pkg.Author, resolved.Version), func() error {
			return resolve.Fetch(resolved, pkgDir)
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s@%s: %s", pkg.Name, pkg.Author, err))
			continue
		}

		m, loadErr := manifest.Load(filepath.Join(pkgDir, manifestFile))
		if loadErr != nil {
			failures = append(failures, fmt.Sprintf("%s@%s: load manifest: %s", pkg.Name, pkg.Author, loadErr))
			continue
		}
		m = manifest.Discover(pkgDir, m)

		successes = append(successes, installedPkg{idx: i, pkg: pkg, manifest: m})
		ui.Done(fmt.Sprintf("Installed %s@%s v%s", pkg.Name, pkg.Author, resolved.Version))
	}

	// Report summary.
	ui.Blank()
	if len(failures) > 0 {
		ui.Warn(fmt.Sprintf("%d package(s) failed:", len(failures)))
		for _, f := range failures {
			ui.Item(f)
		}
		ui.Blank()
	}

	if len(successes) == 0 {
		return fmt.Errorf("no packages were installed successfully")
	}

	// Prompt for activation if any packages are inactive.
	hasInactive := false
	for _, s := range successes {
		if s.pkg.Active.IsNone() {
			hasInactive = true
			break
		}
	}

	if hasInactive {
		if activateFlag != "" {
			activation, err := parseActivateFlag(activateFlag)
			if err != nil {
				return fmt.Errorf("parse --activate: %w", err)
			}
			for _, s := range successes {
				if s.pkg.Active.IsNone() {
					cfg.Packages[s.idx].Active = activation
				}
			}
		} else {
			// Build combined entry-level multi-select.
			if err := promptCombinedActivation(cfg, successes); err != nil {
				return fmt.Errorf("activation prompt: %w", err)
			}
		}

		if err := config.Write(configFile, cfg); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
	}

	// Ask about auto-compilation preference.
	prefs, err := preferences.Load(outputDir)
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	if prefs.AutoCompile == nil {
		var confirmStr string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Auto-compile after adding packages?").
					Description("Automatically recompile documentation when packages are added or changed").
					Options(
						huh.NewOption("Yes", "yes"),
						huh.NewOption("No", "no"),
					).
					Value(&confirmStr),
			),
		).WithTheme(ui.Theme())

		if err := form.Run(); err != nil {
			return fmt.Errorf("prompt: %w", err)
		}

		val := confirmStr == "yes"
		prefs.AutoCompile = &val
		if err := preferences.Write(outputDir, prefs); err != nil {
			return fmt.Errorf("write preferences: %w", err)
		}
	}

	// Sync local manifest: discover new entries, remove stale, infer relationships.
	manifestPath := filepath.Join(docsDir, manifestFile)
	if localManifest, loadErr := manifest.Load(manifestPath); loadErr == nil {
		synced := manifest.Sync(docsDir, localManifest)
		_ = manifest.Write(manifestPath, synced)
	}

	// Run initial compilation.
	ui.Blank()
	var result *compile.Result
	err = ui.SpinErr("Compiling...", func() error {
		var compileErr error
		result, compileErr = compile.Compile(cfg)
		return compileErr
	})
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	ui.Done(fmt.Sprintf("Compiled to %s", result.OutputDir))
	ui.KV("Objects stored", result.ObjectsStored, 16)
	if result.ObjectsPruned > 0 {
		ui.KV("Objects pruned", result.ObjectsPruned, 16)
	}
	ui.KV("Packages", result.Packages, 16)

	return nil
}

// setupDocsDir ensures the docs directory exists and is compatible.
// If the directory exists but is incompatible, prompts for an alternative.
// If it doesn't exist, creates the full structure.
func setupDocsDir(cfg *config.Config, docsDir *string) error {
	if _, err := os.Stat(*docsDir); err == nil {
		// Directory exists — check compatibility.
		issues := checkCompatibility(*docsDir)
		if len(issues) > 0 {
			ui.Blank()
			ui.Warn(fmt.Sprintf("The %s/ directory is not compatible:", *docsDir))
			for _, issue := range issues {
				ui.Item(fmt.Sprintf("%s: %s", issue.path, issue.reason))
			}
			ui.Blank()

			// Prompt for alternative directory.
			var altDir string
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter alternative docs directory").
						Description(fmt.Sprintf("Leave empty for default: %s", defaultBackupDir)).
						Placeholder(defaultBackupDir).
						Value(&altDir),
				),
			).WithTheme(ui.Theme())

			if err := form.Run(); err != nil {
				return fmt.Errorf("prompt: %w", err)
			}

			if altDir == "" {
				altDir = defaultBackupDir
			}
			*docsDir = altDir

			// Update config with new docs dir.
			if cfg.Config == nil {
				cfg.Config = &config.BuildConfig{}
			}
			cfg.Config.DocsDir = *docsDir
			if err := config.Write(configFile, cfg); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
		}
	}

	// Create directory structure.
	dirs := []string{
		*docsDir,
		filepath.Join(*docsDir, "packages"),
		filepath.Join(*docsDir, "schemas"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Write schemas.
	schemasDir := filepath.Join(*docsDir, "schemas")
	if err := schema.WriteAll(schemasDir); err != nil {
		return fmt.Errorf("write schemas: %w", err)
	}

	// Create minimal manifest.yml if it doesn't exist.
	pkgPath := filepath.Join(*docsDir, manifestFile)
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		m := &manifest.Manifest{
			Name:        cfg.Name,
			Author:      "",
			Version:     "0.1.0",
			Description: fmt.Sprintf("Documentation package for %s", cfg.Name),
		}
		if err := manifest.Write(pkgPath, m); err != nil {
			return fmt.Errorf("write package manifest: %w", err)
		}
	}

	return nil
}

// ensureGitignore adds the output directory to .gitignore if not already present.
func ensureGitignore(outputDir string) error {
	entry := outputDir + "/"
	path := ".gitignore"

	if data, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == entry {
				return nil
			}
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open .gitignore: %w", err)
		}
		defer func() { _ = f.Close() }()
		if len(data) > 0 && data[len(data)-1] != '\n' {
			if _, err := f.WriteString("\n"); err != nil {
				return fmt.Errorf("write newline to .gitignore: %w", err)
			}
		}
		if _, err := f.WriteString(entry + "\n"); err != nil {
			return fmt.Errorf("append to .gitignore: %w", err)
		}
		return nil
	}

	return os.WriteFile(path, []byte(entry+"\n"), 0o644)
}

// promptCombinedActivation shows a single multi-select with entries from all
// inactive packages, grouped by package label.
// activationEntry represents a selectable manifest entry for activation prompts.
type activationEntry struct {
	pkgIdx  int // index into cfg.Packages
	section string
	id      string
	label   string
}

// buildActivationEntries collects all selectable manifest entries from
// newly installed packages that have no activation set yet.
func buildActivationEntries(successes []installedPkg) []activationEntry {
	var entries []activationEntry
	for _, s := range successes {
		if !s.pkg.Active.IsNone() {
			continue
		}
		pkgLabel := fmt.Sprintf("%s@%s", s.pkg.Name, s.pkg.Author)
		for _, e := range s.manifest.Foundation {
			entries = append(entries, activationEntry{s.idx, "foundation", e.ID, fmt.Sprintf("[%s / foundation] %s - %s", pkgLabel, e.ID, e.Description)})
		}
		for _, e := range s.manifest.Application {
			entries = append(entries, activationEntry{s.idx, "application", e.ID, fmt.Sprintf("[%s / application] %s - %s", pkgLabel, e.ID, e.Description)})
		}
		for _, e := range s.manifest.Topics {
			entries = append(entries, activationEntry{s.idx, "topics", e.ID, fmt.Sprintf("[%s / topics] %s - %s", pkgLabel, e.ID, e.Description)})
		}
		for _, e := range s.manifest.Prompts {
			entries = append(entries, activationEntry{s.idx, "prompts", e.ID, fmt.Sprintf("[%s / prompts] %s - %s", pkgLabel, e.ID, e.Description)})
		}
		for _, e := range s.manifest.Plans {
			entries = append(entries, activationEntry{s.idx, "plans", e.ID, fmt.Sprintf("[%s / plans] %s - %s", pkgLabel, e.ID, e.Description)})
		}
	}
	return entries
}

// applyActivationSelection takes the user's entry selection and updates
// the config packages with the appropriate activation mode per package.
// If all entries for a package are selected, it uses "all" mode.
// If no entries are selected, it uses "none" mode.
// Otherwise, it uses a granular activation map.
func applyActivationSelection(cfg *config.Config, successes []installedPkg, entries []activationEntry, selected []int) {
	perPkg := make(map[int]*config.ActivationMap)
	totalPerPkg := make(map[int]int)

	for _, s := range successes {
		if !s.pkg.Active.IsNone() {
			continue
		}
		total := len(s.manifest.Foundation) + len(s.manifest.Application) +
			len(s.manifest.Topics) + len(s.manifest.Prompts) + len(s.manifest.Plans)
		totalPerPkg[s.idx] = total
	}

	for _, idx := range selected {
		e := entries[idx]
		if perPkg[e.pkgIdx] == nil {
			perPkg[e.pkgIdx] = &config.ActivationMap{}
		}
		am := perPkg[e.pkgIdx]
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

	for _, s := range successes {
		if !s.pkg.Active.IsNone() {
			continue
		}

		am, hasAny := perPkg[s.idx]
		if !hasAny {
			cfg.Packages[s.idx].Active = config.Activation{Mode: "none"}
			continue
		}

		count := len(am.Foundation) + len(am.Application) + len(am.Topics) + len(am.Prompts) + len(am.Plans)
		if count == totalPerPkg[s.idx] {
			cfg.Packages[s.idx].Active = config.Activation{Mode: "all"}
		} else {
			cfg.Packages[s.idx].Active = config.Activation{Map: am}
		}
	}
}

func promptCombinedActivation(cfg *config.Config, successes []installedPkg) error {
	entries := buildActivationEntries(successes)
	if len(entries) == 0 {
		return nil
	}

	options := make([]huh.Option[int], len(entries))
	for i, e := range entries {
		options[i] = huh.NewOption(e.label, i).Selected(true)
	}

	var selected []int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select entries to activate").
				Description("Entries from all newly installed packages").
				Options(options...).
				Height(min(len(entries)+4, 20)).
				Value(&selected),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return err
	}

	applyActivationSelection(cfg, successes, entries, selected)
	return nil
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
