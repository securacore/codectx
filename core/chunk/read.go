package chunk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// metaEndMarker is the closing tag of the codectx:meta HTML comment header.
const metaEndMarker = "-->"

// ReadChunkContent reads a compiled chunk file and returns only the content
// portion (everything after the <!-- codectx:meta ... --> header).
// The compiledDir is the absolute path to the compiled directory.
func ReadChunkContent(compiledDir, chunkID string) (string, error) {
	path, err := ChunkFilePath(compiledDir, chunkID)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading chunk file %s: %w", chunkID, err)
	}

	return stripMetaHeader(string(data)), nil
}

// ChunkFilePath resolves a chunk ID to its absolute file path within the
// compiled directory. The chunk ID format is "prefix:hash.seq" (e.g.
// "obj:a1b2c3d4.3"), which maps to "compiledDir/objects/a1b2c3d4.3.md".
func ChunkFilePath(compiledDir, chunkID string) (string, error) {
	parts := strings.SplitN(chunkID, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid chunk ID format: %q", chunkID)
	}

	prefix := parts[0]
	hashSeq := parts[1]

	var dir string
	switch prefix {
	case "obj":
		dir = OutputDir(ChunkObject)
	case "spec":
		dir = OutputDir(ChunkSpec)
	case "sys":
		dir = OutputDir(ChunkSystem)
	default:
		return "", fmt.Errorf("unknown chunk prefix: %q", prefix)
	}

	return filepath.Join(compiledDir, dir, hashSeq+".md"), nil
}

// stripMetaHeader removes the <!-- codectx:meta ... --> header from chunk
// file content and returns the remaining content, trimmed of leading whitespace.
func stripMetaHeader(content string) string {
	idx := strings.Index(content, metaEndMarker)
	if idx < 0 {
		// No meta header found — return content as-is.
		return content
	}

	// Skip past "-->" and any trailing newlines before content.
	rest := content[idx+len(metaEndMarker):]
	return strings.TrimLeft(rest, "\n")
}
