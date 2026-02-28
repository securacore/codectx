package new

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var promptCommand = &cli.Command{
	Name:      "prompt",
	Usage:     "Create a new prompt document",
	ArgsUsage: "<name>",
	Flags:     []cli.Flag{packageFlag},
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()
		if args.Len() == 0 {
			return fmt.Errorf("missing required argument: name")
		}
		return scaffold(kindPrompt, args.First(), c.Bool("package"))
	},
}
