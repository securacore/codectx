package search

import (
	"context"
	"fmt"
	"strconv"

	"github.com/securacore/codectx/core/resolve"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

var Command = &cli.Command{
	Name:      "search",
	Aliases:   []string{"s"},
	Usage:     "Search for documentation packages on GitHub",
	Category:  "Package Management",
	ArgsUsage: "<query>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "author",
			Usage: "Filter by GitHub user or organization",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() < 1 {
			return runInteractive(c.String("author"))
		}
		return run(c.Args().First(), c.String("author"))
	},
}

// searchFn is the function type for searching packages.
type searchFn func(query, author string) ([]resolve.SearchResult, error)

func run(query, author string) error {
	return runWith(query, author, resolve.Search)
}

func runWith(query, author string, search searchFn) error {
	var results []resolve.SearchResult
	err := ui.SpinErr(fmt.Sprintf("Searching for packages matching %q...", query), func() error {
		var searchErr error
		results, searchErr = search(query, author)
		return searchErr
	})
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		ui.Done("No packages found.")
		return nil
	}

	// Build table data.
	headers := []string{"PACKAGE", "STARS", "DESCRIPTION"}
	rows := make([][]string, len(results))
	for i, r := range results {
		pkg := fmt.Sprintf("%s@%s", r.Name, r.Author)
		desc := r.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		rows[i] = []string{pkg, strconv.Itoa(r.Stars), desc}
	}

	ui.Blank()
	ui.Table(headers, rows, 2)
	ui.Blank()
	ui.Done(fmt.Sprintf("%d package(s) found. Install with: codectx add <package>", len(results)))
	ui.Blank()

	return nil
}
