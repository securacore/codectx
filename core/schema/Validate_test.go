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
