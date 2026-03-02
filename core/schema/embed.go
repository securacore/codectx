package schema

import "embed"

//go:embed codectx.schema.json manifest.schema.json plan.schema.json compiled.schema.json heuristics.schema.json
var schemas embed.FS

// CodectxSchemaFile is the filename for the codectx.yml schema.
const CodectxSchemaFile = "codectx.schema.json"

// ManifestSchemaFile is the filename for the manifest.yml schema.
const ManifestSchemaFile = "manifest.schema.json"

// PlanSchemaFile is the filename for the plan.yml schema.
const PlanSchemaFile = "plan.schema.json"

// CompiledSchemaFile is the filename for the compiled manifest.yml schema.
const CompiledSchemaFile = "compiled.schema.json"

// HeuristicsSchemaFile is the filename for the heuristics.yml schema.
const HeuristicsSchemaFile = "heuristics.schema.json"
