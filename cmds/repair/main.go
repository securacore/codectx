// Package repair implements the `codectx repair` command which runs scaffold
// maintenance unconditionally — recreating missing directories, restoring
// missing system default files, managing .gitkeep files, and ensuring the
// .gitignore is up to date.
//
// Usage:
//
//	codectx repair
package repair

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/scaffold"
	"github.com/securacore/codectx/core/tui"
	"github.com/securacore/codectx/core/usage"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx repair`.
var Command = &cli.Command{
	Name:  "repair",
	Usage: "Repair project scaffold structure",
	Description: `Recreates missing directories, restores missing system default files,
manages .gitkeep files in content directories, and ensures the .gitignore
is up to date.

This command always runs regardless of the scaffold_maintenance preference.

Examples:
  codectx repair`,
	Action: run,
}

func run(_ context.Context, _ *cli.Command) error {
	// --- Step 1: Discover project ---
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// --- Step 2: Run scaffold maintenance ---
	var result *scaffold.MaintainResult
	var maintainErr error

	if err := shared.RunWithSpinner("Repairing scaffold...", func() {
		result, maintainErr = scaffold.Maintain(projectDir, cfg)
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if maintainErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Repair failed",
			Detail: []string{maintainErr.Error()},
		}.Render())
		return fmt.Errorf("repair failed: %w", maintainErr)
	}

	// --- Step 3: Ensure gitignore ---
	if err := project.EnsureGitignore(projectDir, cfg.Root); err != nil {
		fmt.Print(tui.WarnMsg{
			Title:  "Gitignore update failed",
			Detail: []string{err.Error()},
		}.Render())
	}

	// --- Step 4: Ensure usage files exist ---
	localUsage := usage.LocalPath(projectDir, cfg)
	if initErr := usage.InitLocalFile(localUsage); initErr != nil {
		fmt.Print(tui.WarnMsg{
			Title:  "Local usage file creation failed",
			Detail: []string{initErr.Error()},
		}.Render())
	}
	globalUsage := usage.GlobalPath(projectDir, cfg)
	if initErr := usage.InitGlobalFile(globalUsage, cfg.Name); initErr != nil {
		fmt.Print(tui.WarnMsg{
			Title:  "Global usage file creation failed",
			Detail: []string{initErr.Error()},
		}.Render())
	}

	// --- Step 5: Display summary ---
	if !result.HasActions() {
		fmt.Printf("\n%s %s\n\n",
			tui.Success(),
			tui.StyleBold.Render("Scaffold is intact, no repairs needed"),
		)
		return nil
	}

	fmt.Printf("\n%s %s\n\n",
		tui.Success(),
		tui.StyleBold.Render("Scaffold repaired"),
	)
	for _, item := range []struct {
		label string
		count int
	}{
		{"Directories restored", result.DirsCreated},
		{"System files restored", result.FilesRestored},
		{".gitkeep added", result.GitkeepsAdded},
		{".gitkeep removed", result.GitkeepsRemoved},
	} {
		if item.count > 0 {
			fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue(item.label, tui.FormatNumber(item.count)))
		}
	}
	fmt.Println()

	return nil
}
