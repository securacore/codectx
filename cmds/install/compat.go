package install

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/schema"

	"gopkg.in/yaml.v3"
)

// compatIssue describes a single compatibility problem.
type compatIssue struct {
	path   string
	reason string
}

// nameAtAuthor matches the "name@author" directory naming convention.
var nameAtAuthor = regexp.MustCompile(`^[a-zA-Z0-9_-]+@[a-zA-Z0-9_-]+$`)

// checkCompatibility inspects an existing directory to determine whether
// codectx can use it as a docs directory without conflicts. Returns a list
// of issues; an empty list means the directory is compatible.
func checkCompatibility(dir string) []compatIssue {
	var issues []compatIssue

	// Check package.yml if it exists.
	pkgPath := filepath.Join(dir, "package.yml")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var raw any
		if err := yaml.Unmarshal(data, &raw); err != nil {
			issues = append(issues, compatIssue{
				path:   pkgPath,
				reason: fmt.Sprintf("invalid YAML: %s", err),
			})
		} else if err := schema.Validate(schema.PackageSchemaFile, raw); err != nil {
			issues = append(issues, compatIssue{
				path:   pkgPath,
				reason: fmt.Sprintf("schema validation failed: %s", err),
			})
		}
	}

	// Check packages/ subdirectory for non-codectx content.
	packagesDir := filepath.Join(dir, "packages")
	if entries, err := os.ReadDir(packagesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if !nameAtAuthor.MatchString(e.Name()) {
				issues = append(issues, compatIssue{
					path:   filepath.Join(packagesDir, e.Name()),
					reason: fmt.Sprintf("directory %q does not follow name@author convention", e.Name()),
				})
			}
		}
	}

	// Check schemas/ for conflicting files.
	schemasDir := filepath.Join(dir, "schemas")
	codectxSchemas := map[string]bool{
		"codectx.schema.json":    true,
		"package.schema.json":    true,
		"state.schema.json":      true,
		"compiled.schema.json":   true,
		"heuristics.schema.json": true,
	}
	if entries, err := os.ReadDir(schemasDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if codectxSchemas[e.Name()] {
				// Validate that it's actually a codectx schema by trying to
				// read and verify it has a $schema field.
				schemaPath := filepath.Join(schemasDir, e.Name())
				data, readErr := os.ReadFile(schemaPath)
				if readErr != nil {
					issues = append(issues, compatIssue{
						path:   schemaPath,
						reason: fmt.Sprintf("cannot read schema file: %s", readErr),
					})
					continue
				}
				var raw map[string]any
				if err := yaml.Unmarshal(data, &raw); err != nil {
					issues = append(issues, compatIssue{
						path:   schemaPath,
						reason: fmt.Sprintf("schema file is not valid JSON/YAML: %s", err),
					})
					continue
				}
				if _, ok := raw["$schema"]; !ok {
					issues = append(issues, compatIssue{
						path:   schemaPath,
						reason: "schema file missing $schema field (not a JSON Schema)",
					})
				}
			}
		}
	}

	// Validate existing package.yml can be loaded if it exists and passed schema.
	if _, err := os.Stat(pkgPath); err == nil && len(issues) == 0 {
		if _, err := manifest.Load(pkgPath); err != nil {
			issues = append(issues, compatIssue{
				path:   pkgPath,
				reason: fmt.Sprintf("failed to load: %s", err),
			})
		}
	}

	return issues
}
