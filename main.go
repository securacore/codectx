package main

import (
	"context"
	"log"
	"os"
	"securacore/codectx/cmds/compile"
	initialize "securacore/codectx/cmds/init"
	"securacore/codectx/cmds/version"

	"github.com/urfave/cli/v3"
)

func main() {
	app := cli.Command{
		Name:  "codectx",
		Usage: "AI Code Documentation Package Manager",
		Commands: []*cli.Command{
			compile.Command,
			initialize.Command,
			version.Command,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
