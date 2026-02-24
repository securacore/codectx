package compile

import (
	"github.com/securacore/codectx/core/manifest"
)

// CompiledManifest is the consumption-format manifest written to .codectx/manifest.yml.
// It is the AI's primary interface: a 2-way data map with content-addressed object
// references and provenance tracking. Separate from the source manifest.Manifest type
// which is the authoring format for package.yml files.
//
// A CompiledManifest can contain direct entries, sub-manifest references, or both.
// In single mode (below decomposition threshold), section arrays hold all entries.
// In multi mode (above threshold), the root manifest holds always-load entries and
// Manifests references; section entries live in sub-manifest files.
// Sub-manifests use the same CompiledManifest structure, enabling recursive
// decomposition at any depth.
type CompiledManifest struct {
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description"`
	Foundation  []CompiledFoundationEntry `yaml:"foundation,omitempty"`
	Topics      []CompiledTopicEntry      `yaml:"topics,omitempty"`
	Prompts     []CompiledPromptEntry     `yaml:"prompts,omitempty"`
	Plans       []CompiledPlanEntry       `yaml:"plans,omitempty"`
	Manifests   []ManifestRef             `yaml:"manifests,omitempty"`
}

// ManifestRef is a reference to a sub-manifest file.
// Used when a manifest is decomposed into multiple files for scale.
// At level 1, decomposition is by section (foundation, topics, etc.).
// At level 2+, decomposition is by source package within a section.
type ManifestRef struct {
	Section         string `yaml:"section"`
	Path            string `yaml:"path"`
	Entries         int    `yaml:"entries"`
	EstimatedTokens int    `yaml:"estimated_tokens"`
	Description     string `yaml:"description"`
	Source          string `yaml:"source,omitempty"`
}

// CompiledFoundationEntry is the compiled form of a foundation document.
type CompiledFoundationEntry struct {
	ID          string   `yaml:"id"`
	Object      string   `yaml:"object"`
	Description string   `yaml:"description"`
	Load        string   `yaml:"load,omitempty"`
	Source      string   `yaml:"source"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	RequiredBy  []string `yaml:"required_by,omitempty"`
}

// CompiledTopicEntry is the compiled form of a topic entry.
type CompiledTopicEntry struct {
	ID          string   `yaml:"id"`
	Object      string   `yaml:"object"`
	Description string   `yaml:"description"`
	Spec        string   `yaml:"spec,omitempty"`
	Files       []string `yaml:"files,omitempty"`
	Source      string   `yaml:"source"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	RequiredBy  []string `yaml:"required_by,omitempty"`
}

// CompiledPromptEntry is the compiled form of a prompt entry.
type CompiledPromptEntry struct {
	ID          string   `yaml:"id"`
	Object      string   `yaml:"object"`
	Description string   `yaml:"description"`
	Source      string   `yaml:"source"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	RequiredBy  []string `yaml:"required_by,omitempty"`
}

// CompiledPlanEntry is the compiled form of a plan entry.
type CompiledPlanEntry struct {
	ID          string   `yaml:"id"`
	Object      string   `yaml:"object"`
	Description string   `yaml:"description"`
	State       string   `yaml:"state,omitempty"`
	Source      string   `yaml:"source"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	RequiredBy  []string `yaml:"required_by,omitempty"`
}

// toCompiledManifest converts a unified source manifest into a compiled manifest.
// pathToHash maps source file paths to their 16-char content hashes.
// provenance maps "section:id" keys to source labels ("local" or "name@author").
func toCompiledManifest(
	unified *manifest.Manifest,
	pathToHash map[string]string,
	provenance map[string]string,
) *CompiledManifest {
	cm := &CompiledManifest{
		Name:        unified.Name,
		Description: unified.Description,
	}

	for _, e := range unified.Foundation {
		ce := CompiledFoundationEntry{
			ID:          e.ID,
			Object:      ObjectPath(pathToHash[e.Path]),
			Description: e.Description,
			Load:        e.Load,
			Source:      provenance["foundation:"+e.ID],
			DependsOn:   e.DependsOn,
			RequiredBy:  e.RequiredBy,
		}
		cm.Foundation = append(cm.Foundation, ce)
	}

	for _, e := range unified.Topics {
		ce := CompiledTopicEntry{
			ID:          e.ID,
			Object:      ObjectPath(pathToHash[e.Path]),
			Description: e.Description,
			Source:      provenance["topics:"+e.ID],
			DependsOn:   e.DependsOn,
			RequiredBy:  e.RequiredBy,
		}
		if e.Spec != "" {
			if h, ok := pathToHash[e.Spec]; ok {
				ce.Spec = ObjectPath(h)
			}
		}
		for _, f := range e.Files {
			if h, ok := pathToHash[f]; ok {
				ce.Files = append(ce.Files, ObjectPath(h))
			}
		}
		cm.Topics = append(cm.Topics, ce)
	}

	for _, e := range unified.Prompts {
		ce := CompiledPromptEntry{
			ID:          e.ID,
			Object:      ObjectPath(pathToHash[e.Path]),
			Description: e.Description,
			Source:      provenance["prompts:"+e.ID],
			DependsOn:   e.DependsOn,
			RequiredBy:  e.RequiredBy,
		}
		cm.Prompts = append(cm.Prompts, ce)
	}

	for _, e := range unified.Plans {
		ce := CompiledPlanEntry{
			ID:          e.ID,
			Object:      ObjectPath(pathToHash[e.Path]),
			Description: e.Description,
			Source:      provenance["plans:"+e.ID],
			DependsOn:   e.DependsOn,
			RequiredBy:  e.RequiredBy,
		}
		// State files go to state/{plan-id}.yml, not the object store.
		if e.State != "" {
			ce.State = "state/" + e.ID + ".yml"
		}
		cm.Plans = append(cm.Plans, ce)
	}

	return cm
}
