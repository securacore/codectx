package init

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/defaults"
	"github.com/securacore/codectx/core/gitkeep"
	corelink "github.com/securacore/codectx/core/link"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/core/schema"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"
)

// manifestFile is a local alias for shared.ManifestFile.
const manifestFile = shared.ManifestFile

var Command = &cli.Command{
	Name:      "init",
	Usage:     "Initialize a new codectx project",
	Category:  "Core Workflow",
	ArgsUsage: "[name]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "package",
			Usage: "Initialize as a documentation package (skip default foundation documents)",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		return run(c.Args().First(), nil, c.Bool("package"))
	},
}

// CoreResult holds the outputs of RunCore that downstream callers may need.
type CoreResult struct {
	// Config is the codectx configuration that was written.
	Config *config.Config

	// Preferences is the preferences that were written.
	Preferences *preferences.Preferences

	// DocsDir is the path to the docs directory (relative to cwd).
	DocsDir string
}

// RunCore performs the core initialization: directory scaffold, git init,
// schemas, foundation defaults, config, manifest, and preferences. It does
// NOT perform auto-compile or AI tool linking — call RunPostInit for those.
//
// After RunCore returns, the working directory is inside the new project
// (if a name was provided).
func RunCore(name string, autoCompile *bool, isPackage bool) (*CoreResult, error) {
	// If a name is provided as argument, create the directory and work inside it.
	if name != "" {
		if err := os.MkdirAll(name, 0o755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", name, err)
		}
		if err := os.Chdir(name); err != nil {
			return nil, fmt.Errorf("enter directory %s: %w", name, err)
		}
	}

	// Guard: check if already initialized.
	if _, err := os.Stat(shared.ConfigFile); err == nil {
		return nil, fmt.Errorf("%s already exists: project is already initialized", shared.ConfigFile)
	}

	// If no name provided, prompt interactively.
	if name == "" {
		defaultName := ""
		wd, err := os.Getwd()
		if err == nil {
			defaultName = filepath.Base(wd)
		}

		var prompted string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Project name").
					Description("Name for this codectx project").
					Placeholder(defaultName).
					Value(&prompted),
			),
		).WithTheme(ui.Theme())

		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("prompt: %w", err)
		}

		if prompted != "" {
			name = prompted
		} else {
			name = defaultName
		}

		if name == "" {
			return nil, fmt.Errorf("project name is required")
		}
	}

	// Initialize git if no .git directory exists.
	if err := ensureGit(); err != nil {
		return nil, err
	}

	// Scaffold docs directory structure.
	docsDir := "docs"
	dirs, err := scaffoldDocs(docsDir, isPackage)
	if err != nil {
		return nil, err
	}

	// Create codectx.yml.
	cfg := &config.Config{
		Name:     name,
		Packages: []config.PackageDep{},
	}
	if err := config.Write(shared.ConfigFile, cfg); err != nil {
		return nil, fmt.Errorf("write config: %w", err)
	}

	// Create docs/manifest.yml (local package data map).
	m := &manifest.Manifest{
		Name:        name,
		Author:      "",
		Version:     "0.1.0",
		Description: fmt.Sprintf("Documentation package for %s", name),
	}

	// Pre-populate Foundation with default entries so Sync's merge-missing
	// preserves their load values (which Discover never auto-sets).
	// Packages skip this — they don't ship default foundation documents.
	if !isPackage {
		m.Foundation = defaults.Entries()
	}

	// Sync: discover entries, remove stale, infer relationships from links.
	m = manifest.Sync(docsDir, m)

	packagePath := filepath.Join(docsDir, manifestFile)
	if err := manifest.Write(packagePath, m); err != nil {
		return nil, fmt.Errorf("write package manifest: %w", err)
	}

	// Create .codectx/ directory and write default preferences.
	outputDir := cfg.OutputDir()
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", outputDir, err)
	}

	// Default boolean preferences for new projects.
	if autoCompile == nil {
		autoCompile = preferences.BoolPtr(true)
	}
	compression := preferences.BoolPtr(true)

	// Detect AI tools and prompt for integration (interactive only).
	var aiCfg *preferences.AIConfig
	if ui.IsTTY() {
		var aiErr error
		aiCfg, aiErr = shared.PromptAISetup()
		if aiErr != nil {
			return nil, aiErr
		}
	}

	// Ensure AI config exists so we can set the default model class.
	if aiCfg == nil {
		aiCfg = &preferences.AIConfig{}
	}
	aiCfg.Class = "gpt-4o-class"

	prefs := &preferences.Preferences{
		Compression: compression,
		AutoCompile: autoCompile,
		AI:          aiCfg,
	}
	if err := preferences.Write(outputDir, prefs); err != nil {
		return nil, fmt.Errorf("write preferences: %w", err)
	}

	printInitSummary(name, isPackage, packagePath, outputDir, dirs, prefs)

	return &CoreResult{
		Config:      cfg,
		Preferences: prefs,
		DocsDir:     docsDir,
	}, nil
}

// RunPostInit performs the post-initialization steps: auto-compile and AI tool
// linking. These are interactive (TTY-only) operations that should run after
// the project is fully scaffolded.
func RunPostInit(cfg *config.Config) {
	if !ui.IsTTY() {
		return
	}

	if err := autoCompile(cfg); err != nil {
		ui.Warn(fmt.Sprintf("compile: %s", err))
	}

	if err := promptLink(cfg.OutputDir()); err != nil {
		ui.Warn(fmt.Sprintf("link: %s", err))
	}
}

// run is the private entry point for the init command. It calls RunCore
// followed by RunPostInit.
func run(name string, autoCompile *bool, isPackage bool) error {
	result, err := RunCore(name, autoCompile, isPackage)
	if err != nil {
		return err
	}

	RunPostInit(result.Config)
	return nil
}

// formatBool formats a *bool preference for display.
func formatBool(b *bool) string {
	if b == nil {
		return "(unset)"
	}
	if *b {
		return "true"
	}
	return "false"
}

// ensureGit initializes a git repository and writes a .gitignore
// if no .git directory exists in the current directory.
func ensureGit() error {
	if _, err := os.Stat(".git"); err == nil {
		// Git already initialized; ensure .gitignore has .codectx/ entry.
		return ensureGitignore()
	}

	cmd := exec.Command("git", "init")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	return ensureGitignore()
}

// ensureGitignore creates or appends to .gitignore to include .codectx/.
func ensureGitignore() error {
	return shared.EnsureGitignoreEntry(".gitignore", ".codectx/")
}

// autoCompile compiles the documentation using the shared compile-and-print pattern.
func autoCompile(cfg *config.Config) error {
	ui.Blank()
	return shared.RunCompileAndPrint(cfg)
}

// promptLink offers to set up AI tool integration by creating entry point files.
// If the user declines, a hint is printed.
func promptLink(outputDir string) error {
	ui.Blank()
	var confirm bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Set up AI tool integration?").
				Description("Creates entry point files so your AI tools automatically\nload project documentation on every session.").
				Affirmative("Yes (recommended)").
				Negative("No, I'll run 'codectx ai link' later").
				Value(&confirm),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return nil //nolint:nilerr // User canceled the prompt.
	}

	if !confirm {
		ui.Step("Run 'codectx ai link' when you're ready.")
		return nil
	}

	// Link all available tools with all selected by default.
	results, err := corelink.Link(corelink.Tools, outputDir)
	if err != nil {
		return err
	}

	ui.Done("Linked")
	for _, r := range results {
		if r.BackedUp != "" {
			ui.ItemDetail(r.Path, "backed up to "+r.BackedUp)
		} else {
			ui.Item(r.Path)
		}
	}

	return nil
}

// scaffoldDocs creates the docs directory structure, writes .gitkeep files,
// embedded schemas, and (for non-package projects) default foundation docs.
// Returns the list of created directories.
func scaffoldDocs(docsDir string, isPackage bool) ([]string, error) {
	dirs := []string{
		docsDir,
		filepath.Join(docsDir, "foundation"),
		filepath.Join(docsDir, "topics"),
		filepath.Join(docsDir, "prompts"),
		filepath.Join(docsDir, "plans"),
		filepath.Join(docsDir, "schemas"),
		filepath.Join(docsDir, "packages"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Place .gitkeep files in documentation directories that start empty.
	gitkeepDirs := []string{
		filepath.Join(docsDir, "topics"),
		filepath.Join(docsDir, "prompts"),
		filepath.Join(docsDir, "plans"),
		filepath.Join(docsDir, "packages"),
	}
	for _, dir := range gitkeepDirs {
		if err := gitkeep.Write(dir); err != nil {
			return nil, fmt.Errorf("write .gitkeep in %s: %w", dir, err)
		}
	}

	// Write embedded schemas.
	schemasDir := filepath.Join(docsDir, "schemas")
	if err := schema.WriteAll(schemasDir); err != nil {
		return nil, fmt.Errorf("write schemas: %w", err)
	}

	// Write default foundation documents (project-only, not packages).
	if !isPackage {
		foundationDir := filepath.Join(docsDir, "foundation")
		if err := defaults.WriteAll(foundationDir); err != nil {
			return nil, fmt.Errorf("write defaults: %w", err)
		}
	}

	return dirs, nil
}

// printInitSummary displays the initialization results to the user.
func printInitSummary(name string, isPackage bool, packagePath, outputDir string, dirs []string, prefs *preferences.Preferences) {
	kind := "project"
	if isPackage {
		kind = "package"
	}
	ui.Done(fmt.Sprintf("Initialized codectx %s: %s", kind, name))
	ui.Blank()
	ui.Header("Created:")
	ui.Item(shared.ConfigFile)
	ui.Item(packagePath)
	ui.Item(".gitignore")
	ui.Item(outputDir + "/preferences.yml")
	for _, dir := range dirs {
		ui.Item(dir + "/")
	}
	ui.Blank()
	ui.Header("Preferences:")
	ui.KV("compression", formatBool(prefs.Compression), 16)
	ui.KV("auto_compile", formatBool(prefs.AutoCompile), 16)
	if prefs.AI != nil {
		if prefs.AI.Bin != "" {
			ui.KV("ai.bin", prefs.AI.Bin, 16)
			if prefs.AI.Model != "" {
				ui.KV("ai.model", prefs.AI.Model, 16)
			}
		}
		if prefs.AI.Class != "" {
			ui.KV("ai.class", prefs.AI.Class, 16)
		}
	}
	ui.Blank()
	ui.Item("Adjust preferences: codectx set")
}

// splitLines splits a string into lines, handling both \n and \r\n.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
