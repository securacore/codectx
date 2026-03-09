package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Write serializes a Config to YAML and writes it to the given path.
func Write(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}

	return nil
}
