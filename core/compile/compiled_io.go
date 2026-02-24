package compile

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// WriteCompiledManifest serializes a CompiledManifest to YAML
// and writes it to the given path.
func WriteCompiledManifest(path string, m *CompiledManifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal compiled manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write compiled manifest %s: %w", path, err)
	}

	return nil
}

// LoadCompiledManifest reads and parses a compiled manifest.yml file.
func LoadCompiledManifest(path string) (*CompiledManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read compiled manifest %s: %w", path, err)
	}

	var m CompiledManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal compiled manifest %s: %w", path, err)
	}

	return &m, nil
}
