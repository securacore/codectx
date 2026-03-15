package shared

import (
	"github.com/securacore/codectx/core/project"
)

// LoadPreferencesOrDefault loads preferences.yml, falling back to an empty
// config with default values if the file is missing or malformed. Warnings
// are printed to stderr. Never returns nil.
func LoadPreferencesOrDefault(projectDir string, cfg *project.Config) *project.PreferencesConfig {
	prefsCfg, err := project.LoadPreferencesConfigForProject(projectDir, cfg)
	if err != nil {
		WarnBestEffort("Loading preferences", err)
		prefsCfg = &project.PreferencesConfig{}
	}
	return prefsCfg
}
