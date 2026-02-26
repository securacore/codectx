package cmdx

import (
	"context"
	"fmt"
	"os"

	"github.com/securacore/codectx/core/cmdx"
	"github.com/urfave/cli/v3"
)

var decodeCommand = &cli.Command{
	Name:      "decode",
	Usage:     "Decode CMDX back to Markdown",
	ArgsUsage: "[input.cmdx]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Write output to file instead of stdout",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		return runDecode(c)
	},
}

func runDecode(c *cli.Command) error {
	input, err := readInput(c)
	if err != nil {
		return err
	}

	decoded, err := cmdx.Decode(input)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	outPath := c.String("output")
	if outPath != "" {
		return os.WriteFile(outPath, decoded, 0644)
	}
	_, err = os.Stdout.Write(decoded)
	return err
}
