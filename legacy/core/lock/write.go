package lock

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Write serializes a Lock to YAML and writes it to the given path.
func Write(path string, l *Lock) error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return fmt.Errorf("marshal lock: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write lock %s: %w", path, err)
	}

	return nil
}
