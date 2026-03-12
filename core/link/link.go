// Package link manages AI tool entry point files (CLAUDE.md, AGENTS.md,
// .cursorrules, .github/copilot-instructions.md) that bootstrap AI tools
// into the codectx system by pointing them to the compiled context.md.
//
// Entry point files are created by `codectx init` and `codectx link`.
// They are conditionally updated during `codectx compile` if the context
// path has changed. Existing files are always backed up before overwriting.
package link

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/securacore/codectx/core/project"
)

// codectxMarker is the string used to identify files written by codectx.
// Entry point files are only updated or counted as "linked" if they
// contain this marker.
const codectxMarker = "codectx"

// Integration identifies a specific AI tool entry point.
type Integration int

const (
	// Claude is the Claude Code entry point (CLAUDE.md).
	Claude Integration = iota

	// Agents is the generic agents entry point (AGENTS.md).
	Agents

	// Cursor is the Cursor IDE entry point (.cursorrules).
	Cursor

	// Copilot is the GitHub Copilot entry point (.github/copilot-instructions.md).
	Copilot
)

// Info holds display information about an integration.
type Info struct {
	// Type is the integration identifier.
	Type Integration

	// Name is the human-readable name for display.
	Name string

	// Description is a short description for the selection prompt.
	Description string

	// FilePath is the file path relative to the project root.
	FilePath string
}

// AllIntegrations returns information about all supported integrations.
func AllIntegrations() []Info {
	return []Info{
		{
			Type:        Claude,
			Name:        "Claude Code",
			Description: "CLAUDE.md at project root",
			FilePath:    "CLAUDE.md",
		},
		{
			Type:        Agents,
			Name:        "Agents",
			Description: "AGENTS.md at project root",
			FilePath:    "AGENTS.md",
		},
		{
			Type:        Cursor,
			Name:        "Cursor",
			Description: ".cursorrules at project root",
			FilePath:    ".cursorrules",
		},
		{
			Type:        Copilot,
			Name:        "GitHub Copilot",
			Description: ".github/copilot-instructions.md",
			FilePath:    filepath.Join(".github", "copilot-instructions.md"),
		},
	}
}

// InfoByType returns the Info for a specific integration type.
func InfoByType(t Integration) Info {
	for _, info := range AllIntegrations() {
		if info.Type == t {
			return info
		}
	}
	return Info{}
}

// WriteResult holds the outcome of writing a single entry point file.
type WriteResult struct {
	// Integration is the type of entry point written.
	Integration Integration

	// Name is the human-readable name.
	Name string

	// FilePath is the relative path that was written.
	FilePath string

	// BackedUp is true if an existing file was backed up.
	BackedUp bool

	// BackupPath is the relative path to the backup file.
	// Empty if BackedUp is false.
	BackupPath string
}

// Write writes the selected entry point files to the project directory.
// For each file, if it already exists, it is backed up first.
//
// contextRelPath is the path to context.md relative to the project root
// (e.g., "docs/.codectx/compiled/context.md").
func Write(projectDir, contextRelPath string, integrations []Integration) ([]WriteResult, error) {
	var results []WriteResult

	for _, integration := range integrations {
		info := InfoByType(integration)
		if info.Name == "" {
			continue
		}

		result, err := writeOne(projectDir, contextRelPath, info)
		if err != nil {
			return results, fmt.Errorf("writing %s: %w", info.Name, err)
		}

		results = append(results, result)
	}

	return results, nil
}

// writeOne writes a single entry point file, backing up any existing file.
func writeOne(projectDir, contextRelPath string, info Info) (WriteResult, error) {
	result := WriteResult{
		Integration: info.Type,
		Name:        info.Name,
		FilePath:    info.FilePath,
	}

	absPath := filepath.Join(projectDir, info.FilePath)

	// Ensure parent directory exists (for .github/copilot-instructions.md).
	if err := os.MkdirAll(filepath.Dir(absPath), project.DirPerm); err != nil {
		return result, fmt.Errorf("creating directory: %w", err)
	}

	// Back up existing file if present.
	if _, err := os.Stat(absPath); err == nil {
		backupPath, backupErr := backup(absPath)
		if backupErr != nil {
			return result, fmt.Errorf("backing up %s: %w", info.FilePath, backupErr)
		}

		// Store relative backup path for display.
		relBackup, relErr := filepath.Rel(projectDir, backupPath)
		if relErr != nil {
			relBackup = backupPath
		}

		result.BackedUp = true
		result.BackupPath = relBackup
	}

	// Write the entry point content.
	content := renderTemplate(contextRelPath)
	if err := os.WriteFile(absPath, []byte(content), project.FilePerm); err != nil {
		return result, err
	}

	return result, nil
}

// backup copies a file to a timestamped backup path.
// Returns the absolute path to the backup file.
func backup(absPath string) (string, error) {
	timestamp := time.Now().Unix()
	backupPath := fmt.Sprintf("%s.%d.bak", absPath, timestamp)

	src, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return backupPath, nil
}

// Detect returns the integrations that should be pre-selected based on
// the presence of tool-specific directories or files in the project.
func Detect(projectDir string) []Integration {
	var detected []Integration

	// Claude Code: .claude/ directory or existing CLAUDE.md.
	if dirExists(filepath.Join(projectDir, ".claude")) ||
		fileExists(filepath.Join(projectDir, "CLAUDE.md")) {
		detected = append(detected, Claude)
	}

	// Agents: existing AGENTS.md.
	if fileExists(filepath.Join(projectDir, "AGENTS.md")) {
		detected = append(detected, Agents)
	}

	// Cursor: .cursor/ directory or existing .cursorrules.
	if dirExists(filepath.Join(projectDir, ".cursor")) ||
		fileExists(filepath.Join(projectDir, ".cursorrules")) {
		detected = append(detected, Cursor)
	}

	// GitHub Copilot: .github/ directory.
	if dirExists(filepath.Join(projectDir, ".github")) {
		detected = append(detected, Copilot)
	}

	return detected
}

// NeedsUpdate checks which existing entry point files contain a stale
// context path and returns the integrations that need updating.
func NeedsUpdate(projectDir, contextRelPath string) []Integration {
	var needsUpdate []Integration

	for _, info := range AllIntegrations() {
		absPath := filepath.Join(projectDir, info.FilePath)

		data, err := os.ReadFile(absPath)
		if err != nil {
			// File doesn't exist — nothing to update.
			continue
		}

		content := string(data)

		// Check if the file was written by codectx (contains our marker).
		if !strings.Contains(content, codectxMarker) {
			continue
		}

		// Check if the context path in the file matches the current path.
		if !strings.Contains(content, contextRelPath) {
			needsUpdate = append(needsUpdate, info.Type)
		}
	}

	return needsUpdate
}

// renderTemplate produces the entry point file content.
// The template is deliberately minimal — its only job is to direct the AI
// to read context.md before doing anything else. All project instructions,
// CLI usage, and coding standards live in context.md (compiled from
// session.always_loaded). This keeps entry points as pure infrastructure.
func renderTemplate(contextRelPath string) string {
	// Normalize to forward slashes for display.
	contextRelPath = filepath.ToSlash(contextRelPath)

	var b strings.Builder

	b.WriteString("# codectx\n\n")
	fmt.Fprintf(&b, "STOP. Read [context](%s) now.\n", contextRelPath)
	b.WriteString("Do not proceed with any task until you have read that document\n")
	b.WriteString("completely and followed every instruction it contains.\n")

	return b.String()
}

// dirExists returns true if path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists returns true if path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
