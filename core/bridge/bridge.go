package bridge

import (
	"fmt"
	"sort"
	"strings"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/manifest"
)

// rakeTopN is the number of key phrases to extract from the chunk tail.
const rakeTopN = 4

// tailChars is the approximate character count for the tail window.
// 150 tokens * ~4 chars/token = 600 chars.
const tailChars = 600

// BridgeInput holds the data needed to generate a bridge for a chunk pair.
// This decouples bridge generation from the chunk/manifest data structures
// so the logic is easily testable.
type BridgeInput struct {
	PrevHeading string
	NextHeading string
	PrevContent string
}

// generate produces a deterministic one-line bridge summary from three layers:
//   - Heading transition (when heading paths diverge between chunks)
//   - RAKE key phrase extraction (top terms from the tail of the previous chunk)
//   - Last sentence (concluding prose sentence when available)
//
// Returns empty string if no layers produce output.
func generate(input BridgeInput) string {
	var parts []string

	// Layer 1: heading transition.
	if h := headingBridge(input.PrevHeading, input.NextHeading); h != "" {
		parts = append(parts, h)
	}

	// Layer 2: RAKE key terms from tail of previous chunk.
	tail := tailWindow(input.PrevContent, tailChars)
	phrases := extractKeyPhrases(tail, rakeTopN)
	if len(phrases) > 0 {
		terms := make([]string, len(phrases))
		for i, p := range phrases {
			terms[i] = p.text
		}
		parts = append(parts, "Established: "+strings.Join(terms, ", "))
	}

	// Layer 3: last prose sentence.
	if s := lastSentence(input.PrevContent); s != "" {
		parts = append(parts, s)
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ". ")
}

// GenerateAll produces deterministic bridge summaries for all adjacent chunk
// pairs within the same source file. It returns a map from chunk ID to bridge
// text. Spec chunks are excluded (they use parent_object linking, not adjacency).
//
// The chunks slice must contain all chunks from the compilation. The manifest
// is used to look up entries for chunk metadata. Only entries where
// BridgeToNext is currently nil receive a deterministic bridge.
func GenerateAll(chunks []chunk.Chunk, mfst *manifest.Manifest) map[string]string {
	bridges := make(map[string]string)

	// Group non-spec chunks by source file.
	type chunkRef struct {
		id       string
		sequence int
		heading  string
		content  string
	}

	bySource := make(map[string][]chunkRef)
	for i := range chunks {
		c := &chunks[i]
		if c.Type == chunk.ChunkSpec {
			continue
		}
		bySource[c.Source] = append(bySource[c.Source], chunkRef{
			id:       c.ID,
			sequence: c.Sequence,
			heading:  c.Heading,
			content:  c.Content,
		})
	}

	for _, fileChunks := range bySource {
		sort.Slice(fileChunks, func(i, j int) bool {
			return fileChunks[i].sequence < fileChunks[j].sequence
		})

		for i := 0; i < len(fileChunks)-1; i++ {
			prev := fileChunks[i]
			next := fileChunks[i+1]

			// Skip if this entry already has a bridge (e.g. from incremental cache).
			entry := mfst.LookupEntry(prev.id)
			if entry != nil && entry.BridgeToNext != nil {
				continue
			}

			bridge := generate(BridgeInput{
				PrevHeading: prev.heading,
				NextHeading: next.heading,
				PrevContent: prev.content,
			})

			if bridge != "" {
				bridges[prev.id] = bridge
			}
		}
	}

	return bridges
}

// headingBridge computes the heading transition between two chunks.
// When headings diverge, it reports what section was completed and what
// is being entered. Returns empty string when headings are identical.
func headingBridge(prevHeading, nextHeading string) string {
	if prevHeading == nextHeading {
		return ""
	}

	prevParts := splitHeading(prevHeading)
	nextParts := splitHeading(nextHeading)

	// Find the shared prefix depth.
	shared := 0
	for i := 0; i < min(len(prevParts), len(nextParts)); i++ {
		if prevParts[i] == nextParts[i] {
			shared++
		} else {
			break
		}
	}

	leaving := strings.Join(prevParts[shared:], " > ")
	entering := strings.Join(nextParts[shared:], " > ")

	switch {
	case leaving == "":
		return fmt.Sprintf("Entering: %s", entering)
	case entering == "":
		return fmt.Sprintf("Completed: %s", leaving)
	default:
		return fmt.Sprintf("Completed: %s. Entering: %s", leaving, entering)
	}
}

// splitHeading splits a heading breadcrumb string into its component parts.
func splitHeading(heading string) []string {
	if heading == "" {
		return nil
	}
	return strings.Split(heading, chunk.HeadingSeparator)
}

// tailWindow returns the last ~targetChars characters of text, breaking at
// a word boundary. Returns the full text if it's shorter than the target.
func tailWindow(content string, targetChars int) string {
	if len(content) <= targetChars {
		return content
	}
	cutoff := len(content) - targetChars
	// Find next space after cutoff to break at word boundary.
	idx := strings.IndexByte(content[cutoff:], ' ')
	if idx < 0 {
		return content[cutoff:]
	}
	return content[cutoff+idx+1:]
}

// lastSentence extracts the last prose sentence from chunk content.
// Returns empty string when the content ends with a code block, is too
// short, or looks like a heading rather than prose.
func lastSentence(text string) string {
	cleaned := strings.TrimSpace(text)
	if cleaned == "" {
		return ""
	}

	// Skip if the content ends inside a code block (odd number of fences).
	if strings.Count(cleaned, "```")%2 != 0 {
		return ""
	}

	// Skip if the content ends with a code fence (last block is code).
	lines := strings.Split(cleaned, "\n")
	lastLine := strings.TrimSpace(lines[len(lines)-1])
	if strings.HasPrefix(lastLine, "```") {
		return ""
	}

	// Find the last sentence boundary: period/exclamation/question followed
	// by end-of-string or uppercase letter (new sentence).
	lastBoundary := -1
	for i := len(cleaned) - 1; i >= 0; i-- {
		ch := cleaned[i]
		if ch == '.' || ch == '!' || ch == '?' {
			lastBoundary = i
			break
		}
	}

	if lastBoundary < 0 {
		return ""
	}

	// Walk backwards from the boundary to find the sentence start.
	// The sentence starts after the previous sentence-ending punctuation
	// or at the beginning of the text.
	sentenceStart := 0
	for i := lastBoundary - 1; i >= 0; i-- {
		ch := cleaned[i]
		if ch == '.' || ch == '!' || ch == '?' {
			sentenceStart = i + 1
			break
		}
	}

	sentence := strings.TrimSpace(cleaned[sentenceStart : lastBoundary+1])

	// Skip very short sentences or heading-like lines.
	if len(sentence) < 20 || strings.HasPrefix(sentence, "#") {
		return ""
	}

	// Skip sentences that look like list items or table rows.
	if strings.HasPrefix(sentence, "- ") || strings.HasPrefix(sentence, "* ") || strings.HasPrefix(sentence, "|") {
		return ""
	}

	return sentence
}
