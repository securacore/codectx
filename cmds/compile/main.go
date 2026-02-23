package compile

import (
	"context"
	"fmt"

	"securacore/codectx/core/compile"
	"securacore/codectx/core/config"

	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"

var Command = &cli.Command{
	Name:  "compile",
	Usage: "Build compiled documentation set from all active sources",
	Action: func(ctx context.Context, c *cli.Command) error {
		return run()
	},
}

func run() error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	result, err := compile.Compile(cfg)
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	fmt.Printf("Compiled to %s\n", result.OutputDir)
	fmt.Printf("  Files copied: %d\n", result.FilesCopied)
	fmt.Printf("  Packages:     %d\n", result.Packages)

	if result.Dedup.Total() > 0 {
		if len(result.Dedup.Duplicates) > 0 {
			fmt.Printf("  Deduplicated: %d\n", len(result.Dedup.Duplicates))
		}
		if result.Dedup.HasConflicts() {
			fmt.Printf("\nWarnings (%d conflict(s)):\n", len(result.Dedup.Conflicts))
			for _, c := range result.Dedup.Conflicts {
				fmt.Printf("  [%s] %s: kept from %s, skipped from %s\n",
					c.Section, c.ID, c.WinnerPkg, c.SkippedPkg)
			}
		}
	}

	return nil
}
