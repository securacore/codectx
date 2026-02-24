package compile

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/ui"

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

	var result *compile.Result
	err = ui.SpinErr("Compiling...", func() error {
		var compileErr error
		result, compileErr = compile.Compile(cfg)
		return compileErr
	})
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	if result.UpToDate {
		ui.Done("Already up to date")
		return nil
	}

	ui.Done(fmt.Sprintf("Compiled to %s", result.OutputDir))
	ui.Blank()
	ui.KV("Objects stored", result.ObjectsStored, 16)
	if result.ObjectsPruned > 0 {
		ui.KV("Objects pruned", result.ObjectsPruned, 16)
	}
	ui.KV("Packages", result.Packages, 16)

	if result.Dedup.Total() > 0 {
		if len(result.Dedup.Duplicates) > 0 {
			ui.KV("Deduplicated", len(result.Dedup.Duplicates), 16)
		}
		if result.Dedup.HasConflicts() {
			ui.Blank()
			ui.Warn(fmt.Sprintf("%d conflict(s):", len(result.Dedup.Conflicts)))
			for _, c := range result.Dedup.Conflicts {
				ui.Item(fmt.Sprintf("[%s] %s: kept from %s, skipped from %s",
					c.Section, c.ID, c.WinnerPkg, c.SkippedPkg))
			}
		}
	}

	return nil
}
