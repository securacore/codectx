package version

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/urfave/cli/v3"
)

var Version = "dev"

func init() {
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		Version = v
	}
}

var Command = &cli.Command{
	Name:  "version",
	Usage: "Display the CLI version",
	Action: func(ctx context.Context, c *cli.Command) error {
		fmt.Println(Version)
		return nil
	},
}
