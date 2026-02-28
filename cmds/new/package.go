package new

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	initialize "github.com/securacore/codectx/cmds/init"
	"github.com/securacore/codectx/core/defaults"
	"github.com/securacore/codectx/core/gitkeep"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/packagetpl"
	"github.com/securacore/codectx/core/schema"
	"github.com/securacore/codectx/ui"

	"github.com/urfave/cli/v3"
)

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

	// Re-sync the manifest to pick up the newly added prompt and
	// foundation/prompts documents from the template files.
	docsDir := result.DocsDir
	manifestPath := filepath.Join(docsDir, "manifest.yml")
	existing, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest for re-sync: %w", err)
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
