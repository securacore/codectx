// Package defaults provides embedded foundation documents that are written
// to every new codectx project during initialization. These documents
// establish baseline conventions for documentation management, markdown
// formatting, AI authoring, specification tracking, and decision-making
// philosophy. After init, the files are user-owned and can be customized.
package defaults

import "embed"

//go:embed content/philosophy/README.md content/philosophy/spec/README.md
//go:embed content/documentation/README.md content/documentation/spec/README.md
//go:embed content/markdown/README.md content/markdown/spec/README.md
//go:embed content/specs/README.md content/specs/spec/README.md
//go:embed content/ai-authoring/README.md content/ai-authoring/spec/README.md
//go:embed content/prompts/README.md content/prompts/spec/README.md
//go:embed content/plans/README.md content/plans/spec/README.md
var content embed.FS
