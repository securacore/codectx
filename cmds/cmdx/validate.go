package cmdx

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/core/cmdx"
	"github.com/securacore/codectx/ui"
	"github.com/urfave/cli/v3"
)

var validateCommand = &cli.Command{
	Name:      "validate",
	Usage:     "Validate a CMDX file format",
	ArgsUsage: "[input.cmdx]",
	Action: func(ctx context.Context, c *cli.Command) error {
		return runValidate(c)
	},
}

func runValidate(c *cli.Command) error {
	input, err := readInput(c)
	if err != nil {
		return err
	}

	_, err = cmdx.Parse(input)
	if err != nil {
		return fmt.Errorf("invalid CMDX: %w", err)
	}

	ui.Done("Valid CMDX")
	return nil
}
