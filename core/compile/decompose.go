package compile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Decomposition thresholds. If any threshold is exceeded, the compiled
// manifest is split into sub-manifests per section. The root manifest
// retains always-load entries and ManifestRef pointers.
const (
	ThresholdEntries = 500
	ThresholdBytes   = 50 * 1024  // 50 KB
	ThresholdTokens  = 100 * 1000 // 100k tokens
)

// sectionDescriptions maps section names to human-readable descriptions.
var sectionDescriptions = map[string]string{
	"foundation":  "Core operational context",
	"application": "Product architecture and design documentation",
	"topics":      "Technology and domain conventions",
	"prompts":     "Automated task definitions",
	"plans":       "Implementation plans with state tracking",
}

// shouldDecompose checks whether the compiled manifest exceeds any
// decomposition threshold and should be split into sub-manifests.
func shouldDecompose(h *Heuristics) bool {
	if h == nil {
		return false
	}
	return h.Totals.Entries > ThresholdEntries ||
		h.Totals.SizeBytes > ThresholdBytes ||
		h.Totals.EstimatedTokens > ThresholdTokens
}

// decompose splits a compiled manifest into a root manifest (with
// always-load entries and ManifestRef pointers) and per-section
// sub-manifests. Sub-manifest files are written to outputDir/manifests/.
// The root manifest is modified in place.
func decompose(cm *CompiledManifest, h *Heuristics, outputDir string) error {
	manifestsDir := filepath.Join(outputDir, "manifests")
	if err := os.MkdirAll(manifestsDir, 0o755); err != nil {
		return fmt.Errorf("create manifests directory: %w", err)
	}

	// Extract always-load foundation entries for inlining in root.
	var alwaysLoad []CompiledFoundationEntry
	var nonAlwaysLoad []CompiledFoundationEntry
	for _, e := range cm.Foundation {
		if e.Load == "always" {
			alwaysLoad = append(alwaysLoad, e)
		} else {
			nonAlwaysLoad = append(nonAlwaysLoad, e)
		}
	}

	var refs []ManifestRef

	// Foundation sub-manifest (non-always-load entries only).
	if len(nonAlwaysLoad) > 0 {
		sub := &CompiledManifest{
			Name:        cm.Name + " - Foundation",
			Description: sectionDescriptions["foundation"],
			Foundation:  nonAlwaysLoad,
		}
		subPath := filepath.Join(manifestsDir, "foundation.yml")
		if err := WriteCompiledManifest(subPath, sub); err != nil {
			return fmt.Errorf("write foundation sub-manifest: %w", err)
		}

		tokens := 0
		if h != nil && h.Sections.Foundation != nil {
			tokens = h.Sections.Foundation.EstimatedTokens
		}

		refs = append(refs, ManifestRef{
			Section:         "foundation",
			Path:            "manifests/foundation.yml",
			Entries:         len(nonAlwaysLoad),
			EstimatedTokens: tokens,
			Description:     sectionDescriptions["foundation"],
		})
	}

	// Application sub-manifest.
	if len(cm.Application) > 0 {
		// Extract always-load application entries for inlining in root.
		var alwaysLoadApp []CompiledApplicationEntry
		var nonAlwaysLoadApp []CompiledApplicationEntry
		for _, e := range cm.Application {
			if e.Load == "always" {
				alwaysLoadApp = append(alwaysLoadApp, e)
			} else {
				nonAlwaysLoadApp = append(nonAlwaysLoadApp, e)
			}
		}

		if len(nonAlwaysLoadApp) > 0 {
			sub := &CompiledManifest{
				Name:        cm.Name + " - Application",
				Description: sectionDescriptions["application"],
				Application: nonAlwaysLoadApp,
			}
			subPath := filepath.Join(manifestsDir, "application.yml")
			if err := WriteCompiledManifest(subPath, sub); err != nil {
				return fmt.Errorf("write application sub-manifest: %w", err)
			}

			tokens := 0
			if h != nil && h.Sections.Application != nil {
				tokens = h.Sections.Application.EstimatedTokens
			}

			refs = append(refs, ManifestRef{
				Section:         "application",
				Path:            "manifests/application.yml",
				Entries:         len(nonAlwaysLoadApp),
				EstimatedTokens: tokens,
				Description:     sectionDescriptions["application"],
			})
		}

		// Keep only always-load application entries in root.
		cm.Application = alwaysLoadApp
	}

	// Topics sub-manifest.
	if len(cm.Topics) > 0 {
		sub := &CompiledManifest{
			Name:        cm.Name + " - Topics",
			Description: sectionDescriptions["topics"],
			Topics:      cm.Topics,
		}
		subPath := filepath.Join(manifestsDir, "topics.yml")
		if err := WriteCompiledManifest(subPath, sub); err != nil {
			return fmt.Errorf("write topics sub-manifest: %w", err)
		}

		tokens := 0
		if h != nil && h.Sections.Topics != nil {
			tokens = h.Sections.Topics.EstimatedTokens
		}

		refs = append(refs, ManifestRef{
			Section:         "topics",
			Path:            "manifests/topics.yml",
			Entries:         len(cm.Topics),
			EstimatedTokens: tokens,
			Description:     sectionDescriptions["topics"],
		})
	}

	// Prompts sub-manifest.
	if len(cm.Prompts) > 0 {
		sub := &CompiledManifest{
			Name:        cm.Name + " - Prompts",
			Description: sectionDescriptions["prompts"],
			Prompts:     cm.Prompts,
		}
		subPath := filepath.Join(manifestsDir, "prompts.yml")
		if err := WriteCompiledManifest(subPath, sub); err != nil {
			return fmt.Errorf("write prompts sub-manifest: %w", err)
		}

		tokens := 0
		if h != nil && h.Sections.Prompts != nil {
			tokens = h.Sections.Prompts.EstimatedTokens
		}

		refs = append(refs, ManifestRef{
			Section:         "prompts",
			Path:            "manifests/prompts.yml",
			Entries:         len(cm.Prompts),
			EstimatedTokens: tokens,
			Description:     sectionDescriptions["prompts"],
		})
	}

	// Plans sub-manifest.
	if len(cm.Plans) > 0 {
		sub := &CompiledManifest{
			Name:        cm.Name + " - Plans",
			Description: sectionDescriptions["plans"],
			Plans:       cm.Plans,
		}
		subPath := filepath.Join(manifestsDir, "plans.yml")
		if err := WriteCompiledManifest(subPath, sub); err != nil {
			return fmt.Errorf("write plans sub-manifest: %w", err)
		}

		tokens := 0
		if h != nil && h.Sections.Plans != nil {
			tokens = h.Sections.Plans.EstimatedTokens
		}

		refs = append(refs, ManifestRef{
			Section:         "plans",
			Path:            "manifests/plans.yml",
			Entries:         len(cm.Plans),
			EstimatedTokens: tokens,
			Description:     sectionDescriptions["plans"],
		})
	}

	// Rewrite root manifest: keep only always-load entries + refs.
	cm.Foundation = alwaysLoad
	// cm.Application is already set to alwaysLoadApp above (or left as-is if empty).
	cm.Topics = nil
	cm.Prompts = nil
	cm.Plans = nil
	cm.Manifests = refs

	return nil
}
