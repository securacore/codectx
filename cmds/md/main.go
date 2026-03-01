package md

import "github.com/urfave/cli/v3"

// Command is the parent command for markdown compression operations.
var Command = &cli.Command{
	Name:     "md",
	Usage:    "Markdown compression codec",
	Category: "Development Tools",
	Commands: []*cli.Command{
		encodeCommand,
		statsCommand,
		roundtripCommand,
	},
}
