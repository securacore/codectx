// Package embed provides access to embedded default files used during
// project scaffolding (codectx init) and package scaffolding (codectx new package).
// All default system/ documentation and package templates are bundled into the
// binary so codectx has no external file dependencies.
package embed

import "embed"

//go:embed defaults/*
var defaults embed.FS

//go:embed package/*
var packageTemplates embed.FS

// DefaultFile represents an embedded file with its intended destination
// path relative to the documentation root (e.g., "system/topics/taxonomy-generation/README.md").
type DefaultFile struct {
	// DestPath is the path relative to the documentation root where this file
	// should be written during scaffolding.
	DestPath string

	// EmbedPath is the path within the embedded filesystem to read from.
	EmbedPath string
}

// SystemFiles returns all default system/ documentation files that should be
// created during codectx init. Each entry maps an embedded source to its
// destination path under the documentation root.
func SystemFiles() []DefaultFile {
	return []DefaultFile{
		{
			DestPath:  "system/foundation/compiler-philosophy/README.md",
			EmbedPath: "defaults/compiler-philosophy.md",
		},
		{
			DestPath:  "system/foundation/compiler-philosophy/README.spec.md",
			EmbedPath: "defaults/compiler-philosophy.spec.md",
		},
		{
			DestPath:  "system/topics/taxonomy-generation/README.md",
			EmbedPath: "defaults/taxonomy-generation.md",
		},
		{
			DestPath:  "system/topics/taxonomy-generation/README.spec.md",
			EmbedPath: "defaults/taxonomy-generation.spec.md",
		},
		{
			DestPath:  "system/topics/bridge-summaries/README.md",
			EmbedPath: "defaults/bridge-summaries.md",
		},
		{
			DestPath:  "system/topics/bridge-summaries/README.spec.md",
			EmbedPath: "defaults/bridge-summaries.spec.md",
		},
		{
			DestPath:  "system/topics/context-assembly/README.md",
			EmbedPath: "defaults/context-assembly.md",
		},
		{
			DestPath:  "system/foundation/documentation-protocol/README.md",
			EmbedPath: "defaults/documentation-protocol.md",
		},
		{
			DestPath:  "system/foundation/history/README.md",
			EmbedPath: "defaults/history.md",
		},
	}
}

// ReadFile reads an embedded file by its embed path from the defaults filesystem.
func ReadFile(path string) ([]byte, error) {
	return defaults.ReadFile(path)
}

// PackageTemplateFile represents an embedded template file for package
// scaffolding with its intended destination path relative to the project root.
type PackageTemplateFile struct {
	// DestPath is the path relative to the project root where this file
	// should be written during package scaffolding.
	DestPath string

	// EmbedPath is the path within the embedded filesystem to read from.
	EmbedPath string
}

// PackageTemplateFiles returns all template files for package scaffolding.
// Each entry maps an embedded source to its destination path under the
// project root.
func PackageTemplateFiles() []PackageTemplateFile {
	return []PackageTemplateFile{
		{
			DestPath:  ".github/workflows/release.yml",
			EmbedPath: "package/release.yml",
		},
	}
}

// ReadPackageFile reads an embedded file from the package templates filesystem.
func ReadPackageFile(path string) ([]byte, error) {
	return packageTemplates.ReadFile(path)
}
