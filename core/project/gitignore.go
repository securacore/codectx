package project

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// gitignoreSection is the comment header that marks our managed block.
const gitignoreSection = "# codectx managed entries"

// rootPlaceholder is used in gitignore templates and replaced with the actual
// documentation root directory name. Using a placeholder avoids fragile
// substring replacement that could match unintended text in comments.
const rootPlaceholder = "{{ROOT}}"

// gitignoreTemplate contains the patterns codectx needs in .gitignore,
// with {{ROOT}} as a placeholder for the documentation root directory.
// The order matters: ignore rules must come before negation rules
// so that the negations can override the ignores correctly.
var gitignoreTemplate = []string{
	"# Compiled output and tooling state",
	"{{ROOT}}/.codectx/compiled/",
	"{{ROOT}}/.codectx/packages/",
	"{{ROOT}}/.codectx/ai.local.yml",
	"",
	"# Force-include checked-in config",
	"!{{ROOT}}/.codectx/ai.yml",
	"!{{ROOT}}/.codectx/preferences.yml",
}

// EnsureGitignore ensures the .gitignore at the git repository root contains
// all required codectx entries. It handles:
//   - Creating the file if it doesn't exist
//   - Reading and preserving existing content
//   - Deduplicating entries
//   - Resolving conflicts between ignore and negation rules
//   - Idempotent re-runs (no duplicate entries)
//
// The projectDir is the git repository root. The root parameter is the
// documentation root directory name (e.g. "docs") used to build the correct
// relative paths in the .gitignore patterns.
func EnsureGitignore(projectDir string, root string) error {
	root = ResolveRoot(root)

	path := filepath.Join(projectDir, ".gitignore")

	entries := gitignoreEntriesForRoot(root)

	existing, err := readGitignore(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	merged := mergeGitignore(existing, entries)

	return os.WriteFile(path, []byte(merged), FilePerm)
}

// gitignoreEntriesForRoot returns the codectx gitignore entries with the
// {{ROOT}} placeholder replaced by the actual documentation root directory name.
func gitignoreEntriesForRoot(root string) []string {
	entries := make([]string, len(gitignoreTemplate))
	for i, e := range gitignoreTemplate {
		entries[i] = strings.ReplaceAll(e, rootPlaceholder, root)
	}
	return entries
}

// readGitignore reads an existing .gitignore file and returns its lines.
func readGitignore(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// mergeGitignore merges codectx entries into existing .gitignore content.
// It removes any previous codectx managed block, deduplicates entries,
// resolves conflicts, and appends the managed block at the end.
func mergeGitignore(existing []string, entries []string) string {
	// Remove any previous codectx managed block.
	cleaned := removeManagedBlock(existing)

	// Build a set of patterns that our managed block will add.
	managedPatterns := make(map[string]bool)
	for _, e := range entries {
		trimmed := strings.TrimSpace(e)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			managedPatterns[trimmed] = true
		}
	}

	// Remove duplicates from existing content that conflict with our entries.
	// This handles the case where someone manually added one of our patterns.
	var deduped []string
	for _, line := range cleaned {
		trimmed := strings.TrimSpace(line)
		if managedPatterns[trimmed] {
			continue // we'll add this in our managed block
		}
		// Check for conflicts: if existing has an ignore rule and we have a
		// negation for the same path (or vice versa), remove the existing one.
		if conflictsWithManaged(trimmed, managedPatterns) {
			continue
		}
		deduped = append(deduped, line)
	}

	// Trim trailing blank lines from existing content.
	for len(deduped) > 0 && strings.TrimSpace(deduped[len(deduped)-1]) == "" {
		deduped = deduped[:len(deduped)-1]
	}

	// Build the final content.
	var b strings.Builder

	// Write existing content.
	for _, line := range deduped {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	// Add separator if there's existing content.
	if len(deduped) > 0 {
		b.WriteByte('\n')
	}

	// Write managed block.
	b.WriteString(gitignoreSection)
	b.WriteByte('\n')
	for _, e := range entries {
		b.WriteString(e)
		b.WriteByte('\n')
	}

	return b.String()
}

// removeManagedBlock removes the codectx managed block from existing lines.
// The block starts at the gitignoreSection comment and extends to the next
// blank line followed by a non-codectx comment, or to end of file.
func removeManagedBlock(lines []string) []string {
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == gitignoreSection {
			start = i
			break
		}
	}
	if start == -1 {
		return lines
	}

	// Find the end of the managed block: scan until we hit a line that is
	// a non-empty, non-comment line that doesn't look like a codectx pattern,
	// or until end of file. The block includes all lines from the section
	// header through contiguous codectx-related lines.
	end := len(lines)
	pastHeader := false
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			// Blank lines within the block are fine.
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if pastHeader && !isCodectxComment(trimmed) {
				// New unrelated section — stop here.
				end = i
				break
			}
			pastHeader = true
			continue
		}
		// Pattern line — part of our block if it references codectx paths.
		if isCodectxPattern(trimmed) {
			pastHeader = true
			continue
		}
		// Unrelated line — stop here.
		end = i
		break
	}

	result := make([]string, 0, len(lines))
	result = append(result, lines[:start]...)
	if end < len(lines) {
		result = append(result, lines[end:]...)
	}
	return result
}

// isCodectxComment returns true if the comment line is related to codectx.
func isCodectxComment(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "codectx") ||
		strings.Contains(lower, "compiled") ||
		strings.Contains(lower, "force-include") ||
		strings.Contains(lower, "checked-in config")
}

// isCodectxPattern returns true if the pattern references codectx paths.
func isCodectxPattern(line string) bool {
	return strings.Contains(line, ".codectx/") ||
		strings.Contains(line, ".codectx\\")
}

// conflictsWithManaged returns true if a line conflicts with our managed entries.
// A conflict occurs when:
//   - The existing file has "path/" and we want "!path/" (or vice versa)
//   - This prevents the negation from working correctly.
func conflictsWithManaged(line string, managed map[string]bool) bool {
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	// Check if negating this line is in our managed set.
	if strings.HasPrefix(line, "!") {
		withoutNeg := line[1:]
		if managed[withoutNeg] {
			return true
		}
	} else {
		negated := "!" + line
		if managed[negated] {
			return true
		}
	}
	return false
}
