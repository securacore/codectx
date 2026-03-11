package chunk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/core/markdown"
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

// LoadedChunk holds the metadata needed to reconstruct a Chunk from a
// manifest entry. This is used during incremental compilation to load
// unchanged chunks from disk without re-parsing and re-chunking.
type LoadedChunk struct {
	ID          string
	Type        ChunkType
	Source      string
	Heading     string
	Sequence    int
	TotalInFile int
	Tokens      int
}

// LoadChunkFromDisk reads a compiled chunk file and combines it with
// manifest metadata to produce a Chunk struct suitable for BM25 indexing,
// taxonomy extraction, and manifest rebuilding.
//
// The content is read from disk and reconstructed into blocks for taxonomy
// extraction. Since the on-disk content is stripped markdown (headings lack
// # markers), we synthesize a heading block from the chunk's Heading field
// and re-parse the remaining content to recover paragraph, code, and list
// blocks.
func LoadChunkFromDisk(compiledDir string, lc LoadedChunk) (Chunk, error) {
	content, err := ReadChunkContent(compiledDir, lc.ID)
	if err != nil {
		return Chunk{}, fmt.Errorf("loading chunk %s: %w", lc.ID, err)
	}

	// Reconstruct blocks for taxonomy extraction.
	blocks := reconstructBlocks(lc.Heading, content)

	return Chunk{
		ID:          lc.ID,
		Type:        lc.Type,
		Source:      lc.Source,
		Heading:     lc.Heading,
		Sequence:    lc.Sequence,
		TotalInFile: lc.TotalInFile,
		Tokens:      lc.Tokens,
		Content:     content,
		Blocks:      blocks,
	}, nil
}

// reconstructBlocks builds a block slice from a chunk's heading and content.
//
// The heading field is used to synthesize heading blocks (the last segment
// of the breadcrumb is the primary heading). The content is re-parsed to
// recover paragraph, code, list, and table blocks. Since the content is
// stripped markdown (headings lack # markers), any headings in the content
// are added back with appropriate markers before parsing.
func reconstructBlocks(heading, content string) []markdown.Block {
	var blocks []markdown.Block

	// Add a heading block from the chunk's heading breadcrumb.
	// Use the last segment as the heading text (most specific).
	if heading != "" {
		headingText := heading
		if idx := strings.LastIndex(heading, HeadingSeparator); idx >= 0 {
			headingText = heading[idx+len(HeadingSeparator):]
		}
		blocks = append(blocks, markdown.Block{
			Type:    markdown.BlockHeading,
			Content: headingText,
			Level:   1,
		})
	}

	// Re-parse the content to recover non-heading blocks.
	if content != "" {
		doc := markdown.Parse([]byte(content))
		blocks = append(blocks, doc.Blocks...)
	}

	return blocks
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
