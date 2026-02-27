package init

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/defaults"
	corelink "github.com/securacore/codectx/core/link"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/core/schema"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"
const manifestFile = "manifest.yml"
const gitignoreContent = ".codectx/\n"

var Command = &cli.Command{
	Name:      "init",
	Usage:     "Initialize a new codectx project",
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

func run(name string, autoCompile *bool, isPackage bool) error {
	// If a name is provided as argument, create the directory and work inside it.
	if name != "" {
		if err := os.MkdirAll(name, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", name, err)
		}
		if err := os.Chdir(name); err != nil {
			return fmt.Errorf("enter directory %s: %w", name, err)
		}
	}

	// Guard: check if already initialized.
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("%s already exists: project is already initialized", configFile)
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
			return fmt.Errorf("prompt: %w", err)
		}

		if prompted != "" {
			name = prompted
		} else {
			name = defaultName
		}

		if name == "" {
			return fmt.Errorf("project name is required")
		}
	}

	// Initialize git if no .git directory exists.
	if err := ensureGit(); err != nil {
		return err
	}

	// Create docs directory structure.
	docsDir := "docs"
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
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Write embedded schemas to docs/schemas/.
	schemasDir := filepath.Join(docsDir, "schemas")
	if err := schema.WriteAll(schemasDir); err != nil {
		return fmt.Errorf("write schemas: %w", err)
	}

	// Write embedded default foundation documents to docs/foundation/.
	// Packages skip this step — foundation documents are project-level
	// conventions and should not be duplicated into every published package.
	if !isPackage {
		foundationDir := filepath.Join(docsDir, "foundation")
		if err := defaults.WriteAll(foundationDir); err != nil {
			return fmt.Errorf("write defaults: %w", err)
		}
	}

	// Create codectx.yml.
	cfg := &config.Config{
		Name:     name,
		Packages: []config.PackageDep{},
	}
	if err := config.Write(configFile, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
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
		return fmt.Errorf("write package manifest: %w", err)
	}

	// Create .codectx/ directory and write default preferences.
	outputDir := cfg.OutputDir()
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", outputDir, err)
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
			return aiErr
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
		return fmt.Errorf("write preferences: %w", err)
	}

	kind := "project"
	if isPackage {
		kind = "package"
	}
	ui.Done(fmt.Sprintf("Initialized codectx %s: %s", kind, name))
	ui.Blank()
	ui.Header("Created:")
	ui.Item(configFile)
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
		if prefs.AI.Provider != "" {
			ui.KV("ai.provider", prefs.AI.Provider, 16)
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

	// Auto-compile: since default foundation documents always exist after init,
	// there is always content to compile. This gives the user a complete setup.
	if ui.IsTTY() {
		if err := autoCompileAfterInit(cfg); err != nil {
			ui.Warn(fmt.Sprintf("compile: %s", err))
		}

		// Offer to link AI tools.
		if err := promptLink(cfg.OutputDir()); err != nil {
			ui.Warn(fmt.Sprintf("link: %s", err))
		}
	}

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
	const entry = ".codectx/"
	path := ".gitignore"

	// Check if .gitignore already exists and contains the entry.
	if data, err := os.ReadFile(path); err == nil {
		content := string(data)
		for _, line := range splitLines(content) {
			if line == entry {
				return nil // already present
			}
		}
		// Append the entry.
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

	// Create new .gitignore.
	return os.WriteFile(path, []byte(gitignoreContent), 0o644)
}

// autoCompileAfterInit compiles the initial documentation using the inline
// spinner pattern. Since default foundation documents always exist after init,
// there is always content to compile.
func autoCompileAfterInit(cfg *config.Config) error {
	ui.Blank()
	var result *compile.Result
	err := ui.SpinErr("Compiling...", func() error {
		var compileErr error
		result, compileErr = compile.Compile(cfg)
		return compileErr
	})
	if err != nil {
		return err
	}

	ui.Done(fmt.Sprintf("Compiled to %s", result.OutputDir))
	ui.KV("Objects stored", result.ObjectsStored, 16)
	if result.ObjectsPruned > 0 {
		ui.KV("Objects pruned", result.ObjectsPruned, 16)
	}
	ui.KV("Packages", result.Packages, 16)

	return nil
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
				Negative("No, I'll run 'codectx link' later").
				Value(&confirm),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return nil // User canceled, not an error.
	}

	if !confirm {
		ui.Step("Run 'codectx link' when you're ready.")
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
