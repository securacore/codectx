package new

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var foundationCommand = &cli.Command{
	Name:      "foundation",
	Usage:     "Create a new foundation document",
	ArgsUsage: "<name>",
	Flags:     []cli.Flag{packageFlag},
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()
		if args.Len() == 0 {
			return fmt.Errorf("missing required argument: name")
		}
		return scaffold(kindFoundation, args.First(), c.Bool("package"))
	},
}
