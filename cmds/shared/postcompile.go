package shared

import (
	"fmt"
	"path/filepath"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
)

// ShouldPostInitCompile determines whether auto-compilation should run
// after init or new package, based on CLI flags and the auto_compile preference.
func ShouldPostInitCompile(projectDir, root string, forceCompile, skipCompile bool) bool {
	effectiveRoot := project.ResolveRoot(root)
	codectxDir := filepath.Join(projectDir, effectiveRoot, project.CodectxDir)
	prefsCfg, err := project.LoadPreferencesConfig(filepath.Join(codectxDir, project.PreferencesFile))
	if err != nil {
		// Preferences just written by scaffold — shouldn't fail.
		// Default to compiling if it does.
		prefsCfg = &project.PreferencesConfig{}
	}

	return ShouldAutoCompile(prefsCfg, forceCompile, skipCompile, "initial compile")
}

// RunPostInitCompile loads a freshly created project configuration and runs
// the full compilation pipeline with a spinner. Used after init and new package.
func RunPostInitCompile(projectDir string) error {
	cfg, err := project.LoadConfig(filepath.Join(projectDir, project.ConfigFileName))
	if err != nil {
		return fmt.Errorf("loading project config: %w", err)
	}

	rootDir := project.RootDir(projectDir, cfg)

	aiCfg, err := project.LoadAIConfigForProject(projectDir, cfg)
	if err != nil {
		return fmt.Errorf("loading AI config: %w", err)
	}

	prefsCfg := LoadPreferencesOrDefault(projectDir, cfg)

	compileCfg := compile.BuildConfig(projectDir, rootDir, cfg, aiCfg, prefsCfg)

	fmt.Printf("\n%s Compiling documentation...\n", tui.Arrow())

	var result *compile.Result
	var compileErr error

	if sErr := RunWithSpinner("Compiling...", func() {
		result, compileErr = compile.Run(compileCfg, nil)
	}); sErr != nil {
		return sErr
	}
	if compileErr != nil {
		return compileErr
	}

	// Print compact summary.
	fmt.Print(RenderCompactCompileSummary(result))

	return nil
}
