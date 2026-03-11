package query

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
	"github.com/securacore/codectx/core/project"
	"github.com/securacore/codectx/core/taxonomy"
	"github.com/securacore/codectx/core/tokens"
)

// tmpDir is the directory where generated files are written.
var tmpDir = filepath.Join(os.TempDir(), "codectx")

// maxSlugLength is the maximum character length for topic slugs.
const maxSlugLength = 60

// GenerateResult holds the result of a generate (chunk assembly) operation.
type GenerateResult struct {
	// FilePath is the absolute path to the generated file.
	FilePath string

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
// Reasoning), sorts within groups by source+sequence, writes the assembled
// document to /tmp/codectx/, and returns a summary.
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

	// Count tokens.
	counter, err := tokens.New(encoding)
	if err != nil {
		return nil, fmt.Errorf("creating token counter: %w", err)
	}
	totalTokens, err := counter.Count(doc)
	if err != nil {
		return nil, fmt.Errorf("counting tokens: %w", err)
	}

	// Derive topic slug and write to file.
	slug := topicSlug(resolved)
	filePath, err := writeGeneratedFile(slug, doc)
	if err != nil {
		return nil, fmt.Errorf("writing generated file: %w", err)
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
		FilePath:    filePath,
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
	lastSource := ""

	for _, rc := range resolved {
		// Section change.
		if rc.typeRank != currentRank {
			currentRank = rc.typeRank
			lastSource = ""

			b.WriteString("\n# ")
			b.WriteString(sectionTitle(currentRank))
			b.WriteString("\n")

			if preamble := sectionPreamble(currentRank); preamble != "" {
				b.WriteString("\n")
				b.WriteString(preamble)
			}
		}

		// Source file change within a section — insert separator.
		if lastSource != "" && rc.entry.Source != lastSource {
			b.WriteString("\n---\n")
		}
		lastSource = rc.entry.Source

		// Write heading and content.
		b.WriteString("\n")
		b.WriteString(rc.content)
	}

	// Write related chunks footer.
	b.WriteString("\n---\n")
	b.WriteString("<!-- codectx:related\n")
	b.WriteString("Use codectx generate to load additional chunks listed in the query results.\n")
	b.WriteString("-->\n")

	return b.String()
}

// topicSlug derives a kebab-case topic slug from the heading of the first chunk.
// The slug is used as the filename prefix for generated documents.
// Delegates to taxonomy.NormalizeKey for the core slug transformation,
// then applies truncation and a fallback for empty results.
func topicSlug(resolved []resolvedChunk) string {
	if len(resolved) == 0 {
		return "generated"
	}

	heading := resolved[0].entry.Heading
	result := taxonomy.NormalizeKey(heading)

	if result == "" {
		return "generated"
	}

	// Truncate to maxSlugLength characters.
	if len(result) > maxSlugLength {
		result = result[:maxSlugLength]
	}
	return strings.TrimRight(result, "-")
}

// writeGeneratedFile writes the assembled document to /tmp/codectx/.
func writeGeneratedFile(slug, content string) (string, error) {
	if err := os.MkdirAll(tmpDir, project.DirPerm); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s.%d.md", slug, timestamp)
	path := filepath.Join(tmpDir, filename)

	if err := os.WriteFile(path, []byte(content), project.FilePerm); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	return path, nil
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
