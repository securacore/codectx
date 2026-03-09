// Package scaffold implements the directory and file creation logic for
// codectx init. It creates the full project structure including the
// documentation root, system/ documentation, config files, and .codectx/
// tooling state directory.
package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/embed"
)

// Result holds the outcome of a scaffold operation for reporting.
type Result struct {
	// ProjectDir is the absolute path to the project root (where codectx.yml lives).
	ProjectDir string

	// DocsRoot is the absolute path to the documentation root directory.
	DocsRoot string

	// DirsCreated is the number of directories created.
	DirsCreated int

	// FilesCreated is the number of files written.
	FilesCreated int
}

// Options configures the scaffold operation.
type Options struct {
	// ProjectDir is the directory where codectx.yml will be created.
	// Typically the current working directory.
	ProjectDir string

	// Root is the documentation root directory name relative to ProjectDir.
	// Defaults to "docs" if empty.
	Root string

	// Name is the project name for codectx.yml.
	// Defaults to the base name of ProjectDir if empty.
	Name string
}

// Init creates the full codectx project structure. It is the implementation
// behind `codectx init`.
//
// It creates:
//   - codectx.yml in ProjectDir
//   - The documentation root directory (default: docs/)
//   - system/ subdirectory with default compiler documentation
//   - foundation/, topics/, plans/, prompts/ directories
//   - .codectx/ directory with ai.yml, preferences.yml, and compiled/packages/ subdirs
//   - .gitignore additions for codectx artifacts
//
// Returns ErrAlreadyInitialized if codectx.yml already exists in ProjectDir
// or any parent directory.
func Init(opts Options) (*Result, error) {
	if opts.ProjectDir == "" {
		return nil, errors.New("project directory is required")
	}

	absDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project directory: %w", err)
	}
	opts.ProjectDir = absDir

	// Check if already initialized (codectx.yml exists here or above).
	if _, err := project.Discover(opts.ProjectDir); err == nil {
		return nil, project.ErrAlreadyInitialized
	}

	root := opts.Root
	if root == "" {
		root = project.DefaultRoot
	}

	name := opts.Name
	if name == "" {
		name = filepath.Base(opts.ProjectDir)
	}

	docsRoot := filepath.Join(opts.ProjectDir, root)
	codectxDir := filepath.Join(docsRoot, ".codectx")

	result := &Result{
		ProjectDir: opts.ProjectDir,
		DocsRoot:   docsRoot,
	}

	// Create all directories.
	dirs := directories(docsRoot, codectxDir)
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
		result.DirsCreated++
	}

	// Write config files.
	if err := writeConfigs(opts.ProjectDir, codectxDir, name, root); err != nil {
		return nil, err
	}
	result.FilesCreated += 3 // codectx.yml, ai.yml, preferences.yml

	// Write default system/ documentation files.
	written, err := writeSystemDefaults(docsRoot)
	if err != nil {
		return nil, err
	}
	result.FilesCreated += written

	// Write .gitignore for codectx artifacts.
	if err := writeGitignore(docsRoot); err != nil {
		return nil, err
	}
	result.FilesCreated++

	return result, nil
}

// directories returns the full list of directories to create.
func directories(docsRoot, codectxDir string) []string {
	return []string{
		// Documentation root content directories.
		filepath.Join(docsRoot, "foundation"),
		filepath.Join(docsRoot, "topics"),
		filepath.Join(docsRoot, "plans"),
		filepath.Join(docsRoot, "prompts"),

		// System documentation (compiler instructions).
		filepath.Join(docsRoot, "system", "foundation", "compiler-philosophy"),
		filepath.Join(docsRoot, "system", "topics", "taxonomy-generation"),
		filepath.Join(docsRoot, "system", "topics", "bridge-summaries"),
		filepath.Join(docsRoot, "system", "topics", "context-assembly"),
		filepath.Join(docsRoot, "system", "plans"),
		filepath.Join(docsRoot, "system", "prompts"),

		// Tooling state directory.
		filepath.Join(codectxDir, "compiled", "objects"),
		filepath.Join(codectxDir, "compiled", "specs"),
		filepath.Join(codectxDir, "compiled", "system"),
		filepath.Join(codectxDir, "compiled", "bm25", "objects"),
		filepath.Join(codectxDir, "compiled", "bm25", "specs"),
		filepath.Join(codectxDir, "compiled", "bm25", "system"),
		filepath.Join(codectxDir, "packages"),
	}
}

// writeConfigs creates the three configuration files.
func writeConfigs(projectDir, codectxDir, name, root string) error {
	// codectx.yml at project root.
	cfg := project.DefaultConfig(name, root)
	cfgPath := filepath.Join(projectDir, project.ConfigFileName)
	if err := cfg.WriteToFile(cfgPath); err != nil {
		return fmt.Errorf("writing %s: %w", project.ConfigFileName, err)
	}

	// ai.yml in .codectx/.
	aiCfg := project.DefaultAIConfig()
	aiPath := filepath.Join(codectxDir, "ai.yml")
	if err := aiCfg.WriteToFile(aiPath); err != nil {
		return fmt.Errorf("writing ai.yml: %w", err)
	}

	// preferences.yml in .codectx/.
	prefsCfg := project.DefaultPreferencesConfig()
	prefsPath := filepath.Join(codectxDir, "preferences.yml")
	if err := prefsCfg.WriteToFile(prefsPath); err != nil {
		return fmt.Errorf("writing preferences.yml: %w", err)
	}

	return nil
}

// writeSystemDefaults writes the embedded default system/ documentation files.
func writeSystemDefaults(docsRoot string) (int, error) {
	files := embed.SystemFiles()
	written := 0

	for _, f := range files {
		content, err := embed.ReadFile(f.EmbedPath)
		if err != nil {
			return written, fmt.Errorf("reading embedded file %s: %w", f.EmbedPath, err)
		}

		destPath := filepath.Join(docsRoot, f.DestPath)
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return written, fmt.Errorf("writing %s: %w", f.DestPath, err)
		}
		written++
	}

	return written, nil
}

// writeGitignore creates the .gitignore file in the docs root that ignores
// compiled output, installed packages, and local API key configuration.
func writeGitignore(docsRoot string) error {
	content := `# codectx — tooling state and compiled output
.codectx/compiled/
.codectx/packages/
.codectx/ai.local.yml

# Force-include checked-in config
!.codectx/ai.yml
!.codectx/preferences.yml
`
	path := filepath.Join(docsRoot, ".codectx", ".gitignore")
	return os.WriteFile(path, []byte(content), 0644)
}
