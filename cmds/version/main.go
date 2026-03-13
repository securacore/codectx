// Package version implements the `codectx version` command which prints
// the codectx version, and the `codectx version bump` subcommand for
// incrementing project versions.
package version

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/core/project"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx version`.
var Command = &cli.Command{
	Name:  "version",
	Usage: "Print the codectx version or manage project version",
	Description: `Without arguments, prints the codectx CLI version.

Subcommands:
  bump    Bump the project/package version (major, minor, or patch)`,
	Commands: []*cli.Command{
		bumpCommand,
	},
	Action: func(_ context.Context, _ *cli.Command) error {
		fmt.Println(project.Version)
		return nil
	},
}
