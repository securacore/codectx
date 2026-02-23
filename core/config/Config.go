package config

// Config represents the codectx.yml configuration file.
// It is the sole source of truth for a project's documentation setup.
type Config struct {
	Name     string       `yaml:"name"`
	Config   *BuildConfig `yaml:"config,omitempty"`
	Packages []PackageDep `yaml:"packages"`
}

// BuildConfig holds build and directory configuration.
type BuildConfig struct {
	DocsDir   string `yaml:"docs_dir,omitempty"`
	OutputDir string `yaml:"output_dir,omitempty"`
}

// DocsDir returns the configured documentation source directory,
// defaulting to "docs" when not set.
func (c *Config) DocsDir() string {
	if c.Config != nil && c.Config.DocsDir != "" {
		return c.Config.DocsDir
	}
	return "docs"
}

// OutputDir returns the configured compiled output directory,
// defaulting to ".codectx" when not set.
func (c *Config) OutputDir() string {
	if c.Config != nil && c.Config.OutputDir != "" {
		return c.Config.OutputDir
	}
	return ".codectx"
}
