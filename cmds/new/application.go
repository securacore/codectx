package new

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var applicationCommand = &cli.Command{
	Name:      "application",
	Usage:     "Create a new application document",
	ArgsUsage: "<name>",
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()
		if args.Len() == 0 {
			return fmt.Errorf("missing required argument: name")
		}
		return scaffold(kindApplication, args.First())
	},
}
