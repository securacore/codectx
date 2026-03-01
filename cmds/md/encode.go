package md

import (
	"context"
	"fmt"
	"os"

	coremd "github.com/securacore/codectx/core/md"
	"github.com/urfave/cli/v3"
)

var encodeCommand = &cli.Command{
	Name:      "encode",
	Usage:     "Encode Markdown to compact, normalized Markdown",
	ArgsUsage: "[input.md]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Write output to file instead of stdout",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		return runEncode(c)
	},
}

func runEncode(c *cli.Command) error {
	input, err := readInput(c)
	if err != nil {
		return err
	}

	encoded, err := coremd.Encode(input)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	return writeOutput(c, encoded)
}

// readInput reads from the file argument or stdin.
func readInput(c *cli.Command) ([]byte, error) {
	args := c.Args()
	if args.Len() > 0 {
		return os.ReadFile(args.First())
	}
	return os.ReadFile("/dev/stdin")
}

// writeOutput writes to the -o flag file or stdout.
func writeOutput(c *cli.Command, data []byte) error {
	outPath := c.String("output")
	if outPath != "" {
		return os.WriteFile(outPath, data, 0644)
	}
	_, err := os.Stdout.Write(data)
	return err
}
