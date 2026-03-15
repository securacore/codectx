package shared

import (
	"fmt"

	"github.com/securacore/codectx/core/tui"
)

// BoolCount returns the number of true values among the given bools.
// Used to detect conflicting mutually-exclusive CLI flags.
func BoolCount(vals ...bool) int {
	n := 0
	for _, v := range vals {
		if v {
			n++
		}
	}
	return n
}

// ValidateExclusiveTargetFlags checks that at most one of --project, --package,
// --both is set. Returns a rendered error string if conflicting, empty string if OK.
func ValidateExclusiveTargetFlags(flagProject, flagPackage, flagBoth bool) string {
	if BoolCount(flagProject, flagPackage, flagBoth) <= 1 {
		return ""
	}
	return tui.ErrorMsg{
		Title: "Conflicting flags",
		Detail: []string{
			fmt.Sprintf("Only one of %s, %s, or %s may be specified.",
				tui.StyleCommand.Render("--project"),
				tui.StyleCommand.Render("--package"),
				tui.StyleCommand.Render("--both"),
			),
		},
	}.Render()
}

// WarnNoDependencies prints the standard "no dependencies" warning.
func WarnNoDependencies() {
	fmt.Printf("\n%s No dependencies declared in codectx.yml\n\n", tui.Warning())
}
