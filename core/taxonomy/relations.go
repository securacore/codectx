package taxonomy

import (
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/markdown"
)

// inferRelationships performs Pass 2 of the taxonomy extraction pipeline.
// It adds broader/narrower and related relationships to the taxonomy using
// deterministic structural analysis (no NLP required).
//
// Relationship sources:
//   - Heading hierarchy: H2 under H1 yields parent/child.
//   - Cross-references: markdown links between documents yield lateral
//     "related" relationships between terms found in source and target.
func inferRelationships(tax *Taxonomy, chunks []chunk.Chunk) {
	inferHeadingHierarchy(tax, chunks)
	inferCrossReferences(tax, chunks)
}

// inferHeadingHierarchy derives broader/narrower relationships from heading
// hierarchy within each chunk's blocks.
//
// If a chunk has heading blocks with breadcrumbs like ["Auth", "OAuth"],
// and both "auth" and "oauth" exist as terms, then:
//   - oauth.broader = "auth"
//   - auth.narrower includes "oauth"
func inferHeadingHierarchy(tax *Taxonomy, chunks []chunk.Chunk) {
	// Track which broader/narrower pairs we've already set to avoid duplicates.
	type pair struct{ parent, child string }
	seen := make(map[pair]bool)

	for i := range chunks {
		c := &chunks[i]
		for _, block := range c.Blocks {
			if block.Type != markdown.BlockHeading {
				continue
			}

			hierarchy := block.Heading
			if len(hierarchy) < 2 {
				continue
			}

			// Walk adjacent pairs in the hierarchy.
			for j := 0; j < len(hierarchy)-1; j++ {
				parentKey := NormalizeKey(hierarchy[j])
				childKey := NormalizeKey(hierarchy[j+1])

				if parentKey == childKey || parentKey == "" || childKey == "" {
					continue
				}

				parentTerm := tax.Terms[parentKey]
				childTerm := tax.Terms[childKey]

				if parentTerm == nil || childTerm == nil {
					continue
				}

				p := pair{parentKey, childKey}
				if seen[p] {
					continue
				}
				seen[p] = true

				// Set broader on child (only if not already set to something else).
				if childTerm.Broader == "" {
					childTerm.Broader = parentKey
				}

				// Add to parent's narrower list (if not already present).
				if !slices.Contains(parentTerm.Narrower, childKey) {
					parentTerm.Narrower = append(parentTerm.Narrower, childKey)
				}
			}
		}
	}

	// Sort narrower lists for deterministic output.
	for _, term := range tax.Terms {
		if len(term.Narrower) > 1 {
			sort.Strings(term.Narrower)
		}
	}
}

// mdLinkPattern matches markdown links: [text](url).
var mdLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// inferCrossReferences derives lateral "related" relationships from
// markdown links between documents.
//
// When a chunk in file A links to file B, and both files have heading-derived
// terms, those terms become related.
func inferCrossReferences(tax *Taxonomy, chunks []chunk.Chunk) {
	// Build a map of source file -> heading-derived term keys.
	termsBySource := make(map[string][]string)
	for key, term := range tax.Terms {
		if term.Source != SourceHeading {
			continue
		}
		for _, chunkID := range term.Chunks {
			for i := range chunks {
				if chunks[i].ID == chunkID {
					src := chunks[i].Source
					if !slices.Contains(termsBySource[src], key) {
						termsBySource[src] = append(termsBySource[src], key)
					}
					break
				}
			}
		}
	}

	// Track related pairs to avoid duplicates.
	type pair struct{ a, b string }
	seen := make(map[pair]bool)

	for i := range chunks {
		c := &chunks[i]
		sourceTerms := termsBySource[c.Source]
		if len(sourceTerms) == 0 {
			continue
		}

		for _, block := range c.Blocks {
			links := mdLinkPattern.FindAllStringSubmatch(block.Content, -1)
			for _, match := range links {
				if len(match) < 3 {
					continue
				}
				linkTarget := match[2]

				// Resolve relative link to a source path.
				targetPath := resolveLink(c.Source, linkTarget)
				targetTerms := termsBySource[targetPath]
				if len(targetTerms) == 0 {
					continue
				}

				// Add related relationships between source terms and target terms.
				for _, srcKey := range sourceTerms {
					for _, tgtKey := range targetTerms {
						if srcKey == tgtKey {
							continue
						}

						// Normalize the pair order for deduplication.
						p := pair{srcKey, tgtKey}
						if srcKey > tgtKey {
							p = pair{tgtKey, srcKey}
						}
						if seen[p] {
							continue
						}
						seen[p] = true

						// Skip if already in a broader/narrower relationship.
						srcTerm := tax.Terms[srcKey]
						tgtTerm := tax.Terms[tgtKey]
						if srcTerm.Broader == tgtKey || tgtTerm.Broader == srcKey {
							continue
						}

						if !slices.Contains(srcTerm.Related, tgtKey) {
							srcTerm.Related = append(srcTerm.Related, tgtKey)
						}
						if !slices.Contains(tgtTerm.Related, srcKey) {
							tgtTerm.Related = append(tgtTerm.Related, srcKey)
						}
					}
				}
			}
		}
	}

	// Sort related lists for deterministic output.
	for _, term := range tax.Terms {
		if len(term.Related) > 1 {
			sort.Strings(term.Related)
		}
	}
}

// resolveLink resolves a relative markdown link target against a source path.
// It handles simple relative paths (../foo.md, ./bar.md, baz.md).
// Fragment links (#section) and absolute URLs are ignored (returns empty).
func resolveLink(sourcePath, linkTarget string) string {
	// Ignore fragment-only links.
	if strings.HasPrefix(linkTarget, "#") {
		return ""
	}

	// Ignore absolute URLs.
	if strings.Contains(linkTarget, "://") {
		return ""
	}

	// Strip fragment from the target.
	if idx := strings.Index(linkTarget, "#"); idx >= 0 {
		linkTarget = linkTarget[:idx]
	}

	if linkTarget == "" {
		return ""
	}

	// Resolve relative to the source file's directory.
	sourceDir := ""
	if idx := strings.LastIndex(sourcePath, "/"); idx >= 0 {
		sourceDir = sourcePath[:idx]
	}

	// Join and normalize.
	var resolved string
	if sourceDir == "" {
		resolved = linkTarget
	} else {
		resolved = sourceDir + "/" + linkTarget
	}

	// Normalize path: handle ../ and ./ segments.
	return normalizePath(resolved)
}

// normalizePath resolves . and .. segments in a forward-slash path.
func normalizePath(path string) string {
	parts := strings.Split(path, "/")
	var result []string

	for _, p := range parts {
		switch p {
		case ".":
			continue
		case "..":
			if len(result) > 0 {
				result = result[:len(result)-1]
			}
		default:
			if p != "" {
				result = append(result, p)
			}
		}
	}

	return strings.Join(result, "/")
}
