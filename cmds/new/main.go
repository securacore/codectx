// Package new implements the `codectx new` command which scaffolds new
// codectx resources. Currently supports `codectx new package` for creating
// documentation package repositories.
package new

import (
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx new`.
var Command = &cli.Command{
	Name:  "new",
	Usage: "Create a new codectx resource",
	Description: `Scaffolds new codectx resources. Use a subcommand to specify what to create.

Available resources:
  package    Create a documentation package repository`,
	Commands: []*cli.Command{
		packageCommand,
	},
}
