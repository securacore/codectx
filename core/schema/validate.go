package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// embeddedSchemaLoader loads schemas from the embedded filesystem.
// It implements jsonschema.URLLoader to prevent the library from attempting
// filesystem lookups when schemas are embedded in the binary.
type embeddedSchemaLoader struct {
	schemas map[string]any
}

func (l *embeddedSchemaLoader) Load(url string) (any, error) {
	const prefix = "embed://"
	if !strings.HasPrefix(url, prefix) {
		return nil, fmt.Errorf("unsupported URL scheme: %s", url)
	}

	filename := strings.TrimPrefix(url, prefix)
	doc, ok := l.schemas[filename]
	if !ok {
		return nil, fmt.Errorf("schema %s not found in embedded schemas", filename)
	}
	return doc, nil
}

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

	// Create a custom loader for embedded schemas. This prevents the jsonschema
	// library from converting relative paths to file:// URLs based on the current
	// working directory, which would fail since schemas are embedded in the binary.
	loader := &embeddedSchemaLoader{
		schemas: map[string]any{
			schemaFile: doc,
		},
	}

	// Use embed:// URL scheme to signal this is an embedded schema.
	// This prevents the library from attempting filesystem lookups.
	embedURL := "embed://" + schemaFile

	c := jsonschema.NewCompiler()
	c.UseLoader(loader)

	if err := c.AddResource(embedURL, doc); err != nil {
		return fmt.Errorf("add schema resource %s: %w", schemaFile, err)
	}

	sch, err := c.Compile(embedURL)
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
