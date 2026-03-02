package new

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"

	initialize "github.com/securacore/codectx/cmds/init"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/ai"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/defaults"
	"github.com/securacore/codectx/core/gitkeep"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/packagetpl"
	"github.com/securacore/codectx/core/schema"
	"github.com/securacore/codectx/ui"
)

// githubSlug matches valid GitHub usernames and organization slugs:
// alphanumeric, hyphens, and underscores. Covers GitHub.com and Enterprise.
var githubSlug = regexp.MustCompile(`^[a-zA-Z0-9_]([a-zA-Z0-9_-]*[a-zA-Z0-9_])?$`)

var packageCommand = &cli.Command{
	Name:      "package",
	Usage:     "Scaffold a new codectx documentation package",
	ArgsUsage: "<name>",
	Action: func(ctx context.Context, c *cli.Command) error {
		args := c.Args()
		if args.Len() == 0 {
			return fmt.Errorf("missing required argument: name")
		}
		return runPackage(args.First())
	},
}

func runPackage(name string) error {
	if !kebabCase.MatchString(name) {
		return fmt.Errorf("invalid name %q: must be lowercase kebab-case (e.g. my-package)", name)
	}

	fullName := "codectx-" + name

	// Run core init: creates the directory, chdir into it, sets up docs
	// structure, schemas, foundation defaults, config, manifest, and
	// preferences.
	result, err := initialize.RunCore(fullName, nil, false)
	if err != nil {
		return err
	}

	// Mark this project as a package in codectx.yml so tools and the AI
	// directive know the package/ directory is the authoring target.
	result.Config.Type = "package"
	if err := config.Write(shared.ConfigFile, result.Config); err != nil {
		return fmt.Errorf("update config with package type: %w", err)
	}

	// Append package-specific entries to .gitignore (devbox, opencode).
	for _, entry := range []string{".devbox/", "tui.json"} {
		if err := shared.EnsureGitignoreEntry(".gitignore", entry); err != nil {
			return fmt.Errorf("update .gitignore: %w", err)
		}
	}

	// Prompt for author (org/username) and description (interactive only).
	var author, description string
	if ui.IsTTY() {
		var promptErr error
		author, description, promptErr = promptPackageInfo(name)
		if promptErr != nil {
			return promptErr
		}

		// If AI integration is configured, generate an enhanced description.
		description = maybeGenerateDescription(result, name, description)
	}

	// Determine the AI bin from preferences for template substitution.
	aiBin := "opencode"
	if result.Preferences != nil && result.Preferences.AI != nil && result.Preferences.AI.Bin != "" {
		aiBin = result.Preferences.AI.Bin
	}

	// Write package template files (bin/just/*, bin/release, prompts, etc.).
	if err := packagetpl.WriteAll(".", packagetpl.Options{AIBin: aiBin}); err != nil {
		return fmt.Errorf("write package templates: %w", err)
	}

	// Set up the package/ directory (distributable subset).
	if err := scaffoldPackageDir(result.DocsDir); err != nil {
		return fmt.Errorf("scaffold package directory: %w", err)
	}

	// Write the README.md at the project root.
	if author != "" {
		if err := writeReadme(name, author, description); err != nil {
			return fmt.Errorf("write README.md: %w", err)
		}
	}

	// Re-sync the manifest to pick up the newly added prompt and
	// foundation/prompts documents from the template files.
	docsDir := result.DocsDir
	manifestPath := filepath.Join(docsDir, "manifest.yml")
	existing, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest for re-sync: %w", err)
	}

	// Update manifest with author and description from prompts.
	if author != "" {
		existing.Author = author
	}
	if description != "" {
		existing.Description = description
	}

	synced := manifest.Sync(docsDir, existing)
	if err := manifest.Write(manifestPath, synced); err != nil {
		return fmt.Errorf("write re-synced manifest: %w", err)
	}

	// Write the package/ manifest (same as docs/ but without prompts).
	pkgManifest := &manifest.Manifest{
		Name:        synced.Name,
		Author:      synced.Author,
		Version:     synced.Version,
		Description: synced.Description,
		Foundation:  synced.Foundation,
		Application: synced.Application,
		Topics:      synced.Topics,
	}
	pkgManifestPath := filepath.Join("package", "manifest.yml")
	if err := manifest.Write(pkgManifestPath, pkgManifest); err != nil {
		return fmt.Errorf("write package manifest: %w", err)
	}

	ui.Blank()
	ui.Done(fmt.Sprintf("Scaffolded package: %s", fullName))
	ui.Blank()
	ui.Header("Extra files:")
	ui.Item("README.md")
	ui.Item("bin/just/")
	ui.Item("bin/release")
	ui.Item("docs/prompts/save/README.md")
	ui.Item("docs/foundation/prompts/README.md")
	ui.Item("package/")
	ui.Blank()

	// Run post-init: auto-compile and link AI tools.
	initialize.RunPostInit(result.Config)

	return nil
}

// promptPackageInfo prompts the user for the package author (org/username)
// and a description of what the package covers.
func promptPackageInfo(name string) (author, description string, err error) {
	ui.Blank()

	// Try to detect the authenticated GitHub username so we can pre-fill
	// the author field. If detected, the user can press Enter to accept.
	// The value must be set BEFORE creating the input so huh initializes
	// its internal text model with the detected value.
	detected := detectGitHubUser()
	if detected != "" {
		author = detected
	}

	authorInput := huh.NewInput().
		Title("GitHub username or organization").
		Value(&author).
		Validate(func(s string) error {
			v := strings.TrimSpace(s)
			if v == "" {
				return fmt.Errorf("author is required")
			}
			if !githubSlug.MatchString(v) {
				return fmt.Errorf("must be a GitHub username or organization (no spaces)")
			}
			return nil
		})

	if detected != "" {
		authorInput.Description("Detected: " + detected + " (press Enter to accept)")
	} else {
		authorInput.Description("Used for package identification (" + name + "@author)")
	}

	form := huh.NewForm(
		huh.NewGroup(
			authorInput,
			huh.NewInput().
				Title("Package description").
				Description("What does this documentation package cover?").
				Value(&description).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("description is required")
					}
					return nil
				}),
		),
	).WithTheme(ui.Theme())

	if err := form.Run(); err != nil {
		return "", "", fmt.Errorf("prompt: %w", err)
	}

	author = strings.TrimSpace(author)
	description = strings.TrimSpace(description)

	return author, description, nil
}

// maybeGenerateDescription uses the configured AI tool to enhance the user's
// description. If AI is not configured or generation fails, the original
// description is returned unchanged.
func maybeGenerateDescription(result *initialize.CoreResult, name, description string) string {
	if result.Preferences == nil || result.Preferences.AI == nil || result.Preferences.AI.Bin == "" {
		return description
	}

	bin := result.Preferences.AI.Bin

	// Verify the binary is still available before attempting generation.
	if _, err := exec.LookPath(bin); err != nil {
		ui.Warn(fmt.Sprintf("AI binary %q no longer found on PATH, using your description as-is", bin))
		return description
	}

	prompt := fmt.Sprintf(
		"Write a single concise sentence describing a codectx package called %q. "+
			"The author describes it as: %q. "+
			"The description must include the phrase \"AI documentation package\" naturally, "+
			"with qualifying context about what it covers. "+
			"Output only the description text, no quotes, no markdown formatting.",
		name, description,
	)

	var generated string
	err := ui.SpinErr("Generating description...", func() error {
		var genErr error
		generated, genErr = ai.Generate(bin, prompt)
		return genErr
	})

	if err != nil {
		ui.Warn(fmt.Sprintf("AI description generation failed: %s", err))
		ui.Step("Using your description as-is")
		return description
	}

	return generated
}

// writeReadme writes the package README.md at the project root.
func writeReadme(name, author, description string) error {
	ref := name + "@" + author

	var buf strings.Builder
	buf.WriteString("# " + ref + "\n")
	if description != "" {
		buf.WriteString("\n" + description + "\n")
	}
	buf.WriteString("\n```bash\ncodectx add " + ref + "\n```\n")

	return os.WriteFile("README.md", []byte(buf.String()), 0o644)
}

// detectGitHubUser attempts to resolve the authenticated GitHub username via
// the gh CLI. Returns empty string if gh is not installed or not authenticated.
func detectGitHubUser() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", "api", "user", "--jq", ".login")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// scaffoldPackageDir creates the package/ directory structure. This is the
// distributable subset that contains foundation docs and schemas but no
// topics or prompts content.
func scaffoldPackageDir(docsDir string) error {
	pkgDir := "package"

	// Create all subdirectories.
	dirs := []string{
		pkgDir,
		filepath.Join(pkgDir, "foundation"),
		filepath.Join(pkgDir, "topics"),
		filepath.Join(pkgDir, "prompts"),
		filepath.Join(pkgDir, "schemas"),
		filepath.Join(pkgDir, "packages"),
		filepath.Join(pkgDir, "plans"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Write foundation defaults to package/foundation/.
	if err := defaults.WriteAll(filepath.Join(pkgDir, "foundation")); err != nil {
		return fmt.Errorf("write package foundation defaults: %w", err)
	}

	// Write schemas to package/schemas/.
	if err := schema.WriteAll(filepath.Join(pkgDir, "schemas")); err != nil {
		return fmt.Errorf("write package schemas: %w", err)
	}

	// Place .gitkeep in empty directories.
	emptyDirs := []string{
		filepath.Join(pkgDir, "topics"),
		filepath.Join(pkgDir, "prompts"),
		filepath.Join(pkgDir, "packages"),
		filepath.Join(pkgDir, "plans"),
	}
	for _, dir := range emptyDirs {
		if err := gitkeep.Write(dir); err != nil {
			return fmt.Errorf("write .gitkeep in %s: %w", dir, err)
		}
	}

	return nil
}
