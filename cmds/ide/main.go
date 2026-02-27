package ide

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	coreide "github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/core/ide/launcher"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"
	corewatch "github.com/securacore/codectx/core/watch"
	"github.com/securacore/codectx/ui"
	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

// Command is the codectx ai ide command.
var Command = &cli.Command{
	Name:  "ide",
	Usage: "Launch an AI documentation authoring session",
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
		return fmt.Errorf("codectx ai ide requires an interactive terminal")
	}

	// Load project config.
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	outputDir := cfg.OutputDir()

	// Handle --list flag.
	if c.Bool("list") {
		return listSessions(outputDir)
	}

	// Auto-compile to ensure documentation is fresh.
	ui.Step("Compiling documentation...")
	result, err := compile.Compile(cfg)
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}
	if result.UpToDate {
		ui.Done("Documentation is up to date")
	} else {
		ui.Done(fmt.Sprintf("Compiled (%d objects)", result.ObjectsStored))
	}

	// Load preferences for launcher resolution and prompt context.
	prefs, err := preferences.Load(outputDir)
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	// Resolve AI binary launcher.
	l, err := launcher.Resolve(prefs)
	if err != nil {
		return err
	}

	ui.Done(fmt.Sprintf("AI binary: %s", l.ID()))

	// Clean up old sessions.
	_, _ = coreide.Cleanup(outputDir, 30*24*time.Hour)

	// Determine session: resume explicit, auto-resume, or create new.
	var session *coreide.Session

	if resumeID := c.String("resume"); resumeID != "" {
		session, err = coreide.Load(outputDir, resumeID)
		if err != nil {
			return fmt.Errorf("resume session: %w", err)
		}
	} else {
		session, err = pickOrCreateSession(outputDir, l)
		if err != nil {
			return err
		}
	}

	if session == nil {
		ui.Canceled()
		return nil
	}

	// Assemble the documentation authoring directive.
	prompt, err := assemblePrompt(cfg)
	if err != nil {
		return fmt.Errorf("assemble prompt: %w", err)
	}

	// Build CLI arguments for the AI binary.
	var args []string
	if session.SessionID != "" {
		args = l.ResumeArgs(session.SessionID, prompt)
		ui.Step(fmt.Sprintf("Resuming session: %s", session.Title))
	} else {
		// New session — generate a session ID if the launcher supports it.
		if l.SupportsSessionID() {
			if cl, ok := l.(*launcher.Claude); ok {
				sid := cl.GenerateSessionID()
				session.SessionID = sid
				if err := coreide.Save(outputDir, session); err != nil {
					return fmt.Errorf("save session: %w", err)
				}
			}
		}
		args = l.NewSessionArgs(session.SessionID, prompt)
		ui.Step("Starting new documentation session")
	}

	// Start watch goroutine for live recompilation.
	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()

	w := corewatch.New(configFile)
	go func() {
		_ = w.Run(watchCtx)
	}()

	// Drain watch results silently (the AI binary owns the terminal).
	go func() {
		for {
			select {
			case <-w.Results():
				// Silently consume — recompilation happens automatically.
			case <-watchCtx.Done():
				return
			}
		}
	}()

	ui.Blank()

	// Launch the AI binary as a child process.
	cmd := exec.Command(l.Binary(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward signals to the child process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err = cmd.Run()
	signal.Stop(sigCh)
	close(sigCh)

	// Cancel watch goroutine.
	watchCancel()

	// Post-session: extract session ID if needed and update metadata.
	if !l.SupportsSessionID() && session.SessionID == "" {
		rootDir, _ := os.Getwd()
		if sid, findErr := l.FindLatestSession(rootDir); findErr == nil && sid != "" {
			session.SessionID = sid
		}
	}

	// Always save session state (updates timestamp).
	_ = coreide.Save(outputDir, session)

	ui.Blank()
	if err != nil {
		ui.Warn(fmt.Sprintf("AI session exited: %s", err))
	} else {
		ui.Done("AI session ended")
	}
	ui.Step(fmt.Sprintf("Session saved: %s", session.ID))

	return nil
}

func listSessions(outputDir string) error {
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

// pickOrCreateSession handles automatic session management:
// - 0 active sessions: create new
// - 1 active session: auto-resume
// - N active sessions: show selector with most-recent highlighted
func pickOrCreateSession(outputDir string, l launcher.Launcher) (*coreide.Session, error) {
	active, err := coreide.Active(outputDir)
	if err != nil {
		return nil, err
	}

	if len(active) == 0 {
		// No active sessions — create new.
		s := coreide.NewSession(l.ID())
		if err := coreide.Save(outputDir, s); err != nil {
			return nil, err
		}
		return s, nil
	}

	if len(active) == 1 {
		// Single active session — auto-resume.
		return active[0], nil
	}

	// Multiple active sessions — let user choose.
	// active[0] is already the most recently updated (sorted by List).
	options := make([]huh.Option[string], 0, len(active)+1)
	for i, s := range active {
		var label string
		if s.Category != "" {
			label = fmt.Sprintf("%s  [%s]  phase:%s", s.Title, s.Category, s.Phase)
		} else {
			label = fmt.Sprintf("%s  phase:%s", s.Title, s.Phase)
		}
		if i == 0 {
			label += "  (latest)"
		}
		options = append(options, huh.NewOption(label, s.ID))
	}
	options = append(options, huh.NewOption("Start new session", "__new__"))

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Resume a session or start new?").
				Options(options...).
				Value(&selected),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return nil, nil // User canceled
	}

	if selected == "__new__" {
		s := coreide.NewSession(l.ID())
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
