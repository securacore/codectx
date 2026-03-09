// Package version implements the `codectx version` command.
package version

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// Version is set at build time via ldflags by goreleaser.
var Version = "dev"

// Command is the CLI definition for `codectx version`.
var Command = &cli.Command{
	Name:  "version",
	Usage: "Print the codectx version",
	Action: func(_ context.Context, _ *cli.Command) error {
		fmt.Println(Version)
		return nil
	},
}
