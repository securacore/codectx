// Package plan implements the `codectx plan` command group which provides
// plan state tracking for multi-step AI workflows.
//
// Subcommands:
//   - codectx plan status [plan-name] — Report plan state without loading context
//   - codectx plan resume [plan-name] — Resume a plan with context reconstruction
//
// Plans are stored as plan.yml files in docs/plans/<plan-name>/ directories.
// They track dependencies with content hashes for drift detection and store
// per-step chunk references for instant context replay.
package plan

import (
	"context"
	"fmt"

	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/manifest"
	coreplan "github.com/securacore/codectx/core/plan"
	"github.com/securacore/codectx/core/project"
	corequery "github.com/securacore/codectx/core/query"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx plan`.
var Command = &cli.Command{
	Name:  "plan",
	Usage: "Plan state tracking for multi-step workflows",
	Description: `Manages plan state for multi-step AI workflows.
Plans are stored in docs/plans/<name>/plan.yml and track dependencies,
steps, queries, and chunk references for context reconstruction.`,
	Commands: []*cli.Command{
		statusCommand,
		resumeCommand,
	},
}

// statusCommand reports the current state of a plan.
var statusCommand = &cli.Command{
	Name:      "status",
	Usage:     "Report plan state without loading context",
	ArgsUsage: "[plan-name]",
	Description: `Report the current state of a plan. Shows the current step,
completion percentage, blockers, dependency hash status, and stored queries.

If only one plan exists, the name can be omitted.

Examples:
  codectx plan status auth-migration
  codectx plan status`,
	Action: runStatus,
}

// resumeCommand resumes a plan with context reconstruction.
var resumeCommand = &cli.Command{
	Name:      "resume",
	Usage:     "Resume a plan with context reconstruction",
	ArgsUsage: "[plan-name]",
	Description: `Resume a plan by reconstructing its context. Checks dependency
hashes against current compiled state. If all hashes match, replays the
current step's chunks for instant context reconstruction. If hashes changed,
reports which dependencies drifted and provides stored queries for re-search.

If only one plan exists, the name can be omitted.

Examples:
  codectx plan resume auth-migration
  codectx plan resume`,
	Action: runResume,
}

// runStatus implements the plan status subcommand.
func runStatus(_ context.Context, cmd *cli.Command) error {
	// Discover project.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	rootDir := project.RootDir(projectDir, cfg)
	planName := ""
	if cmd.NArg() > 0 {
		planName = cmd.Args().First()
	}

	// Find, load, and check the plan inside a spinner.
	var p *coreplan.Plan
	var check *coreplan.CheckResult
	var statusErr error
	var hashWarn string

	if err = shared.RunWithSpinner("Loading plan status...", func() {
		_, planPath, findErr := coreplan.FindPlan(rootDir, planName)
		if findErr != nil {
			statusErr = fmt.Errorf("finding plan: %w", findErr)
			return
		}

		var loadErr error
		p, loadErr = coreplan.Load(planPath)
		if loadErr != nil {
			statusErr = fmt.Errorf("loading plan: %w", loadErr)
			return
		}

		if len(p.Dependencies) > 0 {
			compiledDir := corequery.CompiledDir(projectDir, cfg)
			hashesPath := manifest.HashesPath(compiledDir)
			hashes, hashErr := manifest.LoadHashes(hashesPath)
			if hashErr != nil {
				hashWarn = hashErr.Error()
			} else {
				check = coreplan.CheckDependencies(p.Dependencies, hashes)
			}
		}
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if statusErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Plan not found",
			Detail: []string{statusErr.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "List plans in:", Command: fmt.Sprintf("ls %s/plans/", rootDir)},
			},
		}.Render())
		return statusErr
	}

	if hashWarn != "" {
		fmt.Print(tui.WarnMsg{
			Title: "Could not load compiled hashes for dependency checking",
			Detail: []string{
				hashWarn,
				"Dependency status will not be shown.",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Compile the project first:", Command: "codectx compile"},
			},
		}.Render())
	}

	fmt.Print(coreplan.FormatStatus(p, check))
	return nil
}

// runResume implements the plan resume subcommand.
func runResume(_ context.Context, cmd *cli.Command) error {
	// Discover project.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	rootDir := project.RootDir(projectDir, cfg)
	planName := ""
	if cmd.NArg() > 0 {
		planName = cmd.Args().First()
	}

	// Find the plan.
	_, planPath, err := coreplan.FindPlan(rootDir, planName)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Plan not found",
			Detail: []string{err.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "List plans in:", Command: fmt.Sprintf("ls %s/plans/", rootDir)},
			},
		}.Render())
		return fmt.Errorf("finding plan: %w", err)
	}

	// Resolve compiled directory and encoding.
	compiledDir := corequery.CompiledDir(projectDir, cfg)
	encoding := project.ResolveEncoding(projectDir, cfg)

	// Run resume inside a spinner.
	var result *coreplan.ResumeResult
	var resumeErr error

	if err = shared.RunWithSpinner("Resuming plan...", func() {
		result, resumeErr = coreplan.Resume(planPath, compiledDir, encoding)
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if resumeErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Resume failed",
			Detail: []string{resumeErr.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Check plan file:", Command: fmt.Sprintf("cat %s", planPath)},
				{Text: "Compile documentation first:", Command: "codectx compile"},
			},
		}.Render())
		return fmt.Errorf("resume failed: %w", resumeErr)
	}

	fmt.Print(result.Output)
	return nil
}
