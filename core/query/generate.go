package query

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/tokens"
)

// GenerateResult holds the result of a generate (chunk assembly) operation.
type GenerateResult struct {
	// Document is the full assembled markdown content.
	Document string

	// ContentHash is the full hex SHA-256 hash of the assembled document.
	ContentHash string

	// TotalTokens is the token count of the generated document.
	TotalTokens int

	// ChunkIDs is the list of chunk IDs that were assembled.
	ChunkIDs []string

	// Sources is the deduplicated list of source files referenced.
	Sources []string

	// Related contains adjacent chunks not included in the assembly.
	Related []RelatedEntry
}

// resolvedChunk holds a chunk's content and manifest metadata, ready for assembly.
type resolvedChunk struct {
	id       string
	entry    *manifest.ManifestEntry
	content  string
	sortKey  string // "type:source:sequence" for ordering
	typeRank int    // 0=object, 1=system, 2=spec (for section grouping)
}

// RunGenerate assembles requested chunks into a single reading document.
//
// It loads chunk content from disk, groups by type (Instructions, System,
// Reasoning), sorts within groups by source+sequence, and returns the
// assembled document content with metadata. The caller is responsible for
// writing the document to stdout, a file, or history.
//
// Returns an error if any chunk ID is not found in the manifest or on disk.
func RunGenerate(compiledDir, encoding string, chunkIDs []string) (*GenerateResult, error) {
	// Load manifest.
	mfst, err := manifest.LoadManifest(manifest.EntryPath(compiledDir))
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	// Resolve all chunks: validate, read content, gather metadata.
	resolved := make([]resolvedChunk, 0, len(chunkIDs))
	for _, id := range chunkIDs {
		entry := mfst.LookupEntry(id)
		if entry == nil {
			return nil, fmt.Errorf("chunk not found in manifest: %s", id)
		}

		content, readErr := chunk.ReadChunkContent(compiledDir, id)
		if readErr != nil {
			return nil, fmt.Errorf("reading chunk %s: %w", id, readErr)
		}

		resolved = append(resolved, resolvedChunk{
			id:       id,
			entry:    entry,
			content:  content,
			sortKey:  fmt.Sprintf("%s:%04d", entry.Source, entry.Sequence),
			typeRank: typeRankFor(entry.Type),
		})
	}

	// Sort: group by type rank, then by source+sequence within each group.
	sort.SliceStable(resolved, func(i, j int) bool {
		if resolved[i].typeRank != resolved[j].typeRank {
			return resolved[i].typeRank < resolved[j].typeRank
		}
		return resolved[i].sortKey < resolved[j].sortKey
	})

	// Assemble the document.
	doc := assembleDocument(resolved, chunkIDs)

	// Compute content hash.
	hash := sha256.Sum256([]byte(doc))
	contentHash := fmt.Sprintf("%x", hash)

	// Count tokens.
	counter, err := tokens.New(encoding)
	if err != nil {
		return nil, fmt.Errorf("creating token counter: %w", err)
	}
	totalTokens, err := counter.Count(doc)
	if err != nil {
		return nil, fmt.Errorf("counting tokens: %w", err)
	}

	// Collect sources (deduplicated, ordered).
	sources := collectSources(resolved)

	// Collect related chunks.
	seen := make(map[string]bool, len(chunkIDs))
	for _, id := range chunkIDs {
		seen[id] = true
	}
	related := CollectRelated(chunkIDs, mfst, seen)

	return &GenerateResult{
		Document:    doc,
		ContentHash: contentHash,
		TotalTokens: totalTokens,
		ChunkIDs:    chunkIDs,
		Sources:     sources,
		Related:     related,
	}, nil
}

// typeRankFor returns the sort order for chunk type grouping.
// Instructions (object) first, System second, Reasoning (spec) last.
func typeRankFor(chunkType string) int {
	switch chunkType {
	case "object":
		return 0
	case "system":
		return 1
	case "spec":
		return 2
	default:
		return 0
	}
}

// sectionTitle returns the section heading for a type rank.
func sectionTitle(rank int) string {
	switch rank {
	case 0:
		return "Instructions"
	case 1:
		return "System"
	case 2:
		return "Reasoning"
	default:
		return "Instructions"
	}
}

// sectionPreamble returns the introductory note for a section, or empty string.
func sectionPreamble(rank int) string {
	if rank == 2 {
		return "> The following sections contain the reasoning behind the instructions above.\n" +
			"> This is informational context for understanding *why* decisions were made.\n" +
			"> Reason about this content before acting on it.\n"
	}
	return ""
}

// assembleDocument builds the full generated markdown document from resolved chunks.
func assembleDocument(resolved []resolvedChunk, chunkIDs []string) string {
	var b strings.Builder

	// Write metadata header.
	b.WriteString("<!-- codectx:generated\n")
	fmt.Fprintf(&b, "chunks: %s\n", strings.Join(chunkIDs, ", "))
	b.WriteString("sources:\n")
	for _, src := range collectSources(resolved) {
		fmt.Fprintf(&b, "  - %s\n", src)
	}
	fmt.Fprintf(&b, "generated: %s\n", time.Now().UTC().Format(time.RFC3339))
	b.WriteString("-->\n")

	// Group by type rank and assemble sections.
	currentRank := -1
	var prev *resolvedChunk

	for i := range resolved {
		rc := &resolved[i]

		// Section change.
		if rc.typeRank != currentRank {
			currentRank = rc.typeRank
			prev = nil

			b.WriteString("\n# ")
			b.WriteString(sectionTitle(currentRank))
			b.WriteString("\n")

			if preamble := sectionPreamble(currentRank); preamble != "" {
				b.WriteString("\n")
				b.WriteString(preamble)
			}
		}

		// Source file change within a section — insert separator.
		if prev != nil && rc.entry.Source != prev.entry.Source {
			b.WriteString("\n---\n")
		}

		// Non-adjacent chunks from the same file — insert bridge.
		if prev != nil && rc.entry.Source == prev.entry.Source {
			if prev.entry.Sequence+1 != rc.entry.Sequence && prev.entry.BridgeToNext != nil {
				fmt.Fprintf(&b, "\n---\n> **Context bridge**: %s\n---\n", *prev.entry.BridgeToNext)
			}
		}

		// Write heading and content.
		b.WriteString("\n")
		b.WriteString(rc.content)

		prev = rc
	}

	// Write related chunks footer.
	b.WriteString("\n---\n")
	b.WriteString("<!-- codectx:related\n")
	b.WriteString("Use codectx generate to load additional chunks listed in the query results.\n")
	b.WriteString("-->\n")

	return b.String()
}

// ParseChunkIDs splits a comma-separated string of chunk IDs, trims
// whitespace from each, and filters out empty entries.
// Used by cmds/generate and core/plan for chunk ID parsing.
func ParseChunkIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// collectSources returns deduplicated source paths in order of first appearance.
func collectSources(resolved []resolvedChunk) []string {
	seen := make(map[string]bool)
	var sources []string
	for _, rc := range resolved {
		if !seen[rc.entry.Source] {
			seen[rc.entry.Source] = true
			sources = append(sources, rc.entry.Source)
		}
	}
	return sources
}
