// Package link implements the `codectx link` command which sets up AI tool
// entry point files (CLAUDE.md, AGENTS.md, .cursorrules,
// .github/copilot-instructions.md) that bootstrap AI tools into the
// codectx documentation system.
//
// The command auto-detects which AI tools are present and pre-selects
// the corresponding entry points. Existing files are backed up before
// being overwritten.
package link

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/x/term"
	corelink "github.com/securacore/codectx/core/link"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx link`.
var Command = &cli.Command{
	Name:  "link",
	Usage: "Set up AI tool entry point files",
	Description: `Creates entry point files (CLAUDE.md, AGENTS.md, .cursorrules,
.github/copilot-instructions.md) that point AI tools to the compiled
engineering context.

Auto-detects which AI tools are present and pre-selects them.
Existing files are backed up before being overwritten.`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Usage:   "Accept all defaults without prompting",
			Aliases: []string{"y"},
		},
		&cli.BoolFlag{
			Name:  "all",
			Usage: "Link all integrations without prompting",
		},
	},
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	interactive := term.IsTerminal(os.Stdin.Fd()) && !cmd.Bool("yes")

	// --- Step 1: Discover and load the project ---
	projectDir, cfg, err := project.DiscoverAndLoad(".")
	if err != nil {
		fmt.Print(tui.ProjectNotFoundError())
		return fmt.Errorf("project not found: %w", err)
	}

	contextRelPath := project.ContextRelPath(cfg.Root)

	// --- Step 3: Detect and select integrations ---
	var selected []corelink.Integration

	switch {
	case cmd.Bool("all"):
		for _, info := range corelink.AllIntegrations() {
			selected = append(selected, info.Type)
		}
	case interactive:
		var err error
		selected, err = corelink.PromptIntegrations(projectDir, "Which AI tool integrations would you like to set up?")
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			fmt.Printf("\n%s No integrations selected.\n\n", tui.StyleMuted.Render("->"))
			return nil
		}
	default:
		// Non-interactive: use detected integrations.
		selected = corelink.Detect(projectDir)
		if len(selected) == 0 {
			// Default to Claude if nothing detected.
			selected = []corelink.Integration{corelink.Claude}
		}
	}

	// --- Step 4: Show backup warning ---
	hasExisting := false
	for _, integration := range selected {
		info := corelink.InfoByType(integration)
		if _, statErr := os.Stat(filepath.Join(projectDir, info.FilePath)); statErr == nil {
			hasExisting = true
			break
		}
	}

	if hasExisting && interactive {
		fmt.Print(tui.WarnMsg{
			Title: "Existing files will be backed up",
			Detail: []string{
				"Files that already exist will be backed up with a .bak extension",
				"before being overwritten. You can restore them manually if needed.",
			},
		}.Render())
	}

	// --- Step 5: Write entry point files ---
	results, err := corelink.Write(projectDir, contextRelPath, selected)
	if err != nil {
		return fmt.Errorf("writing entry points: %w", err)
	}

	// --- Step 6: Display summary ---
	fmt.Print(corelink.RenderLinkResults(results))

	return nil
}
