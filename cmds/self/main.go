// Package self implements the `codectx self` command group for managing
// the codectx installation itself.
//
// Subcommands:
//   - codectx self update  — Update codectx to the latest version
package self

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/selfupdate"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx self`.
var Command = &cli.Command{
	Name:  "self",
	Usage: "Manage the codectx installation",
	Commands: []*cli.Command{
		updateCommand,
	},
}

// updateCommand checks for and applies updates to the codectx binary.
var updateCommand = &cli.Command{
	Name:  "update",
	Usage: "Update codectx to the latest version",
	Description: `Checks GitHub Releases for the latest version of codectx,
downloads the appropriate binary for your platform, verifies
its SHA-256 checksum, and replaces the current binary.

Examples:
  codectx self update`,
	Action: runUpdate,
}

func runUpdate(ctx context.Context, _ *cli.Command) error {
	current := project.Version
	client := &http.Client{}

	// Step 1: Check for the latest version.
	var latest string
	var checkErr error

	if err := shared.RunWithSpinner("Checking for updates...", func() {
		latest, checkErr = selfupdate.CheckLatest(ctx, client)
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if checkErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to check for updates",
			Detail: []string{checkErr.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Check your network connection and try again"},
				{Text: "Set GITHUB_TOKEN for higher rate limits"},
			},
		}.Render())
		return checkErr
	}

	// Step 2: Compare versions.
	if !selfupdate.NeedsUpdate(current, latest) {
		fmt.Printf("\n%s %s\n\n",
			tui.Success(),
			tui.StyleBold.Render(fmt.Sprintf("Already up to date (v%s)", current)),
		)
		return nil
	}

	// Step 3: Download and verify.
	var binaryPath, tempDir string
	var dlErr error

	if err := shared.RunWithSpinner(
		fmt.Sprintf("Downloading v%s...", latest),
		func() {
			binaryPath, tempDir, dlErr = selfupdate.Download(ctx, client, latest)
		},
	); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if dlErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Download failed",
			Detail: []string{dlErr.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Try again later or install manually:", Command: "curl -fsSL https://raw.githubusercontent.com/securacore/codectx/main/bin/install | sh"},
			},
		}.Render())
		return dlErr
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Step 4: Replace the binary.
	execPath, err := os.Executable()
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Cannot determine binary location",
			Detail: []string{err.Error()},
		}.Render())
		return fmt.Errorf("finding executable: %w", err)
	}

	if err := selfupdate.Replace(execPath, binaryPath); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to replace binary",
			Detail: []string{err.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Check file permissions on your codectx binary"},
				{Text: "Install manually:", Command: "curl -fsSL https://raw.githubusercontent.com/securacore/codectx/main/bin/install | sh"},
			},
		}.Render())
		return fmt.Errorf("replacing binary: %w", err)
	}

	// Step 5: Display result.
	fmt.Printf("\n%s %s\n\n",
		tui.Success(),
		tui.StyleBold.Render("Updated codectx"),
	)
	fmt.Printf("%s%s\n",
		tui.Indent(1),
		tui.KeyValue("Version", fmt.Sprintf("%s %s v%s",
			versionDisplay(current),
			tui.Arrow(),
			tui.StyleBold.Render(latest),
		)),
	)
	fmt.Println()

	return nil
}

// versionDisplay formats the current version for display.
func versionDisplay(v string) string {
	if v == "dev" || v == "" {
		return v
	}
	return "v" + v
}
