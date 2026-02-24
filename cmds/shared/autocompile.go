// Package shared provides utilities shared across CLI commands.
package shared

import (
	"fmt"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/huh"
)

// MaybeAutoCompile loads preferences and runs compile if auto-compile is
// enabled. If the preference is unset (nil), it prompts the user once
// and saves the answer.
func MaybeAutoCompile(cfg *config.Config) error {
	outputDir := cfg.OutputDir()
	prefs, err := preferences.Load(outputDir)
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	if prefs.AutoCompile == nil {
		// Preference unset: prompt once and save.
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

		val := confirmStr == "yes"
		prefs.AutoCompile = &val
		if err := preferences.Write(outputDir, prefs); err != nil {
			return fmt.Errorf("write preferences: %w", err)
		}
	}

	if !*prefs.AutoCompile {
		return nil
	}

	ui.Blank()
	var result *compile.Result
	err = ui.SpinErr("Compiling...", func() error {
		var compileErr error
		result, compileErr = compile.Compile(cfg)
		return compileErr
	})
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	ui.Done(fmt.Sprintf("Compiled to %s", result.OutputDir))
	ui.KV("Objects stored", result.ObjectsStored, 16)
	if result.ObjectsPruned > 0 {
		ui.KV("Objects pruned", result.ObjectsPruned, 16)
	}
	ui.KV("Packages", result.Packages, 16)

	return nil
}
