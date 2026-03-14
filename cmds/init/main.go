// Package init implements the `codectx init` command which scaffolds
// a new codectx documentation project.
//
// The command supports three invocation forms:
//   - codectx init        — initialize in the current working directory
//   - codectx init .      — same as above
//   - codectx init foo    — create foo/ directory and initialize inside it
//
// The TUI flow:
//  1. Resolve target directory from positional args
//  2. Run pre-scaffold checks (already initialized, nested, git, root conflict)
//  3. Prompt for missing information (project name, git init, root conflict)
//  4. Scaffold the project with a spinner
//  5. Display summary and prompt for AI tool entry points
//  6. Auto-compile if enabled
package init

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/scaffold"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx init`.
var Command = &cli.Command{
	Name:      "init",
	Usage:     "Initialize a new codectx documentation project",
	ArgsUsage: "[directory]",
	Description: `Creates the codectx directory structure, default configuration files,
and system documentation.

With no arguments or ".", initializes in the current directory.
With a directory name, creates the directory and initializes inside it.`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "Documentation root directory name (default: docs)",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Project name (default: directory name)",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Usage:   "Accept all defaults without prompting",
			Aliases: []string{"y"},
		},
		&cli.BoolFlag{
			Name:  "compile",
			Usage: "Force initial compilation after project setup",
		},
		&cli.BoolFlag{
			Name:  "no-compile",
			Usage: "Skip initial compilation after project setup",
		},
	},
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	// --- Step 1: Resolve target directory ---
	targetDir, created, err := resolveTarget(cmd.Args().Slice())
	if err != nil {
		return err
	}

	root := cmd.String("root")
	name := cmd.String("name")
	autoYes := cmd.Bool("yes")
	forceCompile := cmd.IsSet("compile") && cmd.Bool("compile")
	skipCompile := cmd.IsSet("no-compile") && cmd.Bool("no-compile")

	// Detect if we have an interactive terminal.
	interactive := term.IsTerminal(os.Stdin.Fd()) && !autoYes

	// --- Step 2: Pre-scaffold checks ---
	check, err := scaffold.Check(targetDir, root)
	if err != nil {
		return err
	}

	if !check.Writable {
		fmt.Print(tui.ErrorMsg{
			Title: "Directory is not writable",
			Detail: []string{
				fmt.Sprintf("Cannot write to %s", tui.StylePath.Render(targetDir)),
			},
			Suggestions: []tui.Suggestion{
				{Text: "Check directory permissions"},
			},
		}.Render())
		return fmt.Errorf("directory not writable: %q", targetDir)
	}

	if check.AlreadyInitialized {
		fmt.Print(tui.ErrorMsg{
			Title: "Already initialized",
			Detail: []string{
				fmt.Sprintf("A codectx project already exists in %s", tui.StylePath.Render(targetDir)),
			},
			Suggestions: []tui.Suggestion{
				{Text: "To reinitialize, remove codectx.yml first"},
			},
		}.Render())
		return fmt.Errorf("already initialized in %q", targetDir)
	}

	// --- Step 3: Handle nested project warning ---
	if check.NestedProject {
		if !interactive {
			return fmt.Errorf("cannot init inside existing project at %q (use --root or a different directory)", check.NestedProjectPath)
		}

		fmt.Print(tui.WarnMsg{
			Title: "Nested project detected",
			Detail: []string{
				fmt.Sprintf("A codectx project exists in a parent directory: %s",
					tui.StylePath.Render(check.NestedProjectPath)),
				"Initializing here will create a nested project.",
			},
		}.Render())

		var proceed bool
		if err := huh.NewConfirm().
			Title("Continue with nested project?").
			Value(&proceed).
			Run(); err != nil {
			return err
		}
		if !proceed {
			return errors.New("initialization canceled")
		}
	}

	// --- Step 4: Handle missing git ---
	needGitInit := false
	if !check.HasGit {
		if !interactive {
			needGitInit = true
		} else {
			if err := huh.NewConfirm().
				Title("No git repository found. Initialize one?").
				Value(&needGitInit).
				Run(); err != nil {
				return err
			}
		}
	}

	// --- Step 5: Handle root conflict ---
	if check.RootConflict {
		if !interactive {
			return errors.New("documentation root conflict: directory already has content (use --root to specify an alternative)")
		}

		effectiveRoot := project.ResolveRoot(root)

		fmt.Print(tui.WarnMsg{
			Title: "Documentation root conflict",
			Detail: []string{
				fmt.Sprintf("The %s/ directory already exists and contains files.",
					tui.StylePath.Render(effectiveRoot)),
			},
		}.Render())

		var choice string
		if err := huh.NewSelect[string]().
			Title("Choose a documentation root:").
			Options(
				huh.NewOption("ai-docs/", "ai-docs"),
				huh.NewOption(".codectx-docs/", ".codectx-docs"),
				huh.NewOption("Enter custom path", "custom"),
			).
			Value(&choice).
			Run(); err != nil {
			return err
		}

		if choice == "custom" {
			if err := huh.NewInput().
				Title("Documentation root directory name:").
				Value(&choice).
				Run(); err != nil {
				return err
			}
		}

		root = choice

		// Re-check for conflict with the new root.
		recheck, err := scaffold.Check(targetDir, root)
		if err != nil {
			return err
		}
		if recheck.RootConflict {
			return fmt.Errorf("the chosen root %q also has existing content", root)
		}
	}

	// --- Step 6: Resolve project name ---
	if name == "" {
		defaultName := filepath.Base(targetDir)
		if !interactive {
			name = defaultName
		} else {
			name = defaultName
			if err := huh.NewInput().
				Title("Project name:").
				Value(&name).
				Run(); err != nil {
				return err
			}
		}
	}

	// --- Step 7: Scaffold with spinner ---
	opts := scaffold.Options{
		ProjectDir: targetDir,
		Root:       root,
		Name:       name,
		GitInit:    needGitInit,
	}

	var result *scaffold.Result
	var initErr error
	if err = shared.RunWithSpinner("Creating project structure...", func() {
		result, initErr = scaffold.Init(opts)
	}); err != nil {
		return err
	}
	if initErr != nil {
		return initErr
	}

	// --- Step 8: Determine auto-compile intent ---
	willAutoCompile := shouldInitAutoCompile(targetDir, root, forceCompile, skipCompile)

	// --- Step 9: Summary ---
	effectiveRoot := project.ResolveRoot(root)

	tree := buildSummaryTree(effectiveRoot)

	nextSteps := []string{
		fmt.Sprintf("Add foundation documents to %s",
			tui.StyleCommand.Render(effectiveRoot+"/foundation/")),
		fmt.Sprintf("Add topic documentation to %s",
			tui.StyleCommand.Render(effectiveRoot+"/topics/")),
	}
	if !willAutoCompile {
		nextSteps = append(nextSteps,
			fmt.Sprintf("Run %s to build the documentation index",
				tui.StyleCommand.Render("codectx compile")),
		)
	}

	fmt.Print(tui.InitSummary(name, tree, nextSteps))

	if created {
		fmt.Printf("%s%s\n", tui.Indent(1),
			tui.KeyValue("Directory", targetDir))
	}
	if result.GitInitialized {
		fmt.Printf("%s%s\n", tui.Indent(1),
			tui.KeyValue("Git", "initialized"))
	}
	fmt.Println()

	// --- Step 10: Link prompt ---
	if interactive {
		if err := promptLink(targetDir); err != nil {
			// Non-fatal: init succeeded even if link was skipped.
			return nil //nolint:nilerr // link setup is optional
		}
	}

	// --- Step 11: Auto-compile ---
	if willAutoCompile {
		if err := runInitCompile(targetDir, root); err != nil {
			// Non-fatal: init succeeded even if compile failed.
			// Print the error but don't return it.
			fmt.Printf("\n%s %s\n%s %s\n\n",
				tui.StyleMuted.Render("-"),
				tui.StyleMuted.Render("Initial compilation failed (non-fatal)"),
				tui.Indent(1),
				tui.StyleMuted.Render("Run: "+tui.StyleCommand.Render("codectx compile")),
			)
			return nil //nolint:nilerr // auto-compile failure is non-fatal; init itself succeeded
		}
	}

	return nil
}

// promptLink prompts the user to set up AI tool entry point files after init.
func promptLink(projectDir string) error {
	return shared.PromptAndWriteLinks(projectDir)
}

// shouldInitAutoCompile determines whether auto-compilation should run
// after init, based on CLI flags and the auto_compile preference.
func shouldInitAutoCompile(
	projectDir, root string,
	forceCompile, skipCompile bool,
) bool {
	return shared.ShouldPostInitCompile(projectDir, root, forceCompile, skipCompile)
}

// runInitCompile loads the freshly created project configuration and runs
// the full compilation pipeline with a spinner.
func runInitCompile(projectDir, root string) error {
	return shared.RunPostInitCompile(projectDir)
}

// resolveTarget determines the target directory from positional arguments.
// Returns (absolutePath, createdNewDir, error).
func resolveTarget(args []string) (string, bool, error) {
	if len(args) == 0 || args[0] == "." {
		// Initialize in CWD.
		cwd, err := os.Getwd()
		if err != nil {
			return "", false, fmt.Errorf("getting working directory: %w", err)
		}
		return cwd, false, nil
	}

	target := args[0]

	// If the path already exists, use it.
	if info, err := os.Stat(target); err == nil {
		if !info.IsDir() {
			return "", false, fmt.Errorf("%q exists but is not a directory", target)
		}
		abs, err := filepath.Abs(target)
		if err != nil {
			return "", false, err
		}
		return abs, false, nil
	}

	// Create the directory.
	if err := os.MkdirAll(target, project.DirPerm); err != nil {
		return "", false, fmt.Errorf("creating directory %q: %w", target, err)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", false, err
	}
	return abs, true, nil
}

// buildSummaryTree creates the tree structure for the init summary display.
func buildSummaryTree(root string) []tui.TreeNode {
	return []tui.TreeNode{
		{
			Name: project.ConfigFileName,
		},
		{
			Name: root + "/",
			Children: []tui.TreeNode{
				{Name: "foundation/"},
				{Name: "topics/"},
				{Name: "plans/"},
				{Name: "prompts/"},
				{
					Name: "system/",
					Children: []tui.TreeNode{
						{Name: "foundation/"},
						{Name: "topics/"},
						{Name: "plans/"},
						{Name: "prompts/"},
					},
				},
				{
					Name: ".codectx/",
					Children: []tui.TreeNode{
						{Name: "ai.yml"},
						{Name: "preferences.yml"},
						{Name: "compiled/"},
						{Name: "packages/"},
					},
				},
			},
		},
	}
}
