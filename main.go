package main

import (
	"context"
	"os"

	"securacore/codectx/cmds/add"
	"securacore/codectx/cmds/compile"
	initialize "securacore/codectx/cmds/init"
	"securacore/codectx/cmds/link"
	"securacore/codectx/cmds/search"
	"securacore/codectx/cmds/version"
	"securacore/codectx/ui"

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

	if err := app.Run(context.Background(), os.Args); err != nil {
		ui.Fail(err.Error())
		os.Exit(1)
	}
}
