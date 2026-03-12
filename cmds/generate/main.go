// Package generate implements the `codectx generate` command which assembles
// specific chunks into a single coherent reading document.
//
// The command accepts comma-separated chunk IDs (from codectx query output),
// loads the corresponding chunk content and manifest metadata, groups by type
// (Instructions, System, Reasoning), and outputs the assembled document.
//
// By default, the document is printed to stdout and the summary goes to
// stderr (Unix pipe-friendly). Use --file to write the document to a file
// instead, in which case the summary prints to stdout.
//
// All generated documents are saved to the project history directory for
// later recall via `codectx history show <hash>`.
//
// Usage:
//
//	codectx generate "obj:a1b2c3.03,obj:a1b2c3.04,spec:f7g8h9.02"
//	codectx generate --file output.md "obj:a1b2c3.03,spec:f7g8h9.02"
package generate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/history"
	"github.com/securacore/codectx/core/project"
	corequery "github.com/securacore/codectx/core/query"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx generate`.
var Command = &cli.Command{
	Name:      "generate",
	Usage:     "Assemble chunks into a reading document",
	ArgsUsage: "<chunk-id>,<chunk-id>,...",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "file",
			Usage: "Write document to file instead of stdout",
		},
	},
	Description: `Assemble specific chunks into a single coherent reading document.
Accepts obj:, spec:, and sys: prefixed chunk IDs from codectx query output.

By default, the document is printed to stdout and the summary goes to
stderr. Use --file to write the document to a specific path.

Examples:
  codectx generate "obj:a1b2c3.03,spec:f7g8h9.02"
  codectx generate --file context.md "obj:a1b2c3.03,spec:f7g8h9.02"`,
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing chunk IDs",
			Detail: []string{
				"Usage: codectx generate \"<chunk-id>,<chunk-id>,...\"",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Find chunk IDs with:", Command: "codectx query \"search terms\""},
				{Text: "Generate from results:", Command: "codectx generate \"obj:a1b2c3.03,spec:f7g8h9.02\""},
			},
		}.Render())
		return fmt.Errorf("missing chunk IDs")
	}

	// Parse comma-separated chunk IDs, trimming whitespace.
	chunkIDs := corequery.ParseChunkIDs(cmd.Args().First())

	if len(chunkIDs) == 0 {
		fmt.Print(tui.ErrorMsg{
			Title: "No valid chunk IDs provided",
			Detail: []string{
				"Expected comma-separated chunk IDs like: obj:a1b2c3.03,spec:f7g8h9.02",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Find chunk IDs with:", Command: "codectx query \"search terms\""},
			},
		}.Render())
		return fmt.Errorf("no valid chunk IDs")
	}

	filePath := cmd.String("file")

	// --- Step 1: Discover and load the project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 2: Resolve compiled directory and load encoding ---
	compiledDir := corequery.CompiledDir(projectDir, cfg)
	encoding := project.ResolveEncoding(projectDir, cfg)

	// --- Step 3: Run generate ---
	var result *corequery.GenerateResult
	var genErr error

	if err = shared.RunWithSpinner("Assembling reading document...", func() {
		result, genErr = corequery.RunGenerate(compiledDir, encoding, chunkIDs)
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if genErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Generate failed",
			Detail: []string{
				genErr.Error(),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Verify chunk IDs exist:", Command: "codectx query \"search terms\""},
				{Text: "Compile documentation first:", Command: "codectx compile"},
			},
		}.Render())
		return fmt.Errorf("generate failed: %w", genErr)
	}

	// --- Step 4: History (best-effort) ---
	histDir := history.HistoryDir(projectDir, cfg)
	historyPath, histErr := history.LogGenerate(
		histDir, projectDir, cfg.Root,
		result.Document, result.ChunkIDs, result.TotalTokens, result.ContentHash,
	)
	if histErr != nil {
		shared.WarnHistory("saving generate result", histErr)
	}

	// Make history path relative to CWD for cleaner display.
	if historyPath != "" {
		if rel, relErr := filepath.Rel(".", historyPath); relErr == nil {
			historyPath = rel
		}
	}

	// --- Step 5: Output ---
	if filePath != "" {
		// --file mode: write document to file, summary to stdout.
		if err := os.WriteFile(filePath, []byte(result.Document), project.FilePerm); err != nil {
			fmt.Print(tui.ErrorMsg{
				Title:  "Failed to write file",
				Detail: []string{err.Error()},
			}.Render())
			return fmt.Errorf("writing file: %w", err)
		}
		fmt.Print(corequery.FormatGenerateSummary(result, historyPath, filePath))
	} else {
		// Default mode: document to stdout, summary to stderr.
		fmt.Print(result.Document)
		fmt.Fprint(os.Stderr, corequery.FormatGenerateSummary(result, historyPath, ""))
	}

	return nil
}
