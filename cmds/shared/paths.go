package shared

import (
	"path/filepath"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/registry"
)

// RegistryPaths holds resolved paths for package operations.
type RegistryPaths struct {
	RootDir     string
	LockPath    string
	PackagesDir string
	Registry    string
}

// ResolveRegistryPaths computes all paths needed for package operations.
func ResolveRegistryPaths(projectDir string, cfg *project.Config) RegistryPaths {
	rootDir := project.RootDir(projectDir, cfg)
	return RegistryPaths{
		RootDir:     rootDir,
		LockPath:    filepath.Join(rootDir, registry.LockFileName),
		PackagesDir: project.PackagesPath(rootDir),
		Registry:    cfg.EffectiveRegistry(),
	}
}
