// Package search implements the `codectx search` command which discovers
// documentation packages on GitHub.
//
// It queries the GitHub API for repositories matching the codectx-* naming
// convention and displays results with name, version, stars, and description.
// Results are annotated with installability status — packages without a
// GitHub Release containing package.tar.gz are marked as not installable.
package search

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
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
Results are sorted by installability and stars. Packages without a GitHub
Release archive are annotated with a warning.

Example:
  codectx search "react patterns"`,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "limit",
			Usage: "Maximum number of results",
			Value: 10,
		},
		&cli.BoolFlag{
			Name:  "show-uninstallable",
			Usage: "Include packages without a release archive in results",
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

	err := shared.RunWithSpinner("Searching packages...", func() {
		results, searchErr = gh.SearchPackages(ctx, query, limit)
	})
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
		fmt.Printf("\n%s No packages found for: %s\n\n",
			tui.Warning(),
			tui.StyleBold.Render(fmt.Sprintf("%q", query)),
		)
		return nil
	}

	// Resolve latest version and check release availability for each result.
	if err = shared.RunWithSpinner("Resolving versions...", func() {
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

			// Check if a GitHub Release with package.tar.gz exists.
			tag := registry.GitTag(r.LatestVersion)
			_, releaseErr := gh.ReleaseAssetURL(ctx, r.Author, r.FullName[strings.Index(r.FullName, "/")+1:], tag)
			r.HasRelease = releaseErr == nil
		}
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}

	// Filter out uninstallable packages unless --show-uninstallable is set
	// or the preference is enabled.
	showUninstallable := cmd.Bool("show-uninstallable")
	if !showUninstallable {
		showUninstallable = shouldShowUninstallable()
	}

	var hiddenCount int
	if !showUninstallable {
		results, hiddenCount = filterInstallable(results)
	}

	// Sort: installable results first, then no-release, then no-tags.
	sortResults(results)

	// Render results.
	fmt.Print(renderSearchResults(query, results, hiddenCount))

	return nil
}

// sortResults orders search results by installability:
// 1. Installable packages (has version + has release) — original star order
// 2. Has version but no release — original star order
// 3. No version tags — original star order
func sortResults(results []registry.SearchResult) {
	sort.SliceStable(results, func(i, j int) bool {
		return resultPriority(results[i]) < resultPriority(results[j])
	})
}

// resultPriority returns a sort key: 0 = installable, 1 = has version but
// no release, 2 = no version at all.
func resultPriority(r registry.SearchResult) int {
	if r.LatestVersion == "" {
		return 2
	}
	if !r.HasRelease {
		return 1
	}
	return 0
}

// countInstallable returns the number of results that have a release archive.
func countInstallable(results []registry.SearchResult) int {
	n := 0
	for _, r := range results {
		if r.HasRelease {
			n++
		}
	}
	return n
}

// filterInstallable removes results that don't have a release archive.
// Returns the filtered list and the count of hidden results.
func filterInstallable(results []registry.SearchResult) ([]registry.SearchResult, int) {
	return shared.FilterInstallable(results)
}

// shouldShowUninstallable checks the project preferences for the
// show_uninstallable setting. Returns false if no project is found
// or if the preference is not set.
func shouldShowUninstallable() bool {
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return false
	}
	prefs, err := project.LoadPreferencesConfigForProject(projectDir, cfg)
	if err != nil {
		return false
	}
	return prefs.Search.EffectiveShowUninstallable()
}

// renderSearchResults formats the search results for terminal display.
//
// Output format:
//
//	-> Results for: "react patterns"
//
//	  Found 3 packages (2 installable)
//
//	  1. react-patterns@community v2.4.0
//	     Repo: community/codectx-react-patterns (* 342)
//	     React component patterns and best practices
//
//	  2. react-testing@community v1.0.0  ⚠ no release archive
//	     Repo: community/codectx-react-testing (* 89)
//
//	  Add with: codectx add react-patterns@community:latest
func renderSearchResults(query string, results []registry.SearchResult, hiddenCount int) string {
	var b strings.Builder

	// Header — matches query command pattern.
	fmt.Fprintf(&b, "\n%s Results for: %s\n",
		tui.Arrow(),
		tui.StyleBold.Render(fmt.Sprintf("%q", query)),
	)

	// Summary count.
	installable := countInstallable(results)
	fmt.Fprintf(&b, "\n%s\n",
		tui.Indent(1)+formatSummaryLine(len(results), installable),
	)

	// Hidden packages note.
	if hiddenCount > 0 {
		noun := "packages"
		if hiddenCount == 1 {
			noun = "package"
		}
		fmt.Fprintf(&b, "%s%s %s\n",
			tui.Indent(1),
			tui.StyleMuted.Render(
				fmt.Sprintf("%d %s hidden (no release archive).", hiddenCount, noun),
			),
			tui.StyleMuted.Render("Use ")+tui.StyleCommand.Render("--show-uninstallable")+tui.StyleMuted.Render(" to include."),
		)
	}

	// Result entries.
	for i, r := range results {
		b.WriteString(formatResult(i+1, r))
	}

	// Install hint — uses first installable result, or first result as fallback.
	if len(results) > 0 {
		example := firstInstallable(results)
		fmt.Fprintf(&b, "%s\n\n",
			tui.Indent(1)+tui.KeyValue("Add with",
				tui.StyleCommand.Render(
					fmt.Sprintf("codectx add %s@%s:latest", example.Name, example.Author),
				),
			),
		)
	}

	return b.String()
}

// formatSummaryLine renders the "Found N packages (M installable)" summary.
func formatSummaryLine(total, installable int) string {
	if installable == total {
		return tui.StyleMuted.Render(fmt.Sprintf("Found %d packages", total))
	}
	return tui.StyleMuted.Render(fmt.Sprintf("Found %d packages (%d installable)", total, installable))
}

// firstInstallable returns the first result with HasRelease true.
// Falls back to the first result if none are installable.
func firstInstallable(results []registry.SearchResult) registry.SearchResult {
	for _, r := range results {
		if r.HasRelease {
			return r
		}
	}
	return results[0]
}

// formatResult formats a single search result entry.
//
// Installable result:
//
//  1. react-patterns@community v2.4.0
//     Repo: community/codectx-react-patterns (* 342)
//     React component patterns and best practices
//
// Non-installable result (no release):
//
//  2. react-testing@community v1.0.0  ⚠ no release archive
//     Repo: community/codectx-react-testing (* 89)
//
// Non-installable result (no tags):
//
//  3. react-utils@someone  ⚠ no version tags
//     Repo: someone/codectx-react-utils
func formatResult(index int, r registry.SearchResult) string {
	var b strings.Builder

	// Line 1: number, package ref, version, optional warning.
	fmt.Fprintf(&b, "\n%s%d. %s",
		tui.Indent(1),
		index,
		tui.StyleAccent.Render(r.Name+"@"+r.Author),
	)

	if r.LatestVersion != "" {
		fmt.Fprintf(&b, " %s", tui.StyleBold.Render("v"+r.LatestVersion))
	}

	// Installability warnings — appended to the title line.
	if r.LatestVersion == "" {
		fmt.Fprintf(&b, "  %s %s", tui.Warning(), tui.StyleMuted.Render("no version tags"))
	} else if !r.HasRelease {
		fmt.Fprintf(&b, "  %s %s", tui.Warning(), tui.StyleMuted.Render("no release archive"))
	}

	b.WriteString("\n")

	// Line 2: Repo as KeyValue with optional star count.
	repoDisplay := tui.StylePath.Render(r.FullName)
	if r.Stars > 0 {
		repoDisplay += " " + tui.StyleMuted.Render(fmt.Sprintf("(* %s)", tui.FormatNumber(r.Stars)))
	}
	fmt.Fprintf(&b, "%s%s\n",
		tui.Indent(2),
		tui.KeyValue("Repo", repoDisplay),
	)

	// Line 3: Description (if present).
	if r.Description != "" {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(2), tui.StyleMuted.Render(r.Description))
	}

	return b.String()
}
