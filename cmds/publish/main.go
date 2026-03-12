// Package publish implements the `codectx publish` command which tags and
// pushes a documentation package to GitHub.
//
// The command reads codectx.yml for name, org, and version, validates the
// directory structure, creates a git tag v[version], and pushes to the
// remote repository at github.com/[org]/codectx-[name].
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

Reads codectx.yml for name, org, and version. Validates directory structure.
Tags the current commit as v[version] and pushes the tag.`,
	Action: run,
}

// validPackageDirs are the allowed directories in a published package.
var validPackageDirs = []string{
	"foundation",
	"topics",
	"plans",
	"prompts",
}

func run(ctx context.Context, _ *cli.Command) error {
	// Step 1: Find and load project config.
	projectDir, cfg, err := project.DiscoverAndLoad(".")
	if err != nil {
		fmt.Print(tui.ProjectNotFoundError())
		return fmt.Errorf("project not found: %w", err)
	}

	// Step 2: Validate required fields.
	if cfg.Name == "" {
		fmt.Print(tui.ErrorMsg{
			Title:  "Missing package name",
			Detail: []string{"codectx.yml must have a 'name' field for publishing."},
		}.Render())
		return fmt.Errorf("missing package name")
	}
	if cfg.Org == "" {
		fmt.Print(tui.ErrorMsg{
			Title:  "Missing organization",
			Detail: []string{"codectx.yml must have an 'org' field for publishing."},
		}.Render())
		return fmt.Errorf("missing org")
	}
	if cfg.Version == "" {
		fmt.Print(tui.ErrorMsg{
			Title:  "Missing version",
			Detail: []string{"codectx.yml must have a 'version' field for publishing."},
		}.Render())
		return fmt.Errorf("missing version")
	}

	tagName := registry.GitTag(cfg.Version)
	repoName := registry.RepoPrefix + cfg.Name
	remoteURL := fmt.Sprintf("https://github.com/%s/%s", cfg.Org, repoName)

	// Step 3: Validate directory structure.
	rootDir := project.RootDir(projectDir, cfg)
	if err := validatePackageStructure(rootDir); err != nil {
		fmt.Print(tui.ErrorMsg{
			Title:  "Invalid package structure",
			Detail: []string{err.Error()},
			Suggestions: []tui.Suggestion{
				{Text: "Packages must contain at least one of: foundation/, topics/, plans/, prompts/"},
			},
		}.Render())
		return err
	}

	// Step 4: Open the git repo.
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

	// Step 5: Check if tag already exists.
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

	// Step 6: Create tag and push.
	fmt.Printf("\n%s Publishing %s v%s\n",
		tui.Arrow(),
		tui.StyleBold.Render(cfg.Name+"@"+cfg.Org),
		cfg.Version,
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

	if err = shared.RunWithSpinner(fmt.Sprintf("Pushing %s to %s...", tagName, cfg.Org+"/"+repoName), func() {
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

	fmt.Printf("\n%s Published %s@%s v%s (%s)\n",
		tui.Success(),
		cfg.Name, cfg.Org, cfg.Version, sha,
	)
	fmt.Printf("%sInstall with: %s\n\n",
		tui.Indent(1),
		tui.StyleCommand.Render(
			fmt.Sprintf("codectx install %s@%s:%s", cfg.Name, cfg.Org, cfg.Version),
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
