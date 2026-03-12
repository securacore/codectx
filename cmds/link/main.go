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
	"github.com/securacore/codectx/cmds/shared"
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
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	contextRelPath := project.ContextRelPath(cfg.Root)

	// --- Step 2: Detect and select integrations ---
	var selected []corelink.Integration

	switch {
	case cmd.Bool("all"):
		selected = selectAll()
	case interactive:
		var err error
		selected, err = corelink.PromptIntegrations(projectDir, "Which AI tool integrations would you like to set up?")
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			fmt.Printf("\n%s No integrations selected.\n\n", tui.StyleMuted.Render(tui.IconArrow))
			return nil
		}
	default:
		selected = selectNonInteractive(projectDir)
	}

	// --- Step 3: Show backup warning ---
	hasExisting := hasExistingFiles(projectDir, selected)

	if hasExisting && interactive {
		fmt.Print(tui.WarnMsg{
			Title: "Existing files will be backed up",
			Detail: []string{
				"Files that already exist will be backed up with a .bak extension",
				"before being overwritten. You can restore them manually if needed.",
			},
		}.Render())
	}

	// --- Step 4: Write entry point files ---
	results, err := corelink.Write(projectDir, contextRelPath, selected)
	if err != nil {
		return fmt.Errorf("writing entry points: %w", err)
	}

	// --- Step 5: Display summary ---
	fmt.Print(corelink.RenderLinkResults(results))

	return nil
}

// selectAll returns all supported integrations (--all flag).
func selectAll() []corelink.Integration {
	all := corelink.AllIntegrations()
	selected := make([]corelink.Integration, len(all))
	for i, info := range all {
		selected[i] = info.Type
	}
	return selected
}

// selectNonInteractive returns integrations based on auto-detection,
// defaulting to Claude if nothing is detected.
func selectNonInteractive(projectDir string) []corelink.Integration {
	detected := corelink.Detect(projectDir)
	if len(detected) == 0 {
		return []corelink.Integration{corelink.Claude}
	}
	return detected
}

// hasExistingFiles checks whether any of the selected integrations already
// have files on disk in the project directory.
func hasExistingFiles(projectDir string, integrations []corelink.Integration) bool {
	for _, integration := range integrations {
		info := corelink.InfoByType(integration)
		if _, err := os.Stat(filepath.Join(projectDir, info.FilePath)); err == nil {
			return true
		}
	}
	return false
}
