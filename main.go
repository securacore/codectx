package main

import (
	"context"
	"log"
	"os"
	"securacore/codectx/cmds/add"
	"securacore/codectx/cmds/compile"
	initialize "securacore/codectx/cmds/init"
	"securacore/codectx/cmds/link"
	"securacore/codectx/cmds/version"

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
			version.Command,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
