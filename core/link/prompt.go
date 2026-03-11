package link

import (
	"fmt"

	"charm.land/huh/v2"
)

// PromptIntegrations shows a multi-select prompt for choosing integrations.
// Auto-detected integrations are pre-selected. The title parameter customizes
// the prompt heading.
func PromptIntegrations(projectDir, title string) ([]Integration, error) {
	detected := Detect(projectDir)
	all := AllIntegrations()

	// Build options and pre-select detected ones.
	options := make([]huh.Option[Integration], len(all))
	preselected := make([]Integration, 0)

	for i, info := range all {
		options[i] = huh.NewOption[Integration](
			fmt.Sprintf("%s (%s)", info.Name, info.Description),
			info.Type,
		)
		for _, d := range detected {
			if d == info.Type {
				preselected = append(preselected, info.Type)
				break
			}
		}
	}

	selected := preselected

	if err := huh.NewMultiSelect[Integration]().
		Title(title).
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return nil, err
	}

	return selected, nil
}
