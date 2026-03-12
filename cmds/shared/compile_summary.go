package shared

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/tui"
)

// RenderCompactCompileSummary formats a compact compilation summary suitable
// for display after auto-compile operations (init, update). Returns the
// formatted string including a success header and key metrics.
func RenderCompactCompileSummary(result *compile.Result) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s %s\n",
		tui.Success(),
		tui.StyleBold.Render("Compilation complete"),
	)

	fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
		tui.KeyValue("Compiled", fmt.Sprintf("%d files -> %s chunks (%s tokens)",
			result.TotalFiles,
			tui.FormatNumber(result.TotalChunks),
			tui.FormatNumber(result.TotalTokens),
		)),
	)

	if result.TaxonomyTerms > 0 {
		fmt.Fprintf(&b, "%s%s\n", tui.Indent(1),
			tui.KeyValue("Taxonomy", fmt.Sprintf("%s terms extracted",
				tui.FormatNumber(result.TaxonomyTerms),
			)),
		)
	}

	fmt.Fprintf(&b, "%s%s\n\n", tui.Indent(1),
		tui.KeyValue("Time", tui.FormatDuration(result.TotalSeconds)),
	)

	return b.String()
}
