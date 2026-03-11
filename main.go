package main

import (
	"context"
	"log"
	"os"

	compilecmd "github.com/securacore/codectx/cmds/compile"
	generatecmd "github.com/securacore/codectx/cmds/generate"
	initcmd "github.com/securacore/codectx/cmds/init"
	installcmd "github.com/securacore/codectx/cmds/install"
	linkcmd "github.com/securacore/codectx/cmds/link"
	plancmd "github.com/securacore/codectx/cmds/plan"
	publishcmd "github.com/securacore/codectx/cmds/publish"
	querycmd "github.com/securacore/codectx/cmds/query"
	searchcmd "github.com/securacore/codectx/cmds/search"
	sessioncmd "github.com/securacore/codectx/cmds/session"
	updatecmd "github.com/securacore/codectx/cmds/update"
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
			installcmd.Command,
			linkcmd.Command,
			plancmd.Command,
			publishcmd.Command,
			querycmd.Command,
			searchcmd.Command,
			sessioncmd.Command,
			updatecmd.Command,
			version.Command,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
