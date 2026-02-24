package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/securacore/codectx/cmds/add"
	"github.com/securacore/codectx/cmds/compile"
	initialize "github.com/securacore/codectx/cmds/init"
	"github.com/securacore/codectx/cmds/link"
	"github.com/securacore/codectx/cmds/search"
	"github.com/securacore/codectx/cmds/version"
	"github.com/securacore/codectx/core/update"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

func main() {
	app := cli.Command{
		Name:  "codectx",
		Usage: "AI Code Documentation Package Manager",
		Commands: []*cli.Command{
			add.Command,
			compile.Command,
			initialize.Command,
			link.Command,
			search.Command,
			version.Command,
		},
	}

	// Start background update check.
	updateCh := make(chan *update.Result, 1)
	go func() {
		updateCh <- update.Check(version.Version)
	}()

	if err := app.Run(context.Background(), os.Args); err != nil {
		ui.Fail(err.Error())
		os.Exit(1)
	}

	// Collect update result with a short timeout.
	if ui.IsTTY() {
		select {
		case result := <-updateCh:
			if result != nil && result.Available {
				fmt.Println()
				ui.Warn(fmt.Sprintf(
					"A new version of codectx is available: v%s -> v%s",
					result.Current, result.Latest,
				))
				ui.Item("Update: curl -fsSL https://raw.githubusercontent.com/securacore/codectx/main/bin/install | sh")
			}
		case <-time.After(500 * time.Millisecond):
			// Don't block.
		}
	}
}
