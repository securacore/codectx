package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"securacore/codectx/core/config"
	"securacore/codectx/core/manifest"
	"securacore/codectx/core/schema"

	"github.com/urfave/cli/v3"
)

const configFile = "codectx.yml"
const packageFile = "package.yml"

var Command = &cli.Command{
	Name:  "init",
	Usage: "Initialize a new codectx project",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "Project name",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		return run(c.String("name"))
	},
}

func run(name string) error {
	// Guard: check if already initialized.
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("%s already exists: project is already initialized", configFile)
	}

	// Infer project name from current directory if not provided.
	if name == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		name = filepath.Base(wd)
	}

	// Create docs directory structure.
	docsDir := "docs"
	dirs := []string{
		docsDir,
		filepath.Join(docsDir, "foundation"),
		filepath.Join(docsDir, "topics"),
		filepath.Join(docsDir, "prompts"),
		filepath.Join(docsDir, "plans"),
		filepath.Join(docsDir, "schemas"),
		filepath.Join(docsDir, "packages"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Write embedded schemas to docs/schemas/.
	schemasDir := filepath.Join(docsDir, "schemas")
	if err := schema.WriteAll(schemasDir); err != nil {
		return fmt.Errorf("write schemas: %w", err)
	}

	// Create codectx.yml.
	cfg := &config.Config{
		Name:     name,
		Packages: []config.PackageDep{},
	}
	if err := config.Write(configFile, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Create docs/package.yml (local package data map).
	m := &manifest.Manifest{
		Name:        name,
		Author:      "",
		Version:     "0.1.0",
		Description: fmt.Sprintf("Documentation package for %s", name),
	}
	packagePath := filepath.Join(docsDir, packageFile)
	if err := manifest.Write(packagePath, m); err != nil {
		return fmt.Errorf("write package manifest: %w", err)
	}

	fmt.Printf("Initialized codectx project: %s\n", name)
	fmt.Println()
	fmt.Println("Created:")
	fmt.Printf("  %s\n", configFile)
	fmt.Printf("  %s\n", packagePath)
	for _, dir := range dirs {
		fmt.Printf("  %s/\n", dir)
	}

	return nil
}
