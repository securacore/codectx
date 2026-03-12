package shared

import (
	"fmt"

	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/tui"
)

// DiscoverProject locates and loads the nearest codectx project configuration
// starting from the current directory. On failure it prints a styled error
// message and returns a wrapped error suitable for command-level returns.
func DiscoverProject() (string, *project.Config, error) {
	projectDir, cfg, err := project.DiscoverAndLoad(".")
	if err != nil {
		fmt.Print(tui.ProjectNotFoundError())
		return "", nil, fmt.Errorf("project not found: %w", err)
	}
	return projectDir, cfg, nil
}
