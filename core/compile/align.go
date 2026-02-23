package compile

import (
	"securacore/codectx/core/config"
	"securacore/codectx/core/manifest"
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
		ids := toSet(am.Foundation)
		for _, e := range m.Foundation {
			if ids[e.ID] {
				filtered.Foundation = append(filtered.Foundation, e)
			}
		}
	}

	if am.Topics != nil {
		ids := toSet(am.Topics)
		for _, e := range m.Topics {
			if ids[e.ID] {
				filtered.Topics = append(filtered.Topics, e)
			}
		}
	}

	if am.Prompts != nil {
		ids := toSet(am.Prompts)
		for _, e := range m.Prompts {
			if ids[e.ID] {
				filtered.Prompts = append(filtered.Prompts, e)
			}
		}
	}

	if am.Plans != nil {
		ids := toSet(am.Plans)
		for _, e := range m.Plans {
			if ids[e.ID] {
				filtered.Plans = append(filtered.Plans, e)
			}
		}
	}

	return filtered
}

// mergeManifest appends all entries from src into dst.
func mergeManifest(dst, src *manifest.Manifest) {
	dst.Foundation = append(dst.Foundation, src.Foundation...)
	dst.Topics = append(dst.Topics, src.Topics...)
	dst.Prompts = append(dst.Prompts, src.Prompts...)
	dst.Plans = append(dst.Plans, src.Plans...)
}

// toSet converts a string slice to a set for O(1) lookups.
func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
