package cmdx

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/core/cmdx"
	"github.com/securacore/codectx/ui"
	"github.com/urfave/cli/v3"
)

var statsCommand = &cli.Command{
	Name:      "stats",
	Usage:     "Show compression statistics for a Markdown file",
	ArgsUsage: "[input.md]",
	Action: func(ctx context.Context, c *cli.Command) error {
		return runStats(c)
	},
}

func runStats(c *cli.Command) error {
	input, err := readInput(c)
	if err != nil {
		return err
	}

	stats, err := cmdx.Analyze(input)
	if err != nil {
		return fmt.Errorf("stats: %w", err)
	}

	ui.Header("Compression statistics:")
	ui.KV("Original", fmt.Sprintf("%d bytes (%d est. tokens)", stats.OriginalBytes, stats.EstTokensBefore), 18)
	ui.KV("Compressed", fmt.Sprintf("%d bytes (%d est. tokens)", stats.CompressedBytes, stats.EstTokensAfter), 18)
	ui.KV("Byte savings", fmt.Sprintf("%.1f%%", stats.ByteSavings), 18)
	ui.KV("Token savings", fmt.Sprintf("%.1f%%", stats.TokenSavings), 18)
	ui.KV("Dict entries", fmt.Sprintf("%d (saved %d bytes)", stats.DictEntries, stats.DictSavings), 18)
	ui.KV("Domain savings", fmt.Sprintf("%d bytes", stats.DomainSavings), 18)
	ui.Blank()

	return nil
}
