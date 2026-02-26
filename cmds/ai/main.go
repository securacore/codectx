package ai

import "github.com/urfave/cli/v3"

// Command is the parent command for AI tool integration management.
var Command = &cli.Command{
	Name:  "ai",
	Usage: "Manage AI tool integration",
	Commands: []*cli.Command{
		setupCommand,
		statusCommand,
	},
}
