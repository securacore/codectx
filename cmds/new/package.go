package new

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/scaffold"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// packageCommand is the CLI definition for `codectx new package`.
var packageCommand = &cli.Command{
	Name:      "package",
	Usage:     "Create a new documentation package repository",
	ArgsUsage: "[directory]",
	Description: `Creates a documentation package repository with:
  - package/ directory for publishable content (foundation, topics, plans, prompts)
  - docs/ directory as a full codectx authoring project
  - GitHub Actions workflow for automated releases
  - README.md with install instructions

With no arguments or ".", initializes in the current directory.
With a directory name, creates the directory and initializes inside it.`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "Package name (default: directory name)",
		},
		&cli.StringFlag{
			Name:  "author",
			Usage: "GitHub username or organization that owns this package",
		},
		&cli.StringFlag{
			Name:  "description",
			Usage: "One-line package description",
		},
		&cli.StringFlag{
			Name:  "root",
			Usage: "Documentation root directory name for authoring project (default: docs)",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Usage:   "Accept all defaults without prompting",
			Aliases: []string{"y"},
		},
		&cli.BoolFlag{
			Name:  "compile",
			Usage: "Force initial compilation after setup",
		},
		&cli.BoolFlag{
			Name:  "no-compile",
			Usage: "Skip initial compilation after setup",
		},
	},
	Action: runPackage,
}

func runPackage(_ context.Context, cmd *cli.Command) error {
	// --- Step 1: Resolve target directory ---
	targetDir, created, err := resolveTarget(cmd.Args().Slice())
	if err != nil {
		return err
	}

	name := cmd.String("name")
	author := cmd.String("author")
	description := cmd.String("description")
	root := cmd.String("root")
	autoYes := cmd.Bool("yes")
	forceCompile := cmd.IsSet("compile") && cmd.Bool("compile")
	skipCompile := cmd.IsSet("no-compile") && cmd.Bool("no-compile")

	interactive := term.IsTerminal(os.Stdin.Fd()) && !autoYes

	// --- Step 1 (cont): Warn if directory doesn't follow codectx- convention ---
	dirBase := filepath.Base(targetDir)
	if !strings.HasPrefix(dirBase, registry.RepoPrefix) {
		fmt.Print(tui.WarnMsg{
			Title: "Directory name missing codectx- prefix",
			Detail: []string{
				fmt.Sprintf("The directory %s does not follow the %s naming convention.",
					tui.StylePath.Render(dirBase),
					tui.StyleAccent.Render("codectx-<name>")),
				"Packages must be hosted in a GitHub repository named codectx-<name>",
				"to be discoverable and installable via codectx add.",
			},
		}.Render())

		if interactive {
			var proceed bool
			if err := huh.NewConfirm().
				Title("Continue anyway?").
				Value(&proceed).
				Run(); err != nil {
				return err
			}
			if !proceed {
				return errors.New("initialization canceled")
			}
		}
	}

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
			return fmt.Errorf("cannot init inside existing project at %q", check.NestedProjectPath)
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

	// --- Step 5: Resolve package name ---
	if name == "" {
		defaultName := filepath.Base(targetDir)
		if !interactive {
			name = defaultName
		} else {
			name = defaultName
			if err := huh.NewInput().
				Title("Package name:").
				Value(&name).
				Run(); err != nil {
				return err
			}
		}
	}

	// Always strip the codectx- prefix from the name — the config stores
	// the bare name (e.g., "react" not "codectx-react"), matching DepKey.Name.
	name = stripPackagePrefix(name)

	// --- Step 6: Resolve author ---
	if author == "" && interactive {
		if err := huh.NewInput().
			Title("Author (GitHub username or organization):").
			Description("Used to construct the package URL: github.com/<author>/codectx-<name>").
			Value(&author).
			Run(); err != nil {
			return err
		}
	}

	// --- Step 7: Resolve description ---
	if description == "" && interactive {
		if err := huh.NewInput().
			Title("Package description (optional):").
			Value(&description).
			Run(); err != nil {
			return err
		}
	}

	// --- Step 8: Scaffold with spinner ---
	opts := scaffold.PackageOptions{
		ProjectDir:  targetDir,
		Root:        root,
		Name:        name,
		Author:      author,
		Description: description,
		GitInit:     needGitInit,
	}

	var result *scaffold.PackageResult
	var initErr error
	if err = shared.RunWithSpinner("Creating package structure...", func() {
		result, initErr = scaffold.InitPackage(opts)
	}); err != nil {
		return err
	}
	if initErr != nil {
		return initErr
	}

	// --- Step 9: Summary ---
	effectiveRoot := project.ResolveRoot(root)
	tree := buildPackageSummaryTree(effectiveRoot)
	willCompile := shouldPackageAutoCompile(targetDir, root, forceCompile, skipCompile)

	nextSteps := []string{
		fmt.Sprintf("Add documentation to %s",
			tui.StyleCommand.Render("package/topics/")),
		fmt.Sprintf("Use %s for AI-assisted authoring",
			tui.StyleCommand.Render(effectiveRoot+"/")),
	}
	if !willCompile {
		nextSteps = append(nextSteps,
			fmt.Sprintf("Run %s to build the documentation index",
				tui.StyleCommand.Render("codectx compile")),
		)
	}

	ref := name
	if author != "" {
		ref = name + "@" + author
	}

	fmt.Print(tui.InitSummary(ref, tree, nextSteps))

	// Print key configuration values.
	fmt.Printf("%s%s\n", tui.Indent(1),
		tui.KeyValue("Type", "documentation package"))
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
			return nil //nolint:nilerr // link setup is optional
		}
	}

	// --- Step 11: Auto-compile ---
	if willCompile {
		if err := shared.RunPostInitCompile(targetDir); err != nil {
			fmt.Printf("\n%s %s\n%s %s\n\n",
				tui.StyleMuted.Render("-"),
				tui.StyleMuted.Render("Initial compilation failed (non-fatal)"),
				tui.Indent(1),
				tui.StyleMuted.Render("Run: "+tui.StyleCommand.Render("codectx compile")),
			)
			return nil //nolint:nilerr // auto-compile failure is non-fatal
		}
	}

	return nil
}

// resolveTarget determines the target directory from positional arguments.
// For package repositories, bare names (no path separators, not ".", not an
// existing directory) are automatically prefixed with "codectx-" to follow
// the naming convention (e.g., "react" → "codectx-react").
//
// Returns (absolutePath, createdNewDir, error).
func resolveTarget(args []string) (string, bool, error) {
	if len(args) == 0 || args[0] == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", false, fmt.Errorf("getting working directory: %w", err)
		}
		return cwd, false, nil
	}

	target := args[0]

	// If the path already exists, use it as-is.
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

	// Bare name (no path separators): apply codectx- prefix.
	if !strings.Contains(target, string(filepath.Separator)) && !strings.Contains(target, "/") {
		target = ensurePackagePrefix(target)
	}

	if err := os.MkdirAll(target, project.DirPerm); err != nil {
		return "", false, fmt.Errorf("creating directory %q: %w", target, err)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", false, err
	}
	return abs, true, nil
}

// buildPackageSummaryTree creates the tree structure for the package init summary.
func buildPackageSummaryTree(root string) []tui.TreeNode {
	return []tui.TreeNode{
		{
			Name: project.ConfigFileName,
		},
		{
			Name: project.PackageContentDir + "/",
			Children: []tui.TreeNode{
				{Name: project.ConfigFileName},
				{Name: "foundation/"},
				{Name: "topics/"},
				{Name: "plans/"},
				{Name: "prompts/"},
			},
		},
		{
			Name: root + "/",
			Children: []tui.TreeNode{
				{Name: "foundation/"},
				{Name: "topics/"},
				{Name: "plans/"},
				{Name: "prompts/"},
				{Name: "system/"},
				{Name: ".codectx/"},
			},
		},
		{
			Name: ".github/workflows/",
			Children: []tui.TreeNode{
				{Name: "release.yml"},
			},
		},
		{
			Name: "README.md",
		},
	}
}

// promptLink prompts the user to set up AI tool entry point files.
func promptLink(projectDir string) error {
	return shared.PromptAndWriteLinks(projectDir)
}

// shouldPackageAutoCompile determines whether auto-compilation should run
// after package init.
func shouldPackageAutoCompile(
	projectDir, root string,
	forceCompile, skipCompile bool,
) bool {
	return shared.ShouldPostInitCompile(projectDir, root, forceCompile, skipCompile)
}

// ensurePackagePrefix prepends the "codectx-" prefix if not already present.
func ensurePackagePrefix(s string) string {
	if strings.HasPrefix(s, registry.RepoPrefix) {
		return s
	}
	return registry.RepoPrefix + s
}

// stripPackagePrefix removes the "codectx-" prefix if present.
func stripPackagePrefix(s string) string {
	return strings.TrimPrefix(s, registry.RepoPrefix)
}
