package compile

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/gitkeep"
	"github.com/securacore/codectx/ui"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

var Command = &cli.Command{
	Name:  "compile",
	Usage: "Build compiled documentation set from all active sources",
	Action: func(ctx context.Context, c *cli.Command) error {
		return run()
	},
}

// compileMsg is sent when the compile goroutine completes.
type compileMsg struct {
	result *compile.Result
	err    error
}

// progressMsg wraps a compile progress event.
type progressMsg compile.ProgressEvent

// compileModel is the bubbletea model for the compile progress spinner.
type compileModel struct {
	spinner spinner.Model
	message string
	done    bool
	result  *compile.Result
	err     error
}

func newCompileModel() compileModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = ui.DimStyle
	return compileModel{
		spinner: s,
		message: "Compiling...",
	}
}

func (m compileModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m compileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case compileMsg:
		m.done = true
		m.result = msg.result
		m.err = msg.err
		return m, tea.Quit

	case progressMsg:
		m.message = msg.Message
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m compileModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("  %s %s", m.spinner.View(), m.message)
}

func run() error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Clean up .gitkeep files in documentation directories that now contain
	// content. This keeps the working tree tidy as documentation is added.
	if err := gitkeep.Clean(cfg.DocsDir()); err != nil {
		ui.Warn(fmt.Sprintf("gitkeep cleanup: %s", err))
	}

	var result *compile.Result

	if ui.IsTTY() {
		// Use bubbletea inline spinner for live progress.
		m := newCompileModel()
		p := tea.NewProgram(m, tea.WithOutput(ui.Writer()))

		// Run compile in background, sending progress and final result.
		go func() {
			var compileErr error
			result, compileErr = compile.Compile(cfg, func(ev compile.ProgressEvent) {
				p.Send(progressMsg(ev))
			})
			p.Send(compileMsg{result: result, err: compileErr})
		}()

		finalModel, runErr := p.Run()
		if runErr != nil {
			return fmt.Errorf("tui: %w", runErr)
		}

		fm := finalModel.(compileModel)
		if fm.err != nil {
			return fmt.Errorf("compile: %w", fm.err)
		}
		result = fm.result
	} else {
		// Non-TTY: simple fallback.
		ui.Step("Compiling...")
		result, err = compile.Compile(cfg)
		if err != nil {
			return fmt.Errorf("compile: %w", err)
		}
	}

	if result.UpToDate {
		ui.Done("Already up to date")
		return nil
	}

	ui.Done(fmt.Sprintf("Compiled to %s", result.OutputDir))

	// File listing (Vite/Rollup style — show all objects per section).
	if len(result.Entries) > 0 {
		ui.Blank()
		for _, entry := range result.Entries {
			sizeStr := formatSize(entry.Size)
			ui.Item(fmt.Sprintf("%-14s %-20s %s %s",
				ui.DimStyle.Render(entry.Section),
				entry.ID,
				ui.DimStyle.Render(entry.Object),
				ui.DimStyle.Render(sizeStr)))
		}
	}

	// Heuristics summary.
	ui.Blank()
	ui.KV("Objects stored", result.ObjectsStored, 18)
	if result.ObjectsPruned > 0 {
		ui.KV("Objects pruned", result.ObjectsPruned, 18)
	}
	ui.KV("Packages", result.Packages, 18)
	if result.Compressed {
		ui.KV("Compression", "cmdx", 18)
	}

	if result.Heuristics != nil {
		h := result.Heuristics
		ui.KV("Total size", formatSize(h.Totals.SizeBytes), 18)
		ui.KV("Est. tokens", formatTokens(h.Totals.EstimatedTokens), 18)
		if h.Totals.AlwaysLoad > 0 {
			ui.KV("Always-load", h.Totals.AlwaysLoad, 18)
		}
	}

	if result.Dedup.Total() > 0 {
		if len(result.Dedup.Duplicates) > 0 {
			ui.KV("Deduplicated", len(result.Dedup.Duplicates), 18)
		}
		if result.Dedup.HasConflicts() {
			ui.Blank()
			ui.Warn(fmt.Sprintf("%d conflict(s):", len(result.Dedup.Conflicts)))
			for _, c := range result.Dedup.Conflicts {
				ui.Item(fmt.Sprintf("[%s] %s: kept from %s, skipped from %s",
					c.Section, c.ID, c.WinnerPkg, c.SkippedPkg))
			}
		}
	}
	ui.Blank()

	return nil
}

// formatSize formats bytes to a human-readable string.
func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f kB", kb)
	}
	mb := kb / 1024
	return fmt.Sprintf("%.1f MB", mb)
}

// formatTokens formats token count to a human-readable string.
func formatTokens(tokens int) string {
	if tokens >= 1000 {
		k := float64(tokens) / 1000
		if k == float64(int(k)) {
			return fmt.Sprintf("~%dk", int(k))
		}
		return fmt.Sprintf("~%.1fk", k)
	}
	return fmt.Sprintf("~%d", tokens)
}
