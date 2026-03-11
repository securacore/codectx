package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
)

// contextFile is the filename for the assembled session context.
const contextFile = "context.md"

// WriteContextMD writes the assembled context.md to the compiled directory.
// The file includes a metadata header with token count, budget, timestamp,
// and source information.
func WriteContextMD(compiledDir string, result *AssemblyResult) error {
	if result == nil {
		return fmt.Errorf("assembly result is nil")
	}

	// Ensure the compiled directory exists.
	if err := os.MkdirAll(compiledDir, project.DirPerm); err != nil {
		return fmt.Errorf("creating compiled directory: %w", err)
	}

	content := renderContextMD(result)

	path := filepath.Join(compiledDir, contextFile)
	if err := os.WriteFile(path, []byte(content), project.FilePerm); err != nil {
		return fmt.Errorf("writing %s: %w", contextFile, err)
	}

	return nil
}

// ContextPath returns the path to context.md within the compiled directory.
func ContextPath(compiledDir string) string {
	return filepath.Join(compiledDir, contextFile)
}

// renderContextMD produces the full context.md content with metadata header.
func renderContextMD(result *AssemblyResult) string {
	var b strings.Builder

	b.WriteString("# Project Engineering Context\n\n")

	// Metadata block.
	b.WriteString("> This document is automatically compiled from session context entries.\n")
	b.WriteString("> Source: codectx.yml session.always_loaded\n")

	if result.Budget > 0 {
		fmt.Fprintf(&b, "> Token count: %s / %s budget\n",
			tui.FormatNumber(result.TotalTokens),
			tui.FormatNumber(result.Budget),
		)
	} else {
		fmt.Fprintf(&b, "> Token count: %s\n",
			tui.FormatNumber(result.TotalTokens),
		)
	}

	fmt.Fprintf(&b, "> Compiled: %s\n", time.Now().UTC().Format(time.RFC3339))

	// Add the assembled content.
	if result.Content != "" {
		b.WriteString("\n")
		b.WriteString(result.Content)
		b.WriteString("\n")
	}

	return b.String()
}
