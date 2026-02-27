package ide

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDocumentBlocks_single(t *testing.T) {
	text := `Here is your document:

<document path="docs/topics/go-error-handling/README.md">
# Go Error Handling

Handle errors explicitly.
</document>

Let me know if you want changes.`

	blocks := ParseDocumentBlocks(text)
	require.Len(t, blocks, 1)
	assert.Equal(t, "docs/topics/go-error-handling/README.md", blocks[0].Path)
	assert.Contains(t, blocks[0].Content, "# Go Error Handling")
	assert.Contains(t, blocks[0].Content, "Handle errors explicitly.")
}

func TestParseDocumentBlocks_multiple(t *testing.T) {
	text := `<document path="docs/topics/example/README.md">
# Example
Content here.
</document>

<document path="docs/topics/example/spec/README.md">
# Example Spec
Spec content.
</document>`

	blocks := ParseDocumentBlocks(text)
	require.Len(t, blocks, 2)
	assert.Equal(t, "docs/topics/example/README.md", blocks[0].Path)
	assert.Equal(t, "docs/topics/example/spec/README.md", blocks[1].Path)
	assert.Contains(t, blocks[0].Content, "# Example")
	assert.Contains(t, blocks[1].Content, "# Example Spec")
}

func TestParseDocumentBlocks_none(t *testing.T) {
	text := "Just a regular conversation with no document blocks."
	blocks := ParseDocumentBlocks(text)
	assert.Empty(t, blocks)
}

func TestParseDocumentBlocks_preservesContent(t *testing.T) {
	text := `<document path="docs/foundation/test/README.md">
# Test

## Section One

- Item one
- Item two

## Section Two

` + "```go" + `
func main() {}
` + "```" + `

A [link](../other/README.md) to another doc.
</document>`

	blocks := ParseDocumentBlocks(text)
	require.Len(t, blocks, 1)
	assert.Contains(t, blocks[0].Content, "## Section One")
	assert.Contains(t, blocks[0].Content, "- Item one")
	assert.Contains(t, blocks[0].Content, "func main() {}")
	assert.Contains(t, blocks[0].Content, "[link](../other/README.md)")
}

func TestWriteDocuments_createsDirectory(t *testing.T) {
	root := t.TempDir()

	blocks := []DocumentBlock{
		{Path: "docs/topics/example/README.md", Content: "# Example\n"},
	}

	written, err := WriteDocuments(root, blocks)
	require.NoError(t, err)
	require.Len(t, written, 1)

	content, err := os.ReadFile(filepath.Join(root, "docs/topics/example/README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Example\n", string(content))
}

func TestWriteDocuments_writesMultipleFiles(t *testing.T) {
	root := t.TempDir()

	blocks := []DocumentBlock{
		{Path: "docs/topics/test/README.md", Content: "# Test"},
		{Path: "docs/topics/test/spec/README.md", Content: "# Test Spec"},
	}

	written, err := WriteDocuments(root, blocks)
	require.NoError(t, err)
	assert.Len(t, written, 2)

	// Both files should exist.
	_, err = os.Stat(filepath.Join(root, "docs/topics/test/README.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, "docs/topics/test/spec/README.md"))
	assert.NoError(t, err)
}

func TestWriteDocuments_addsTrailingNewline(t *testing.T) {
	root := t.TempDir()

	blocks := []DocumentBlock{
		{Path: "docs/test.md", Content: "# No newline"},
	}

	_, err := WriteDocuments(root, blocks)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(root, "docs/test.md"))
	require.NoError(t, err)
	assert.True(t, content[len(content)-1] == '\n')
}

func TestWriteDocuments_existingDirectory(t *testing.T) {
	root := t.TempDir()

	// Create the directory first.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs/topics/existing"), 0o755))

	blocks := []DocumentBlock{
		{Path: "docs/topics/existing/README.md", Content: "# Existing\n"},
	}

	written, err := WriteDocuments(root, blocks)
	require.NoError(t, err)
	assert.Len(t, written, 1)
}

func TestWriteDocuments_empty(t *testing.T) {
	root := t.TempDir()
	written, err := WriteDocuments(root, nil)
	require.NoError(t, err)
	assert.Empty(t, written)
}

func TestWriteAndSync_syncsManifest(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o755))

	blocks := []DocumentBlock{
		{Path: "docs/topics/test-sync/README.md", Content: "# Test Sync\n\nContent for syncing.\n"},
	}

	written, err := WriteAndSync(root, docsDir, blocks)
	require.NoError(t, err)
	assert.Len(t, written, 1)

	// Manifest should have been created/updated.
	manifestPath := filepath.Join(docsDir, "manifest.yml")
	_, err = os.Stat(manifestPath)
	assert.NoError(t, err, "manifest.yml should exist after WriteAndSync")
}
