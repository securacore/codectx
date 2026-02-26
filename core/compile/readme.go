package compile

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/manifest"
)

// generateReadme builds the compiled README.md content dynamically
// from the unified manifest and heuristics data. Only sections with
// entries are included. When heuristics are available, token estimates
// and size information are included. The compressed flag adds a format
// note about CMDX encoding.
func generateReadme(m *manifest.Manifest, h *Heuristics, compressed ...bool) string {
	isCmdx := len(compressed) > 0 && compressed[0]

	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", m.Name)
	b.WriteString("> Documentation managed by codectx.\n\n")
	b.WriteString("Load [manifest.yml](manifest.yml) and all foundation documents marked `load: always` at the start of every session.\n\n")

	if isCmdx {
		b.WriteString("> **Format**: Objects use CMDX compression (`.cmdx`). CMDX is a compact binary encoding of markdown optimized for AI context loading. Decode with `codectx cmdx decode` or load directly — content is semantically equivalent to the source markdown.\n\n")
	}

	b.WriteString("## Loading Protocol\n\n")
	b.WriteString("1. Load this file (done).\n")
	b.WriteString("2. Load [manifest.yml](manifest.yml). This is the data map indexing all documentation.\n")
	b.WriteString("3. Load all foundation entries with `load: always`. These are required context.\n")
	b.WriteString("4. As the task progresses, consult the data map to load relevant topics, prompts, or plans.\n")

	// Build sections list dynamically.
	hasSections := len(m.Foundation) > 0 || len(m.Application) > 0 || len(m.Topics) > 0 || len(m.Prompts) > 0 || len(m.Plans) > 0
	if hasSections {
		b.WriteString("\n## Sections\n\n")

		if len(m.Foundation) > 0 {
			line := fmt.Sprintf("- **Foundation**: %d %s.",
				len(m.Foundation), pluralize(len(m.Foundation), "document", "documents"))
			if h != nil && h.Sections.Foundation != nil {
				line += fmt.Sprintf(" ~%s.", formatTokens(h.Sections.Foundation.EstimatedTokens))
			}
			line += " Core operational context."
			if h != nil && h.Totals.AlwaysLoad > 0 {
				line += fmt.Sprintf(" %d %s auto-loaded.",
					h.Totals.AlwaysLoad, pluralize(h.Totals.AlwaysLoad, "is", "are"))
			}
			b.WriteString(line + "\n")
		}

		if len(m.Application) > 0 {
			line := fmt.Sprintf("- **Application**: %d %s.",
				len(m.Application), pluralize(len(m.Application), "entry", "entries"))
			if h != nil && h.Sections.Application != nil {
				line += fmt.Sprintf(" ~%s.", formatTokens(h.Sections.Application.EstimatedTokens))
			}
			line += " Product architecture and design documentation."
			b.WriteString(line + "\n")
		}

		if len(m.Topics) > 0 {
			line := fmt.Sprintf("- **Topics**: %d %s.",
				len(m.Topics), pluralize(len(m.Topics), "entry", "entries"))
			if h != nil && h.Sections.Topics != nil {
				line += fmt.Sprintf(" ~%s.", formatTokens(h.Sections.Topics.EstimatedTokens))
			}
			line += " Technology and domain conventions."
			b.WriteString(line + "\n")
		}

		if len(m.Prompts) > 0 {
			line := fmt.Sprintf("- **Prompts**: %d %s.",
				len(m.Prompts), pluralize(len(m.Prompts), "entry", "entries"))
			if h != nil && h.Sections.Prompts != nil {
				line += fmt.Sprintf(" ~%s.", formatTokens(h.Sections.Prompts.EstimatedTokens))
			}
			line += " Automated task definitions."
			b.WriteString(line + "\n")
		}

		if len(m.Plans) > 0 {
			line := fmt.Sprintf("- **Plans**: %d %s.",
				len(m.Plans), pluralize(len(m.Plans), "entry", "entries"))
			if h != nil && h.Sections.Plans != nil {
				line += fmt.Sprintf(" ~%s.", formatTokens(h.Sections.Plans.EstimatedTokens))
			}
			line += " Implementation plans with state tracking."
			b.WriteString(line + "\n")
		}
	}

	// Total documentation size.
	if h != nil && h.Totals.EstimatedTokens > 0 {
		fmt.Fprintf(&b, "\nTotal documentation: ~%s across %d %s.\n",
			formatTokens(h.Totals.EstimatedTokens),
			h.Totals.Objects,
			pluralize(h.Totals.Objects, "object", "objects"))
	}

	return b.String()
}

// formatTokens formats a token count as a human-readable string.
// e.g., 1500 -> "1.5k tokens", 250 -> "250 tokens".
func formatTokens(tokens int) string {
	if tokens >= 1000 {
		k := float64(tokens) / 1000
		if k == float64(int(k)) {
			return fmt.Sprintf("%dk tokens", int(k))
		}
		return fmt.Sprintf("%.1fk tokens", k)
	}
	return fmt.Sprintf("%d tokens", tokens)
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
