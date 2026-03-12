// Package manifest generates the four compiled output files that describe
// the documentation landscape: manifest.yml (chunk navigation map),
// metadata.yml (document relationship graph), hashes.yml (incremental
// compilation state), and heuristics.yml (compilation diagnostics).
//
// These files are written to .codectx/compiled/ and consumed by the AI
// at query time for orientation, navigation, and context assembly.
package manifest

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/securacore/codectx/core/project"
)

// Compiled output filenames.
const (
	// ManifestFile is the filename for the manifest output.
	ManifestFile = "manifest.yml"

	// MetadataFile is the filename for the metadata output.
	MetadataFile = "metadata.yml"

	// HashesFile is the filename for the hashes output.
	HashesFile = "hashes.yml"

	// HeuristicsFile is the filename for the heuristics output.
	HeuristicsFile = "heuristics.yml"
)

// DocumentType classifies a source file by its directory convention.
type DocumentType string

const (
	// DocFoundation is a foundation document under docs/foundation/.
	DocFoundation DocumentType = "foundation"

	// DocTopic is a topic document under docs/topics/.
	DocTopic DocumentType = "topic"

	// DocPlan is a plan document under docs/plans/.
	DocPlan DocumentType = "plan"

	// DocPrompt is a prompt document under docs/prompts/.
	DocPrompt DocumentType = "prompt"

	// DocSystem is a system/compiler document under system/.
	DocSystem DocumentType = "system"

	// DocPackage is a document from an installed dependency package.
	DocPackage DocumentType = "package"
)

// Adjacency describes the previous/next chunk relationship within a source
// file. Pointer-to-string fields marshal to YAML null when nil.
type Adjacency struct {
	// Previous is the ID of the preceding chunk in the same file, or nil.
	Previous *string `yaml:"previous"`

	// Next is the ID of the following chunk in the same file, or nil.
	Next *string `yaml:"next"`
}

// Reference describes a cross-reference between documents.
// Populated by future cross-reference extraction; nil/empty for now.
type Reference struct {
	// Path is the relative path to the referenced document.
	Path string `yaml:"path"`

	// Reason describes why this reference exists.
	Reason string `yaml:"reason"`
}

// ClassifyDocType determines the DocumentType from a source file path.
// The path should be relative to the project root (e.g. "docs/topics/auth/jwt.md").
//
// Rules:
//   - foundation/ -> DocFoundation
//   - topics/     -> DocTopic
//   - plans/      -> DocPlan
//   - prompts/    -> DocPrompt
//   - system/     -> DocSystem
//   - .codectx/packages/ -> DocPackage
//   - default     -> DocTopic
//
// Source paths are expected to be relative to the documentation root
// (e.g., "foundation/overview.md", "topics/auth.md", "system/topics/README.md").
func ClassifyDocType(sourcePath string) DocumentType {
	normalized := filepath.ToSlash(sourcePath)

	prefixes := []struct {
		prefix  string
		docType DocumentType
	}{
		{"foundation/", DocFoundation},
		{"topics/", DocTopic},
		{"plans/", DocPlan},
		{"prompts/", DocPrompt},
		{project.SystemDir + "/", DocSystem},
		{project.CodectxDir + "/" + project.PackagesDir + "/", DocPackage},
	}

	for _, p := range prefixes {
		if strings.HasPrefix(normalized, p.prefix) {
			return p.docType
		}
	}

	return DocTopic
}

// StringPtr returns a pointer to the given string. Convenience helper for
// building Adjacency and other pointer-to-string fields.
func StringPtr(s string) *string {
	return &s
}

// CompiledAtNow returns the current UTC time in RFC 3339 format.
// Used as the canonical timestamp for all compiled output files.
func CompiledAtNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
