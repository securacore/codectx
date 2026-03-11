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

	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/cmds/version"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx compile`.
var Command = &cli.Command{
	Name:  "compile",
	Usage: "Compile documentation into searchable chunks",
	Description: `Runs the full compilation pipeline: parse, strip, chunk, index,
and generate manifest files. Produces compiled output in .codectx/compiled/.`,
	Action: run,
}

func run(_ context.Context, _ *cli.Command) error {
	interactive := term.IsTerminal(os.Stdin.Fd())

	// --- Step 1: Discover the project ---
	projectDir, err := project.Discover(".")
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "No codectx project found",
			Detail: []string{
				"Could not find codectx.yml in the current directory or any parent.",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Initialize a project first:", Command: "codectx init"},
			},
		}.Render())
		return fmt.Errorf("project not found: %w", err)
	}

	// --- Step 2: Load configuration ---
	cfg, err := project.LoadConfig(filepath.Join(projectDir, project.ConfigFileName))
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Failed to load project configuration",
			Detail: []string{
				fmt.Sprintf("Error reading %s", tui.StylePath.Render(project.ConfigFileName)),
				err.Error(),
			},
		}.Render())
		return fmt.Errorf("loading config: %w", err)
	}

	rootDir := project.RootDir(projectDir, cfg)
	codectxDir := filepath.Join(rootDir, project.CodectxDir)

	aiCfg, err := project.LoadAIConfig(filepath.Join(codectxDir, project.AIConfigFile))
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Failed to load AI configuration",
			Detail: []string{
				fmt.Sprintf("Error reading %s", tui.StylePath.Render(project.CodectxDir+"/"+project.AIConfigFile)),
				err.Error(),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Reinitialize the project:", Command: "codectx init"},
			},
		}.Render())
		return fmt.Errorf("loading AI config: %w", err)
	}

	prefsCfg, err := project.LoadPreferencesConfig(filepath.Join(codectxDir, project.PreferencesFile))
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Failed to load preferences",
			Detail: []string{
				fmt.Sprintf("Error reading %s", tui.StylePath.Render(project.CodectxDir+"/"+project.PreferencesFile)),
				err.Error(),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Reinitialize the project:", Command: "codectx init"},
			},
		}.Render())
		return fmt.Errorf("loading preferences: %w", err)
	}

	// Build active dependencies map from config.
	activeDeps := make(map[string]bool)
	for name, dep := range cfg.Dependencies {
		if dep != nil && dep.Active {
			activeDeps[name] = true
		}
	}

	compiledDir := filepath.Join(codectxDir, project.CompiledDir)

	compileCfg := compile.Config{
		ProjectDir:  projectDir,
		RootDir:     rootDir,
		CompiledDir: compiledDir,
		SystemDir:   project.SystemDir,
		Encoding:    aiCfg.Compilation.Encoding,
		Version:     version.Version,
		Chunking:    prefsCfg.Chunking,
		BM25:        prefsCfg.BM25,
		Validation:  prefsCfg.Validation,
		ActiveDeps:  activeDeps,
	}

	// --- Step 3: Run compilation pipeline ---
	var result *compile.Result

	if interactive {
		var compileErr error

		err = spinner.New().
			Title("Compiling documentation...").
			ActionWithErr(func(_ context.Context) error {
				result, compileErr = compile.Run(compileCfg, nil)
				return compileErr
			}).
			Run()
		if err != nil {
			return renderCompileError(err)
		}
	} else {
		result, err = compile.Run(compileCfg, func(stage, detail string) {
			fmt.Printf("%s %s\n", tui.StyleMuted.Render("["+stage+"]"), detail)
		})
		if err != nil {
			return renderCompileError(err)
		}
	}

	// --- Step 4: Display summary ---
	fmt.Print(renderSummary(result, cfg.Name, aiCfg.Compilation.Model, compiledDir, projectDir))

	// --- Step 5: Display warnings ---
	if len(result.Warnings) > 0 {
		fmt.Print(renderWarnings(result.Warnings))
	}

	return nil
}

// stageTitle returns a human-readable title for a pipeline stage.
func stageTitle(stage, detail string) string {
	titles := map[string]string{
		compile.StagePrepare:   "Preparing output directories...",
		compile.StageDiscover:  "Discovering source files...",
		compile.StageParse:     "Parsing and validating...",
		compile.StageChunk:     "Chunking documents...",
		compile.StageWrite:     "Writing chunk files...",
		compile.StageIndex:     "Building search index...",
		compile.StageManifest:  "Generating manifests...",
		compile.StageHeuristic: "Computing heuristics...",
	}

	title, ok := titles[stage]
	if !ok {
		title = stage
	}

	if detail != "" {
		return fmt.Sprintf("%s (%s)", title, detail)
	}
	return title
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
func renderSummary(result *compile.Result, projectName, model, compiledDir, projectDir string) string {
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

	// Index line: breakdown by type.
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
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Index", fmt.Sprintf("%d indexes (%s)",
				countNonZero(result.ObjectChunks, result.SpecChunks, result.SystemChunks),
				strings.Join(indexParts, ", "),
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
