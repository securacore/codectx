package shared

import (
	"fmt"
	"path/filepath"

	"github.com/securacore/codectx/core/link"
	"github.com/securacore/codectx/core/project"
)

// PromptAndWriteLinks prompts the user to select AI tool integrations and
// writes the entry point files. Used after init and new package scaffolding.
func PromptAndWriteLinks(projectDir string) error {
	selected, err := link.PromptIntegrations(projectDir, "Set up AI tool entry points?")
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		return nil
	}

	// Load fresh config (may have been just written by scaffold).
	cfg, err := project.LoadConfig(filepath.Join(projectDir, project.ConfigFileName))
	if err != nil {
		return err
	}

	contextRelPath := project.ContextRelPath(cfg.Root)

	results, err := link.Write(projectDir, contextRelPath, selected)
	if err != nil {
		return err
	}

	fmt.Print(link.RenderLinkResults(results))
	return nil
}
