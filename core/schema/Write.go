package schema

import (
	"fmt"
	"os"
	"path/filepath"
)

// schemaFiles lists all embedded schema files.
var schemaFiles = []string{
	CodectxSchemaFile,
	PackageSchemaFile,
	StateSchemaFile,
}

// WriteAll writes all embedded schema files to the target directory.
// The directory is created if it does not exist.
func WriteAll(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create schema directory %s: %w", dir, err)
	}

	for _, name := range schemaFiles {
		data, err := schemas.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read embedded schema %s: %w", name, err)
		}

		dst := filepath.Join(dir, name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write schema %s: %w", dst, err)
		}
	}

	return nil
}
