package config

import "fmt"

// PackageDep represents a documentation package dependency in codectx.yml.
type PackageDep struct {
	Name    string     `yaml:"name"`
	Author  string     `yaml:"author"`
	Version string     `yaml:"version"`
	Source  string     `yaml:"source,omitempty"`
	Active  Activation `yaml:"active,omitempty"`
}

// Activation represents the activation state of a package.
// It can be a string ("all" or "none") or a granular ActivationMap.
type Activation struct {
	// Mode is set when active is a string: "all" or "none".
	Mode string

	// Map is set when active is a granular activation object.
	Map *ActivationMap
}

// ActivationMap provides granular activation by section.
type ActivationMap struct {
	Foundation  []string `yaml:"foundation,omitempty"`
	Application []string `yaml:"application,omitempty"`
	Topics      []string `yaml:"topics,omitempty"`
	Prompts     []string `yaml:"prompts,omitempty"`
	Plans       []string `yaml:"plans,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling for Activation.
// It handles both string values ("all", "none") and object values (ActivationMap).
func (a *Activation) UnmarshalYAML(unmarshal func(any) error) error {
	var mode string
	if err := unmarshal(&mode); err == nil {
		if mode != "all" && mode != "none" {
			return fmt.Errorf("invalid activation mode %q: must be \"all\" or \"none\"", mode)
		}
		a.Mode = mode
		return nil
	}

	var m ActivationMap
	if err := unmarshal(&m); err != nil {
		return fmt.Errorf("invalid activation value: must be \"all\", \"none\", or an activation map: %w", err)
	}
	a.Map = &m
	return nil
}

// MarshalYAML implements custom YAML marshaling for Activation.
func (a Activation) MarshalYAML() (any, error) {
	if a.Map != nil {
		return a.Map, nil
	}
	if a.Mode != "" {
		return a.Mode, nil
	}
	return "none", nil
}

// IsAll returns true if the activation mode is "all".
func (a *Activation) IsAll() bool {
	return a.Mode == "all"
}

// IsNone returns true if the activation is "none" or unset.
func (a *Activation) IsNone() bool {
	return a.Map == nil && (a.Mode == "" || a.Mode == "none")
}

// IsGranular returns true if the activation uses a granular map.
func (a *Activation) IsGranular() bool {
	return a.Map != nil
}
