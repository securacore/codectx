package ai

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

var setupCommand = &cli.Command{
	Name:  "setup",
	Usage: "Detect and configure AI tool integration",
	Action: func(ctx context.Context, c *cli.Command) error {
		return runSetup()
	},
}

func runSetup() error {
	// Load config to determine output directory.
	cfg, err := config.Load(shared.ConfigFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	outputDir := cfg.OutputDir()

	// Load existing preferences.
	prefs, err := preferences.Load(outputDir)
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	// Run detection and prompt for selection.
	aiCfg, err := shared.PromptAISetup()
	if err != nil {
		return err
	}

	if aiCfg == nil {
		return nil
	}

	// Save to preferences.
	prefs.AI = aiCfg
	if err := preferences.Write(outputDir, prefs); err != nil {
		return fmt.Errorf("write preferences: %w", err)
	}

	ui.Blank()
	ui.Done(fmt.Sprintf("AI integration enabled: %s", aiCfg.Bin))
	ui.Blank()
	return nil
}
