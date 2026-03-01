package md

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	coremd "github.com/securacore/codectx/core/md"
	"github.com/securacore/codectx/ui"
	"github.com/urfave/cli/v3"
)

var statsCommand = &cli.Command{
	Name:      "stats",
	Usage:     "Show compression statistics for a Markdown file or directory",
	ArgsUsage: "[input.md]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "dir",
			Usage: "Analyze all .md files in a directory recursively",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.String("dir") != "" {
			return runStatsDir(c)
		}
		return runStats(c)
	},
}

func runStats(c *cli.Command) error {
	input, err := readInput(c)
	if err != nil {
		return err
	}

	stats, err := coremd.Analyze(input)
	if err != nil {
		return fmt.Errorf("stats: %w", err)
	}

	printStats(stats)
	return nil
}

func printStats(s *coremd.Stats) {
	ui.Header("Compression statistics:")
	ui.KV("Original", fmt.Sprintf("%d bytes (%d tokens)", s.OriginalBytes, s.EstTokensBefore), 18)
	ui.KV("Compressed", fmt.Sprintf("%d bytes (%d tokens)", s.CompressedBytes, s.EstTokensAfter), 18)
	ui.KV("Byte savings", fmt.Sprintf("%.1f%%", s.ByteSavings), 18)
	ui.KV("Token savings", fmt.Sprintf("%.1f%%", s.TokenSavings), 18)
	ui.Blank()
}

// runStatsDir analyzes all .md files in a directory and prints a summary table.
func runStatsDir(c *cli.Command) error {
	dir := c.String("dir")

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", dir, err)
	}

	if len(files) == 0 {
		ui.Warn(fmt.Sprintf("No .md files found in %s", dir))
		return nil
	}

	// Print table header.
	fmt.Printf("%-55s %7s %7s %7s %7s\n",
		"File", "Orig", "Compact", "Byte%", "Tok%")
	fmt.Println(strings.Repeat("-", 89))

	// Accumulators for totals.
	var totalOrig, totalCompressed int
	var totalTokensBefore, totalTokensAfter int
	var fileCount int

	for _, path := range files {
		input, err := os.ReadFile(path)
		if err != nil {
			ui.Fail(fmt.Sprintf("read %s: %s", path, err))
			continue
		}

		stats, err := coremd.Analyze(input)
		if err != nil {
			ui.Fail(fmt.Sprintf("analyze %s: %s", path, err))
			continue
		}

		relPath, _ := filepath.Rel(dir, path)
		if len(relPath) > 55 {
			relPath = "..." + relPath[len(relPath)-52:]
		}

		fmt.Printf("%-55s %7d %7d %6.1f%% %6.1f%%\n",
			relPath,
			stats.OriginalBytes,
			stats.CompressedBytes,
			stats.ByteSavings,
			stats.TokenSavings,
		)

		totalOrig += stats.OriginalBytes
		totalCompressed += stats.CompressedBytes
		totalTokensBefore += stats.EstTokensBefore
		totalTokensAfter += stats.EstTokensAfter
		fileCount++
	}

	// Print summary.
	fmt.Println(strings.Repeat("-", 89))

	var totalByteSavings, totalTokenSavings float64
	if totalOrig > 0 {
		totalByteSavings = float64(totalOrig-totalCompressed) / float64(totalOrig) * 100
	}
	if totalTokensBefore > 0 {
		totalTokenSavings = float64(totalTokensBefore-totalTokensAfter) / float64(totalTokensBefore) * 100
	}

	fmt.Printf("%-55s %7d %7d %6.1f%% %6.1f%%\n",
		fmt.Sprintf("TOTAL (%d files)", fileCount),
		totalOrig,
		totalCompressed,
		totalByteSavings,
		totalTokenSavings,
	)
	fmt.Println()

	ui.Header("Aggregate:")
	ui.KV("Raw markdown", fmt.Sprintf("%d tokens", totalTokensBefore), 20)
	ui.KV("Compact markdown", fmt.Sprintf("%d tokens (%+d)", totalTokensAfter, totalTokensAfter-totalTokensBefore), 20)
	ui.Blank()

	return nil
}
