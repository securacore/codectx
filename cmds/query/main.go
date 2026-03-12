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

	"charm.land/huh/v2/spinner"
	"github.com/securacore/codectx/core/project"
	corequery "github.com/securacore/codectx/core/query"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx query`.
var Command = &cli.Command{
	Name:      "query",
	Usage:     "Search compiled documentation",
	ArgsUsage: "<search terms>",
	Description: `Search all compiled BM25 indexes (objects, specs, system) and return
ranked results grouped by type with manifest metadata.

Results include chunk IDs that can be passed to codectx generate.

Examples:
  codectx query "jwt refresh token"
  codectx query "error handling middleware" --top 5`,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "top",
			Usage: "Maximum number of results per index type",
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
	projectDir, cfg, err := project.DiscoverAndLoad(".")
	if err != nil {
		fmt.Print(tui.ProjectNotFoundError())
		return fmt.Errorf("project not found: %w", err)
	}

	// --- Step 2: Resolve compiled directory ---
	compiledDir := corequery.CompiledDir(projectDir, cfg)

	// --- Step 3: Determine topN ---
	topN := resolveTopN(int(cmd.Int("top")), projectDir, cfg)

	// --- Step 4: Run the query ---
	var result *corequery.QueryResult
	var queryErr error

	err = spinner.New().
		Title("Searching compiled documentation...").
		Action(func() {
			result, queryErr = corequery.RunQuery(compiledDir, queryStr, topN)
		}).
		Run()
	if err != nil {
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

	// --- Step 5: Display results ---
	fmt.Print(corequery.FormatQueryResults(result))

	return nil
}

// resolveTopN determines the number of results per index type.
// If flagValue is positive, it's used directly. Otherwise, the default
// is loaded from the AI config or falls back to project.DefaultResultsCount.
func resolveTopN(flagValue int, projectDir string, cfg *project.Config) int {
	if flagValue > 0 {
		return flagValue
	}

	if cfg != nil {
		aiCfg, aiErr := project.LoadAIConfigForProject(projectDir, cfg)
		if aiErr == nil && aiCfg.Consumption.ResultsCount > 0 {
			return aiCfg.Consumption.ResultsCount
		}
	}

	return project.DefaultResultsCount
}
