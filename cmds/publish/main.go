// Package publish implements the `codectx publish` command which tags and
// pushes a documentation package to GitHub.
//
// The command reads codectx.yml for name, author, and version, validates the
// directory structure, creates a git tag v[version], and pushes to the
// remote repository at github.com/[author]/codectx-[name].
package publish

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/securacore/codectx/cmds/shared"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
	"github.com/securacore/codectx/core/tui"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx publish`.
var Command = &cli.Command{
	Name:  "publish",
	Usage: "Publish the current package to GitHub",
	Description: `Publish a documentation package by tagging the current commit and
pushing to GitHub. The repo must already exist on GitHub.

Reads codectx.yml for name, author, and version. Validates directory structure.
Tags the current commit as v[version] and pushes the tag.

For package authoring projects (type: "package"), validates the package/
directory and uses the package manifest for identity fields.

Use --validate to run all validation checks without publishing.`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "validate",
			Usage:   "Validate package structure without publishing (dry run)",
			Aliases: []string{"dry-run"},
		},
	},
	Action: run,
}

// validPackageDirs are the allowed directories in a published package.
var validPackageDirs = []string{
	"foundation",
	"topics",
	"plans",
	"prompts",
}

func run(ctx context.Context, cmd *cli.Command) error {
	validateOnly := cmd.Bool("validate")

	// Step 1: Find and load project config.
	projectDir, cfg, err := shared.DiscoverProject()
	if err != nil {
		return err
	}

	// Step 2: Determine validation source.
	// For package authoring projects, validate the package/ directory.
	// For standard projects, validate the documentation root.
	name := cfg.Name
	author := cfg.Author
	version := cfg.Version
	var structureDir string

	if cfg.IsPackage() {
		// Package authoring project — validate package/ directory.
		pkgContentDir := project.PackageContentPath(projectDir)
		structureDir = pkgContentDir

		// Load and validate the package manifest.
		manifestPath := project.PackageConfigPath(projectDir)
		manifest, loadErr := project.LoadPackageManifest(manifestPath)
		if loadErr != nil {
			fmt.Print(tui.ErrorMsg{
				Title: "Missing package manifest",
				Detail: []string{
					loadErr.Error(),
					fmt.Sprintf("Expected at: %s", tui.StylePath.Render(manifestPath)),
				},
				Suggestions: []tui.Suggestion{
					{Text: "Ensure package/codectx.yml exists with name, author, and version fields"},
				},
			}.Render())
			return fmt.Errorf("missing package manifest: %w", loadErr)
		}

		// Use manifest fields as source of truth for publishing identity.
		name = manifest.Name
		author = manifest.Author
		version = manifest.Version

		// Validate version consistency.
		if cfg.Version != "" && cfg.Version != manifest.Version {
			fmt.Print(tui.WarnMsg{
				Title: "Version mismatch",
				Detail: []string{
					fmt.Sprintf("Root codectx.yml version: %s", cfg.Version),
					fmt.Sprintf("Package manifest version: %s", manifest.Version),
					"The package manifest version will be used for publishing.",
				},
			}.Render())
		}
	} else {
		structureDir = project.RootDir(projectDir, cfg)
	}

	// Step 3: Validate required fields.
	if name == "" {
		fmt.Print(tui.ErrorMsg{
			Title:  "Missing package name",
			Detail: []string{"codectx.yml must have a 'name' field for publishing."},
		}.Render())
		return fmt.Errorf("missing package name")
	}
	if author == "" {
		fmt.Print(tui.ErrorMsg{
			Title: "Missing author",
			Detail: []string{
				"codectx.yml must have an 'author' field for publishing.",
			},
		}.Render())
		return fmt.Errorf("missing author")
	}
	if version == "" {
		fmt.Print(tui.ErrorMsg{
			Title:  "Missing version",
			Detail: []string{"codectx.yml must have a 'version' field for publishing."},
		}.Render())
		return fmt.Errorf("missing version")
	}

	tagName := registry.GitTag(version)
	repoName := registry.RepoPrefix + name
	remoteURL := fmt.Sprintf("https://github.com/%s/%s", author, repoName)

	// Step 4: Validate directory structure.
	if err := validatePackageStructure(structureDir); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Invalid package structure",
			Detail: []string{err.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Packages must contain at least one of: foundation/, topics/, plans/, prompts/"},
			},
		}.Render())
		return err
	}

	// Check for .codectx/ in the package directory (shouldn't be published).
	if cfg.IsPackage() {
		codectxInPkg := filepath.Join(structureDir, project.CodectxDir)
		if info, statErr := os.Stat(codectxInPkg); statErr == nil && info.IsDir() {
			fmt.Print(tui.WarnMsg{
				Title: "Package contains .codectx/ directory",
				Detail: []string{
					"The .codectx/ directory should not be in package/.",
					"It will be excluded from the release archive by the GitHub Action,",
					"but you may want to remove it.",
				},
			}.Render())
		}
	}

	// If validate-only, print summary and exit.
	if validateOnly {
		fmt.Printf("\n%s Package validation passed\n\n", tui.Success())
		fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue("Name", name+"@"+author))
		fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue("Version", version))
		fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue("Tag", tagName))
		fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue("Repo", remoteURL))
		if cfg.IsPackage() {
			fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue("Package dir", structureDir))
		}
		fmt.Println()
		return nil
	}

	// Step 5: Open the git repo.
	repo, err := git.PlainOpen(projectDir)
	if err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Not a git repository",
			Detail: []string{"codectx publish requires a git repository."},
			Suggestions: []tui.Suggestion{
				{Text: "Initialize git:", Command: "git init"},
			},
		}.Render())
		return fmt.Errorf("not a git repo: %w", err)
	}

	gc := registry.NewGitClient(registry.GitHubToken())

	// Step 6: Check if tag already exists.
	if gc.TagExists(repo, tagName) {
		fmt.Print(tui.ErrorMsg{
			Title: fmt.Sprintf("Tag %s already exists", tagName),
			Detail: []string{
				"This version has already been published.",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Bump the version in codectx.yml and try again"},
			},
		}.Render())
		return fmt.Errorf("tag %s already exists", tagName)
	}

	// Step 7: Create tag and push.
	fmt.Printf("\n%s Publishing %s v%s\n",
		tui.Arrow(),
		tui.StyleBold.Render(name+"@"+author),
		version,
	)
	fmt.Printf("%s%s\n", tui.Indent(1), tui.KeyValue("Repo", tui.StyleMuted.Render(remoteURL)))

	var tagErr, pushErr error

	if err = shared.RunWithSpinner(fmt.Sprintf("Creating tag %s...", tagName), func() {
		tagErr = gc.CreateLightweightTag(repo, tagName)
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if tagErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Failed to create tag",
			Detail: []string{tagErr.Error()},
		}.Render())
		return tagErr
	}

	if err = shared.RunWithSpinner(fmt.Sprintf("Pushing %s to %s...", tagName, author+"/"+repoName), func() {
		pushErr = gc.PushTag(ctx, repo, tagName)
	}); err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if pushErr != nil {
		fmt.Print(tui.ErrorMsg{
			Title: "Failed to push tag",
			Detail: []string{
				pushErr.Error(),
				"The tag was created locally but could not be pushed.",
			},
			Suggestions: []tui.Suggestion{
				{Text: "Check that the remote repository exists:", Command: remoteURL},
				{Text: "Push manually:", Command: fmt.Sprintf("git push origin %s", tagName)},
			},
		}.Render())
		return pushErr
	}

	sha, _ := gc.HeadCommitSHA(repo)
	if len(sha) > 8 {
		sha = sha[:8]
	}

	fmt.Printf("\n%s Published %s v%s (%s)\n\n",
		tui.Success(),
		tui.StyleBold.Render(name+"@"+author), version, sha,
	)
	fmt.Printf("%sInstall with: %s\n\n",
		tui.Indent(1),
		tui.StyleCommand.Render(
			fmt.Sprintf("codectx add %s@%s:%s", name, author, version),
		),
	)

	return nil
}

// validatePackageStructure checks that the documentation root contains at
// least one of the valid package directories (foundation/, topics/, plans/,
// prompts/) and has a codectx.yml.
func validatePackageStructure(rootDir string) error {
	found := false
	for _, dir := range validPackageDirs {
		info, err := os.Stat(filepath.Join(rootDir, dir))
		if err == nil && info.IsDir() {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no package content directories found in %s", rootDir)
	}
	return nil
}
