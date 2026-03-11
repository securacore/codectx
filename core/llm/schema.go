// Package llm implements LLM augmentation for the codectx compilation pipeline.
//
// Two LLM-powered tasks run during compilation:
//
//  1. Alias generation: For each canonical taxonomy term, generate semantic
//     aliases (abbreviations, synonyms, related acronyms) via batched API calls.
//
//  2. Bridge summaries: For each pair of adjacent chunks from the same source
//     file, generate a one-line semantic bridge describing the knowledge that
//     carries forward across the boundary.
//
// Both tasks support two providers:
//
//   - "api": Direct Anthropic Messages API via the official Go SDK. Requires
//     ANTHROPIC_API_KEY environment variable.
//   - "cli": Local Claude CLI binary invoked in headless mode with structured
//     JSON output. Requires the claude binary on PATH.
//
// The LLM augmentation stage degrades gracefully. If the provider is unavailable,
// the API key is missing, or any API call fails, compilation completes
// successfully with empty aliases and nil bridges. The LLM pass never causes
// a compile failure.
package llm

// AliasResponse is the structured output from an alias generation batch.
// Both the API provider (via tool-use) and the CLI provider (via --json-schema)
// return data in this shape.
type AliasResponse struct {
	Terms []AliasTermResponse `json:"terms"`
}

// AliasTermResponse holds the generated aliases for a single taxonomy term.
type AliasTermResponse struct {
	// Key is the normalized term key (e.g. "authentication").
	Key string `json:"key"`

	// Aliases are the generated alternative labels for the term.
	Aliases []string `json:"aliases"`
}

// BridgeResponse is the structured output from a bridge generation batch.
type BridgeResponse struct {
	Bridges []BridgeEntryResponse `json:"bridges"`
}

// BridgeEntryResponse holds the generated bridge summary for a single chunk boundary.
type BridgeEntryResponse struct {
	// ChunkID is the ID of the "from" chunk (e.g. "obj:a1b2c3.01").
	ChunkID string `json:"chunk_id"`

	// Summary is the one-line bridge text (under 30 words, past tense).
	Summary string `json:"summary"`
}

// aliasJSONSchema is the JSON Schema string for alias responses.
// Used by the CLI provider with --json-schema.
const aliasJSONSchema = `{
  "type": "object",
  "properties": {
    "terms": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "key": {"type": "string"},
          "aliases": {"type": "array", "items": {"type": "string"}}
        },
        "required": ["key", "aliases"]
      }
    }
  },
  "required": ["terms"]
}`

// bridgeJSONSchema is the JSON Schema string for bridge responses.
// Used by the CLI provider with --json-schema.
const bridgeJSONSchema = `{
  "type": "object",
  "properties": {
    "bridges": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "chunk_id": {"type": "string"},
          "summary": {"type": "string"}
        },
        "required": ["chunk_id", "summary"]
      }
    }
  },
  "required": ["bridges"]
}`
