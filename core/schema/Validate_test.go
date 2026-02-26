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
	err := Validate(ManifestSchemaFile, v)
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
	err := Validate(ManifestSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_invalidPackage_missingRequired(t *testing.T) {
	v := map[string]any{
		"name": "test-pkg",
		// missing author, version, description
	}
	err := Validate(ManifestSchemaFile, v)
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
	err := Validate(PlanSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_invalidState_badStatus(t *testing.T) {
	v := map[string]any{
		"plan":    "migration",
		"status":  "unknown",
		"summary": "halfway done",
	}
	err := Validate(PlanSchemaFile, v)
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
				"plan_state":  "plans/migration/plan.yml",
				"description": "Migration plan",
			},
		},
	}
	err := Validate(ManifestSchemaFile, v)
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
	err := Validate(ManifestSchemaFile, v)
	assert.NoError(t, err)
}

// --- Compiled manifest schema ---

func TestValidate_validCompiledManifest(t *testing.T) {
	v := map[string]any{
		"name":        "test-project",
		"description": "Compiled output",
		"foundation": []any{
			map[string]any{
				"id":          "philosophy",
				"object":      "objects/a1b2c3d4e5f67890.md",
				"description": "Core philosophy",
				"load":        "always",
				"source":      "local",
			},
		},
		"topics": []any{
			map[string]any{
				"id":          "go",
				"object":      "objects/b2c3d4e5f6789012.md",
				"description": "Go conventions",
				"source":      "go@org",
				"spec":        "objects/c3d4e5f678901234.md",
			},
		},
	}
	err := Validate(CompiledSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_compiledManifest_withManifestRefs(t *testing.T) {
	v := map[string]any{
		"name": "decomposed-project",
		"foundation": []any{
			map[string]any{
				"id":          "philosophy",
				"object":      "objects/a1b2c3d4e5f67890.md",
				"description": "Philosophy",
				"load":        "always",
				"source":      "local",
			},
		},
		"manifests": []any{
			map[string]any{
				"section":          "topics",
				"path":             "manifests/topics.yml",
				"entries":          25,
				"estimated_tokens": 30000,
				"description":      "Technology and domain conventions",
			},
		},
	}
	err := Validate(CompiledSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_compiledManifest_invalidObjectPath(t *testing.T) {
	v := map[string]any{
		"name": "test",
		"foundation": []any{
			map[string]any{
				"id":          "a",
				"object":      "bad/path.md",
				"description": "A",
				"source":      "local",
			},
		},
	}
	err := Validate(CompiledSchemaFile, v)
	assert.Error(t, err)
}

func TestValidate_compiledManifest_missingSource(t *testing.T) {
	v := map[string]any{
		"name": "test",
		"foundation": []any{
			map[string]any{
				"id":          "a",
				"object":      "objects/a1b2c3d4e5f67890.md",
				"description": "A",
				// source is required
			},
		},
	}
	err := Validate(CompiledSchemaFile, v)
	assert.Error(t, err)
}

// --- Heuristics schema ---

func TestValidate_validHeuristics(t *testing.T) {
	v := map[string]any{
		"compiled_at": "2026-02-23T12:00:00Z",
		"totals": map[string]any{
			"entries":          10,
			"objects":          8,
			"size_bytes":       50000,
			"estimated_tokens": 12500,
			"always_load":      2,
		},
		"sections": map[string]any{
			"foundation": map[string]any{
				"entries":          3,
				"size_bytes":       15000,
				"estimated_tokens": 3750,
				"always_load":      2,
			},
			"topics": map[string]any{
				"entries":          5,
				"size_bytes":       25000,
				"estimated_tokens": 6250,
			},
		},
		"packages": []any{
			map[string]any{
				"name":             "local",
				"entries":          6,
				"size_bytes":       30000,
				"estimated_tokens": 7500,
			},
		},
	}
	err := Validate(HeuristicsSchemaFile, v)
	assert.NoError(t, err)
}

func TestValidate_heuristics_missingTotals(t *testing.T) {
	v := map[string]any{
		"compiled_at": "2026-02-23T12:00:00Z",
		// missing totals
		"sections": map[string]any{},
		"packages": []any{},
	}
	err := Validate(HeuristicsSchemaFile, v)
	assert.Error(t, err)
}

func TestValidate_heuristics_emptySections(t *testing.T) {
	v := map[string]any{
		"compiled_at": "2026-02-23T12:00:00Z",
		"totals": map[string]any{
			"entries":          0,
			"objects":          0,
			"size_bytes":       0,
			"estimated_tokens": 0,
			"always_load":      0,
		},
		"sections": map[string]any{},
		"packages": []any{},
	}
	err := Validate(HeuristicsSchemaFile, v)
	assert.NoError(t, err)
}

func TestEmbeddedSchemas_readable(t *testing.T) {
	schemaConstants := []string{
		CodectxSchemaFile,
		ManifestSchemaFile,
		PlanSchemaFile,
		CompiledSchemaFile,
		HeuristicsSchemaFile,
	}
	for _, name := range schemaConstants {
		t.Run(name, func(t *testing.T) {
			data, err := schemas.ReadFile(name)
			require.NoError(t, err, "embedded schema %s should be readable", name)
			assert.True(t, len(data) > 0, "embedded schema %s should not be empty", name)
		})
	}
}
