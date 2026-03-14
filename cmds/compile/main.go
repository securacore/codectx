// Package compile implements the `codectx compile` command which runs the
// full documentation compilation pipeline.
//
// The command discovers markdown files, parses and normalizes them, chunks
// them into token-counted semantic blocks, builds BM25 search indexes, and
// generates all manifest files.
//
// The TUI flow:
//  1. Discover the project (walk up to codectx.yml)
//  2. Load all configuration files (codectx.yml, ai.yml, preferences.yml)
//  3. Run the compilation pipeline with per-stage spinners
//  4. Display a formatted summary with statistics
//  5. Display any validation warnings
package compile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/scaffold"
	"github.com/securacore/codectx/core/tui"
	"github.com/securacore/codectx/core/usage"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx compile`.
var Command = &cli.Command{
	Name:  "compile",
	Usage: "Compile documentation into searchable chunks",
	Description: `Runs the full compilation pipeline: parse, strip, chunk, index,
and generate manifest files. Produces compiled output in .codectx/compiled/.`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "incremental",
			Usage: "Only reprocess changed files (default: true)",
			Value: true,
		},
	},
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	interactive := term.IsTerminal(os.Stdin.Fd())

	// --- Step 1: Discover and load the project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	rootDir := project.RootDir(projectDir, cfg)

	aiCfg, err := project.LoadAIConfigForProject(projectDir, cfg)
	if err != nil {
		fmt.Print(renderConfigError("AI configuration", project.AIConfigFile, err))
		return fmt.Errorf("loading AI config: %w", err)
	}

	prefsCfg, err := project.LoadPreferencesConfigForProject(projectDir, cfg)
	if err != nil {
		fmt.Print(renderConfigError("preferences", project.PreferencesFile, err))
		return fmt.Errorf("loading preferences: %w", err)
	}

	// --- Step 2b: Scaffold maintenance (if enabled) ---
	if prefsCfg.EffectiveScaffoldMaintenance() {
		mr, mrErr := scaffold.Maintain(projectDir, cfg)
		if mrErr != nil {
			fmt.Print(tui.WarnMsg{
				Title:  "Scaffold maintenance failed",
				Detail: []string{mrErr.Error()},
				Suggestions: []tui.Suggestion{
					{Text: "Run manual repair:", Command: "codectx repair"},
				},
			}.Render())
		} else if mr.HasActions() {
			var parts []string
			if mr.DirsCreated > 0 {
				parts = append(parts, fmt.Sprintf("%d dirs", mr.DirsCreated))
			}
			if mr.FilesRestored > 0 {
				parts = append(parts, fmt.Sprintf("%d files", mr.FilesRestored))
			}
			if mr.GitkeepsAdded > 0 {
				parts = append(parts, fmt.Sprintf("+%d .gitkeep", mr.GitkeepsAdded))
			}
			if mr.GitkeepsRemoved > 0 {
				parts = append(parts, fmt.Sprintf("-%d .gitkeep", mr.GitkeepsRemoved))
			}
			fmt.Printf("%s Scaffold: %s\n", tui.Arrow(), strings.Join(parts, ", "))
		}
	}

	compileCfg := compile.BuildConfig(projectDir, rootDir, cfg, aiCfg, prefsCfg)
	compileCfg.Incremental = cmd.Bool("incremental")

	// --- Step 3: Run compilation pipeline ---
	var result *compile.Result

	progress := func(stage, detail string) {
		fmt.Printf("%s %s\n", tui.StyleMuted.Render("["+stage+"]"), detail)
	}

	if interactive {
		// Print a header before per-stage progress lines.
		fmt.Printf("\n%s Compiling documentation...\n\n", tui.Arrow())
	}

	result, err = compile.Run(compileCfg, progress)
	if err != nil {
		return renderCompileError(err)
	}

	// --- Step 4: Display summary ---
	fmt.Print(renderSummary(result, cfg.Name, aiCfg.Compilation.Model, compileCfg.CompiledDir, projectDir, prefsCfg))

	// --- Step 5: Display warnings ---
	if len(result.Warnings) > 0 {
		fmt.Print(renderWarnings(result.Warnings))
	}

	// --- Step 6: Sync usage metrics (best-effort) ---
	localUsage := usage.LocalPath(projectDir, cfg)
	globalUsage := usage.GlobalPath(projectDir, cfg)
	if syncErr := usage.SyncGlobal(localUsage, globalUsage, cfg.Name); syncErr != nil {
		shared.WarnBestEffort("Syncing usage metrics", syncErr)
	}

	return nil
}

// renderConfigError formats a configuration loading error for terminal display.
func renderConfigError(configName, fileName string, err error) string {
	return tui.ErrorMsg{
		Title: fmt.Sprintf("Failed to load %s", configName),
		Detail: []string{
			fmt.Sprintf("Error reading %s", tui.StylePath.Render(project.CodectxDir+"/"+fileName)),
			err.Error(),
		},
		Suggestions: []tui.Suggestion{
			{Text: "Reinitialize the project:", Command: "codectx init"},
		},
	}.Render()
}

// renderCompileError formats a compilation error for terminal display.
func renderCompileError(err error) error {
	fmt.Print(tui.ErrorMsg{
		Title: "Compilation failed",
		Detail: []string{
			err.Error(),
		},
		Suggestions: []tui.Suggestion{
			{Text: "Check your markdown files for syntax errors"},
			{Text: "Verify your configuration files are valid"},
		},
	}.Render())
	return fmt.Errorf("compilation failed: %w", err)
}

// renderSummary formats the post-compilation summary.
func renderSummary(result *compile.Result, projectName, model, compiledDir, projectDir string, prefsCfg *project.PreferencesConfig) string {
	var b strings.Builder

	// Success header.
	fmt.Fprintf(&b, "\n%s %s\n\n",
		tui.Success(),
		tui.StyleBold.Render("Compilation complete"),
	)

	// Compiled line: files -> chunks (tokens)
	fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
		tui.KeyValue("Compiled", fmt.Sprintf("%d files -> %s chunks (%s tokens)",
			result.TotalFiles,
			tui.FormatNumber(result.TotalChunks),
			tui.FormatNumber(result.TotalTokens),
		)),
	)

	// Session line: token count vs budget (only if session context was assembled).
	if result.SessionBudget > 0 && result.SessionTokens > 0 {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Session", tui.FormatBudget(result.SessionTokens, result.SessionBudget)),
		)
	}

	// Index line: breakdown by type + active indexer.
	indexParts := []string{}
	if result.ObjectChunks > 0 {
		indexParts = append(indexParts, fmt.Sprintf("objects: %d", result.ObjectChunks))
	}
	if result.SpecChunks > 0 {
		indexParts = append(indexParts, fmt.Sprintf("specs: %d", result.SpecChunks))
	}
	if result.SystemChunks > 0 {
		indexParts = append(indexParts, fmt.Sprintf("system: %d", result.SystemChunks))
	}
	if len(indexParts) > 0 {
		activeIndexer := string(project.IndexerBM25F)
		if prefsCfg != nil {
			activeIndexer = string(prefsCfg.EffectiveIndexer())
		}
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Index", fmt.Sprintf("bm25 + bm25f (%s, active: %s)",
				strings.Join(indexParts, ", "),
				activeIndexer,
			)),
		)
	}

	// Tokens line: avg/min/max.
	if result.TotalChunks > 0 {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Tokens", fmt.Sprintf("avg %d, min %d, max %d per chunk",
				result.AvgTokens, result.MinTokens, result.MaxTokens,
			)),
		)
	}

	// Taxonomy terms.
	if result.TaxonomyTerms > 0 {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Taxonomy", fmt.Sprintf("%s terms extracted",
				tui.FormatNumber(result.TaxonomyTerms),
			)),
		)
	}

	// Deterministic bridges.
	if result.DetBridgeCount > 0 {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Bridges", fmt.Sprintf("%s deterministic",
				tui.FormatNumber(result.DetBridgeCount),
			)),
		)
	}

	// LLM augmentation.
	if result.LLMSkipped {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("LLM", fmt.Sprintf("skipped (%s)", result.LLMSkipReason)),
		)
	} else if result.LLMAliasCount > 0 || result.LLMBridgeCount > 0 {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("LLM", fmt.Sprintf("%d aliases, %d bridges (%s)",
				result.LLMAliasCount, result.LLMBridgeCount,
				tui.FormatDuration(result.LLMSeconds),
			)),
		)
	}

	// Changes line (incremental mode).
	if result.IncrementalMode {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Changes", fmt.Sprintf("%d new, %d modified, %d unchanged",
				result.NewFiles, result.ModifiedFiles, result.UnchangedFiles,
			)),
		)
	}

	// Oversized chunks warning.
	if result.Oversized > 0 {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Oversized", fmt.Sprintf("%d chunks exceed max_tokens", result.Oversized)),
		)
	}

	// Timing.
	fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
		tui.KeyValue("Time", tui.FormatDuration(result.TotalSeconds)),
	)

	// Configuration info.
	fmt.Fprintf(&b, "\n%s%s\n", tui.Indent(1),
		tui.KeyValue("Model", model),
	)

	// Output path relative to project dir.
	relOutput, err := filepath.Rel(projectDir, compiledDir)
	if err != nil {
		relOutput = compiledDir
	}
	fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
		tui.KeyValue("Output", relOutput),
	)

	b.WriteString("\n")
	return b.String()
}

// renderWarnings formats validation warnings for display.
func renderWarnings(warnings []string) string {
	// Group by file for cleaner display.
	if len(warnings) == 0 {
		return ""
	}

	detail := make([]string, 0, len(warnings))
	for _, w := range warnings {
		detail = append(detail, tui.StyleMuted.Render(w))
	}

	return tui.WarnMsg{
		Title:  fmt.Sprintf("%d validation warning(s)", len(warnings)),
		Detail: detail,
		Suggestions: []tui.Suggestion{
			{Text: "Fix the warnings and recompile:", Command: "codectx compile"},
		},
	}.Render()
}

// countNonZero returns how many of the given ints are greater than zero.
func countNonZero(values ...int) int {
	count := 0
	for _, v := range values {
		if v > 0 {
			count++
		}
	}
	return count
}
