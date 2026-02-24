package compile

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/manifest"
)

// generateReadme builds the compiled README.md content dynamically
// from the unified manifest. Only sections with entries are included.
func generateReadme(m *manifest.Manifest) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", m.Name)
	b.WriteString("> Documentation managed by codectx.\n\n")
	b.WriteString("Load [package.yml](package.yml) and all foundation documents marked `load: always` at the start of every session.\n\n")

	b.WriteString("## Loading Protocol\n\n")
	b.WriteString("1. Load this file (done).\n")
	b.WriteString("2. Load [package.yml](package.yml). This is the data map indexing all documentation.\n")
	b.WriteString("3. Load all foundation entries with `load: always`. These are required context.\n")
	b.WriteString("4. As the task progresses, consult the data map to load relevant topics, prompts, or plans.\n")

	// Build sections list dynamically.
	hasSections := len(m.Foundation) > 0 || len(m.Topics) > 0 || len(m.Prompts) > 0 || len(m.Plans) > 0
	if hasSections {
		b.WriteString("\n## Sections\n\n")

		if len(m.Foundation) > 0 {
			fmt.Fprintf(&b, "- **Foundation**: %d %s. Core operational context.\n",
				len(m.Foundation), pluralize(len(m.Foundation), "document", "documents"))
		}

		if len(m.Topics) > 0 {
			fmt.Fprintf(&b, "- **Topics**: %d %s. Technology and domain conventions.\n",
				len(m.Topics), pluralize(len(m.Topics), "entry", "entries"))
		}

		if len(m.Prompts) > 0 {
			fmt.Fprintf(&b, "- **Prompts**: %d %s. Automated task definitions.\n",
				len(m.Prompts), pluralize(len(m.Prompts), "entry", "entries"))
		}

		if len(m.Plans) > 0 {
			fmt.Fprintf(&b, "- **Plans**: %d %s. Implementation plans with state tracking. Read `state.yml` before loading full plans.\n",
				len(m.Plans), pluralize(len(m.Plans), "entry", "entries"))
		}
	}

	return b.String()
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
