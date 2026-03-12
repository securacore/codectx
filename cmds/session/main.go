// Package session implements the `codectx session` command group which manages
// always-loaded session context entries in codectx.yml.
//
// Subcommands:
//   - codectx session add <reference>    — Add a reference to always_loaded
//   - codectx session remove <reference> — Remove a reference from always_loaded
//   - codectx session list               — List entries with token counts
//
// These commands are convenience wrappers that modify codectx.yml directly.
// They provide token cost feedback that hand-editing wouldn't give.
package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/cmds/shared"
	codectx "github.com/securacore/codectx/core/context"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx session`.
var Command = &cli.Command{
	Name:  "session",
	Usage: "Manage always-loaded session context",
	Description: `Manages the session.always_loaded list in codectx.yml.
These references are compiled into context.md and loaded at the
start of every AI session.`,
	Commands: []*cli.Command{
		addCommand,
		removeCommand,
		listCommand,
	},
}

// addCommand adds a reference to always_loaded.
var addCommand = &cli.Command{
	Name:      "add",
	Usage:     "Add a reference to always-loaded session context",
	ArgsUsage: "<reference>",
	Description: `Add a local path or package reference to the always-loaded session context.
Reports the token cost of the added entry and the new total against the budget.

Examples:
  codectx session add foundation/coding-standards
  codectx session add react-patterns@community/foundation/component-principles
  codectx session add company-standards@acme`,
	Action: runAdd,
}

// removeCommand removes a reference from always_loaded.
var removeCommand = &cli.Command{
	Name:      "remove",
	Usage:     "Remove a reference from always-loaded session context",
	ArgsUsage: "<reference>",
	Description: `Remove an entry from the always-loaded session context.

Examples:
  codectx session remove foundation/coding-standards
  codectx session remove company-standards@acme`,
	Action: runRemove,
}

// listCommand lists all always_loaded entries with token counts.
var listCommand = &cli.Command{
	Name:  "list",
	Usage: "List always-loaded session context entries",
	Description: `List all always-loaded session context entries with individual
token counts and total against the budget.`,
	Action: runList,
}

func runAdd(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing reference argument",
			Detail: []string{
				"Usage: codectx session add <reference>",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Add a local path:", Command: "codectx session add foundation/coding-standards"},
				{Text: "Add a package:", Command: "codectx session add company-standards@acme"},
			},
		}.Render())
		return fmt.Errorf("missing reference argument")
	}

	ref := cmd.Args().First()

	// Discover and load the project.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// Initialize session config if nil.
	if cfg.Session == nil {
		cfg.Session = &project.SessionConfig{
			AlwaysLoaded: []string{},
			Budget:       project.DefaultSessionBudget,
		}
	}

	// Check for duplicates.
	if isDuplicate(cfg.Session.AlwaysLoaded, ref) {
		fmt.Print(tui.WarnMsg{
			Title: "Already in session context",
			Detail: []string{
				fmt.Sprintf("%s is already in always_loaded.", tui.StyleAccent.Render(ref)),
			},
			Suggestions: []tui.Suggestion{
				{Text: "List current session entries:", Command: "codectx session list"},
			},
		}.Render())
		return nil
	}

	// Resolve the reference to validate it and count tokens.
	rootDir := project.RootDir(projectDir, cfg)
	packagesDir := project.PackagesPath(rootDir)

	resolved, err := codectx.Resolve(rootDir, packagesDir, []string{ref})
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Failed to resolve reference",
			Detail: []string{
				fmt.Sprintf("Could not resolve %s", tui.StyleAccent.Render(ref)),
				err.Error(),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Check that the path exists under your documentation root"},
				{Text: "For packages, ensure the package is installed"},
			},
		}.Render())
		return fmt.Errorf("resolving reference: %w", err)
	}

	// Load AI config for encoding.
	aiCfg, err := project.LoadAIConfigForProject(projectDir, cfg)
	if err != nil {
		return fmt.Errorf("loading AI config: %w", err)
	}

	// Assemble to count tokens for the new entry.
	budget := cfg.Session.EffectiveBudget()

	assembly, err := codectx.Assemble(resolved, aiCfg.Compilation.Encoding, budget)
	if err != nil {
		return fmt.Errorf("counting tokens: %w", err)
	}

	newEntryTokens := 0
	if len(assembly.Entries) > 0 {
		newEntryTokens = assembly.Entries[0].Tokens
	}

	// Add the reference to config and save.
	cfg.Session.AlwaysLoaded = append(cfg.Session.AlwaysLoaded, ref)
	configPath := filepath.Join(projectDir, project.ConfigFileName)
	if err := cfg.WriteToFile(configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Compute new total by resolving all entries.
	totalTokens, err := computeSessionTotal(cfg, rootDir, packagesDir, aiCfg.Compilation.Encoding, budget)
	if err != nil {
		// Non-fatal: we still added the entry successfully, just can't show totals.
		fmt.Printf("\n%s Added %s (%s tokens)\n\n",
			tui.Success(),
			tui.StyleAccent.Render(ref),
			tui.FormatNumber(newEntryTokens),
		)
		return nil //nolint:nilerr // intentionally non-fatal, entry was saved
	}

	// Display result.
	fmt.Printf("\n%s Added %s (%s tokens)\n\n",
		tui.Success(),
		tui.StyleAccent.Render(ref),
		tui.FormatNumber(newEntryTokens),
	)

	fmt.Printf("%s%s\n\n",
		tui.Indent(1),
		tui.KeyValue("Session", tui.FormatBudget(totalTokens, budget)),
	)

	if totalTokens > budget {
		fmt.Print(tui.WarnMsg{
			Title: "Session context exceeds budget",
			Detail: []string{
				fmt.Sprintf("Consider removing entries or increasing the budget in %s.",
					tui.StylePath.Render(project.ConfigFileName)),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Remove an entry:", Command: "codectx session remove <ref>"},
			},
		}.Render())
	}

	return nil
}

func runRemove(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing reference argument",
			Detail: []string{
				"Usage: codectx session remove <reference>",
			},
			Suggestions: []tui.Suggestion{
				{Text: "List current entries:", Command: "codectx session list"},
			},
		}.Render())
		return fmt.Errorf("missing reference argument")
	}

	ref := cmd.Args().First()

	// Discover and load the project.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	if cfg.Session == nil || len(cfg.Session.AlwaysLoaded) == 0 {
		fmt.Print(tui.WarnMsg{
			Title: "No session context configured",
			Detail: []string{
				"There are no entries in session.always_loaded to remove.",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Add an entry first:", Command: "codectx session add <ref>"},
			},
		}.Render())
		return nil
	}

	// Find and remove the reference.
	filtered, found := removeRef(cfg.Session.AlwaysLoaded, ref)
	if !found {
		fmt.Print(tui.ErrorMsg{
			Title: "Reference not found",
			Detail: []string{
				fmt.Sprintf("%s is not in always_loaded.", tui.StyleAccent.Render(ref)),
			},
			Suggestions: []tui.Suggestion{
				{Text: "List current entries:", Command: "codectx session list"},
			},
		}.Render())
		return fmt.Errorf("reference not found: %s", ref)
	}

	cfg.Session.AlwaysLoaded = filtered
	configPath := filepath.Join(projectDir, project.ConfigFileName)
	if err := cfg.WriteToFile(configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\n%s Removed %s\n\n",
		tui.Success(),
		tui.StyleAccent.Render(ref),
	)

	return nil
}

func runList(_ context.Context, _ *cli.Command) error {
	interactive := term.IsTerminal(os.Stdin.Fd())

	// Discover and load the project.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	if cfg.Session == nil || len(cfg.Session.AlwaysLoaded) == 0 {
		if interactive {
			fmt.Printf("\n%s No always-loaded session context configured.\n\n",
				tui.StyleMuted.Render(tui.IconArrow))
			fmt.Printf("%s%s\n\n",
				tui.Indent(1),
				tui.StyleMuted.Render(fmt.Sprintf("Add entries with: %s",
					tui.StyleCommand.Render("codectx session add <reference>"))),
			)
		}
		return nil
	}

	// Resolve and count tokens for all entries.
	rootDir := project.RootDir(projectDir, cfg)
	packagesDir := project.PackagesPath(rootDir)

	// Load AI config for encoding.
	aiCfg, err := project.LoadAIConfigForProject(projectDir, cfg)
	if err != nil {
		return fmt.Errorf("loading AI config: %w", err)
	}

	budget := cfg.Session.EffectiveBudget()

	resolved, err := codectx.Resolve(rootDir, packagesDir, cfg.Session.AlwaysLoaded)
	if err != nil {
		return fmt.Errorf("resolving session context: %w", err)
	}

	assembly, err := codectx.Assemble(resolved, aiCfg.Compilation.Encoding, budget)
	if err != nil {
		return fmt.Errorf("assembling session context: %w", err)
	}

	// Render the list output matching the spec format.
	fmt.Print(renderSessionList(assembly, budget))

	return nil
}

// renderSessionList formats the session list output.
// Matches the spec format:
//
//	Always-loaded session context (28,450 / 30,000 tokens):
//
//	  foundation/coding-standards                              8,200 tokens
//	  ...
func renderSessionList(assembly *codectx.AssemblyResult, budget int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s %s\n\n",
		tui.Arrow(),
		tui.StyleBold.Render(fmt.Sprintf("Always-loaded session context (%s)",
			tui.FormatBudget(assembly.TotalTokens, budget),
		)),
	)

	// Find the longest reference for alignment.
	maxRefLen := 0
	for _, entry := range assembly.Entries {
		if len(entry.Reference) > maxRefLen {
			maxRefLen = len(entry.Reference)
		}
	}

	for _, entry := range assembly.Entries {
		padding := strings.Repeat(" ", maxRefLen-len(entry.Reference)+2)
		fmt.Fprintf(&b, "%s%s%s%s tokens\n",
			tui.Indent(1),
			tui.StyleAccent.Render(entry.Reference),
			padding,
			tui.FormatNumber(entry.Tokens),
		)
	}

	b.WriteString("\n")

	if budget > 0 && assembly.TotalTokens > budget {
		utilization := float64(assembly.TotalTokens) / float64(budget) * 100.0
		fmt.Fprintf(&b, "%s%s\n\n",
			tui.Indent(1),
			tui.StyleWarning.Render(fmt.Sprintf("Budget exceeded: %.1f%%", utilization)),
		)
	}

	return b.String()
}

// isDuplicate checks whether ref already exists in the list.
func isDuplicate(list []string, ref string) bool {
	for _, existing := range list {
		if existing == ref {
			return true
		}
	}
	return false
}

// removeRef removes the first occurrence of ref from the list.
// Returns the filtered list and whether the ref was found.
func removeRef(list []string, ref string) ([]string, bool) {
	found := false
	filtered := make([]string, 0, len(list))
	for _, existing := range list {
		if existing == ref && !found {
			found = true
			continue
		}
		filtered = append(filtered, existing)
	}
	return filtered, found
}

// computeSessionTotal resolves and assembles all session entries to get the total token count.
func computeSessionTotal(cfg *project.Config, rootDir, packagesDir, encoding string, budget int) (int, error) {
	if cfg.Session == nil || len(cfg.Session.AlwaysLoaded) == 0 {
		return 0, nil
	}

	resolved, err := codectx.Resolve(rootDir, packagesDir, cfg.Session.AlwaysLoaded)
	if err != nil {
		return 0, err
	}

	assembly, err := codectx.Assemble(resolved, encoding, budget)
	if err != nil {
		return 0, err
	}

	return assembly.TotalTokens, nil
}
