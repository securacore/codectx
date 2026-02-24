package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_validCodectx(t *testing.T) {
	v := map[string]any{
		"name":     "test-project",
		"packages": []any{},
	}
	err := Validate(CodectxSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_validCodectxWithPackages(t *testing.T) {
	v := map[string]any{
		"name": "test-project",
		"packages": []any{
			map[string]any{
				"name":    "react",
				"author":  "facebook",
				"version": "^1.0.0",
				"active":  "all",
			},
		},
	}
	err := Validate(CodectxSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_invalidCodectx_missingName(t *testing.T) {
	v := map[string]any{
		"packages": []any{},
	}
	err := Validate(CodectxSchemaFile, v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidate_invalidCodectx_missingPackages(t *testing.T) {
	v := map[string]any{
		"name": "test",
	}
	err := Validate(CodectxSchemaFile, v)
	assert.Error(t, err)
}

func TestValidate_invalidCodectx_additionalProperties(t *testing.T) {
	v := map[string]any{
		"name":     "test",
		"packages": []any{},
		"extra":    "not allowed",
	}
	err := Validate(CodectxSchemaFile, v)
	assert.Error(t, err)
}

func TestValidate_validPackage(t *testing.T) {
	v := map[string]any{
		"name":        "test-pkg",
		"author":      "test-author",
		"version":     "1.0.0",
		"description": "A test package",
	}
	err := Validate(PackageSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_validPackageWithEntries(t *testing.T) {
	v := map[string]any{
		"name":        "test-pkg",
		"author":      "test-author",
		"version":     "1.0.0",
		"description": "A test package",
		"foundation": []any{
			map[string]any{
				"id":          "philosophy",
				"path":        "foundation/philosophy.md",
				"description": "Core philosophy",
				"load":        "always",
			},
		},
		"topics": []any{
			map[string]any{
				"id":          "react",
				"path":        "topics/react/README.md",
				"description": "React conventions",
			},
		},
	}
	err := Validate(PackageSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_invalidPackage_missingRequired(t *testing.T) {
	v := map[string]any{
		"name": "test-pkg",
		// missing author, version, description
	}
	err := Validate(PackageSchemaFile, v)
	assert.Error(t, err)
}

func TestValidate_unknownSchema(t *testing.T) {
	v := map[string]any{"name": "test"}
	err := Validate("nonexistent.schema.json", v)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read embedded schema")
}

func TestValidate_validState(t *testing.T) {
	v := map[string]any{
		"plan":    "migration",
		"status":  "in_progress",
		"summary": "halfway done",
	}
	err := Validate(StateSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_invalidState_badStatus(t *testing.T) {
	v := map[string]any{
		"plan":    "migration",
		"status":  "unknown",
		"summary": "halfway done",
	}
	err := Validate(StateSchemaFile, v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidate_packageWithPromptsPlans(t *testing.T) {
	v := map[string]any{
		"name":        "test-pkg",
		"author":      "test-author",
		"version":     "1.0.0",
		"description": "A test package",
		"prompts": []any{
			map[string]any{
				"id":          "review",
				"path":        "prompts/review/README.md",
				"description": "Code review prompt",
			},
		},
		"plans": []any{
			map[string]any{
				"id":          "migration",
				"path":        "plans/migration/README.md",
				"state":       "plans/migration/state.yml",
				"description": "Migration plan",
			},
		},
	}
	err := Validate(PackageSchemaFile, v)
	assert.NoError(t, err)
}

func TestNormalizeForJSONSchema_unmarshalable(t *testing.T) {
	// Channels cannot be marshaled to JSON.
	ch := make(chan int)
	_, err := normalizeForJSONSchema(ch)
	assert.Error(t, err)
}

func TestNormalizeForJSONSchema_roundTrip(t *testing.T) {
	// Integers from YAML should become float64 after normalization.
	input := map[string]any{
		"count": 42,
		"name":  "test",
	}
	result, err := normalizeForJSONSchema(input)
	require.NoError(t, err)

	m, ok := result.(map[string]any)
	require.True(t, ok)
	// JSON round-trip converts int to float64.
	assert.Equal(t, float64(42), m["count"])
	assert.Equal(t, "test", m["name"])
}

func TestValidate_packageWithDependsOn(t *testing.T) {
	v := map[string]any{
		"name":        "test-pkg",
		"author":      "test-author",
		"version":     "1.0.0",
		"description": "A test package",
		"foundation": []any{
			map[string]any{
				"id":          "conventions",
				"path":        "foundation/conventions.md",
				"description": "Coding conventions",
				"depends_on":  []any{"other"},
			},
		},
	}
	err := Validate(PackageSchemaFile, v)
	assert.NoError(t, err)
}
