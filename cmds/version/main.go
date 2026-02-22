package cmd_version

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var Command = &cli.Command{
	Name:  "version",
	Usage: "CLI version",
	Action: func(ctx context.Context, c *cli.Command) error {
		fmt.Println("VERSION")
		return nil
	},
}
