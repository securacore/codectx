package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Write serializes a Manifest to YAML and writes it to the given path.
func Write(path string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest %s: %w", path, err)
	}

	return nil
}
