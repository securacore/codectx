// Package prompt implements the `codectx prompt` command which combines
// query and generate into a single atomic operation.
//
// The command searches compiled documentation, auto-selects the top results
// within a computed token budget, generates a reading document from them,
// and outputs the full content. This eliminates the failure mode where AI
// agents run `codectx query` but skip `codectx generate`.
//
// The token budget is computed as:
//
//	budget = chunk_target × budget_multiplier × (1 + budget_delta)
//
// Both multiplier and delta are configurable in preferences.yml under the
// `prompt` section. The --delta flag allows per-command overrides, and
// --budget bypasses the formula entirely.
//
// Usage:
//
//	codectx prompt "jwt refresh token validation"
//	codectx prompt --delta 0.2 "error handling middleware"
//	codectx prompt --budget 2000 "architecture patterns"
//	codectx prompt --file output.md "React component patterns"
package prompt

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

// Command is the CLI definition for `codectx prompt`.
var Command = &cli.Command{
	Name:      "prompt",
	Usage:     "Query and generate documentation in one step",
	ArgsUsage: "<search terms>",
	Description: `Search compiled documentation and automatically generate the top results
into a single reading document. Combines codectx query and codectx generate
into one atomic operation.

The token budget controls how many chunks are auto-selected. It is computed
from the project's chunk target size, a multiplier, and an optional delta:

  budget = chunk_target × multiplier × (1 + delta)

Configure defaults in preferences.yml under the 'prompt' section.

Examples:
  codectx prompt "jwt refresh token"
  codectx prompt --delta 0.2 "error handling"
  codectx prompt --budget 2000 "architecture patterns"
  codectx prompt --file context.md "React component patterns"`,
	Flags: []cli.Flag{
		&cli.FloatFlag{
			Name:  "delta",
			Usage: "Override budget delta for this invocation (e.g. 0.1 = +10%, -0.2 = -20%)",
		},
		&cli.IntFlag{
			Name:  "budget",
			Usage: "Hard override: set token budget directly, bypassing the formula",
		},
		&cli.StringFlag{
			Name:  "file",
			Usage: "Write generated document to file instead of stdout",
		},
		&cli.IntFlag{
			Name:  "top",
			Usage: "Maximum number of query results to consider",
		},
		&cli.BoolFlag{
			Name:  "no-cache",
			Usage: "Bypass generate cache lookup",
		},
	},
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing search terms",
			Detail: []string{
				"Usage: codectx prompt \"<search terms>\" [flags]",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Quick search:", Command: "codectx prompt \"jwt refresh token\""},
				{Text: "Adjust budget:", Command: "codectx prompt --delta 0.2 \"error handling\""},
				{Text: "Fixed budget:", Command: "codectx prompt --budget 2000 \"architecture\""},
			},
		}.Render())
		return fmt.Errorf("missing search terms")
	}

	queryStr := cmd.Args().First()
	filePath := cmd.String("file")
	noCache := cmd.Bool("no-cache")

	// --- Step 1: Discover and load the project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 2: Resolve paths and config ---
	compiledDir := corequery.CompiledDir(projectDir, cfg)
	encoding := project.ResolveEncoding(projectDir, cfg)
	histDir := history.HistoryDir(projectDir, cfg)
	usageFile := usage.LocalPath(projectDir, cfg)
	caller := history.ResolveCallerContext()

	prefsCfg := shared.LoadPreferencesOrDefault(projectDir, cfg)

	// --- Step 3: Compute budget ---
	chunkTarget := prefsCfg.Chunking.TargetTokens
	if chunkTarget <= 0 {
		chunkTarget = 450 // fallback to default
	}

	var budget int
	var budgetFormula string

	budgetFlag := int(cmd.Int("budget"))
	if budgetFlag > 0 {
		budget = budgetFlag
		budgetFormula = fmt.Sprintf("%d (override)", budgetFlag)
	} else {
		var deltaOverride *float64
		if cmd.IsSet("delta") {
			d := cmd.Float("delta")
			deltaOverride = &d
		}
		budget = prefsCfg.Prompt.EffectiveBudget(chunkTarget, deltaOverride)

		// Build formula string for display.
		mult := prefsCfg.Prompt.BudgetMultiplier
		if mult <= 0 {
			mult = project.DefaultPromptBudgetMultiplier
		}
		delta := prefsCfg.Prompt.BudgetDelta
		if deltaOverride != nil {
			delta = *deltaOverride
		}
		budgetFormula = fmt.Sprintf("%d × %.0f × %.1f", chunkTarget, mult, 1+delta)
	}

	// --- Step 4: Determine topN ---
	topN := shared.ResolveTopN(int(cmd.Int("top")), projectDir, cfg)

	// --- Step 5: Run query ---
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

	// --- Step 6: Log query to history (best-effort) ---
	compileHash, _ := history.CompileHash(compiledDir)
	allResults := collectAllResults(result)
	totalResults := len(allResults)

	if logErr := history.LogQuery(histDir, projectDir, cfg.Root, queryStr, result.ExpandedQuery, totalResults, compileHash, caller); logErr != nil {
		shared.WarnHistory("logging query", logErr)
	}
	if usageErr := usage.UpdateQuery(usageFile); usageErr != nil {
		shared.WarnBestEffort("Updating usage metrics", usageErr)
	}

	// --- Step 7: Check for results ---
	if totalResults == 0 {
		fmt.Print(corequery.FormatPromptNoResults(queryStr))
		return nil
	}

	// --- Step 8: Auto-select chunks within budget ---
	chunkIDs, selectedTokens := selectChunks(allResults, budget)

	// --- Step 9: Cache lookup (unless --no-cache) ---
	if !noCache {
		if docPath, hit := history.GenerateCacheLookup(histDir, chunkIDs, compiledDir); hit {
			res, cacheErr := shared.ServeCacheHit(shared.CacheHitParams{
				DocPath:     docPath,
				ChunkIDs:    chunkIDs,
				HistDir:     histDir,
				CompiledDir: compiledDir,
				UsageFile:   usageFile,
				Caller:      caller,
			})
			if cacheErr != nil {
				return cacheErr
			}

			// Use recovered token count for display if available.
			displayTokens := selectedTokens
			if res.TokenCount > 0 {
				displayTokens = res.TokenCount
			}

			header := corequery.FormatPromptHeader(&corequery.PromptSummary{
				RawQuery:      queryStr,
				ExpandedQuery: result.ExpandedQuery,
				SelectedCount: len(chunkIDs),
				SelectedTotal: displayTokens,
				QueryTotal:    totalResults,
				Budget:        budget,
				BudgetFormula: budgetFormula,
			})

			cacheResult := &corequery.GenerateResult{
				TotalTokens: res.TokenCount,
				ContentHash: res.Hash,
				ChunkIDs:    chunkIDs,
			}
			historyPath := shared.BuildHistoryPath(histDir, res.DocFile)
			footer := corequery.FormatPromptFooter(cacheResult, historyPath, filePath, true)

			return shared.OutputDocument(shared.OutputDocumentParams{
				Content:  res.Content,
				FilePath: filePath,
				Header:   header,
				Footer:   footer,
			})
		}
	}

	// --- Step 10: Run generate ---
	var genResult *corequery.GenerateResult
	var genErr error

	if err = shared.RunWithSpinner("Generating document...", func() {
		genResult, genErr = corequery.RunGenerate(compiledDir, encoding, chunkIDs)
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
				{Text: "Verify compilation:", Command: "codectx compile"},
				{Text: "Try manual query:", Command: "codectx query \"" + queryStr + "\""},
			},
		}.Render())
		return fmt.Errorf("generate failed: %w", genErr)
	}

	// --- Step 11: Log generate to history (best-effort) ---
	contentHash := history.ContentHash([]byte(genResult.Document))
	docFile, histErr := history.LogGenerate(
		histDir, projectDir, cfg.Root,
		[]byte(genResult.Document), genResult.ChunkIDs, genResult.TotalTokens,
		contentHash, compileHash, false, caller,
	)
	if histErr != nil {
		shared.WarnHistory("saving generate result", histErr)
	}
	if usageErr := usage.UpdateGenerate(usageFile, genResult.TotalTokens, false, caller); usageErr != nil {
		shared.WarnBestEffort("Updating usage metrics", usageErr)
	}

	// --- Step 12: Output ---
	header := corequery.FormatPromptHeader(&corequery.PromptSummary{
		RawQuery:      queryStr,
		ExpandedQuery: result.ExpandedQuery,
		SelectedCount: len(chunkIDs),
		SelectedTotal: genResult.TotalTokens,
		QueryTotal:    totalResults,
		Budget:        budget,
		BudgetFormula: budgetFormula,
	})

	historyPath := shared.BuildHistoryPath(histDir, docFile)
	footer := corequery.FormatPromptFooter(genResult, historyPath, filePath, false)

	return shared.OutputDocument(shared.OutputDocumentParams{
		Content:  []byte(genResult.Document),
		FilePath: filePath,
		Header:   header,
		Footer:   footer,
	})
}

// collectAllResults returns a flat slice of all query results in score order.
// In unified (BM25F) mode, this is the Unified slice. In BM25 mode,
// results are interleaved from Instructions, Reasoning, and System by score.
func collectAllResults(r *corequery.QueryResult) []corequery.ResultEntry {
	if len(r.Unified) > 0 {
		return r.Unified
	}

	// BM25 mode: merge and sort by score descending.
	var all []corequery.ResultEntry
	all = append(all, r.Instructions...)
	all = append(all, r.Reasoning...)
	all = append(all, r.System...)

	// Sort by score descending (stable to preserve per-type order on ties).
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].Score > all[j-1].Score; j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	return all
}

// selectChunks picks chunk IDs from scored results until the token budget
// is exceeded. The first result is always included even if it exceeds the
// budget. Returns the selected IDs and total token count.
func selectChunks(results []corequery.ResultEntry, budget int) ([]string, int) {
	var selected []string
	total := 0
	for _, r := range results {
		if len(selected) > 0 && total+r.Tokens > budget {
			break
		}
		selected = append(selected, r.ChunkID)
		total += r.Tokens
	}
	return selected, total
}
