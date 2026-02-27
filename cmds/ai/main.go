package ai

import (
	"github.com/securacore/codectx/cmds/ide"
	"github.com/urfave/cli/v3"
)

// Command is the parent command for AI tool integration management.
var Command = &cli.Command{
	Name:  "ai",
	Usage: "Manage AI tool integration",
	Commands: []*cli.Command{
		ide.Command,
		setupCommand,
		statusCommand,
	},
}
