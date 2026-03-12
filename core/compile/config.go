package compile

import (
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/project"
)

// BuildConfig assembles a Config from the project, AI, and preferences
// configurations. This is a pure data-mapping function used by both the
// compile and update commands.
func BuildConfig(
	projectDir, rootDir string,
	cfg *project.Config,
	aiCfg *project.AIConfig,
	prefsCfg *project.PreferencesConfig,
) Config {
	activeDeps := make(map[string]bool)
	for name, dep := range cfg.Dependencies {
		if dep != nil && dep.Active {
			activeDeps[name] = true
		}
	}

	codectxDir := filepath.Join(rootDir, project.CodectxDir)
	compiledDir := filepath.Join(codectxDir, project.CompiledDir)

	return Config{
		ProjectDir:  projectDir,
		RootDir:     rootDir,
		CompiledDir: compiledDir,
		SystemDir:   project.SystemDir,
		Encoding:    aiCfg.Compilation.Encoding,
		Version:     project.Version,
		Chunking:    prefsCfg.Chunking,
		BM25:        prefsCfg.BM25,
		Validation:  prefsCfg.Validation,
		Taxonomy:    prefsCfg.Taxonomy,
		Model:       aiCfg.Compilation.Model,
		Provider:    aiCfg.Compilation.Provider,
		APIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		ActiveDeps:  activeDeps,
		Session:     cfg.Session,
	}
}
