package shared

import (
	"fmt"

	"github.com/securacore/codectx/core/ai"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/huh"
)

// PromptAISetup runs AI tool detection and prompts the user to select
// a provider. Returns nil if no tools are found or the user declines.
// This function is shared between `codectx ai setup` and `codectx init`.
func PromptAISetup() (*preferences.AIConfig, error) {
	// Detect available providers.
	results := ai.Detect()
	found := ai.Found(results)

	// Display detection results.
	ui.Blank()
	ui.Header("AI tool detection:")
	for _, r := range results {
		if r.Found {
			ui.Done(fmt.Sprintf("%s (%s)", r.Provider.Name, r.Path))
		} else {
			ui.Fail(fmt.Sprintf("%s (not found)", r.Provider.Name))
		}
	}
	ui.Blank()

	if len(found) == 0 {
		ui.Warn("No supported AI tools detected. Install claude, opencode, or ollama to enable AI features.")
		return nil, nil
	}

	// Ask if user wants to enable AI integration.
	var enable bool
	enableForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable AI integration?").
				Description("Connect codectx with a detected AI tool for assistance and doc generation.").
				Affirmative("Yes").
				Negative("No").
				Value(&enable),
		),
	).WithTheme(ui.Theme())

	if err := enableForm.Run(); err != nil {
		return nil, fmt.Errorf("prompt: %w", err)
	}

	if !enable {
		return nil, nil
	}

	// Build selection options from found providers.
	options := make([]huh.Option[string], len(found))
	for i, r := range found {
		options[i] = huh.NewOption(r.Provider.Name, r.Provider.ID)
	}

	var selectedID string
	selectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select AI provider").
				Description("Choose which AI tool codectx should use.").
				Options(options...).
				Value(&selectedID),
		),
	).WithTheme(ui.Theme())

	if err := selectForm.Run(); err != nil {
		return nil, fmt.Errorf("prompt: %w", err)
	}

	cfg := &preferences.AIConfig{
		Bin: selectedID,
	}

	// Ollama-specific: prompt for model selection if service is running.
	if selectedID == "ollama" {
		promptOllamaModel(cfg, results)
	}

	return cfg, nil
}

// promptOllamaModel checks the Ollama service and prompts the user
// to select a model if models are available.
func promptOllamaModel(cfg *preferences.AIConfig, results []ai.DetectionResult) {
	// Find the ollama detection result.
	var ollamaResult ai.DetectionResult
	for _, r := range results {
		if r.Provider.ID == "ollama" {
			ollamaResult = r
			break
		}
	}

	status, err := ai.OllamaReady(ollamaResult)
	if err != nil {
		ui.Warn(err.Error())
		ui.Item("You can configure the model later with: codectx ai setup")
		return
	}

	if len(status.Models) == 1 {
		cfg.Model = status.Models[0]
		ui.Done(fmt.Sprintf("Using Ollama model: %s", cfg.Model))
		return
	}

	// Multiple models available — let user choose.
	options := make([]huh.Option[string], len(status.Models))
	for i, m := range status.Models {
		options[i] = huh.NewOption(m, m)
	}

	var selectedModel string
	modelForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Ollama model").
				Description("Choose which model to use for AI features.").
				Options(options...).
				Value(&selectedModel),
		),
	).WithTheme(ui.Theme())

	if err := modelForm.Run(); err != nil {
		ui.Warn(fmt.Sprintf("model selection: %s", err.Error()))
		return
	}

	cfg.Model = selectedModel
}
