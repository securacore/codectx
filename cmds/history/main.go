// Package history implements the `codectx history` command group which
// provides access to query and generate history.
//
// Subcommands:
//   - codectx history            (default: show recent activity)
//   - codectx history queries    (show query history)
//   - codectx history chunks     (show generate history)
//   - codectx history show <hash> (print a history document)
//   - codectx history clear      (wipe all history)
package history

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/history"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// recentCount is the number of entries shown in the default overview.
const recentCount = 10

// Command is the CLI definition for `codectx history`.
var Command = &cli.Command{
	Name:  "history",
	Usage: "View query and generate history",
	Description: `View recent query and generate activity, recall previously generated
documents, or clear the history.

Examples:
  codectx history
  codectx history queries
  codectx history show a1b2c3
  codectx history clear`,
	Action: runOverview,
	Commands: []*cli.Command{
		queriesCommand,
		chunksCommand,
		showCommand,
		clearCommand,
	},
}

// queriesCommand shows query history entries.
var queriesCommand = &cli.Command{
	Name:   "queries",
	Usage:  "Show query history",
	Action: runQueries,
}

// chunksCommand shows generate/chunks history entries.
var chunksCommand = &cli.Command{
	Name:   "chunks",
	Usage:  "Show generate history",
	Action: runChunks,
}

// showCommand prints a previously generated document to stdout.
var showCommand = &cli.Command{
	Name:      "show",
	Usage:     "Print a history document to stdout",
	ArgsUsage: "<hash>",
	Action:    runShow,
}

// clearCommand removes all history data after confirmation.
var clearCommand = &cli.Command{
	Name:   "clear",
	Usage:  "Remove all history data",
	Action: runClear,
}

// runOverview shows recent queries and generates (default action).
func runOverview(_ context.Context, _ *cli.Command) error {
	histDir, err := resolveHistDir()
	if err != nil {
		return err
	}

	queries, err := history.ReadQueryHistory(histDir, recentCount)
	if err != nil {
		return fmt.Errorf("reading query history: %w", err)
	}

	chunks, err := history.ReadChunksHistory(histDir, recentCount)
	if err != nil {
		return fmt.Errorf("reading chunks history: %w", err)
	}

	if len(queries) == 0 && len(chunks) == 0 {
		fmt.Printf("\n%s No history entries found.\n\n", tui.Warning())
		return nil
	}

	if len(queries) > 0 {
		fmt.Printf("\n%s %s\n\n",
			tui.Arrow(),
			tui.StyleBold.Render(fmt.Sprintf("Recent queries (last %d)", len(queries))),
		)
		printQueryEntries(queries)
	}

	if len(chunks) > 0 {
		fmt.Printf("\n%s %s\n\n",
			tui.Arrow(),
			tui.StyleBold.Render(fmt.Sprintf("Recent generates (last %d)", len(chunks))),
		)
		printChunksEntries(chunks)
	}

	fmt.Println()
	return nil
}

// runQueries shows query history.
func runQueries(_ context.Context, _ *cli.Command) error {
	histDir, err := resolveHistDir()
	if err != nil {
		return err
	}

	entries, err := history.ReadQueryHistory(histDir, 0)
	if err != nil {
		return fmt.Errorf("reading query history: %w", err)
	}

	if len(entries) == 0 {
		fmt.Printf("\n%s No query history found.\n\n", tui.Warning())
		return nil
	}

	fmt.Printf("\n%s %s\n\n",
		tui.Arrow(),
		tui.StyleBold.Render(fmt.Sprintf("Query history (%d entries)", len(entries))),
	)
	printQueryEntries(entries)
	fmt.Println()
	return nil
}

// runChunks shows generate/chunks history.
func runChunks(_ context.Context, _ *cli.Command) error {
	histDir, err := resolveHistDir()
	if err != nil {
		return err
	}

	entries, err := history.ReadChunksHistory(histDir, 0)
	if err != nil {
		return fmt.Errorf("reading chunks history: %w", err)
	}

	if len(entries) == 0 {
		fmt.Printf("\n%s No generate history found.\n\n", tui.Warning())
		return nil
	}

	fmt.Printf("\n%s %s\n\n",
		tui.Arrow(),
		tui.StyleBold.Render(fmt.Sprintf("Generate history (%d entries)", len(entries))),
	)
	printChunksEntries(entries)
	fmt.Println()
	return nil
}

// runShow prints a history document to stdout.
func runShow(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing hash",
			Detail: []string{
				"Usage: codectx history show <hash>",
			},
			Suggestions: []tui.Suggestion{
				{Text: "View recent generates:", Command: "codectx history chunks"},
			},
		}.Render())
		return fmt.Errorf("missing hash argument")
	}

	hashPrefix := cmd.Args().First()

	histDir, err := resolveHistDir()
	if err != nil {
		return err
	}

	content, err := history.ShowDocument(histDir, hashPrefix)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Document not found",
			Detail: []string{err.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "List available documents:", Command: "codectx history chunks"},
			},
		}.Render())
		return fmt.Errorf("showing document: %w", err)
	}

	fmt.Print(content)
	return nil
}

// runClear removes all history data after confirmation.
func runClear(_ context.Context, _ *cli.Command) error {
	histDir, err := resolveHistDir()
	if err != nil {
		return err
	}

	// Require interactive terminal for confirmation prompt.
	if !term.IsTerminal(os.Stdin.Fd()) {
		fmt.Print(tui.ErrorMsg{
			Title: "Cannot clear history in non-interactive mode",
			Detail: []string{
				"The clear command requires interactive confirmation.",
			},
		}.Render())
		return fmt.Errorf("non-interactive mode")
	}

	var confirmed bool
	if err := huh.NewConfirm().
		Title("Clear all history data?").
		Value(&confirmed).
		Run(); err != nil {
		return err
	}
	if !confirmed {
		fmt.Printf("\n%s Clear canceled.\n\n",
			tui.StyleMuted.Render("-"),
		)
		return nil
	}

	if err := history.Clear(histDir); err != nil {
		return fmt.Errorf("clearing history: %w", err)
	}

	fmt.Printf("\n%s %s\n\n",
		tui.Success(),
		tui.StyleBold.Render("History cleared"),
	)
	return nil
}

// resolveHistDir discovers the project and returns the history directory.
func resolveHistDir() (string, error) {
	histDir, _, _, err := shared.ResolveHistoryDir()
	if err != nil {
		return "", err
	}
	return histDir, nil
}

// printQueryEntries formats and prints query history entries.
func printQueryEntries(entries []history.QueryEntry) {
	for _, e := range entries {
		ts := time.Unix(0, e.Timestamp)
		fmt.Printf("%s%s  %s  %s\n",
			tui.Indent(1),
			tui.StyleMuted.Render(ts.Format("2006-01-02 15:04")),
			tui.StyleBold.Render(fmt.Sprintf("%q", e.RawQuery)),
			tui.StyleMuted.Render(fmt.Sprintf("%d results", e.ResultCount)),
		)
		if e.ExpandedQuery != "" && e.ExpandedQuery != e.RawQuery {
			fmt.Printf("%s%s\n",
				tui.Indent(2),
				tui.KeyValue("Expanded", tui.StyleMuted.Render(e.ExpandedQuery)),
			)
		}
	}
}

// printChunksEntries formats and prints chunks/generate history entries.
func printChunksEntries(entries []history.ChunksEntry) {
	for _, e := range entries {
		ts := time.Unix(0, e.Timestamp)
		hash := history.ShortHash(e.ContentHash)
		fmt.Printf("%s%s  %s  %s\n",
			tui.Indent(1),
			tui.StyleMuted.Render(ts.Format("2006-01-02 15:04")),
			tui.StyleAccent.Render(hash),
			tui.StyleMuted.Render(fmt.Sprintf("%s tokens", tui.FormatNumber(e.TokenCount))),
		)

		// Show chunk IDs on a second line for readability.
		styledIDs := make([]string, len(e.ChunkIDs))
		for i, id := range e.ChunkIDs {
			styledIDs[i] = tui.StyleAccent.Render(id)
		}
		fmt.Printf("%s%s\n",
			tui.Indent(2),
			tui.KeyValue("Chunks", strings.Join(styledIDs, ", ")),
		)
	}
}
