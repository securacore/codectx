package new

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	"github.com/charmbracelet/x/term"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/detect"
	"github.com/securacore/codectx/core/link"
	"github.com/securacore/codectx/core/project"
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
			Name:  "org",
			Usage: "Organization or author namespace",
		},
		&cli.StringFlag{
			Name:  "description",
			Usage: "One-line package description",
		},
		&cli.StringFlag{
			Name:  "root",
			Usage: "Documentation root directory name for authoring project (default: docs)",
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
	org := cmd.String("org")
	description := cmd.String("description")
	root := cmd.String("root")
	model := cmd.String("model")
	autoYes := cmd.Bool("yes")
	forceCompile := cmd.IsSet("compile") && cmd.Bool("compile")
	skipCompile := cmd.IsSet("no-compile") && cmd.Bool("no-compile")

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

	// --- Step 6: Resolve org ---
	if org == "" && interactive {
		if err := huh.NewInput().
			Title("Organization (GitHub user or org):").
			Value(&org).
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

	// --- Step 8: AI tool detection ---
	var detection detect.Result

	if err = shared.RunWithSpinner("Scanning for AI tools...", func() {
		detection = detect.Scan()
	}); err != nil {
		return err
	}

	if detection.HasAnything() {
		fmt.Println()
		if len(detection.Tools) > 0 {
			fmt.Printf("%s%s\n", tui.Indent(1), tui.StyleMuted.Render("Detected tools:"))
			for _, tool := range detection.Tools {
				fmt.Printf("%s%s\n", tui.Indent(2), tui.DetectedTool(tool.Name, tool.Version))
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

	// --- Step 9: Model selection ---
	effectiveModel := model
	effectiveEncoding := detection.RecommendedEncoding
	if effectiveModel == "" {
		effectiveModel = detection.RecommendedModel
	}

	if interactive && model == "" {
		type modelChoice struct {
			model    string
			encoding string
		}

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

	// --- Step 9b: Provider selection ---
	hasCLI, hasAPI := detectProviderCapabilities(detection)
	effectiveProvider := autoSelectProvider(hasCLI, hasAPI)

	if hasCLI && hasAPI && interactive {
		if err := huh.NewSelect[string]().
			Title("LLM provider for compilation:").
			Options(
				huh.NewOption("Claude CLI (recommended, uses existing subscription)", project.ProviderCLI),
				huh.NewOption("Anthropic API (direct API calls)", project.ProviderAPI),
			).
			Value(&effectiveProvider).
			Run(); err != nil {
			return err
		}
	}

	// --- Step 10: Scaffold with spinner ---
	opts := scaffold.PackageOptions{
		ProjectDir:  targetDir,
		Root:        root,
		Name:        name,
		Org:         org,
		Description: description,
		GitInit:     needGitInit,
		Model:       effectiveModel,
		Encoding:    effectiveEncoding,
		Provider:    effectiveProvider,
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

	// --- Step 11: Summary ---
	effectiveRoot := project.ResolveRoot(root)
	tree := buildPackageSummaryTree(effectiveRoot)

	nextSteps := []string{
		fmt.Sprintf("Add documentation to %s",
			tui.StyleCommand.Render("package/topics/")),
		fmt.Sprintf("Use %s for AI-assisted authoring",
			tui.StyleCommand.Render(effectiveRoot+"/")),
	}
	if !shouldPackageAutoCompile(targetDir, root, forceCompile, skipCompile) {
		nextSteps = append(nextSteps,
			fmt.Sprintf("Run %s to build the documentation index",
				tui.StyleCommand.Render("codectx compile")),
		)
	}

	ref := name
	if org != "" {
		ref = name + "@" + org
	}

	fmt.Print(tui.InitSummary(ref, tree, nextSteps))

	// Print key configuration values.
	fmt.Printf("%s%s\n", tui.Indent(1),
		tui.KeyValue("Type", "documentation package"))
	fmt.Printf("%s%s\n", tui.Indent(1),
		tui.KeyValue("Model", effectiveModel))
	if effectiveProvider != "" {
		var providerLabel string
		switch effectiveProvider {
		case project.ProviderCLI:
			providerLabel = "cli (Claude Code)"
		case project.ProviderAPI:
			providerLabel = "api (Anthropic API)"
		default:
			providerLabel = effectiveProvider
		}
		fmt.Printf("%s%s\n", tui.Indent(1),
			tui.KeyValue("Provider", providerLabel))
	}
	if created {
		fmt.Printf("%s%s\n", tui.Indent(1),
			tui.KeyValue("Directory", targetDir))
	}
	if result.GitInitialized {
		fmt.Printf("%s%s\n", tui.Indent(1),
			tui.KeyValue("Git", "initialized"))
	}
	fmt.Println()

	// --- Step 12: Link prompt ---
	if interactive {
		if err := promptLink(targetDir); err != nil {
			return nil //nolint:nilerr // link setup is optional
		}
	}

	// --- Step 13: Auto-compile ---
	if shouldPackageAutoCompile(targetDir, root, forceCompile, skipCompile) {
		if err := runPackageCompile(targetDir, root); err != nil {
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

	if err := os.MkdirAll(target, project.DirPerm); err != nil {
		return "", false, fmt.Errorf("creating directory %q: %w", target, err)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", false, err
	}
	return abs, true, nil
}

// detectProviderCapabilities checks the detection results for Claude CLI
// binary and Anthropic API key availability.
func detectProviderCapabilities(detection detect.Result) (hasCLI, hasAPI bool) {
	for _, t := range detection.Tools {
		if t.Binary == "claude" {
			hasCLI = true
			break
		}
	}
	for _, p := range detection.Providers {
		if p.Name == "Anthropic" {
			hasAPI = true
			break
		}
	}
	return hasCLI, hasAPI
}

// autoSelectProvider returns the appropriate provider string based on
// detected capabilities.
func autoSelectProvider(hasCLI, hasAPI bool) string {
	switch {
	case hasCLI:
		return project.ProviderCLI
	case hasAPI:
		return project.ProviderAPI
	default:
		return ""
	}
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
	selected, err := link.PromptIntegrations(projectDir, "Set up AI tool entry points?")
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		return nil
	}

	cfg, err := project.LoadConfig(filepath.Join(projectDir, project.ConfigFileName))
	if err != nil {
		return err
	}

	contextRelPath := project.ContextRelPath(cfg.Root)

	results, err := link.Write(projectDir, contextRelPath, selected)
	if err != nil {
		return err
	}

	fmt.Print(link.RenderLinkResults(results))
	return nil
}

// shouldPackageAutoCompile determines whether auto-compilation should run
// after package init.
func shouldPackageAutoCompile(
	projectDir, root string,
	forceCompile, skipCompile bool,
) bool {
	effectiveRoot := project.ResolveRoot(root)
	codectxDir := filepath.Join(projectDir, effectiveRoot, project.CodectxDir)
	prefsCfg, err := project.LoadPreferencesConfig(filepath.Join(codectxDir, project.PreferencesFile))
	if err != nil {
		prefsCfg = &project.PreferencesConfig{}
	}

	return shared.ShouldAutoCompile(prefsCfg, forceCompile, skipCompile, "initial compile")
}

// runPackageCompile loads the freshly created project configuration and runs
// the full compilation pipeline.
func runPackageCompile(projectDir, root string) error {
	cfg, err := project.LoadConfig(filepath.Join(projectDir, project.ConfigFileName))
	if err != nil {
		return fmt.Errorf("loading project config: %w", err)
	}

	rootDir := project.RootDir(projectDir, cfg)

	aiCfg, err := project.LoadAIConfigForProject(projectDir, cfg)
	if err != nil {
		return fmt.Errorf("loading AI config: %w", err)
	}

	prefsCfg, err := project.LoadPreferencesConfigForProject(projectDir, cfg)
	if err != nil {
		return fmt.Errorf("loading preferences: %w", err)
	}

	compileCfg := compile.BuildConfig(projectDir, rootDir, cfg, aiCfg, prefsCfg)

	fmt.Printf("\n%s Compiling documentation...\n", tui.Arrow())

	var result *compile.Result
	var compileErr error

	if sErr := shared.RunWithSpinner("Compiling...", func() {
		result, compileErr = compile.Run(compileCfg, nil)
	}); sErr != nil {
		return sErr
	}
	if compileErr != nil {
		return compileErr
	}

	fmt.Print(shared.RenderCompactCompileSummary(result))

	return nil
}
