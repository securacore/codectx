// Package set implements the `codectx set` command for viewing and
// changing user-local preferences stored in .codectx/preferences.yml.
package set

import (
	"context"
	"fmt"
	"strings"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

// preferenceKey describes a single settable preference.
type preferenceKey struct {
	Key         string
	Description string
	Type        string // "bool", "string"
}

// registry is the static list of all known preference keys.
var registry = []preferenceKey{
	{Key: "compression", Description: "Encode compiled objects to CMDX format", Type: "bool"},
	{Key: "auto_compile", Description: "Recompile automatically after changes", Type: "bool"},
	{Key: "ai.provider", Description: "AI provider (claude, opencode, ollama)", Type: "string"},
	{Key: "ai.model", Description: "AI model name (ollama only)", Type: "string"},
	{Key: "ai.class", Description: "Documentation target model class", Type: "string"},
}

var Command = &cli.Command{
	Name:      "set",
	Usage:     "View or change project preferences",
	ArgsUsage: "[key=value]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "interactive",
			Aliases: []string{"i"},
			Usage:   "Select preferences interactively (coming soon)",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.Bool("interactive") {
			ui.Warn("Interactive mode is not yet implemented.")
			ui.Item("Use: codectx set key=value")
			return nil
		}
		if c.NArg() == 0 {
			return showAll()
		}
		return setKeyValue(c.Args().First())
	},
}

// showAll lists every known preference with its current value.
func showAll() error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	prefs, err := preferences.Load(cfg.OutputDir())
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	ui.Header("Preferences:")
	ui.Blank()

	for _, k := range registry {
		value := readValue(prefs, k.Key)
		line := fmt.Sprintf("%-16s %-10s %s", k.Key, value, ui.DimStyle.Render(k.Description))
		fmt.Printf("  %s\n", line)
	}

	ui.Blank()
	ui.Item("Set a value: codectx set key=value")
	ui.Blank()

	return nil
}

// setKeyValue parses "key=value" and applies the change.
func setKeyValue(arg string) error {
	eq := strings.IndexByte(arg, '=')
	if eq < 0 {
		return fmt.Errorf("expected key=value format (e.g., codectx set compression=true)")
	}
	key := arg[:eq]
	value := arg[eq+1:]

	// Validate key exists.
	var entry *preferenceKey
	for i := range registry {
		if registry[i].Key == key {
			entry = &registry[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("unknown preference %q — run codectx set to see available keys", key)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	outputDir := cfg.OutputDir()
	prefs, err := preferences.Load(outputDir)
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	if err := applyValue(prefs, entry, value); err != nil {
		return err
	}

	if err := preferences.Write(outputDir, prefs); err != nil {
		return fmt.Errorf("write preferences: %w", err)
	}

	ui.Done(fmt.Sprintf("%s = %s", key, value))
	return nil
}

// applyValue validates and applies a value to the preferences struct.
func applyValue(prefs *preferences.Preferences, entry *preferenceKey, value string) error {
	switch entry.Key {
	case "compression":
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("compression: %w", err)
		}
		prefs.Compression = preferences.BoolPtr(b)

	case "auto_compile":
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("auto_compile: %w", err)
		}
		prefs.AutoCompile = preferences.BoolPtr(b)

	case "ai.provider":
		if value == "" {
			// Clear the provider (and model).
			prefs.AI = nil
			return nil
		}
		if err := validateAIProvider(value); err != nil {
			return err
		}
		if prefs.AI == nil {
			prefs.AI = &preferences.AIConfig{}
		}
		prefs.AI.Provider = value

	case "ai.model":
		if prefs.AI == nil {
			prefs.AI = &preferences.AIConfig{}
		}
		prefs.AI.Model = value

	case "ai.class":
		if value == "" {
			// Clear the class.
			if prefs.AI != nil {
				prefs.AI.Class = ""
			}
			return nil
		}
		if err := validateAIClass(value); err != nil {
			return err
		}
		if prefs.AI == nil {
			prefs.AI = &preferences.AIConfig{}
		}
		prefs.AI.Class = value
	}

	return nil
}

// validateAIProvider delegates to the shared validator.
func validateAIProvider(id string) error {
	return shared.ValidateAIProvider(id)
}

// validateAIClass delegates to the shared validator.
func validateAIClass(id string) error {
	return shared.ValidateAIClass(id)
}

// readValue returns the current value for a preference key as a display string.
func readValue(prefs *preferences.Preferences, key string) string {
	switch key {
	case "compression":
		return formatBoolPtr(prefs.Compression)
	case "auto_compile":
		return formatBoolPtr(prefs.AutoCompile)
	case "ai.provider":
		if prefs.AI != nil && prefs.AI.Provider != "" {
			return prefs.AI.Provider
		}
		return ui.DimStyle.Render("(unset)")
	case "ai.model":
		if prefs.AI != nil && prefs.AI.Model != "" {
			return prefs.AI.Model
		}
		return ui.DimStyle.Render("(unset)")
	case "ai.class":
		if prefs.AI != nil && prefs.AI.Class != "" {
			return prefs.AI.Class
		}
		return ui.DimStyle.Render("(unset)")
	default:
		return ui.DimStyle.Render("(unset)")
	}
}

// formatBoolPtr formats a *bool for display.
func formatBoolPtr(b *bool) string {
	if b == nil {
		return ui.DimStyle.Render("(unset)")
	}
	if *b {
		return "true"
	}
	return "false"
}

// parseBool accepts "true", "false", "1", "0", "yes", "no".
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected true or false, got %q", s)
	}
}
