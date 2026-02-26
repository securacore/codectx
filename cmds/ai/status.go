package ai

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/core/ai"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

var statusCommand = &cli.Command{
	Name:  "status",
	Usage: "Show AI integration status and detected tools",
	Action: func(ctx context.Context, c *cli.Command) error {
		return runStatus()
	},
}

func runStatus() error {
	// Load config to determine output directory.
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	prefs, err := preferences.Load(cfg.OutputDir())
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	// Show current configuration.
	ui.Blank()
	ui.Header("AI integration:")

	if prefs.AI == nil {
		ui.Warn("Not configured. Run 'codectx ai setup' to enable.")
	} else {
		provider, ok := ai.ProviderByID(prefs.AI.Provider)
		if !ok {
			ui.Fail(fmt.Sprintf("Unknown provider: %s", prefs.AI.Provider))
		} else {
			result := ai.DetectProvider(provider)
			if result.Found {
				ui.Done(fmt.Sprintf("Provider: %s (%s)", provider.Name, result.Path))
			} else {
				ui.Fail(fmt.Sprintf("Provider: %s (binary not found — was it uninstalled?)", provider.Name))
			}
		}
		if prefs.AI.Model != "" {
			ui.Item(fmt.Sprintf("Model: %s", prefs.AI.Model))
		}

		// Ollama-specific: show service status.
		if prefs.AI.Provider == "ollama" {
			printOllamaStatus()
		}
	}

	// Show detection results for all providers.
	ui.Blank()
	ui.Header("Detected tools:")
	results := ai.Detect()
	for _, r := range results {
		if r.Found {
			ui.Done(fmt.Sprintf("%s (%s)", r.Provider.Name, r.Path))
		} else {
			ui.Fail(fmt.Sprintf("%s (not found)", r.Provider.Name))
		}
	}
	ui.Blank()

	return nil
}

// printOllamaStatus shows Ollama service and model information.
func printOllamaStatus() {
	status := ai.CheckOllama()
	if !status.Running {
		ui.Warn("Ollama service is not running")
		return
	}

	ui.Item(fmt.Sprintf("Ollama service: running (%d model(s) available)", len(status.Models)))
	for _, m := range status.Models {
		ui.Item(fmt.Sprintf("  %s", m))
	}
}
