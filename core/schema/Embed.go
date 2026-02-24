package schema

import "embed"

//go:embed codectx.schema.json package.schema.json state.schema.json compiled.schema.json heuristics.schema.json
var schemas embed.FS

// CodectxSchemaFile is the filename for the codectx.yml schema.
const CodectxSchemaFile = "codectx.schema.json"

// PackageSchemaFile is the filename for the package.yml schema.
const PackageSchemaFile = "package.schema.json"

// StateSchemaFile is the filename for the state.yml schema.
const StateSchemaFile = "state.schema.json"

// CompiledSchemaFile is the filename for the compiled manifest.yml schema.
const CompiledSchemaFile = "compiled.schema.json"

// HeuristicsSchemaFile is the filename for the heuristics.yml schema.
const HeuristicsSchemaFile = "heuristics.schema.json"
