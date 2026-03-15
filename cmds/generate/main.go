// Package generate implements the `codectx generate` command which assembles
// specific chunks into a single coherent reading document.
//
// The command accepts comma-separated chunk IDs (from codectx query output),
// loads the corresponding chunk content and manifest metadata, groups by type
// (Instructions, System, Reasoning), and outputs the assembled document.
//
// Before assembly, the command checks the generate cache. If the same chunk
// set was previously assembled against the current compilation state, the
// cached document is served directly. Use --no-cache to bypass this.
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
//	codectx generate --no-cache "obj:a1b2c3.03,spec:f7g8h9.02"
package generate

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/history"
	"github.com/securacore/codectx/core/project"
	corequery "github.com/securacore/codectx/core/query"
	"github.com/securacore/codectx/core/tui"
	"github.com/securacore/codectx/core/usage"
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
		&cli.BoolFlag{
			Name:  "no-cache",
			Usage: "Bypass cache lookup and always run the full generate pipeline",
		},
	},
	Description: `Assemble specific chunks into a single coherent reading document.
Accepts obj:, spec:, and sys: prefixed chunk IDs from codectx query output.

By default, the document is printed to stdout and the summary goes to
stderr. Use --file to write the document to a specific path.

Examples:
  codectx generate "obj:a1b2c3.03,spec:f7g8h9.02"
  codectx generate --file context.md "obj:a1b2c3.03,spec:f7g8h9.02"
  codectx generate --no-cache "obj:a1b2c3.03,spec:f7g8h9.02"`,
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
	noCache := cmd.Bool("no-cache")

	// --- Step 1: Discover and load the project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 2: Resolve paths ---
	compiledDir := corequery.CompiledDir(projectDir, cfg)
	encoding := project.ResolveEncoding(projectDir, cfg)
	histDir := history.HistoryDir(projectDir, cfg)
	usageFile := usage.LocalPath(projectDir, cfg)
	caller := history.ResolveCallerContext()

	// --- Step 3: Cache lookup (unless --no-cache) ---
	if !noCache {
		if docPath, hit := history.GenerateCacheLookup(histDir, chunkIDs, compiledDir); hit {
			res, err := shared.ServeCacheHit(shared.CacheHitParams{
				DocPath:     docPath,
				ChunkIDs:    chunkIDs,
				HistDir:     histDir,
				CompiledDir: compiledDir,
				UsageFile:   usageFile,
				Caller:      caller,
			})
			if err != nil {
				return err
			}
			cacheResult := &corequery.GenerateResult{
				TotalTokens: res.TokenCount,
				ContentHash: res.Hash,
				ChunkIDs:    chunkIDs,
			}
			historyPath := shared.BuildHistoryPath(histDir, res.DocFile)
			summary := corequery.FormatGenerateSummary(cacheResult, historyPath, filePath, true)
			return shared.OutputDocument(shared.OutputDocumentParams{
				Content:  res.Content,
				FilePath: filePath,
				Footer:   summary,
			})
		}
	}

	// --- Step 4: Run generate ---
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

	// --- Step 5: History (best-effort) ---
	compileHash, _ := history.CompileHash(compiledDir)
	contentHash := history.ContentHash([]byte(result.Document))

	docFile, histErr := history.LogGenerate(
		histDir, projectDir, cfg.Root,
		[]byte(result.Document), result.ChunkIDs, result.TotalTokens,
		contentHash, compileHash, false, caller,
	)
	if histErr != nil {
		shared.WarnHistory("saving generate result", histErr)
	}

	// --- Step 6: Usage (best-effort) ---
	if usageErr := usage.UpdateGenerate(usageFile, result.TotalTokens, false, caller); usageErr != nil {
		shared.WarnBestEffort("Updating usage metrics", usageErr)
	}

	// --- Step 7: Output ---
	historyPath := shared.BuildHistoryPath(histDir, docFile)
	summary := corequery.FormatGenerateSummary(result, historyPath, filePath, false)

	return shared.OutputDocument(shared.OutputDocumentParams{
		Content:  []byte(result.Document),
		FilePath: filePath,
		Footer:   summary,
	})
}
