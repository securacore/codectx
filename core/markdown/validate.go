package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidationResult holds warnings and errors from structural validation.
type ValidationResult struct {
	// Warnings are non-fatal issues that should be reported but don't
	// prevent compilation.
	Warnings []string

	// Errors are fatal issues that must be resolved.
	Errors []string
}

// OK returns true if there are no errors (warnings are acceptable).
func (v *ValidationResult) OK() bool {
	return len(v.Errors) == 0
}

// HasWarnings returns true if there are any warnings.
func (v *ValidationResult) HasWarnings() bool {
	return len(v.Warnings) > 0
}

// Merge combines another ValidationResult into this one.
func (v *ValidationResult) Merge(other *ValidationResult) {
	if other == nil {
		return
	}
	v.Warnings = append(v.Warnings, other.Warnings...)
	v.Errors = append(v.Errors, other.Errors...)
}

// ValidateDir checks structural requirements for a documentation directory tree.
//
// Checks performed:
//   - requireReadme: every subdirectory containing .md files must have a README.md
//   - requireHeadings: every .md file must contain at least one heading
//
// The dir parameter should be the documentation root (e.g., "docs/").
// Walks recursively through all subdirectories.
func ValidateDir(dir string, requireReadme, requireHeadings bool) (*ValidationResult, error) {
	result := &ValidationResult{}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving directory: %w", err)
	}

	// Walk the directory tree.
	err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories (like .codectx/).
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != absDir {
			return filepath.SkipDir
		}

		// For directories: check for README.md if required.
		if d.IsDir() && requireReadme && path != absDir {
			if err := checkReadme(path, absDir, result); err != nil {
				return err
			}
			return nil
		}

		// For files: validate .md files.
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") && requireHeadings {
			if err := checkHeadings(path, absDir, result); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", dir, err)
	}

	return result, nil
}

// ValidateFile checks a single parsed document for structural issues.
//
// Currently checks:
//   - requireHeadings: the document must contain at least one heading block
func ValidateFile(doc *Document, requireHeadings bool) *ValidationResult {
	result := &ValidationResult{}

	if requireHeadings {
		hasHeading := false
		for _, b := range doc.Blocks {
			if b.Type == BlockHeading {
				hasHeading = true
				break
			}
		}
		if !hasHeading {
			result.Warnings = append(result.Warnings, "file has no headings")
		}
	}

	return result
}

// checkReadme verifies that a directory containing .md files has a README.md.
func checkReadme(dir, rootDir string, result *ValidationResult) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	hasMD := false
	hasReadme := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") {
			hasMD = true
		}
		if strings.EqualFold(name, "readme.md") {
			hasReadme = true
		}
	}

	if hasMD && !hasReadme {
		relPath, err := filepath.Rel(rootDir, dir)
		if err != nil || relPath == "" {
			relPath = dir
		}
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("directory %s has .md files but no README.md", relPath))
	}

	return nil
}

// checkHeadings reads and parses a markdown file, checking for at least one heading.
func checkHeadings(path, rootDir string, result *ValidationResult) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	doc := Parse(data)
	fileResult := ValidateFile(doc, true)

	if fileResult.HasWarnings() {
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			relPath = path
		}
		for _, w := range fileResult.Warnings {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s: %s", relPath, w))
		}
	}

	return nil
}
