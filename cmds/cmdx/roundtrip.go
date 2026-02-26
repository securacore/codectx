package cmdx

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/core/cmdx"
	"github.com/securacore/codectx/ui"
	"github.com/urfave/cli/v3"
)

var roundtripCommand = &cli.Command{
	Name:      "roundtrip",
	Usage:     "Encode, decode, and compare — exit 0 if lossless",
	ArgsUsage: "[input.md]",
	Action: func(ctx context.Context, c *cli.Command) error {
		return runRoundtrip(c)
	},
}

func runRoundtrip(c *cli.Command) error {
	input, err := readInput(c)
	if err != nil {
		return err
	}

	encoded, err := cmdx.Encode(input)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	decoded, err := cmdx.Decode(encoded)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	equal, diff, err := cmdx.CompareASTs(input, decoded)
	if err != nil {
		return fmt.Errorf("compare: %w", err)
	}

	if !equal {
		return fmt.Errorf("round-trip mismatch:\n%s", diff)
	}

	ui.Done("Round-trip OK: lossless")
	return nil
}
