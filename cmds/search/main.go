// Package search implements the `codectx search` command which discovers
// documentation packages on GitHub.
//
// It queries the GitHub API for repositories matching the codectx-* naming
// convention and displays results with name, version, stars, and description.
package search

import (
	"context"
	"fmt"
	"strings"

	"charm.land/huh/v2/spinner"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx search`.
var Command = &cli.Command{
	Name:      "search",
	Usage:     "Search for documentation packages on GitHub",
	ArgsUsage: "<query>",
	Description: `Search for codectx packages on GitHub using the codectx-* naming convention.
Results are sorted by stars and include version information.

Example:
  codectx search "react patterns"`,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "limit",
			Usage: "Maximum number of results",
			Value: 10,
		},
	},
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	query := strings.Join(cmd.Args().Slice(), " ")
	if query == "" {
		fmt.Print(tui.ErrorMsg{
			Title: "No search query provided",
			Suggestions: []tui.Suggestion{
				{Text: "Provide a search term:", Command: `codectx search "react patterns"`},
			},
		}.Render())
		return fmt.Errorf("no search query provided")
	}

	limit := int(cmd.Int("limit"))

	gh := registry.NewGitHubClient()
	gitClient := registry.NewGitClient()

	var results []registry.SearchResult
	var searchErr error

	err := spinner.New().
		Title("Searching packages...").
		Action(func() {
			results, searchErr = gh.SearchPackages(ctx, query, limit)
		}).
		Run()
	if err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if searchErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Search failed",
			Detail: []string{
				searchErr.Error(),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Check your network connection and try again"},
			},
		}.Render())
		return searchErr
	}

	if len(results) == 0 {
		fmt.Printf("\n%s No packages found for: %q\n\n", tui.Warning(), query)
		return nil
	}

	// Resolve latest version for each result.
	err = spinner.New().
		Title("Resolving versions...").
		Action(func() {
			for i := range results {
				r := &results[i]
				tags, tagErr := gitClient.ListRemoteTags(ctx, "https://github.com/"+r.FullName)
				if tagErr != nil {
					continue
				}
				resolved, verErr := registry.ResolveVersion(tags, registry.LatestVersion)
				if verErr != nil {
					continue
				}
				r.LatestVersion = registry.VersionFromTag(resolved)
			}
		}).
		Run()
	if err != nil {
		return fmt.Errorf("spinner: %w", err)
	}

	// Render results.
	fmt.Printf("\nSearch results for: %q\n\n", query)

	for i, r := range results {
		version := r.LatestVersion
		if version == "" {
			version = "no tags"
		}

		// 1. react-patterns@community (v2.4.0) ★ 342
		fmt.Printf("%s%d. %s (v%s)",
			tui.Indent(1),
			i+1,
			tui.StyleAccent.Render(r.Name+"@"+r.Org),
			version,
		)
		if r.Stars > 0 {
			fmt.Printf(" %s %d", tui.StyleMuted.Render("*"), r.Stars)
		}
		fmt.Println()

		// github.com/community/codectx-react-patterns
		fmt.Printf("%s%s\n", tui.Indent(2), tui.StyleMuted.Render(r.FullName))

		// Description
		if r.Description != "" {
			fmt.Printf("%s%s\n", tui.Indent(2), r.Description)
		}

		fmt.Println()
	}

	if len(results) > 0 {
		example := results[0]
		fmt.Printf("Install with: %s\n\n",
			tui.StyleCommand.Render(
				fmt.Sprintf("codectx install %s@%s:latest", example.Name, example.Org),
			),
		)
	}

	return nil
}
