package config

import (
	"fmt"
	"os"

	"github.com/securacore/codectx/core/schema"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a codectx.yml file from the given path.
// It validates the parsed content against the codectx JSON schema.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	// Parse into any for schema validation.
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if err := schema.Validate(schema.CodectxSchemaFile, raw); err != nil {
		return nil, fmt.Errorf("validate config %s: %w", path, err)
	}

	// Parse into typed struct.
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config %s: %w", path, err)
	}

	return &cfg, nil
}
