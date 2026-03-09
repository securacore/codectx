package defaults

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/manifest"
)

// defaultDoc describes one embedded foundation document.
type defaultDoc struct {
	// ID is the manifest entry identifier.
	ID string

	// Dir is the directory name under foundation/ (e.g., "philosophy").
	Dir string

	// Load is the manifest load value ("always" or "documentation").
	Load string

	// Description is a short summary for the manifest entry.
	Description string
}

// docs lists all embedded default documents in manifest order.
var docs = []defaultDoc{
	{
		ID:          "philosophy",
		Dir:         "philosophy",
		Load:        "always",
		Description: "Guiding principles for decision-making",
	},
	{
		ID:          "documentation",
		Dir:         "documentation",
		Load:        "documentation",
		Description: "Documentation management and organization",
	},
	{
		ID:          "markdown",
		Dir:         "markdown",
		Load:        "documentation",
		Description: "Markdown formatting conventions",
	},
	{
		ID:          "specs",
		Dir:         "specs",
		Load:        "documentation",
		Description: "Specification template and process",
	},
	{
		ID:          "ai-authoring",
		Dir:         "ai-authoring",
		Load:        "documentation",
		Description: "Cross-model AI authoring conventions",
	},
	{
		ID:          "prompts",
		Dir:         "prompts",
		Load:        "documentation",
		Description: "Prompt lifecycle management",
	},
	{
		ID:          "plans",
		Dir:         "plans",
		Load:        "documentation",
		Description: "Plan lifecycle management",
	},
}

// files lists the relative paths within each default document directory.
var files = []string{
	"README.md",
	filepath.Join("spec", "README.md"),
}

// WriteAll writes all embedded default foundation documents to the target
// directory. Each document gets a subdirectory with a README.md and a
// spec/README.md. Existing files are not overwritten (user-owned after init).
// The target directory is typically docs/foundation/.
func WriteAll(dir string) error {
	for _, doc := range docs {
		for _, file := range files {
			srcPath := filepath.Join("content", doc.Dir, file)
			dstPath := filepath.Join(dir, doc.Dir, file)

			// Skip if file already exists (user-owned after init).
			if _, err := os.Stat(dstPath); err == nil {
				continue
			}

			// Create parent directory.
			dstDir := filepath.Dir(dstPath)
			if err := os.MkdirAll(dstDir, 0o755); err != nil {
				return fmt.Errorf("create directory %s: %w", dstDir, err)
			}

			data, err := content.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("read embedded default %s: %w", srcPath, err)
			}

			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return fmt.Errorf("write default %s: %w", dstPath, err)
			}
		}
	}

	return nil
}

// Entries returns manifest FoundationEntry values for all embedded defaults.
// These entries are pre-populated with the correct load value and spec path
// so that manifest.Sync preserves them via merge-missing.
func Entries() []manifest.FoundationEntry {
	entries := make([]manifest.FoundationEntry, len(docs))
	for i, doc := range docs {
		entries[i] = manifest.FoundationEntry{
			ID:          doc.ID,
			Path:        filepath.Join("foundation", doc.Dir, "README.md"),
			Load:        doc.Load,
			Description: doc.Description,
			Spec:        filepath.Join("foundation", doc.Dir, "spec", "README.md"),
		}
	}
	return entries
}
