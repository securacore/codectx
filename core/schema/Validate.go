package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validate validates a Go value against the named schema file.
// The value should be the result of unmarshaling YAML into any.
func Validate(schemaFile string, v any) error {
	raw, err := schemas.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("read embedded schema %s: %w", schemaFile, err)
	}

	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(raw)))
	if err != nil {
		return fmt.Errorf("parse schema %s: %w", schemaFile, err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource(schemaFile, doc); err != nil {
		return fmt.Errorf("add schema resource %s: %w", schemaFile, err)
	}

	sch, err := c.Compile(schemaFile)
	if err != nil {
		return fmt.Errorf("compile schema %s: %w", schemaFile, err)
	}

	// YAML unmarshals numbers as int or float64 depending on format.
	// JSON Schema expects float64 for numbers. Normalize via JSON round-trip.
	normalized, err := normalizeForJSONSchema(v)
	if err != nil {
		return fmt.Errorf("normalize value for schema validation: %w", err)
	}

	if err := sch.Validate(normalized); err != nil {
		return fmt.Errorf("validation failed against %s: %w", schemaFile, err)
	}

	return nil
}

// normalizeForJSONSchema converts a YAML-unmarshaled value to JSON-compatible
// types by round-tripping through JSON. This ensures integers become float64
// and map keys become strings, matching what JSON Schema validators expect.
func normalizeForJSONSchema(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}
