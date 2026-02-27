package self

import (
	"context"
	"fmt"
	"strings"

	"github.com/securacore/codectx/cmds/version"
	"github.com/securacore/codectx/core/update"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

var upgradeCommand = &cli.Command{
	Name:  "upgrade",
	Usage: "Upgrade codectx to the latest version",
	Action: func(ctx context.Context, c *cli.Command) error {
		return runUpgrade(version.Version, update.FetchLatest, update.Upgrade)
	},
}

// runUpgrade checks for the latest version and upgrades the binary.
// fetchLatest and upgrade are injected for testability.
func runUpgrade(current string, fetchLatest func() (string, error), upgrade func(string) error) error {
	ui.Blank()
	if current == "dev" {
		return fmt.Errorf("cannot upgrade a dev build — install from a release first")
	}

	current = strings.TrimPrefix(current, "v")

	// Fetch latest version.
	var tag string
	var fetchErr error
	ui.Spin("Checking for updates...", func() {
		tag, fetchErr = fetchLatest()
	})
	if fetchErr != nil {
		return fmt.Errorf("check latest version: %w", fetchErr)
	}

	latest := strings.TrimPrefix(tag, "v")
	if latest == current {
		ui.Done(fmt.Sprintf("Already on latest version (v%s)", current))
		return nil
	}

	// Download, verify, extract, replace.
	err := ui.SpinErr(fmt.Sprintf("Downloading v%s...", latest), func() error {
		return upgrade(tag)
	})
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	ui.Blank()
	ui.Done(fmt.Sprintf("Updated codectx: v%s -> v%s", current, latest))
	ui.Blank()

	return nil
}
