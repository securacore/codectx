// Package shared provides common helpers used by multiple CLI commands.
// This package contains display-layer utilities that cannot live in core/
// because they depend on the tui package for styled output.
package shared

import (
	"fmt"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
)

// ShouldAutoCompile determines whether auto-compilation should run
// based on CLI flags and the auto_compile preference. It prints an
// informational message explaining the decision when relevant.
//
// The actionLabel parameter is a human-readable description of what's
// being skipped or enabled (e.g., "initial compile" or "recompile").
func ShouldAutoCompile(
	prefsCfg *project.PreferencesConfig,
	forceCompile, skipCompile bool,
	actionLabel string,
) bool {
	if skipCompile {
		fmt.Printf("\n%s Skipping %s %s\n",
			tui.StyleMuted.Render("-"),
			actionLabel,
			tui.StyleMuted.Render("(--no-compile)"),
		)
		return false
	}

	if forceCompile {
		return true
	}

	if prefsCfg.AutoCompileIsDefault() {
		fmt.Printf("\n%s %s\n",
			tui.StyleMuted.Render("-"),
			tui.StyleMuted.Render("Auto-compile enabled (default). Set auto_compile in preferences.yml to configure."),
		)
		return true
	}

	if !prefsCfg.EffectiveAutoCompile() {
		fmt.Printf("\n%s Skipping %s %s\n%s %s\n",
			tui.StyleMuted.Render("-"),
			actionLabel,
			tui.StyleMuted.Render("(auto_compile: false)"),
			tui.Indent(1),
			tui.StyleMuted.Render("Run: "+tui.StyleCommand.Render("codectx compile")),
		)
		return false
	}

	// auto_compile is explicitly true — compile silently.
	return true
}
