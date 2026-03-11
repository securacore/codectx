package llm

import (
	"testing"
)

// TestSchemaFromJSON_AliasSchema verifies the alias JSON schema parses correctly.
func TestSchemaFromJSON_AliasSchema(t *testing.T) {
	schema, err := schemaFromJSON(aliasJSONSchema)
	if err != nil {
		t.Fatalf("schemaFromJSON(aliasJSONSchema): %v", err)
	}

	if schema.Properties == nil {
		t.Fatal("expected non-nil Properties")
	}

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatalf("expected Properties to be map[string]any, got %T", schema.Properties)
	}

	terms, ok := props["terms"]
	if !ok {
		t.Fatal("expected 'terms' property in schema")
	}

	termsMap, ok := terms.(map[string]any)
	if !ok {
		t.Fatalf("expected terms to be map[string]any, got %T", terms)
	}
	if termsMap["type"] != "array" {
		t.Errorf("expected terms type 'array', got %v", termsMap["type"])
	}
}

// TestSchemaFromJSON_BridgeSchema verifies the bridge JSON schema parses correctly.
func TestSchemaFromJSON_BridgeSchema(t *testing.T) {
	schema, err := schemaFromJSON(bridgeJSONSchema)
	if err != nil {
		t.Fatalf("schemaFromJSON(bridgeJSONSchema): %v", err)
	}

	if schema.Properties == nil {
		t.Fatal("expected non-nil Properties")
	}

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatalf("expected Properties to be map[string]any, got %T", schema.Properties)
	}

	bridges, ok := props["bridges"]
	if !ok {
		t.Fatal("expected 'bridges' property in schema")
	}

	bridgesMap, ok := bridges.(map[string]any)
	if !ok {
		t.Fatalf("expected bridges to be map[string]any, got %T", bridges)
	}
	if bridgesMap["type"] != "array" {
		t.Errorf("expected bridges type 'array', got %v", bridgesMap["type"])
	}
}

// TestSchemaFromJSON_InvalidJSON verifies error handling for invalid JSON.
func TestSchemaFromJSON_InvalidJSON(t *testing.T) {
	_, err := schemaFromJSON("not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestSchemaFromJSON_NoProperties verifies handling of schema with no properties field.
func TestSchemaFromJSON_NoProperties(t *testing.T) {
	schema, err := schemaFromJSON(`{"type": "object"}`)
	if err != nil {
		t.Fatalf("schemaFromJSON: %v", err)
	}

	// When "properties" is absent from JSON, unmarshaling produces nil map.
	props, ok := schema.Properties.(map[string]any)
	if ok && len(props) > 0 {
		t.Errorf("expected empty or nil Properties, got %v", schema.Properties)
	}
}
