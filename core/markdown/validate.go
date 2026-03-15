package markdown

// ValidationResult holds warnings and errors from structural validation.
type ValidationResult struct {
	// Warnings are non-fatal issues that should be reported but don't
	// prevent compilation.
	Warnings []string

	// Errors are fatal issues that must be resolved.
	Errors []string
}

// ok returns true if there are no errors (warnings are acceptable).
func (v *ValidationResult) ok() bool {
	return len(v.Errors) == 0
}

// hasWarnings returns true if there are any warnings.
func (v *ValidationResult) hasWarnings() bool {
	return len(v.Warnings) > 0
}

// merge combines another ValidationResult into this one.
func (v *ValidationResult) merge(other *ValidationResult) {
	if other == nil {
		return
	}
	v.Warnings = append(v.Warnings, other.Warnings...)
	v.Errors = append(v.Errors, other.Errors...)
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
