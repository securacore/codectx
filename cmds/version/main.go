package version

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var Command = cli.Command{
	Name: "version",
	Action: func(ctx context.Context, c *cli.Command) error {
		fmt.Println("VERSION")
		return nil
	},
}
