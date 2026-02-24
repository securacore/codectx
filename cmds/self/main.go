package self

import "github.com/urfave/cli/v3"

// Command is the parent command for CLI self-management operations.
var Command = &cli.Command{
	Name:  "self",
	Usage: "Manage the codectx CLI",
	Commands: []*cli.Command{
		upgradeCommand,
	},
}
