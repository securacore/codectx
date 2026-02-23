package manifest

import (
	"fmt"
	"os"

	"securacore/codectx/core/schema"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a package.yml file from the given path.
// It validates the parsed content against the package JSON schema.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}

	// Parse into any for schema validation.
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	if err := schema.Validate(schema.PackageSchemaFile, raw); err != nil {
		return nil, fmt.Errorf("validate manifest %s: %w", path, err)
	}

	// Parse into typed struct.
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal manifest %s: %w", path, err)
	}

	return &m, nil
}
