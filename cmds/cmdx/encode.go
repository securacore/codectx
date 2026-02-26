package cmdx

import (
	"context"
	"fmt"
	"os"

	"github.com/securacore/codectx/core/cmdx"
	"github.com/urfave/cli/v3"
)

var encodeCommand = &cli.Command{
	Name:      "encode",
	Usage:     "Encode Markdown to CMDX",
	ArgsUsage: "[input.md]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Write output to file instead of stdout",
		},
		&cli.IntFlag{
			Name:  "dict-max",
			Usage: "Maximum dictionary entries",
			Value: 50,
		},
		&cli.IntFlag{
			Name:  "min-freq",
			Usage: "Minimum frequency for dictionary candidates",
			Value: 2,
		},
		&cli.IntFlag{
			Name:  "min-len",
			Usage: "Minimum string length for dictionary candidates",
			Value: 10,
		},
		&cli.BoolFlag{
			Name:  "no-domain",
			Usage: "Disable domain-specific block detection",
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

	opts := cmdx.DefaultEncoderOptions()
	opts.MaxDictEntries = int(c.Int("dict-max"))
	opts.MinFrequency = int(c.Int("min-freq"))
	opts.MinStringLength = int(c.Int("min-len"))
	opts.EnableDomainBlocks = !c.Bool("no-domain")

	encoded, err := cmdx.Encode(input, opts)
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
