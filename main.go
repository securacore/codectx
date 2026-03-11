package main

import (
	"context"
	"log"
	"os"

	compilecmd "github.com/securacore/codectx/cmds/compile"
	initcmd "github.com/securacore/codectx/cmds/init"
	"github.com/securacore/codectx/cmds/version"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "codectx",
		Usage: "Documentation compiler for AI-driven development",
		Commands: []*cli.Command{
			compilecmd.Command,
			initcmd.Command,
			version.Command,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
