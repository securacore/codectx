package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/securacore/codectx/core/manifest"
)

// DocumentBlock represents a parsed <document> tag from AI output.
type DocumentBlock struct {
	Path    string // Relative path (e.g., docs/topics/example/README.md)
	Content string // Document content
}

// documentTagOpen matches <document path="...">.
var documentTagOpen = regexp.MustCompile(`<document\s+path="([^"]+)"[^>]*>`)

// ParseDocumentBlocks extracts <document path="...">content</document> blocks
// from the AI's response text. Returns an empty slice if no blocks are found.
func ParseDocumentBlocks(text string) []DocumentBlock {
	var blocks []DocumentBlock

	remaining := text
	for {
		loc := documentTagOpen.FindStringSubmatchIndex(remaining)
		if loc == nil {
			break
		}

		path := remaining[loc[2]:loc[3]]
		afterOpen := remaining[loc[1]:]

		closeIdx := strings.Index(afterOpen, "</document>")
		if closeIdx == -1 {
			// Unclosed tag: take everything remaining as content.
			content := strings.TrimSpace(afterOpen)
			blocks = append(blocks, DocumentBlock{Path: path, Content: content})
			break
		}

		content := strings.TrimSpace(afterOpen[:closeIdx])
		blocks = append(blocks, DocumentBlock{Path: path, Content: content})

		remaining = afterOpen[closeIdx+len("</document>"):]
	}

	return blocks
}

// WriteDocuments writes document blocks to disk, creating directories as needed.
// The rootDir is the project root (parent of docs/). Each block's Path is
// relative to rootDir. Returns the list of files written.
func WriteDocuments(rootDir string, blocks []DocumentBlock) ([]string, error) {
	var written []string

	for _, block := range blocks {
		fullPath := filepath.Join(rootDir, block.Path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0o755); err != nil {
			return written, fmt.Errorf("create directory %s: %w", dir, err)
		}

		// Ensure content ends with a newline.
		content := block.Content
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return written, fmt.Errorf("write %s: %w", block.Path, err)
		}

		written = append(written, block.Path)
	}

	return written, nil
}

// WriteAndSync writes document blocks to disk, then syncs the manifest to
// discover the new entries. The docsDir is the documentation directory
// (e.g., "docs/"). Returns the list of files written.
func WriteAndSync(rootDir, docsDir string, blocks []DocumentBlock) ([]string, error) {
	written, err := WriteDocuments(rootDir, blocks)
	if err != nil {
		return written, err
	}

	// Sync manifest to discover new entries.
	manifestPath := filepath.Join(docsDir, "manifest.yml")
	m, loadErr := manifest.Load(manifestPath)
	if loadErr != nil {
		// Manifest may not exist yet; start fresh.
		m = &manifest.Manifest{}
	}

	synced := manifest.Sync(docsDir, m)
	if err := manifest.Write(manifestPath, synced); err != nil {
		return written, fmt.Errorf("sync manifest: %w", err)
	}

	return written, nil
}
