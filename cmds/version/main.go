package version

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var Version = "dev"

var Command = &cli.Command{
	Name:  "version",
	Usage: "Display the CLI version",
	Action: func(ctx context.Context, c *cli.Command) error {
		fmt.Println(Version)
		return nil
	},
}
