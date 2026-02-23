package search

import (
	"context"
	"fmt"

	"securacore/codectx/core/resolve"

	"github.com/urfave/cli/v3"
)

var Command = &cli.Command{
	Name:      "search",
	Usage:     "Search for documentation packages on GitHub",
	ArgsUsage: "<query>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "author",
			Usage: "Filter by GitHub user or organization",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() < 1 {
			return fmt.Errorf("query argument required (e.g., codectx search react)")
		}
		return run(c.Args().First(), c.String("author"))
	},
}

func run(query, author string) error {
	fmt.Printf("Searching for codectx packages matching %q...\n\n", query)

	results, err := resolve.Search(query, author)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No packages found.")
		return nil
	}

	// Print results as a formatted table.
	fmt.Printf("%-30s %-6s %s\n", "PACKAGE", "STARS", "DESCRIPTION")
	fmt.Printf("%-30s %-6s %s\n", "-------", "-----", "-----------")

	for _, r := range results {
		pkg := fmt.Sprintf("%s@%s", r.Name, r.Author)
		desc := r.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Printf("%-30s %-6d %s\n", pkg, r.Stars, desc)
	}

	fmt.Printf("\n%d package(s) found. Install with: codectx add <package>\n", len(results))

	return nil
}
