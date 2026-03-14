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
	"github.com/securacore/codectx/core/usage"
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

	// Encoding is the tokenizer encoding to write to ai.yml. Uses default if empty.
	Encoding string

	// ProjectType controls the type field in codectx.yml.
	// Use project.TypePackage for package authoring projects.
	// Empty or project.TypeProject for standard projects.
	ProjectType string
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
	if err := writeConfigs(configPaths{
		projectDir: opts.ProjectDir, codectxDir: codectxDir,
		name: name, root: root,
		encoding:    opts.Encoding,
		projectType: opts.ProjectType,
	}); err != nil {
		return nil, err
	}
	result.FilesCreated += 3 // codectx.yml, ai.yml, preferences.yml

	// Write default system/ documentation files.
	written, err := writeSystemDefaults(docsRoot)
	if err != nil {
		return nil, err
	}
	result.FilesCreated += written

	// Create .gitkeep in empty content directories.
	for _, dir := range contentDirs(docsRoot) {
		added, _, err := manageGitkeep(dir)
		if err != nil {
			return nil, fmt.Errorf("creating .gitkeep in %s: %w", dir, err)
		}
		if added {
			result.FilesCreated++
		}
	}

	// Ensure .gitignore at repo root contains codectx entries.
	if err := project.EnsureGitignore(opts.ProjectDir, root); err != nil {
		return nil, err
	}
	result.FilesCreated++

	// Create usage files (global_usage.yml is checked in, usage.yml is gitignored).
	globalPath := filepath.Join(codectxDir, usage.GlobalFile)
	if err := usage.InitGlobalFile(globalPath, name); err != nil {
		return nil, fmt.Errorf("creating %s: %w", usage.GlobalFile, err)
	}
	result.FilesCreated++

	localPath := filepath.Join(codectxDir, usage.LocalFile)
	if err := usage.InitLocalFile(localPath); err != nil {
		return nil, fmt.Errorf("creating %s: %w", usage.LocalFile, err)
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
		filepath.Join(docsRoot, project.SystemDir, "foundation", "documentation-protocol"),
		filepath.Join(docsRoot, project.SystemDir, "foundation", "history"),
		filepath.Join(docsRoot, project.SystemDir, "topics", "context-assembly"),
		filepath.Join(docsRoot, project.SystemDir, "plans"),
		filepath.Join(docsRoot, project.SystemDir, "prompts"),

		// Tooling state directory.
		filepath.Join(codectxDir, project.PackagesDir),

		// History directory for query/generate entries and document snapshots.
		filepath.Join(codectxDir, "history", "queries"),
		filepath.Join(codectxDir, "history", "chunks"),
		filepath.Join(codectxDir, "history", "docs"),
	}

	// Compiled output directories (chunk + BM25 subdirs).
	for _, sub := range chunk.CompiledOutputDirs() {
		dirs = append(dirs, filepath.Join(codectxDir, project.CompiledDir, sub))
	}

	return dirs
}

// configPaths holds the resolved paths and values for writing config files.
type configPaths struct {
	projectDir  string
	codectxDir  string
	name        string
	root        string
	encoding    string
	projectType string // "project" or "package"
}

// writeConfigs creates the three configuration files.
func writeConfigs(cp configPaths) error {
	// codectx.yml at project root.
	cfg := project.DefaultConfig(cp.name, cp.root, cp.projectType)
	cfgPath := filepath.Join(cp.projectDir, project.ConfigFileName)
	if err := cfg.WriteToFile(cfgPath); err != nil {
		return fmt.Errorf("writing %s: %w", project.ConfigFileName, err)
	}

	// ai.yml in .codectx/.
	aiCfg := project.DefaultAIConfig()
	if cp.encoding != "" {
		aiCfg.Compilation.Encoding = cp.encoding
	}
	aiPath := filepath.Join(cp.codectxDir, project.AIConfigFile)
	if err := aiCfg.WriteToFile(aiPath); err != nil {
		return fmt.Errorf("writing %s: %w", project.AIConfigFile, err)
	}

	// preferences.yml in .codectx/.
	prefsCfg := project.DefaultPreferencesConfig()
	prefsPath := filepath.Join(cp.codectxDir, project.PreferencesFile)
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

// PackageOptions configures the package scaffold operation.
type PackageOptions struct {
	// ProjectDir is the directory where the package repo will be created.
	ProjectDir string

	// Root is the documentation root for the authoring project.
	// Defaults to "docs" if empty.
	Root string

	// Name is the package name (e.g., "react").
	Name string

	// Author is the GitHub username or organization (e.g., "community").
	Author string

	// Description is a one-line package description.
	Description string

	// GitInit controls whether to run `git init` before scaffolding.
	GitInit bool

	// Model is the AI model to write to ai.yml for the authoring project.
	Model string

	// Encoding is the tokenizer encoding for the authoring project.
	Encoding string
}

// PackageResult holds the outcome of a package scaffold operation.
type PackageResult struct {
	// ProjectDir is the absolute path to the project root.
	ProjectDir string

	// PackageDir is the absolute path to the package/ directory.
	PackageDir string

	// DocsRoot is the absolute path to the docs/ authoring project root.
	DocsRoot string

	// DirsCreated is the number of directories created.
	DirsCreated int

	// FilesCreated is the number of files written.
	FilesCreated int

	// GitInitialized is true if git init was run during scaffolding.
	GitInitialized bool

	// InitResult holds the result of the docs/ authoring project init.
	InitResult *Result
}

// InitPackage creates the full documentation package repository structure.
// It creates:
//   - package/ directory with content dirs (foundation, topics, plans, prompts)
//   - package/codectx.yml with package-only manifest
//   - .github/workflows/release.yml for automated publishing
//   - README.md at repo root
//   - .gitignore with codectx entries
//   - docs/ authoring project via Init()
//
// The calling command is responsible for performing checks before calling InitPackage.
func InitPackage(opts PackageOptions) (*PackageResult, error) {
	if opts.ProjectDir == "" {
		return nil, errors.New("project directory is required")
	}

	absDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project directory: %w", err)
	}
	opts.ProjectDir = absDir

	result := &PackageResult{
		ProjectDir: absDir,
		PackageDir: filepath.Join(absDir, project.PackageContentDir),
	}

	// Optionally run git init first.
	if opts.GitInit {
		if err := gitInit(absDir); err != nil {
			return nil, fmt.Errorf("running git init: %w", err)
		}
		result.GitInitialized = true
	}

	// 1. Create package/ content directories.
	pkgDir := filepath.Join(absDir, project.PackageContentDir)
	pkgContentDirs := []string{
		filepath.Join(pkgDir, "foundation"),
		filepath.Join(pkgDir, "topics"),
		filepath.Join(pkgDir, "plans"),
		filepath.Join(pkgDir, "prompts"),
	}
	for _, dir := range pkgContentDirs {
		if err := os.MkdirAll(dir, project.DirPerm); err != nil {
			return nil, fmt.Errorf("creating package directory %s: %w", dir, err)
		}
		result.DirsCreated++
	}

	// 2. Write package/codectx.yml.
	manifest := project.DefaultPackageManifest(opts.Name, opts.Author, opts.Description)
	manifestPath := project.PackageConfigPath(absDir)
	if err := manifest.WriteToFile(manifestPath); err != nil {
		return nil, fmt.Errorf("writing package manifest: %w", err)
	}
	result.FilesCreated++

	// 3. Write .gitkeep in empty package content directories.
	for _, dir := range pkgContentDirs {
		added, _, err := manageGitkeep(dir)
		if err != nil {
			return nil, fmt.Errorf("creating .gitkeep in %s: %w", dir, err)
		}
		if added {
			result.FilesCreated++
		}
	}

	// 4. Write package template files (GHA workflow, etc.).
	for _, tmpl := range embed.PackageTemplateFiles() {
		content, err := embed.ReadPackageFile(tmpl.EmbedPath)
		if err != nil {
			return nil, fmt.Errorf("reading embedded template %s: %w", tmpl.EmbedPath, err)
		}

		destPath := filepath.Join(absDir, tmpl.DestPath)
		if err := os.MkdirAll(filepath.Dir(destPath), project.DirPerm); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", tmpl.DestPath, err)
		}

		if err := os.WriteFile(destPath, content, project.FilePerm); err != nil {
			return nil, fmt.Errorf("writing %s: %w", tmpl.DestPath, err)
		}
		result.FilesCreated++
	}

	// 5. Write README.md at repo root.
	readmeContent := generatePackageReadme(opts.Name, opts.Author, opts.Description)
	readmePath := filepath.Join(absDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), project.FilePerm); err != nil {
		return nil, fmt.Errorf("writing README.md: %w", err)
	}
	result.FilesCreated++

	// 6. Initialize the docs/ authoring project.
	initOpts := Options{
		ProjectDir:  absDir,
		Root:        opts.Root,
		Name:        opts.Name,
		Encoding:    opts.Encoding,
		ProjectType: project.TypePackage,
	}

	initResult, err := Init(initOpts)
	if err != nil {
		return nil, fmt.Errorf("initializing authoring project: %w", err)
	}

	result.DocsRoot = initResult.DocsRoot
	result.DirsCreated += initResult.DirsCreated
	result.FilesCreated += initResult.FilesCreated
	result.InitResult = initResult

	return result, nil
}

// generatePackageReadme creates a README.md for a documentation package repo.
func generatePackageReadme(name, author, description string) string {
	ref := name + "@" + author
	if author == "" {
		ref = name
	}

	readme := "# " + ref + "\n\n"

	if description != "" {
		readme += description + "\n\n"
	}

	readme += "## Install\n\n"
	readme += "```bash\n"
	readme += "codectx add " + ref + ":latest\n"
	readme += "```\n"

	return readme
}

// MaintainResult holds the outcome of a scaffold maintenance operation.
type MaintainResult struct {
	// DirsCreated is the number of directories that were recreated.
	DirsCreated int

	// FilesRestored is the number of system default files that were restored.
	FilesRestored int

	// GitkeepsAdded is the number of .gitkeep files added to empty content dirs.
	GitkeepsAdded int

	// GitkeepsRemoved is the number of .gitkeep files removed from non-empty dirs.
	GitkeepsRemoved int
}

// HasActions reports whether any maintenance actions were taken.
func (r *MaintainResult) HasActions() bool {
	return r.DirsCreated > 0 || r.FilesRestored > 0 || r.GitkeepsAdded > 0 || r.GitkeepsRemoved > 0
}

// Maintain ensures the scaffold structure is intact. It recreates missing
// directories, restores missing system default files, and manages .gitkeep
// files in the four top-level content directories (foundation, topics, plans,
// prompts). Returns a summary of actions taken.
//
// This is called by `codectx compile` (when scaffold_maintenance is enabled)
// and by `codectx repair` (unconditionally).
func Maintain(projectDir string, cfg *project.Config) (*MaintainResult, error) {
	root := project.ResolveRoot(cfg.Root)
	docsRoot := filepath.Join(projectDir, root)
	codectxDir := filepath.Join(docsRoot, project.CodectxDir)
	result := &MaintainResult{}

	// 1. Ensure all directories exist.
	dirs := directories(docsRoot, codectxDir)
	for _, dir := range dirs {
		created, err := ensureDir(dir)
		if err != nil {
			return result, fmt.Errorf("ensuring directory %s: %w", dir, err)
		}
		if created {
			result.DirsCreated++
		}
	}

	// 2. Restore missing system default files.
	restored, err := restoreMissingDefaults(docsRoot)
	if err != nil {
		return result, err
	}
	result.FilesRestored = restored

	// 3. Manage .gitkeep in top-level content directories.
	for _, dir := range contentDirs(docsRoot) {
		added, removed, err := manageGitkeep(dir)
		if err != nil {
			return result, fmt.Errorf("managing .gitkeep in %s: %w", dir, err)
		}
		if added {
			result.GitkeepsAdded++
		}
		if removed {
			result.GitkeepsRemoved++
		}
	}

	return result, nil
}

// contentDirs returns the four top-level content directories that receive
// .gitkeep management.
func contentDirs(docsRoot string) []string {
	return []string{
		filepath.Join(docsRoot, "foundation"),
		filepath.Join(docsRoot, "topics"),
		filepath.Join(docsRoot, "plans"),
		filepath.Join(docsRoot, "prompts"),
	}
}

// ensureDir creates a directory if it doesn't exist.
// Returns true if the directory was created.
func ensureDir(dir string) (bool, error) {
	if _, err := os.Stat(dir); err == nil {
		return false, nil
	}
	if err := os.MkdirAll(dir, project.DirPerm); err != nil {
		return false, err
	}
	return true, nil
}

// restoreMissingDefaults checks each embedded system default file and writes
// it if the destination file is missing. Returns the count of restored files.
func restoreMissingDefaults(docsRoot string) (int, error) {
	files := embed.SystemFiles()
	restored := 0

	for _, f := range files {
		destPath := filepath.Join(docsRoot, f.DestPath)

		// Skip if file already exists.
		if _, err := os.Stat(destPath); err == nil {
			continue
		}

		content, err := embed.ReadFile(f.EmbedPath)
		if err != nil {
			return restored, fmt.Errorf("reading embedded file %s: %w", f.EmbedPath, err)
		}

		if err := os.WriteFile(destPath, content, project.FilePerm); err != nil {
			return restored, fmt.Errorf("restoring %s: %w", f.DestPath, err)
		}
		restored++
	}

	return restored, nil
}

// manageGitkeep manages a .gitkeep file in a directory:
//   - If the directory is empty (or only contains .gitkeep): ensure .gitkeep exists
//   - If the directory has real content: remove .gitkeep if it exists
//
// Returns (added, removed, error).
func manageGitkeep(dir string) (bool, bool, error) {
	gitkeepPath := filepath.Join(dir, ".gitkeep")

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil // dir doesn't exist, handled by ensureDir
		}
		return false, false, err
	}

	// Count non-.gitkeep entries.
	realContent := 0
	for _, e := range entries {
		if e.Name() != ".gitkeep" {
			realContent++
		}
	}

	if realContent == 0 {
		// Empty directory — ensure .gitkeep exists.
		if _, err := os.Stat(gitkeepPath); err == nil {
			return false, false, nil // already exists
		}
		if err := os.WriteFile(gitkeepPath, nil, project.FilePerm); err != nil {
			return false, false, err
		}
		return true, false, nil
	}

	// Directory has content — remove .gitkeep if it exists.
	if _, err := os.Stat(gitkeepPath); err == nil {
		if err := os.Remove(gitkeepPath); err != nil {
			return false, false, err
		}
		return false, true, nil
	}
	return false, false, nil
}
