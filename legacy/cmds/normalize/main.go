package normalize

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	coreide "github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/core/ide/launcher"
	"github.com/securacore/codectx/core/manifest"
	corenorm "github.com/securacore/codectx/core/normalize"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/ui"
	"github.com/urfave/cli/v3"
)

// Command is the codectx normalize command.
var Command = &cli.Command{
	Name:  "normalize",
	Usage: "AI-driven terminology normalization across documentation",
	Description: `Launches an AI session that reads all documentation in docs/, identifies
inconsistent terminology, and normalizes to a consistent vocabulary.

Requires AI integration to be configured (codectx ai setup or codectx set ai.bin=<binary>).
The AI modifies documentation files in-place. Review changes with git diff and commit.`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Show what would be analyzed without launching AI",
		},
	},
	Action: run,
}

func run(_ context.Context, c *cli.Command) error {
	if !ui.IsTTY() {
		return fmt.Errorf("codectx normalize requires an interactive terminal")
	}

	// Load project config.
	cfg, err := config.Load(shared.ConfigFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	outputDir := cfg.OutputDir()

	// Load preferences and check AI integration.
	prefs, err := preferences.Load(outputDir)
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	if prefs.AI == nil || prefs.AI.Bin == "" {
		ui.Warn("AI integration is not configured.")
		ui.Blank()
		ui.Item("Terminology normalization requires an AI binary (Claude or OpenCode).")
		ui.Item("Run: codectx ai setup")
		ui.Item("  or: codectx set ai.bin=claude")
		ui.Blank()
		return nil
	}

	// Resolve AI binary launcher.
	l, err := launcher.Resolve(prefs)
	if err != nil {
		// Graceful degradation when AI binary is not available.
		ui.Warn("AI binary not available.")
		ui.Blank()
		ui.Item(fmt.Sprintf("Configured binary %q was not found on PATH.", prefs.AI.Bin))
		ui.Item("Run: codectx ai setup")
		ui.Blank()
		return nil //nolint:nilerr // Intentional: missing binary is not a fatal error.
	}

	// Verify docs directory exists.
	docsDir := cfg.DocsDir()
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		return fmt.Errorf("documentation directory not found: %s", docsDir)
	}

	// Load manifest for context.
	m, err := manifest.Load(filepath.Join(docsDir, "manifest.yml"))
	if err != nil {
		m = &manifest.Manifest{}
	}

	summary := coreide.BuildManifestSummary(m)

	// Dry-run mode: show what would be analyzed.
	if c.Bool("dry-run") {
		return dryRun(docsDir, l, summary)
	}

	// Assemble the normalization prompt.
	prompt := corenorm.AssemblePrompt(docsDir, summary)

	ui.Done(fmt.Sprintf("AI binary: %s", l.ID()))
	ui.Step("Launching terminology normalization session...")
	ui.Item("The AI will read all documentation, identify inconsistent terminology,")
	ui.Item("and normalize to a consistent vocabulary. Review changes with git diff.")
	ui.Blank()

	// Launch the AI binary as a child process.
	args := l.NewSessionArgs("", prompt)
	err = shared.RunAIProcess(l.Binary(), args)

	ui.Blank()
	if err != nil {
		ui.Warn(fmt.Sprintf("AI session exited: %s", err))
	} else {
		ui.Done("Normalization session ended")
		ui.Item("Review changes: git diff docs/")
	}

	return nil
}

// countDocsFiles counts markdown files under docsDir, excluding the packages/ subdirectory.
func countDocsFiles(docsDir string) int {
	count := 0
	_ = filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // Skip inaccessible files during count.
		}
		// Skip packages directory (installed packages, not owned by this project).
		rel, _ := filepath.Rel(docsDir, path)
		if strings.HasPrefix(rel, "packages/") || rel == "packages" {
			return filepath.SkipDir
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			count++
		}
		return nil
	})
	return count
}

// dryRun shows what would be analyzed without launching AI.
func dryRun(docsDir string, l launcher.Launcher, summary string) error {
	ui.Header("Dry run: codectx normalize")
	ui.Blank()
	ui.KV("AI binary", l.ID(), 18)
	ui.KV("Docs directory", docsDir, 18)
	ui.Blank()

	count := countDocsFiles(docsDir)
	ui.KV("Markdown files", fmt.Sprintf("%d", count), 18)
	ui.Blank()

	if summary != "" && summary != "No existing documentation." {
		ui.Header("Documentation map:")
		fmt.Println(summary)
	}

	ui.Item("Run without --dry-run to launch the AI normalization session.")
	ui.Blank()

	return nil
}
