package main

import (
	"context"
	"log"
	"os"

	addcmd "github.com/securacore/codectx/cmds/add"
	compilecmd "github.com/securacore/codectx/cmds/compile"
	generatecmd "github.com/securacore/codectx/cmds/generate"
	historycmd "github.com/securacore/codectx/cmds/history"
	initcmd "github.com/securacore/codectx/cmds/init"
	installcmd "github.com/securacore/codectx/cmds/install"
	linkcmd "github.com/securacore/codectx/cmds/link"
	newcmd "github.com/securacore/codectx/cmds/new"
	plancmd "github.com/securacore/codectx/cmds/plan"
	publishcmd "github.com/securacore/codectx/cmds/publish"
	querycmd "github.com/securacore/codectx/cmds/query"
	removecmd "github.com/securacore/codectx/cmds/remove"
	repaircmd "github.com/securacore/codectx/cmds/repair"
	searchcmd "github.com/securacore/codectx/cmds/search"
	selfcmd "github.com/securacore/codectx/cmds/self"
	sessioncmd "github.com/securacore/codectx/cmds/session"
	updatecmd "github.com/securacore/codectx/cmds/update"
	usagecmd "github.com/securacore/codectx/cmds/usage"
	versioncmd "github.com/securacore/codectx/cmds/version"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "codectx",
		Usage: "Documentation compiler for AI-driven development",
		Commands: []*cli.Command{
			addcmd.Command,
			compilecmd.Command,
			generatecmd.Command,
			historycmd.Command,
			initcmd.Command,
			installcmd.Command,
			linkcmd.Command,
			newcmd.Command,
			plancmd.Command,
			publishcmd.Command,
			querycmd.Command,
			removecmd.Command,
			repaircmd.Command,
			searchcmd.Command,
			selfcmd.Command,
			sessioncmd.Command,
			updatecmd.Command,
			usagecmd.Command,
			versioncmd.Command,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
