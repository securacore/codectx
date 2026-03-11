package main

import (
	"context"
	"log"
	"os"

	compilecmd "github.com/securacore/codectx/cmds/compile"
	generatecmd "github.com/securacore/codectx/cmds/generate"
	initcmd "github.com/securacore/codectx/cmds/init"
	linkcmd "github.com/securacore/codectx/cmds/link"
	querycmd "github.com/securacore/codectx/cmds/query"
	sessioncmd "github.com/securacore/codectx/cmds/session"
	"github.com/securacore/codectx/cmds/version"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "codectx",
		Usage: "Documentation compiler for AI-driven development",
		Commands: []*cli.Command{
			compilecmd.Command,
			generatecmd.Command,
			initcmd.Command,
			linkcmd.Command,
			querycmd.Command,
			sessioncmd.Command,
			version.Command,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
