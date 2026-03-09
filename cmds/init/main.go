// Package init implements the `codectx init` command which scaffolds
// a new codectx documentation project in the current directory.
package init

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/scaffold"
	"github.com/urfave/cli/v3"
)

// Command is the CLI definition for `codectx init`.
var Command = &cli.Command{
	Name:  "init",
	Usage: "Initialize a new codectx documentation project",
	Description: `Creates the codectx directory structure, default configuration files,
and system documentation in the current directory.

The documentation root defaults to "docs/". Use --root to override if
"docs/" is already in use for other purposes.

This command will not run if a codectx.yml already exists in the current
directory or any parent directory.`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "Documentation root directory name (default: docs)",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Project name (default: current directory name)",
		},
	},
	Action: run,
}

func run(_ context.Context, cmd *cli.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	opts := scaffold.Options{
		ProjectDir: cwd,
		Root:       cmd.String("root"),
		Name:       cmd.String("name"),
	}

	result, err := scaffold.Init(opts)
	if err != nil {
		if errors.Is(err, project.ErrAlreadyInitialized) {
			return fmt.Errorf("this directory is already part of a codectx project")
		}
		return err
	}

	root := opts.Root
	if root == "" {
		root = project.DefaultRoot
	}

	fmt.Printf("Initialized codectx project in %s\n", result.ProjectDir)
	fmt.Printf("  Documentation root: %s/\n", root)
	fmt.Printf("  Config: %s\n", project.ConfigFileName)
	fmt.Printf("  Created: %d directories, %d files\n", result.DirsCreated, result.FilesCreated)
	fmt.Println()
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Edit %s to set your project name and description\n", project.ConfigFileName)
	fmt.Printf("  2. Add foundation documents to %s/foundation/\n", root)
	fmt.Printf("  3. Add topic documentation to %s/topics/\n", root)
	fmt.Printf("  4. Run 'codectx compile' to build the documentation index\n")

	return nil
}
