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
	Action: func(ctx context.Context, c *cli.Command) error {
		return run(c.Args().First(), nil)
	},
}

func run(name string, autoCompile *bool) error {
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
	foundationDir := filepath.Join(docsDir, "foundation")
	if err := defaults.WriteAll(foundationDir); err != nil {
		return fmt.Errorf("write defaults: %w", err)
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
	// Pre-populate Foundation with default entries so Sync's merge-missing
	// preserves their load values (which Discover never auto-sets).
	m := &manifest.Manifest{
		Name:        name,
		Author:      "",
		Version:     "0.1.0",
		Description: fmt.Sprintf("Documentation package for %s", name),
		Foundation:  defaults.Entries(),
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

	// Prompt for auto-compile preference (skip if value provided).
	if autoCompile == nil {
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
		autoCompile = preferences.BoolPtr(confirmStr == "yes")
	}

	// Detect AI tools and prompt for integration (skip if non-interactive).
	var aiCfg *preferences.AIConfig
	if autoCompile == nil {
		// Interactive mode: run AI detection and prompt.
		var aiErr error
		aiCfg, aiErr = shared.PromptAISetup()
		if aiErr != nil {
			return aiErr
		}
	}

	prefs := &preferences.Preferences{
		AutoCompile: autoCompile,
		AI:          aiCfg,
	}
	if err := preferences.Write(outputDir, prefs); err != nil {
		return fmt.Errorf("write preferences: %w", err)
	}

	ui.Done(fmt.Sprintf("Initialized codectx project: %s", name))
	ui.Blank()
	ui.Header("Created:")
	ui.Item(configFile)
	ui.Item(packagePath)
	ui.Item(".gitignore")
	ui.Item(outputDir + "/preferences.yml")
	for _, dir := range dirs {
		ui.Item(dir + "/")
	}

	return nil
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
