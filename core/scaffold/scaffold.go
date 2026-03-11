// Package scaffold implements the directory and file creation logic for
// codectx init. It creates the full project structure including the
// documentation root, system/ documentation, config files, and .codectx/
// tooling state directory.
//
// The scaffold package is the engine — it assumes all capability checks
// (git, already initialized, writable, root conflicts) have been performed
// by the calling command. It creates what it's told to create.
package scaffold

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/embed"
)

// Result holds the outcome of a scaffold operation for reporting.
type Result struct {
	// ProjectDir is the absolute path to the project root (where codectx.yml lives).
	ProjectDir string

	// DocsRoot is the absolute path to the documentation root directory.
	DocsRoot string

	// Root is the documentation root directory name relative to ProjectDir.
	Root string

	// DirsCreated is the number of directories created.
	DirsCreated int

	// FilesCreated is the number of files written.
	FilesCreated int

	// GitInitialized is true if git init was run during scaffolding.
	GitInitialized bool
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

	// GitInit controls whether to run `git init` in ProjectDir before scaffolding.
	GitInit bool

	// Model is the AI model to write to ai.yml. Uses default if empty.
	Model string

	// Encoding is the tokenizer encoding to write to ai.yml. Uses default if empty.
	Encoding string

	// Provider is the LLM provider for compilation tasks ("cli" or "api").
	// Empty means auto-detect at compile time.
	Provider string
}

// CheckResult holds the result of pre-scaffold capability checks.
type CheckResult struct {
	// AlreadyInitialized is true if codectx.yml exists in the target dir.
	AlreadyInitialized bool

	// NestedProject is true if codectx.yml exists in a parent directory.
	// The path to the parent project is in NestedProjectPath.
	NestedProject bool

	// NestedProjectPath is the absolute path to the parent project's directory,
	// if a nested project was detected.
	NestedProjectPath string

	// HasGit is true if .git/ exists in the target directory.
	HasGit bool

	// RootConflict is true if the target documentation root directory already
	// exists and contains files.
	RootConflict bool

	// Writable is true if the target directory is writable.
	Writable bool
}

// Check performs pre-scaffold capability checks on the target directory.
// The command layer uses these results to display appropriate prompts and errors
// before calling Init.
func Check(projectDir, root string) (*CheckResult, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project directory: %w", err)
	}

	root = project.ResolveRoot(root)

	result := &CheckResult{}

	// Check if codectx.yml exists in the target directory.
	configPath := filepath.Join(absDir, project.ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		result.AlreadyInitialized = true
	}

	// Check if a parent directory has codectx.yml.
	if !result.AlreadyInitialized {
		parentDir := filepath.Dir(absDir)
		if parentDir != absDir {
			if found, err := project.Discover(parentDir); err == nil {
				result.NestedProject = true
				result.NestedProjectPath = found
			}
		}
	}

	// Check for .git/ directory.
	gitDir := filepath.Join(absDir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		result.HasGit = true
	}

	// Check if documentation root already exists with content.
	docsRoot := filepath.Join(absDir, root)
	if entries, err := os.ReadDir(docsRoot); err == nil && len(entries) > 0 {
		result.RootConflict = true
	}

	// Check if directory is writable by attempting to create a temp file.
	result.Writable = isWritable(absDir)

	return result, nil
}

// isWritable tests if a directory is writable by creating and removing a temp file.
func isWritable(dir string) bool {
	testFile := filepath.Join(dir, ".codectx-write-test")
	if err := os.WriteFile(testFile, []byte{}, project.FilePerm); err != nil {
		return false
	}
	_ = os.Remove(testFile)
	return true
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
//   - .gitignore at the git repo root with codectx entries (merged with existing)
//
// The calling command is responsible for performing Check() first and handling
// any capability issues (already initialized, root conflict, etc.) before
// calling Init.
func Init(opts Options) (*Result, error) {
	if opts.ProjectDir == "" {
		return nil, errors.New("project directory is required")
	}

	absDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project directory: %w", err)
	}
	opts.ProjectDir = absDir

	root := project.ResolveRoot(opts.Root)

	name := opts.Name
	if name == "" {
		name = filepath.Base(opts.ProjectDir)
	}

	docsRoot := filepath.Join(opts.ProjectDir, root)
	codectxDir := filepath.Join(docsRoot, project.CodectxDir)

	result := &Result{
		ProjectDir: opts.ProjectDir,
		DocsRoot:   docsRoot,
		Root:       root,
	}

	// Optionally run git init.
	if opts.GitInit {
		if err := gitInit(opts.ProjectDir); err != nil {
			return nil, fmt.Errorf("running git init: %w", err)
		}
		result.GitInitialized = true
	}

	// Create all directories.
	dirs := directories(docsRoot, codectxDir)
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, project.DirPerm); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
		result.DirsCreated++
	}

	// Write config files.
	if err := writeConfigs(opts.ProjectDir, codectxDir, name, root, opts.Model, opts.Encoding, opts.Provider); err != nil {
		return nil, err
	}
	result.FilesCreated += 3 // codectx.yml, ai.yml, preferences.yml

	// Write default system/ documentation files.
	written, err := writeSystemDefaults(docsRoot)
	if err != nil {
		return nil, err
	}
	result.FilesCreated += written

	// Ensure .gitignore at repo root contains codectx entries.
	if err := project.EnsureGitignore(opts.ProjectDir, root); err != nil {
		return nil, err
	}
	result.FilesCreated++

	return result, nil
}

// gitInit runs `git init` in the given directory.
func gitInit(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// directories returns the full list of directories to create.
func directories(docsRoot, codectxDir string) []string {
	dirs := []string{
		// Documentation root content directories.
		filepath.Join(docsRoot, "foundation"),
		filepath.Join(docsRoot, "topics"),
		filepath.Join(docsRoot, "plans"),
		filepath.Join(docsRoot, "prompts"),

		// System documentation (compiler instructions).
		filepath.Join(docsRoot, project.SystemDir, "foundation", "compiler-philosophy"),
		filepath.Join(docsRoot, project.SystemDir, "topics", "taxonomy-generation"),
		filepath.Join(docsRoot, project.SystemDir, "topics", "bridge-summaries"),
		filepath.Join(docsRoot, project.SystemDir, "topics", "context-assembly"),
		filepath.Join(docsRoot, project.SystemDir, "plans"),
		filepath.Join(docsRoot, project.SystemDir, "prompts"),

		// Tooling state directory.
		filepath.Join(codectxDir, project.PackagesDir),
	}

	// Compiled output directories (chunk + BM25 subdirs).
	for _, sub := range chunk.CompiledOutputDirs() {
		dirs = append(dirs, filepath.Join(codectxDir, project.CompiledDir, sub))
	}

	return dirs
}

// writeConfigs creates the three configuration files.
func writeConfigs(projectDir, codectxDir, name, root, model, encoding, provider string) error {
	// codectx.yml at project root.
	cfg := project.DefaultConfig(name, root)
	cfgPath := filepath.Join(projectDir, project.ConfigFileName)
	if err := cfg.WriteToFile(cfgPath); err != nil {
		return fmt.Errorf("writing %s: %w", project.ConfigFileName, err)
	}

	// ai.yml in .codectx/.
	aiCfg := project.DefaultAIConfig()
	if model != "" {
		aiCfg.Compilation.Model = model
		aiCfg.Consumption.Model = model
	}
	if encoding != "" {
		aiCfg.Compilation.Encoding = encoding
	}
	if provider != "" {
		aiCfg.Compilation.Provider = provider
	}
	aiPath := filepath.Join(codectxDir, project.AIConfigFile)
	if err := aiCfg.WriteToFile(aiPath); err != nil {
		return fmt.Errorf("writing %s: %w", project.AIConfigFile, err)
	}

	// preferences.yml in .codectx/.
	prefsCfg := project.DefaultPreferencesConfig()
	prefsPath := filepath.Join(codectxDir, project.PreferencesFile)
	if err := prefsCfg.WriteToFile(prefsPath); err != nil {
		return fmt.Errorf("writing %s: %w", project.PreferencesFile, err)
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
		if err := os.WriteFile(destPath, content, project.FilePerm); err != nil {
			return written, fmt.Errorf("writing %s: %w", f.DestPath, err)
		}
		written++
	}

	return written, nil
}
