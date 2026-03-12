// Package version implements the `codectx version` command.
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
	Usage: "Print the codectx version",
	Action: func(_ context.Context, _ *cli.Command) error {
		fmt.Println(project.Version)
		return nil
	},
}
