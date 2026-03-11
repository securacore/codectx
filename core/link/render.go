package link

import (
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/tui"
)

// RenderLinkResults formats the results of a link/write operation as a
// styled summary string. Used by both `codectx link` and `codectx init`.
func RenderLinkResults(results []WriteResult) string {
	if len(results) == 0 {
		return ""
	}

	var b strings.Builder

	fmt.Fprintf(&b, "\n%s %s\n\n",
		tui.Success(),
		tui.StyleBold.Render("Entry points linked"),
	)

	for _, r := range results {
		if r.BackedUp {
			fmt.Fprintf(&b, "%s%s %s %s\n",
				tui.Indent(1),
				tui.Success(),
				tui.StylePath.Render(r.FilePath),
				tui.StyleMuted.Render(fmt.Sprintf("(backed up to %s)", r.BackupPath)),
			)
		} else {
			fmt.Fprintf(&b, "%s%s %s\n",
				tui.Indent(1),
				tui.Success(),
				tui.StylePath.Render(r.FilePath),
			)
		}
	}

	b.WriteString("\n")
	return b.String()
}
