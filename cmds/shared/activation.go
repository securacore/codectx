package shared

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/securacore/codectx/core/compile"
	"github.com/securacore/codectx/core/config"
	"github.com/securacore/codectx/core/manifest"
)

// ParseActivateFlag parses the --activate flag value into an Activation.
// Accepted values: "all", "none", or "section:id,section:id,..."
func ParseActivateFlag(value string) (config.Activation, error) {
	if value == "all" {
		return config.Activation{Mode: "all"}, nil
	}
	if value == "none" {
		return config.Activation{Mode: "none"}, nil
	}

	// Parse granular: "topics:react,foundation:philosophy"
	am := &config.ActivationMap{}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			return config.Activation{}, fmt.Errorf("invalid format %q: expected section:id", part)
		}
		section := part[:colonIdx]
		id := part[colonIdx+1:]
		if id == "" {
			return config.Activation{}, fmt.Errorf("empty id in %q", part)
		}

		switch section {
		case "foundation":
			am.Foundation = append(am.Foundation, id)
		case "application":
			am.Application = append(am.Application, id)
		case "topics":
			am.Topics = append(am.Topics, id)
		case "prompts":
			am.Prompts = append(am.Prompts, id)
		case "plans":
			am.Plans = append(am.Plans, id)
		default:
			return config.Activation{}, fmt.Errorf("unknown section %q in %q", section, part)
		}
	}

	return config.Activation{Map: am}, nil
}

// Collision represents a single entry ID that collides with an already-active entry.
type Collision struct {
	Section string
	ID      string
	Pkg     string // "local" or "name@author"
}

// DetectCollisions checks if activating a package manifest would collide
// with entries already active in the local manifest or other installed packages.
// Use skipIdx >= 0 to exclude a specific config package index from the check
// (e.g., when re-activating an existing package). Use skipIdx < 0 to check
// against all packages.
func DetectCollisions(cfg *config.Config, skipIdx int, pkgManifest *manifest.Manifest, activation config.Activation) []Collision {
	activeIDs := make(map[string]string) // "section:id" -> source label

	docsDir := cfg.DocsDir()
	localManifestPath := filepath.Join(docsDir, "manifest.yml")
	if localManifest, err := manifest.Load(localManifestPath); err == nil {
		localManifest = manifest.Sync(docsDir, localManifest)
		for key := range compile.CollectActiveIDs(localManifest) {
			activeIDs[key] = "local"
		}
	}

	for i, pkg := range cfg.Packages {
		if i == skipIdx || pkg.Active.IsNone() {
			continue
		}
		pkgDir := filepath.Join(docsDir, "packages", fmt.Sprintf("%s@%s", pkg.Name, pkg.Author))
		pkgPath := filepath.Join(pkgDir, "manifest.yml")
		m, err := manifest.Load(pkgPath)
		if err != nil {
			continue
		}
		m = manifest.Discover(pkgDir, m)
		filtered := FilterManifestForIDs(m, pkg.Active)
		pkgLabel := fmt.Sprintf("%s@%s", pkg.Name, pkg.Author)
		for key := range compile.CollectActiveIDs(filtered) {
			activeIDs[key] = pkgLabel
		}
	}

	filtered := FilterManifestForIDs(pkgManifest, activation)
	newIDs := compile.CollectActiveIDs(filtered)

	var collisions []Collision
	for key := range newIDs {
		if pkg, exists := activeIDs[key]; exists {
			section, id := SplitKey(key)
			collisions = append(collisions, Collision{Section: section, ID: id, Pkg: pkg})
		}
	}

	return collisions
}

// FilterManifestForIDs applies activation filtering to a manifest for ID
// collection. This is the CLI-layer filter that does not preserve manifest
// metadata (Name, Author, etc). Use compile.filterManifest for the
// build-pipeline version that preserves metadata.
func FilterManifestForIDs(m *manifest.Manifest, activation config.Activation) *manifest.Manifest {
	if activation.IsAll() {
		return m
	}
	if activation.IsNone() {
		return &manifest.Manifest{}
	}

	am := activation.Map
	filtered := &manifest.Manifest{}

	if am.Foundation != nil {
		ids := ToSet(am.Foundation)
		for _, e := range m.Foundation {
			if ids[e.ID] {
				filtered.Foundation = append(filtered.Foundation, e)
			}
		}
	}
	if am.Application != nil {
		ids := ToSet(am.Application)
		for _, e := range m.Application {
			if ids[e.ID] {
				filtered.Application = append(filtered.Application, e)
			}
		}
	}
	if am.Topics != nil {
		ids := ToSet(am.Topics)
		for _, e := range m.Topics {
			if ids[e.ID] {
				filtered.Topics = append(filtered.Topics, e)
			}
		}
	}
	if am.Prompts != nil {
		ids := ToSet(am.Prompts)
		for _, e := range m.Prompts {
			if ids[e.ID] {
				filtered.Prompts = append(filtered.Prompts, e)
			}
		}
	}
	if am.Plans != nil {
		ids := ToSet(am.Plans)
		for _, e := range m.Plans {
			if ids[e.ID] {
				filtered.Plans = append(filtered.Plans, e)
			}
		}
	}

	return filtered
}

// ToSet converts a string slice to a set for O(1) lookups.
func ToSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

// SplitKey splits "section:id" into its parts.
func SplitKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
