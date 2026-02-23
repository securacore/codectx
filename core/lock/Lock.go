package lock

import "securacore/codectx/core/config"

// Lock represents the codectx.lock file.
// It records the exact resolved state from a compile so the output
// can be reproduced deterministically.
type Lock struct {
	CompiledAt string          `yaml:"compiled_at"`
	Packages   []LockedPackage `yaml:"packages"`
}

// LockedPackage records the resolved state of a single package dependency.
type LockedPackage struct {
	Name    string            `yaml:"name"`
	Author  string            `yaml:"author"`
	Version string            `yaml:"version"`
	Source  string            `yaml:"source,omitempty"`
	Active  config.Activation `yaml:"active"`
}
