package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/securacore/codectx/cmds/activate"
	"github.com/securacore/codectx/cmds/add"
	"github.com/securacore/codectx/cmds/compile"
	initialize "github.com/securacore/codectx/cmds/init"
	"github.com/securacore/codectx/cmds/install"
	"github.com/securacore/codectx/cmds/link"
	new "github.com/securacore/codectx/cmds/new"
	"github.com/securacore/codectx/cmds/search"
	"github.com/securacore/codectx/cmds/self"
	"github.com/securacore/codectx/cmds/sync"
	"github.com/securacore/codectx/cmds/version"
	"github.com/securacore/codectx/cmds/watch"
	"github.com/securacore/codectx/core/update"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

func main() {
	app := cli.Command{
		Name:  "codectx",
		Usage: "AI Code Documentation Package Manager",
		Commands: []*cli.Command{
			activate.Command,
			add.Command,
			compile.Command,
			initialize.Command,
			install.Command,
			link.Command,
			new.Command,
			search.Command,
			self.Command,
			sync.Command,
			version.Command,
			watch.Command,
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
				ui.Item("Run: codectx self upgrade")
			}
		case <-time.After(500 * time.Millisecond):
			// Don't block.
		}
	}
}
