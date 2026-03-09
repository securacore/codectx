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

// writeSubManifest writes a sub-manifest to disk and returns the ManifestRef.
func writeSubManifest(manifestsDir, section string, sub *CompiledManifest, entryCount, tokens int) (ManifestRef, error) {
	subPath := filepath.Join(manifestsDir, section+".yml")
	if err := WriteCompiledManifest(subPath, sub); err != nil {
		return ManifestRef{}, fmt.Errorf("write %s sub-manifest: %w", section, err)
	}
	return ManifestRef{
		Section:         section,
		Path:            "manifests/" + section + ".yml",
		Entries:         entryCount,
		EstimatedTokens: tokens,
		Description:     sectionDescriptions[section],
	}, nil
}

// sectionTokens returns the estimated token count for a section from heuristics.
func sectionTokens(h *Heuristics, section string) int {
	if h == nil {
		return 0
	}
	switch section {
	case "foundation":
		if h.Sections.Foundation != nil {
			return h.Sections.Foundation.EstimatedTokens
		}
	case "application":
		if h.Sections.Application != nil {
			return h.Sections.Application.EstimatedTokens
		}
	case "topics":
		if h.Sections.Topics != nil {
			return h.Sections.Topics.EstimatedTokens
		}
	case "prompts":
		if h.Sections.Prompts != nil {
			return h.Sections.Prompts.EstimatedTokens
		}
	case "plans":
		if h.Sections.Plans != nil {
			return h.Sections.Plans.EstimatedTokens
		}
	default:
	}
	return 0
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

	// Separate always-load foundation entries for inlining in root.
	var alwaysLoad, nonAlwaysLoad []CompiledFoundationEntry
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
			Name: cm.Name + " - Foundation", Description: sectionDescriptions["foundation"],
			Foundation: nonAlwaysLoad,
		}
		ref, err := writeSubManifest(manifestsDir, "foundation", sub, len(nonAlwaysLoad), sectionTokens(h, "foundation"))
		if err != nil {
			return err
		}
		refs = append(refs, ref)
	}

	// Application sub-manifest (split always-load like foundation).
	if len(cm.Application) > 0 {
		var alwaysLoadApp, nonAlwaysLoadApp []CompiledApplicationEntry
		for _, e := range cm.Application {
			if e.Load == "always" {
				alwaysLoadApp = append(alwaysLoadApp, e)
			} else {
				nonAlwaysLoadApp = append(nonAlwaysLoadApp, e)
			}
		}
		if len(nonAlwaysLoadApp) > 0 {
			sub := &CompiledManifest{
				Name: cm.Name + " - Application", Description: sectionDescriptions["application"],
				Application: nonAlwaysLoadApp,
			}
			ref, err := writeSubManifest(manifestsDir, "application", sub, len(nonAlwaysLoadApp), sectionTokens(h, "application"))
			if err != nil {
				return err
			}
			refs = append(refs, ref)
		}
		cm.Application = alwaysLoadApp
	}

	// Topics sub-manifest.
	if len(cm.Topics) > 0 {
		sub := &CompiledManifest{
			Name: cm.Name + " - Topics", Description: sectionDescriptions["topics"],
			Topics: cm.Topics,
		}
		ref, err := writeSubManifest(manifestsDir, "topics", sub, len(cm.Topics), sectionTokens(h, "topics"))
		if err != nil {
			return err
		}
		refs = append(refs, ref)
	}

	// Prompts sub-manifest.
	if len(cm.Prompts) > 0 {
		sub := &CompiledManifest{
			Name: cm.Name + " - Prompts", Description: sectionDescriptions["prompts"],
			Prompts: cm.Prompts,
		}
		ref, err := writeSubManifest(manifestsDir, "prompts", sub, len(cm.Prompts), sectionTokens(h, "prompts"))
		if err != nil {
			return err
		}
		refs = append(refs, ref)
	}

	// Plans sub-manifest.
	if len(cm.Plans) > 0 {
		sub := &CompiledManifest{
			Name: cm.Name + " - Plans", Description: sectionDescriptions["plans"],
			Plans: cm.Plans,
		}
		ref, err := writeSubManifest(manifestsDir, "plans", sub, len(cm.Plans), sectionTokens(h, "plans"))
		if err != nil {
			return err
		}
		refs = append(refs, ref)
	}

	// Rewrite root manifest: keep only always-load entries + refs.
	cm.Foundation = alwaysLoad
	cm.Topics = nil
	cm.Prompts = nil
	cm.Plans = nil
	cm.Manifests = refs

	return nil
}
