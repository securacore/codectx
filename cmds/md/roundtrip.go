package md

import (
	"context"
	"fmt"

	coremd "github.com/securacore/codectx/core/md"
	"github.com/securacore/codectx/ui"
	"github.com/urfave/cli/v3"
)

var roundtripCommand = &cli.Command{
	Name:      "roundtrip",
	Usage:     "Encode and compare ASTs — exit 0 if lossless",
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

	encoded, err := coremd.Encode(input)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	equal, diff, err := coremd.CompareASTs(input, encoded)
	if err != nil {
		return fmt.Errorf("compare: %w", err)
	}

	if !equal {
		return fmt.Errorf("round-trip mismatch:\n%s", diff)
	}

	ui.Done("Round-trip OK: lossless")
	return nil
}
