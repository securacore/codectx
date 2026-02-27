package ide

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/preferences"
)

// BuildManifestSummary formats a manifest into a readable summary for the
// system prompt. Each entry shows its ID, description, load value (foundation),
// and dependency relationships.
func BuildManifestSummary(m *manifest.Manifest) string {
	if m == nil {
		return "No existing documentation."
	}

	var b strings.Builder
	empty := true

	if len(m.Foundation) > 0 {
		empty = false
		b.WriteString("### Foundation\n\n")
		for _, e := range m.Foundation {
			fmt.Fprintf(&b, "- **%s** (load:%s): %s\n", e.ID, e.Load, e.Description)
			if len(e.DependsOn) > 0 {
				fmt.Fprintf(&b, "  depends_on: %s\n", strings.Join(e.DependsOn, ", "))
			}
		}
		b.WriteString("\n")
	}

	if len(m.Topics) > 0 {
		empty = false
		b.WriteString("### Topics\n\n")
		for _, e := range m.Topics {
			fmt.Fprintf(&b, "- **%s**: %s\n", e.ID, e.Description)
			if len(e.DependsOn) > 0 {
				fmt.Fprintf(&b, "  depends_on: %s\n", strings.Join(e.DependsOn, ", "))
			}
		}
		b.WriteString("\n")
	}

	if len(m.Application) > 0 {
		empty = false
		b.WriteString("### Application\n\n")
		for _, e := range m.Application {
			fmt.Fprintf(&b, "- **%s**: %s\n", e.ID, e.Description)
			if len(e.DependsOn) > 0 {
				fmt.Fprintf(&b, "  depends_on: %s\n", strings.Join(e.DependsOn, ", "))
			}
		}
		b.WriteString("\n")
	}

	if len(m.Prompts) > 0 {
		empty = false
		b.WriteString("### Prompts\n\n")
		for _, e := range m.Prompts {
			fmt.Fprintf(&b, "- **%s**: %s\n", e.ID, e.Description)
		}
		b.WriteString("\n")
	}

	if empty {
		return "No existing documentation."
	}

	return b.String()
}

// BuildPreferencesContext formats preferences into a string for the system prompt.
func BuildPreferencesContext(p *preferences.Preferences) string {
	if p == nil {
		return ""
	}

	var parts []string

	if p.Compression != nil {
		if *p.Compression {
			parts = append(parts, "- Compression is **enabled** (CMDX format). Documentation will be compressed during compilation.")
		} else {
			parts = append(parts, "- Compression is **disabled**. Documentation is stored as plain Markdown.")
		}
	}

	if p.AI != nil && p.AI.Class != "" {
		parts = append(parts, fmt.Sprintf("- Model class target: **%s**. Write documentation appropriate for this model tier.", p.AI.Class))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n")
}
