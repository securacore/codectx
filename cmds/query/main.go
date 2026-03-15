// Package query implements the `codectx query` command which searches
// compiled documentation using BM25 indexes.
//
// The command discovers the project, loads compiled BM25 indexes and
// manifest metadata, runs the search query against all three index types
// (objects, specs, system), and displays ranked results with metadata.
//
// Usage:
//
//	codectx query "jwt refresh token validation" [--top N]
package query

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

// Command is the CLI definition for `codectx query`.
var Command = &cli.Command{
	Name:      "query",
	Usage:     "Search compiled documentation",
	ArgsUsage: "<search terms>",
	Description: `Search compiled documentation using BM25 or BM25F indexes.

In BM25F mode (default), results are fused across object, spec, and system
indexes using Reciprocal Rank Fusion (RRF) and graph-based re-ranking into
a single unified ranked list. In BM25 mode, results are grouped by type.

The active indexer is controlled by the 'indexer' field in preferences.yml.

Results include chunk IDs that can be passed to codectx generate.

Examples:
  codectx query "jwt refresh token"
  codectx query "error handling middleware" --top 5`,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "top",
			Usage: "Maximum number of results (total in bm25f mode, per-index in bm25 mode)",
		},
	},
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing search terms",
			Detail: []string{
				"Usage: codectx query \"<search terms>\" [--top N]",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Search for a topic:", Command: "codectx query \"jwt refresh token\""},
				{Text: "Limit results:", Command: "codectx query \"error handling\" --top 5"},
			},
		}.Render())
		return fmt.Errorf("missing search terms")
	}

	queryStr := cmd.Args().First()

	// --- Step 1: Discover and load the project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 2: Resolve compiled directory ---
	compiledDir := corequery.CompiledDir(projectDir, cfg)

	// --- Step 3: Determine topN ---
	topN := shared.ResolveTopN(int(cmd.Int("top")), projectDir, cfg)

	// --- Step 4: Load preferences for indexer selection ---
	prefsCfg := shared.LoadPreferencesOrDefault(projectDir, cfg)

	// --- Step 5: Run the query ---
	var result *corequery.QueryResult
	var queryErr error

	indexer := prefsCfg.EffectiveIndexer()
	if err = shared.RunWithSpinner("Searching compiled documentation...", func() {
		if indexer == project.IndexerBM25F {
			result, queryErr = corequery.RunQueryUnified(compiledDir, queryStr, topN, prefsCfg.Query)
		} else {
			result, queryErr = corequery.RunQuery(compiledDir, queryStr, topN)
		}
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if queryErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Query failed",
			Detail: []string{
				queryErr.Error(),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Compile documentation first:", Command: "codectx compile"},
			},
		}.Render())
		return fmt.Errorf("query failed: %w", queryErr)
	}

	// --- Step 6: Display results ---
	fmt.Print(corequery.FormatQueryResults(result))

	// --- Step 7: History logging (best-effort) ---
	histDir := history.HistoryDir(projectDir, cfg)
	compileHash, _ := history.CompileHash(compiledDir)
	caller := history.ResolveCallerContext()
	totalResults := len(result.Instructions) + len(result.Reasoning) + len(result.System) + len(result.Unified)

	if logErr := history.LogQuery(histDir, projectDir, cfg.Root, queryStr, result.ExpandedQuery, totalResults, compileHash, caller); logErr != nil {
		shared.WarnHistory("logging query", logErr)
	}

	// --- Step 8: Usage (best-effort) ---
	usageFile := usage.LocalPath(projectDir, cfg)
	if usageErr := usage.UpdateQuery(usageFile); usageErr != nil {
		shared.WarnBestEffort("Updating usage metrics", usageErr)
	}

	return nil
}
