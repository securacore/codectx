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

	token := registry.GitHubToken()
	gh := registry.NewGitHubClient(token)
	gitClient := registry.NewGitClient(token)

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
	fmt.Print(renderSearchResults(query, results))

	return nil
}

// renderSearchResults formats the search results for terminal display.
func renderSearchResults(query string, results []registry.SearchResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s Results for: %s\n\n",
		tui.Arrow(),
		tui.StyleBold.Render(fmt.Sprintf("%q", query)),
	)

	for i, r := range results {
		b.WriteString(formatResult(i+1, r))
	}

	if len(results) > 0 {
		example := results[0]
		fmt.Fprintf(&b, "Install with: %s\n\n",
			tui.StyleCommand.Render(
				fmt.Sprintf("codectx install %s@%s:latest", example.Name, example.Org),
			),
		)
	}

	return b.String()
}

// formatResult formats a single search result entry.
func formatResult(index int, r registry.SearchResult) string {
	var b strings.Builder

	version := r.LatestVersion
	if version == "" {
		version = "no tags"
	}

	fmt.Fprintf(&b, "%s%d. %s (v%s)",
		tui.Indent(1),
		index,
		tui.StyleAccent.Render(r.Name+"@"+r.Org),
		version,
	)
	if r.Stars > 0 {
		fmt.Fprintf(&b, " %s %d", tui.StyleMuted.Render("*"), r.Stars)
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "%s%s\n", tui.Indent(2), tui.StyleMuted.Render(r.FullName))

	if r.Description != "" {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(2), r.Description)
	}

	b.WriteString("\n")
	return b.String()
}
