// Package usage implements the `codectx usage` command which displays
// local and global usage metrics.
//
// Usage:
//
//	codectx usage
//	codectx usage --local
//	codectx usage --global
//	codectx usage --reset-local
package usage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/tui"
	coreusage "github.com/securacore/codectx/core/usage"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx usage`.
var Command = &cli.Command{
	Name:  "usage",
	Usage: "Display token usage metrics",
	Description: `Show local machine and project lifetime usage metrics including
token counts, invocation counts, cache hit rates, and usage by caller/model.

Examples:
  codectx usage
  codectx usage --local
  codectx usage --global
  codectx usage --reset-local`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "local",
			Usage: "Show only local machine metrics",
		},
		&cli.BoolFlag{
			Name:  "global",
			Usage: "Show only project lifetime metrics",
		},
		&cli.BoolFlag{
			Name:  "reset-local",
			Usage: "Reset local usage.yml to zero without syncing to global (debugging)",
		},
	},
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	localPath := coreusage.LocalPath(projectDir, cfg)
	globalPath := coreusage.GlobalPath(projectDir, cfg)

	// Handle --reset-local.
	if cmd.Bool("reset-local") {
		if err := coreusage.InitLocalFile(localPath); err != nil {
			return fmt.Errorf("resetting local usage: %w", err)
		}
		fmt.Printf("\n%s %s\n\n",
			tui.Success(),
			tui.StyleBold.Render("Local usage reset to zero"),
		)
		return nil
	}

	showLocal := !cmd.Bool("global")
	showGlobal := !cmd.Bool("local")

	if showLocal {
		local := coreusage.ReadLocal(localPath)
		fmt.Print(formatSection("Token usage (local machine)", local))
	}

	if showGlobal {
		global := coreusage.ReadGlobal(globalPath)
		fmt.Print(formatSection("Project lifetime (global_usage.yml)", global))
	}

	return nil
}

// formatSection renders a usage metrics section with the standard TUI pattern:
// section header, one KeyValue per line at indent 1, optional breakdowns at
// indent 2, and timestamp footer.
func formatSection(title string, m coreusage.Metrics) string {
	var b strings.Builder

	// Section header.
	fmt.Fprintf(&b, "\n%s %s\n\n",
		tui.Arrow(),
		tui.StyleBold.Render(title),
	)

	// Summary stats — one KeyValue per line.
	fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
		tui.KeyValue("Total tokens generated", tui.FormatNumber(m.TotalTokens)))
	fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
		tui.KeyValue("Query invocations", tui.FormatNumber(m.QueryInvocations)))
	fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
		tui.KeyValue("Generate invocations", tui.FormatNumber(m.GenerateInvocations)))

	// Cache hit rate (only meaningful with generate invocations).
	if m.GenerateInvocations > 0 {
		rate := float64(m.CacheHits) / float64(m.GenerateInvocations) * 100.0
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Cache hit rate",
				fmt.Sprintf("%.1f%%  %s",
					rate,
					tui.StyleMuted.Render(fmt.Sprintf("(%s / %s)",
						tui.FormatNumber(m.CacheHits),
						tui.FormatNumber(m.GenerateInvocations),
					)),
				),
			),
		)
	}

	// Caller breakdown.
	if len(m.TokensByCaller) > 0 {
		fmt.Fprintf(&b, "\n%s%s\n", tui.Indent(1), tui.StyleMuted.Render("By caller:"))
		formatBreakdown(&b, m.TokensByCaller, m.TotalTokens)
	}

	// Model breakdown.
	if len(m.TokensByModel) > 0 {
		fmt.Fprintf(&b, "\n%s%s\n", tui.Indent(1), tui.StyleMuted.Render("By model:"))
		formatBreakdown(&b, m.TokensByModel, m.TotalTokens)
	}

	// Timestamps.
	if m.FirstSeen > 0 {
		fmt.Fprintf(&b, "\n%s%s\n", tui.Indent(1),
			tui.KeyValue("Tracking since", time.Unix(0, m.FirstSeen).Format("2006-01-02")))
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Last updated", time.Unix(0, m.LastUpdated).Format("2006-01-02")))
	}

	if m.LastCompile > 0 {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Last compile sync", time.Unix(0, m.LastCompile).Format("2006-01-02")))
	}

	b.WriteString("\n")
	return b.String()
}

// formatBreakdown renders a sorted breakdown of tokens by key at indent 2.
// Each entry shows the key, token count, and percentage of total.
func formatBreakdown(b *strings.Builder, breakdown map[string]int, total int) {
	type entry struct {
		key    string
		tokens int
	}
	entries := make([]entry, 0, len(breakdown))
	for k, v := range breakdown {
		entries = append(entries, entry{k, v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].tokens > entries[j].tokens
	})

	for _, e := range entries {
		pct := 0.0
		if total > 0 {
			pct = float64(e.tokens) / float64(total) * 100.0
		}
		fmt.Fprintf(b, "%s%s\n", tui.Indent(2),
			tui.KeyValue(e.key,
				fmt.Sprintf("%s tokens  %s",
					tui.FormatNumber(e.tokens),
					tui.StyleMuted.Render(fmt.Sprintf("(%.1f%%)", pct)),
				),
			),
		)
	}
}
