package new

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var planCommand = &cli.Command{
	Name:      "plan",
	Usage:     "Create a new plan document",
	ArgsUsage: "<name>",
	Flags:     []cli.Flag{packageFlag},
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()
		if args.Len() == 0 {
			return fmt.Errorf("missing required argument: name")
		}
		return scaffold(kindPlan, args.First(), c.Bool("package"))
	},
}
