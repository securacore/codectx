package link

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LinkResult holds the outcome of linking a single tool.
type LinkResult struct {
	Tool     Tool
	Path     string
	BackedUp string // path of backup file, empty if no backup
}

// Link creates entry point files for the given tools.
// Each entry point is a single line referencing the compiled data map.
// Existing files are backed up as [file].[timestamp].bak.
func Link(tools []Tool, outputDir string) ([]LinkResult, error) {
	var results []LinkResult
	timestamp := time.Now().Format("20060102-150405")

	for _, tool := range tools {
		path := tool.File
		if tool.SubDir != "" {
			if err := os.MkdirAll(tool.SubDir, 0o755); err != nil {
				return results, fmt.Errorf("create directory %s: %w", tool.SubDir, err)
			}
			path = filepath.Join(tool.SubDir, tool.File)
		}

		result := LinkResult{
			Tool: tool,
			Path: path,
		}

		// Back up existing file.
		if _, err := os.Stat(path); err == nil {
			backupPath := fmt.Sprintf("%s.%s.bak", path, timestamp)
			if err := os.Rename(path, backupPath); err != nil {
				return results, fmt.Errorf("backup %s: %w", path, err)
			}
			result.BackedUp = backupPath
		}

		// Write entry point.
		readmePath := filepath.Join(outputDir, "README.md")
		content := fmt.Sprintf("Read [README.md](%s) before continuing.\n", readmePath)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return results, fmt.Errorf("write %s: %w", path, err)
		}

		results = append(results, result)
	}

	return results, nil
}
