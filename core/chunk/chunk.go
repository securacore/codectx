// Package chunk implements semantic block accumulation with token-based
// windows. It splits parsed markdown documents into chunks suitable for
// BM25 indexing and AI context assembly.
//
// The chunking algorithm walks an ordered list of semantic blocks (paragraphs,
// code blocks, lists, tables, blockquotes), accumulating them into chunks
// that target a configurable token count. It never splits within an atomic
// block. Headings always start new chunks when possible.
//
// Chunk IDs are derived from content hashes (SHA-256), making them stable
// across machines given identical source + preferences + tokenizer encoding.
package chunk

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/markdown"
	"github.com/securacore/codectx/core/project"
)

// ChunkType identifies the routing category for a chunk.
type ChunkType string

const (
	// ChunkObject is an instruction/documentation chunk from .md files.
	ChunkObject ChunkType = "object"

	// ChunkSpec is a reasoning chunk from .spec.md files.
	ChunkSpec ChunkType = "spec"

	// ChunkSystem is a system/compiler documentation chunk.
	ChunkSystem ChunkType = "system"
)

// HeadingSeparator is the delimiter used in heading breadcrumb strings.
// E.g. "Authentication > JWT Tokens > Refresh Flow".
const HeadingSeparator = " > "

// SpecFileSuffix is the file extension that identifies spec (reasoning) files.
const SpecFileSuffix = ".spec.md"

// chunkTypeMeta maps each ChunkType to its ID prefix and output directory.
// This is the single source of truth — idPrefix() and OutputDir() both delegate here.
type chunkTypeMeta struct {
	prefix string
	outDir string
}

var chunkTypeRegistry = map[ChunkType]chunkTypeMeta{
	ChunkObject: {prefix: "obj", outDir: "objects"},
	ChunkSpec:   {prefix: "spec", outDir: "specs"},
	ChunkSystem: {prefix: "sys", outDir: "system"},
}

// defaultChunkTypeMeta is used when a ChunkType is not in the registry.
var defaultChunkTypeMeta = chunkTypeMeta{prefix: "obj", outDir: "objects"}

// lookupMeta returns the metadata for a chunk type, falling back to defaults.
func lookupMeta(ct ChunkType) chunkTypeMeta {
	if m, ok := chunkTypeRegistry[ct]; ok {
		return m
	}
	return defaultChunkTypeMeta
}

// Chunk represents a compiled chunk of semantic blocks with metadata.
type Chunk struct {
	// ID is the unique chunk identifier, e.g. "obj:a1b2c3d4e5f67890.3".
	ID string

	// Type is the routing category (object, spec, system).
	Type ChunkType

	// Source is the relative path of the source file.
	Source string

	// Heading is the heading breadcrumb at the chunk start position,
	// e.g. "Authentication > JWT Tokens > Refresh Flow".
	Heading string

	// Sequence is the 1-based chunk index within the source file.
	Sequence int

	// TotalInFile is the total number of chunks from the source file.
	// Set after all chunks for the file have been generated.
	TotalInFile int

	// Tokens is the token count of the chunk content (excluding meta header).
	Tokens int

	// Blocks is the ordered list of semantic blocks in this chunk.
	Blocks []markdown.Block

	// Content is the joined block content text.
	Content string

	// Oversized is true if this chunk contains a single atomic block
	// that exceeds max_tokens on its own.
	Oversized bool
}

// ContentHash computes a truncated SHA-256 hex digest of the given content.
// The hashLength parameter controls how many hex characters to use,
// clamped to [MinHashLength, MaxHashLength].
//
// Delegates to markdown.Hash for the full SHA-256 computation, then truncates
// to the requested length.
func ContentHash(content string, hashLength int) string {
	hashLength = project.ClampHashLength(hashLength)
	full := markdown.Hash([]byte(content))
	return full[:hashLength]
}

// FormatID produces a chunk ID in the format "prefix:hash.seq".
// The prefix is derived from the chunk type: obj, spec, or sys.
func FormatID(ct ChunkType, hash string, seq int) string {
	return fmt.Sprintf("%s:%s.%d", idPrefix(ct), hash, seq)
}

// idPrefix returns the short ID prefix for a chunk type.
func idPrefix(ct ChunkType) string {
	return lookupMeta(ct).prefix
}

// FormatHeading joins a heading hierarchy into a breadcrumb string.
// E.g. ["Authentication", "JWT", "Refresh"] becomes "Authentication > JWT > Refresh".
// Returns empty string for nil or empty input.
func FormatHeading(hierarchy []string) string {
	if len(hierarchy) == 0 {
		return ""
	}
	return strings.Join(hierarchy, HeadingSeparator)
}

// JoinContent concatenates block content with double-newline separators.
// This is the canonical way to produce chunk content from blocks.
func JoinContent(blocks []markdown.Block) string {
	if len(blocks) == 0 {
		return ""
	}
	parts := make([]string, len(blocks))
	for i, b := range blocks {
		parts[i] = b.Content
	}
	return strings.Join(parts, "\n\n")
}
