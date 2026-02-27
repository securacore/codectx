package ide

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/config"
	coreide "github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/core/llm"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"
	"github.com/securacore/codectx/ui"
	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

// Command is the codectx ide command.
var Command = &cli.Command{
	Name:  "ide",
	Usage: "AI documentation authoring",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "resume",
			Usage: "Resume a session by ID",
		},
		&cli.BoolFlag{
			Name:  "list",
			Usage: "List all sessions",
		},
	},
	Action: run,
}

func run(_ context.Context, c *cli.Command) error {
	if !ui.IsTTY() {
		return fmt.Errorf("codectx ide requires an interactive terminal")
	}

	// Load project config.
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	outputDir := cfg.OutputDir()
	rootDir, _ := os.Getwd()

	// Handle --list flag.
	if c.Bool("list") {
		return listSessions(outputDir)
	}

	// Resolve AI provider.
	provider, err := llm.Resolve()
	if err != nil {
		return err
	}

	// Clean up old sessions.
	_, _ = coreide.Cleanup(outputDir, 30*24*time.Hour)

	// Determine session: resume or new.
	var session *coreide.Session

	if resumeID := c.String("resume"); resumeID != "" {
		session, err = coreide.Load(outputDir, resumeID)
		if err != nil {
			return fmt.Errorf("resume session: %w", err)
		}
	} else {
		session, err = pickOrCreateSession(outputDir, provider.ID())
		if err != nil {
			return err
		}
	}

	if session == nil {
		ui.Canceled()
		return nil
	}

	// Assemble system prompt.
	prompt, err := assemblePrompt(cfg)
	if err != nil {
		return fmt.Errorf("assemble prompt: %w", err)
	}

	// Launch the TUI.
	docsDir := cfg.DocsDir()
	m := newModel(session, provider, prompt, outputDir, docsDir, rootDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("ide error: %w", err)
	}

	fm := final.(ideModel)

	if fm.saved {
		ui.Done("Document written")
		for _, block := range fm.docBlocks {
			ui.Item(block.Path)
		}

		// Auto-compile if enabled (runs outside TUI, after exit).
		if err := shared.MaybeAutoCompile(cfg); err != nil {
			ui.Warn(fmt.Sprintf("auto-compile: %s", err))
		}
	} else if !fm.quitting {
		ui.Step("Session saved. Resume with: codectx ide --resume " + session.ID)
	}
	ui.Blank()

	return nil
}

func listSessions(outputDir string) error {
	ui.Blank()
	sessions, err := coreide.List(outputDir)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		ui.Step("No sessions found")
		return nil
	}

	for _, s := range sessions {
		status := s.Phase.String()
		age := time.Since(s.Updated).Truncate(time.Minute)
		detail := fmt.Sprintf("[%s] phase:%s  %s ago", s.Category, status, age)
		if s.Category == "" {
			detail = fmt.Sprintf("phase:%s  %s ago", status, age)
		}
		ui.KV(s.Title, detail, 20)
	}
	ui.Blank()

	return nil
}

func pickOrCreateSession(outputDir, providerID string) (*coreide.Session, error) {
	active, err := coreide.Active(outputDir)
	if err != nil {
		return nil, err
	}

	if len(active) == 0 {
		// No active sessions, create new directly.
		s := coreide.NewSession(providerID)
		if err := coreide.Save(outputDir, s); err != nil {
			return nil, err
		}
		return s, nil
	}

	// Build options: active sessions + "new" option.
	options := make([]huh.Option[string], 0, len(active)+1)
	for _, s := range active {
		var label string
		if s.Category != "" {
			label = fmt.Sprintf("%s  [%s]  phase:%s", s.Title, s.Category, s.Phase)
		} else {
			label = fmt.Sprintf("%s  phase:%s", s.Title, s.Phase)
		}
		options = append(options, huh.NewOption(label, s.ID))
	}
	options = append(options, huh.NewOption("Start new conversation", "__new__"))

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Continue a session or start new?").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return nil, nil // User canceled
	}

	if selected == "__new__" {
		s := coreide.NewSession(providerID)
		if err := coreide.Save(outputDir, s); err != nil {
			return nil, err
		}
		return s, nil
	}

	return coreide.Load(outputDir, selected)
}

func assemblePrompt(cfg *config.Config) (string, error) {
	docsDir := cfg.DocsDir()

	// Load manifest for context.
	m, err := manifest.Load(docsDir)
	if err != nil {
		// Manifest might not exist yet; that's OK.
		m = &manifest.Manifest{}
	}

	summary := coreide.BuildManifestSummary(m)

	// Load preferences for context.
	prefs, err := preferences.Load(cfg.OutputDir())
	if err != nil {
		prefs = &preferences.Preferences{}
	}

	prefsCtx := coreide.BuildPreferencesContext(prefs)

	return coreide.AssemblePrompt(summary, prefsCtx), nil
}
