package manifest

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Write serializes a Manifest to YAML and writes it to the given path.
// If the file already exists with identical content, the write is skipped
// to avoid unnecessary filesystem events (important for watch mode).
func Write(path string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	// Skip write if the file already has identical content.
	if existing, readErr := os.ReadFile(path); readErr == nil {
		if bytes.Equal(data, existing) {
			return nil
		}
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest %s: %w", path, err)
	}

	return nil
}
