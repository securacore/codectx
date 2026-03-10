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
//  4. Detect AI tools and API keys
//  5. Confirm or override the recommended model
//  6. Scaffold the project with a spinner
//  7. Display a formatted summary
package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/core/detect"
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
		&cli.StringFlag{
			Name:  "model",
			Usage: "AI model for compilation (auto-detected if omitted)",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Usage:   "Accept all defaults without prompting",
			Aliases: []string{"y"},
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
	model := cmd.String("model")
	autoYes := cmd.Bool("yes")

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
		return fmt.Errorf("directory not writable: %s", targetDir)
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
		return fmt.Errorf("already initialized in %s", targetDir)
	}

	// --- Step 3: Handle nested project warning ---
	if check.NestedProject {
		if !interactive {
			return fmt.Errorf("cannot init inside existing project at %s (use --root or a different directory)", check.NestedProjectPath)
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
			return fmt.Errorf("initialization canceled")
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
			return fmt.Errorf("documentation root conflict: directory already has content (use --root to specify an alternative)")
		}

		effectiveRoot := root
		if effectiveRoot == "" {
			effectiveRoot = "docs"
		}

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

	// --- Step 7: AI tool detection ---
	var detection detect.Result

	if interactive {
		err = spinner.New().
			Title("Scanning for AI tools...").
			ActionWithErr(func(_ context.Context) error {
				detection = detect.Scan()
				return nil
			}).
			Run()
		if err != nil {
			return err
		}
	} else {
		detection = detect.Scan()
	}

	// Display detection results.
	if detection.HasAnything() {
		fmt.Println()
		if len(detection.Tools) > 0 {
			fmt.Printf("%s%s\n", tui.Indent(1), tui.StyleMuted.Render("Detected tools:"))
			for _, tool := range detection.Tools {
				version := ""
				if tool.Version != "" {
					version = tool.Version
				}
				fmt.Printf("%s%s\n", tui.Indent(2), tui.DetectedTool(tool.Name, version))
			}
		}
		if len(detection.Providers) > 0 {
			fmt.Printf("%s%s\n", tui.Indent(1), tui.StyleMuted.Render("API providers:"))
			for _, p := range detection.Providers {
				fmt.Printf("%s%s %s\n", tui.Indent(2),
					tui.Success(),
					tui.StyleBold.Render(p.Name))
			}
		}
		fmt.Println()
	}

	// --- Step 8: Model selection ---
	effectiveModel := model
	effectiveEncoding := detection.RecommendedEncoding
	if effectiveModel == "" {
		effectiveModel = detection.RecommendedModel
	}

	if interactive && model == "" {
		// Build model options from detection results + the recommended default.
		type modelChoice struct {
			model    string
			encoding string
		}

		// Collect unique models from providers.
		seen := map[string]bool{effectiveModel: true}
		options := []huh.Option[modelChoice]{
			huh.NewOption(
				fmt.Sprintf("%s (recommended)", effectiveModel),
				modelChoice{model: effectiveModel, encoding: detection.RecommendedEncoding},
			),
		}

		for _, p := range detection.Providers {
			if !seen[p.DefaultModel] {
				seen[p.DefaultModel] = true
				enc := detect.EncodingForModel(p.DefaultModel)
				options = append(options, huh.NewOption(
					fmt.Sprintf("%s (%s)", p.DefaultModel, p.Name),
					modelChoice{model: p.DefaultModel, encoding: enc},
				))
			}
		}

		// Only show select if there are multiple choices.
		if len(options) > 1 {
			var selected modelChoice
			if err := huh.NewSelect[modelChoice]().
				Title("Compilation model:").
				Options(options...).
				Value(&selected).
				Run(); err != nil {
				return err
			}
			effectiveModel = selected.model
			effectiveEncoding = selected.encoding
		}
	}

	// --- Step 9: Scaffold with spinner ---
	opts := scaffold.Options{
		ProjectDir: targetDir,
		Root:       root,
		Name:       name,
		GitInit:    needGitInit,
		Model:      effectiveModel,
		Encoding:   effectiveEncoding,
	}

	var result *scaffold.Result
	if interactive {
		err = spinner.New().
			Title("Creating project structure...").
			ActionWithErr(func(_ context.Context) error {
				var initErr error
				result, initErr = scaffold.Init(opts)
				return initErr
			}).
			Run()
		if err != nil {
			return err
		}
	} else {
		result, err = scaffold.Init(opts)
		if err != nil {
			return err
		}
	}

	// --- Step 10: Summary ---
	effectiveRoot := root
	if effectiveRoot == "" {
		effectiveRoot = "docs"
	}

	tree := buildSummaryTree(effectiveRoot, result)

	nextSteps := []string{
		fmt.Sprintf("Add foundation documents to %s",
			tui.StyleCommand.Render(effectiveRoot+"/foundation/")),
		fmt.Sprintf("Add topic documentation to %s",
			tui.StyleCommand.Render(effectiveRoot+"/topics/")),
		fmt.Sprintf("Run %s to build the documentation index",
			tui.StyleCommand.Render("codectx compile")),
	}

	fmt.Print(tui.InitSummary(name, tree, nextSteps))

	// Print key configuration values.
	fmt.Printf("%s%s\n", tui.Indent(1),
		tui.KeyValue("Model", effectiveModel))
	if created {
		fmt.Printf("%s%s\n", tui.Indent(1),
			tui.KeyValue("Directory", targetDir))
	}
	if result.GitInitialized {
		fmt.Printf("%s%s\n", tui.Indent(1),
			tui.KeyValue("Git", "initialized"))
	}
	fmt.Println()

	return nil
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
			return "", false, fmt.Errorf("%s exists but is not a directory", target)
		}
		abs, err := filepath.Abs(target)
		if err != nil {
			return "", false, err
		}
		return abs, false, nil
	}

	// Create the directory.
	if err := os.MkdirAll(target, 0755); err != nil {
		return "", false, fmt.Errorf("creating directory %s: %w", target, err)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", false, err
	}
	return abs, true, nil
}

// buildSummaryTree creates the tree structure for the init summary display.
func buildSummaryTree(root string, result *scaffold.Result) []tui.TreeNode {
	return []tui.TreeNode{
		{
			Name: "codectx.yml",
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
