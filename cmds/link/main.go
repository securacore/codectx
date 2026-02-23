package link

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"securacore/codectx/core/config"
	corelink "securacore/codectx/core/link"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

var Command = &cli.Command{
	Name:  "link",
	Usage: "Create AI tool entry point files",
	Action: func(ctx context.Context, c *cli.Command) error {
		return run()
	},
}

func run() error {
	// Load config.
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Guard: compiled output must exist.
	outputDir := cfg.OutputDir()
	packageYML := filepath.Join(outputDir, "package.yml")
	if _, err := os.Stat(packageYML); os.IsNotExist(err) {
		return fmt.Errorf("compiled output not found at %s: run 'codectx compile' first", outputDir)
	}

	// Prompt: select which tools to link.
	tools := corelink.Tools
	options := make([]huh.Option[int], len(tools))
	for i, t := range tools {
		options[i] = huh.NewOption(t.Name, i).Selected(true)
	}

	var selectedIdxs []int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select AI tools to link").
				Description("Entry point files will be created for each selected tool.").
				Options(options...).
				Value(&selectedIdxs),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if len(selectedIdxs) == 0 {
		fmt.Println("No tools selected.")
		return nil
	}

	// Check for existing files and prompt for confirmation.
	selectedTools := make([]corelink.Tool, len(selectedIdxs))
	for i, idx := range selectedIdxs {
		selectedTools[i] = tools[idx]
	}

	var collisions []string
	for _, t := range selectedTools {
		path := t.File
		if t.SubDir != "" {
			path = filepath.Join(t.SubDir, t.File)
		}
		if _, err := os.Stat(path); err == nil {
			collisions = append(collisions, path)
		}
	}

	if len(collisions) > 0 {
		var confirm bool
		desc := fmt.Sprintf("The following files will be backed up:\n  %s", strings.Join(collisions, "\n  "))
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Existing files detected").
					Description(desc).
					Affirmative("Yes, back up and replace").
					Negative("Cancel").
					Value(&confirm),
			),
		)

		if err := confirmForm.Run(); err != nil {
			return err
		}

		if !confirm {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Perform the link operation.
	results, err := corelink.Link(selectedTools, outputDir)
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}

	fmt.Println("\nLinked:")
	for _, r := range results {
		if r.BackedUp != "" {
			fmt.Printf("  %s (backed up to %s)\n", r.Path, r.BackedUp)
		} else {
			fmt.Printf("  %s\n", r.Path)
		}
	}

	return nil
}
