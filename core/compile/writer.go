package compile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/project"
)

// PrepareOutputDirs deletes and recreates the compiled output directories
// (objects/, specs/, system/) and BM25 index directories under compiledDir.
// This ensures a clean slate for each full compilation.
func PrepareOutputDirs(compiledDir string) error {
	subdirs := chunk.CompiledOutputDirs()
	allDirs := make([]string, len(subdirs))
	for i, sub := range subdirs {
		allDirs[i] = filepath.Join(compiledDir, sub)
	}

	for _, dir := range allDirs {
		// Remove existing directory and all contents.
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("removing %s: %w", dir, err)
		}

		// Recreate the directory.
		if err := os.MkdirAll(dir, project.DirPerm); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	return nil
}

// WriteChunkFile writes a single chunk's rendered content to its output path
// under the compiled directory.
func WriteChunkFile(compiledDir string, c *chunk.Chunk) error {
	if c == nil {
		return fmt.Errorf("cannot write nil chunk")
	}

	outPath := filepath.Join(compiledDir, chunk.OutputDir(c.Type), chunk.OutputFilename(c))

	content := chunk.Render(c)
	if err := os.WriteFile(outPath, []byte(content), project.FilePerm); err != nil {
		return fmt.Errorf("writing chunk %s: %w", c.ID, err)
	}

	return nil
}

// WriteChunkFiles writes all chunks to their output paths under the compiled
// directory. Returns the number of files written.
func WriteChunkFiles(compiledDir string, chunks []chunk.Chunk) (int, error) {
	written := 0
	for i := range chunks {
		if err := WriteChunkFile(compiledDir, &chunks[i]); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}
