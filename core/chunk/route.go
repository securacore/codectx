package chunk

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/core/project"
)

// OutputDir returns the compiled subdirectory name for a chunk type.
// Returns "objects", "specs", or "system".
func OutputDir(ct ChunkType) string {
	return lookupMeta(ct).outDir
}

// OutputFilename returns the filename for a chunk file: [hash].[seq].md.
// The hash is extracted from the chunk ID (after the prefix and colon).
// Returns an empty string for nil chunks.
func OutputFilename(c *Chunk) string {
	if c == nil {
		return ""
	}
	// ID format: "prefix:hash.seq"
	// We want "hash.seq.md"
	parts := strings.SplitN(c.ID, ":", 2)
	if len(parts) != 2 {
		return fmt.Sprintf("unknown.%d.md", c.Sequence)
	}
	return parts[1] + ".md"
}

// CompiledOutputDirs returns the list of all compiled output subdirectory
// paths (chunk dirs + BM25 dirs) relative to the compiled directory.
// Used by both scaffold (to create dirs) and compile (to clean/recreate).
func CompiledOutputDirs() []string {
	types := []ChunkType{ChunkObject, ChunkSpec, ChunkSystem}
	dirs := make([]string, 0, len(types)*2)
	for _, ct := range types {
		dirs = append(dirs, OutputDir(ct))
		dirs = append(dirs, filepath.Join(project.BM25Dir, OutputDir(ct)))
	}
	return dirs
}

// ClassifySource determines the ChunkType from a source file path.
//
// Rules (in priority order):
//  1. Any file ending in .spec.md → ChunkSpec
//  2. Files under the system directory (non-spec) → ChunkSystem
//  3. All other .md files → ChunkObject
//
// The systemDir parameter is the system directory name relative to the
// documentation root (typically "system").
func ClassifySource(sourcePath string, systemDir string) ChunkType {
	// Normalize to forward slashes for consistent matching.
	normalized := filepath.ToSlash(sourcePath)

	// Rule 1: spec files always produce spec chunks.
	if strings.HasSuffix(normalized, SpecFileSuffix) {
		return ChunkSpec
	}

	// Rule 2: files under system/ directory (non-spec) produce system chunks.
	if systemDir != "" {
		prefix := systemDir + "/"
		if strings.HasPrefix(normalized, prefix) || normalized == systemDir {
			return ChunkSystem
		}
	}

	// Rule 3: everything else is an object chunk.
	return ChunkObject
}
