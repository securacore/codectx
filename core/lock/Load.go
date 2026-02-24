package lock

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a codectx.lock file from the given path.
// Returns nil, nil if the file does not exist.
func Load(path string) (*Lock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read lock %s: %w", path, err)
	}

	var l Lock
	if err := yaml.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parse lock %s: %w", path, err)
	}

	return &l, nil
}
