package cmdx

import "github.com/urfave/cli/v3"

// Command is the parent command for CMDX compression operations.
var Command = &cli.Command{
	Name:     "cmdx",
	Usage:    "CMDX compression codec for Markdown",
	Category: "Development Tools",
	Commands: []*cli.Command{
		encodeCommand,
		decodeCommand,
		statsCommand,
		validateCommand,
		roundtripCommand,
	},
}
