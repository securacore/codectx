// Package preferences manages user-specific settings stored in
// .codectx/preferences.yml. These settings are personal, not checked
// into version control (.codectx/ is gitignored).
package preferences

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const fileName = "preferences.yml"

// Preferences holds user-specific settings for the project.
// Pointer fields distinguish "unset" from "false".
type Preferences struct {
	AutoCompile *bool `yaml:"auto_compile,omitempty"`
}

// Load reads preferences from the output directory.
// Returns a zero Preferences (all fields nil) if the file does not exist.
func Load(outputDir string) (*Preferences, error) {
	path := filepath.Join(outputDir, fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Preferences{}, nil
		}
		return nil, fmt.Errorf("read preferences: %w", err)
	}

	var p Preferences
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse preferences: %w", err)
	}

	return &p, nil
}

// Write saves preferences to the output directory.
// Creates the directory if it does not exist.
func Write(outputDir string, p *Preferences) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}

	path := filepath.Join(outputDir, fileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write preferences: %w", err)
	}

	return nil
}

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(v bool) *bool {
	return &v
}
