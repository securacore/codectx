package compile

import (
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/internal/util"
)

// filterManifest returns a new manifest containing only the entries
// that match the given activation state.
func filterManifest(m *manifest.Manifest, activation config.Activation) *manifest.Manifest {
	if activation.IsNone() {
		return &manifest.Manifest{
			Name:        m.Name,
			Author:      m.Author,
			Version:     m.Version,
			Description: m.Description,
		}
	}

	if activation.IsAll() {
		return m
	}

	// Granular activation: filter by listed IDs.
	am := activation.Map
	filtered := &manifest.Manifest{
		Name:        m.Name,
		Author:      m.Author,
		Version:     m.Version,
		Description: m.Description,
	}

	if am.Foundation != nil {
		ids := util.ToSet(am.Foundation)
		for _, e := range m.Foundation {
			if ids[e.ID] {
				filtered.Foundation = append(filtered.Foundation, e)
			}
		}
	}

	if am.Application != nil {
		ids := util.ToSet(am.Application)
		for _, e := range m.Application {
			if ids[e.ID] {
				filtered.Application = append(filtered.Application, e)
			}
		}
	}

	if am.Topics != nil {
		ids := util.ToSet(am.Topics)
		for _, e := range m.Topics {
			if ids[e.ID] {
				filtered.Topics = append(filtered.Topics, e)
			}
		}
	}

	if am.Prompts != nil {
		ids := util.ToSet(am.Prompts)
		for _, e := range m.Prompts {
			if ids[e.ID] {
				filtered.Prompts = append(filtered.Prompts, e)
			}
		}
	}

	if am.Plans != nil {
		ids := util.ToSet(am.Plans)
		for _, e := range m.Plans {
			if ids[e.ID] {
				filtered.Plans = append(filtered.Plans, e)
			}
		}
	}

	return filtered
}
