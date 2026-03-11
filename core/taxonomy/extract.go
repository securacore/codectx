package taxonomy

import (
	"regexp"
	"strings"

	"github.com/securacore/codectx/core/chunk"
	"github.com/securacore/codectx/core/markdown"
)

// candidate is a raw term extraction before deduplication.
type candidate struct {
	// canonical is the display form of the term.
	canonical string

	// source is the discovery method.
	source string

	// chunkID is the chunk where this term was found.
	chunkID string
}

// extractStructural performs Pass 1 of the taxonomy extraction pipeline.
// It walks all chunks and extracts terms from structural positions:
//
//   - Headings (highest confidence)
//   - Code identifiers (function names, type names, class names)
//   - Bold/emphasized terms in definition-like positions
//   - Structured positions (list headers, table headers)
//
// Returns a slice of raw candidates for further processing.
func extractStructural(chunks []chunk.Chunk) []candidate {
	var candidates []candidate

	for i := range chunks {
		c := &chunks[i]
		for _, block := range c.Blocks {
			switch block.Type {
			case markdown.BlockHeading:
				candidates = appendHeadingTerms(candidates, block, c.ID)

			case markdown.BlockCodeBlock:
				candidates = appendCodeIdentifiers(candidates, block, c.ID)

			case markdown.BlockParagraph:
				candidates = appendBoldTerms(candidates, block.Content, c.ID)

			case markdown.BlockList:
				candidates = appendListHeaders(candidates, block.Content, c.ID)
				candidates = appendBoldTerms(candidates, block.Content, c.ID)

			case markdown.BlockTable:
				candidates = appendTableHeaders(candidates, block.Content, c.ID)

			case markdown.BlockBlockquote, markdown.BlockThematicBreak, markdown.BlockHTMLBlock:
				// No term extraction for these block types.
			}
		}
	}

	return candidates
}

// appendHeadingTerms extracts heading text as a candidate term.
// Headings are the highest-confidence source for canonical terms.
func appendHeadingTerms(candidates []candidate, block markdown.Block, chunkID string) []candidate {
	text := strings.TrimSpace(block.Content)
	if text == "" {
		return candidates
	}

	return append(candidates, candidate{
		canonical: text,
		source:    SourceHeading,
		chunkID:   chunkID,
	})
}

// Code identifier patterns for common languages.
// Each pattern matches the identifier name in capture group 1.
var codeIdentifierPatterns = []*regexp.Regexp{
	// Go: func Name, type Name, func (r Receiver) Name
	regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s+)?(\w+)`),
	// Go: type Name struct/interface/...
	regexp.MustCompile(`(?m)^type\s+(\w+)\s+`),
	// Python: def name, class Name
	regexp.MustCompile(`(?m)^(?:def|class)\s+(\w+)`),
	// JavaScript/TypeScript: function name, class Name, export function name
	regexp.MustCompile(`(?m)^(?:export\s+)?(?:function|class)\s+(\w+)`),
	// Rust: fn name, struct Name, enum Name, trait Name, impl Name
	regexp.MustCompile(`(?m)^(?:pub\s+)?(?:fn|struct|enum|trait|impl)\s+(\w+)`),
	// Java/C#/Kotlin: class Name, interface Name
	regexp.MustCompile(`(?m)^(?:public\s+|private\s+|protected\s+)?(?:static\s+)?(?:abstract\s+)?(?:class|interface|enum)\s+(\w+)`),
}

// minIdentifierLen is the minimum length for a code identifier to be
// considered as a taxonomy term. Short identifiers (i, j, x, etc.)
// are too generic to be useful.
const minIdentifierLen = 3

// appendCodeIdentifiers extracts function names, type names, and class names
// from code blocks using lightweight regex patterns.
func appendCodeIdentifiers(candidates []candidate, block markdown.Block, chunkID string) []candidate {
	content := block.Content
	if content == "" {
		return candidates
	}

	seen := make(map[string]bool)

	for _, pattern := range codeIdentifierPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			name := match[1]
			if len(name) < minIdentifierLen {
				continue
			}
			lower := strings.ToLower(name)
			if seen[lower] {
				continue
			}
			seen[lower] = true

			candidates = append(candidates, candidate{
				canonical: name,
				source:    SourceCodeIdentifier,
				chunkID:   chunkID,
			})
		}
	}

	return candidates
}

// boldPattern matches **bold** or __bold__ text in markdown content.
// It captures the text between the markers.
var boldPattern = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)

// maxBoldTermLen is the maximum length for a bold term to be considered
// a taxonomy candidate. Very long bold phrases are typically emphasis
// on sentences, not term definitions.
const maxBoldTermLen = 60

// appendBoldTerms extracts bold/emphasized terms from paragraph and list
// content. Only terms that appear in definition-like positions (start of
// a line or list item) are considered candidates.
func appendBoldTerms(candidates []candidate, content string, chunkID string) []candidate {
	matches := boldPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return candidates
	}

	seen := make(map[string]bool)

	for _, loc := range matches {
		// Extract the matched bold text from whichever group matched.
		var text string
		if loc[2] >= 0 {
			text = content[loc[2]:loc[3]] // **text**
		} else if loc[4] >= 0 {
			text = content[loc[4]:loc[5]] // __text__
		}

		text = strings.TrimSpace(text)
		if text == "" || len(text) > maxBoldTermLen {
			continue
		}

		// Check if this is in a definition-like position:
		// at the start of content, at the start of a line, or near a line start.
		matchStart := loc[0]
		if !isDefinitionPosition(content, matchStart) {
			continue
		}

		lower := strings.ToLower(text)
		if seen[lower] {
			continue
		}
		seen[lower] = true

		candidates = append(candidates, candidate{
			canonical: text,
			source:    SourceBoldTerm,
			chunkID:   chunkID,
		})
	}

	return candidates
}

// isDefinitionPosition checks whether a match at the given offset is in
// a definition-like position: at line start, or preceded only by
// whitespace and/or list markers on the same line.
func isDefinitionPosition(content string, offset int) bool {
	if offset == 0 {
		return true
	}

	// Walk backwards to find the start of the line.
	lineStart := strings.LastIndex(content[:offset], "\n")
	if lineStart < 0 {
		lineStart = 0
	} else {
		lineStart++ // skip past the newline
	}

	prefix := strings.TrimSpace(content[lineStart:offset])

	// Empty prefix means it's at line start (after whitespace).
	if prefix == "" {
		return true
	}

	// List marker prefixes: "- ", "* ", "1. ", "2. ", etc.
	if isListMarker(prefix) {
		return true
	}

	return false
}

// listMarkerPattern matches common list markers.
var listMarkerPattern = regexp.MustCompile(`^(?:[-*+]|\d+[.)])$`)

// isListMarker checks if a string is a list marker.
func isListMarker(s string) bool {
	return listMarkerPattern.MatchString(s)
}

// listHeaderPattern matches list item headers: text before `: ` or ` -- ` or ` — `.
// Only matches the first item-like line.
var listHeaderPattern = regexp.MustCompile(`(?m)^[-*+]\s+([^:\n]+?)(?::\s|(?:\s+--|(?:\s+\x{2014})))`)

// appendListHeaders extracts terms from list item header positions
// (text before a colon or dash in a list item).
func appendListHeaders(candidates []candidate, content string, chunkID string) []candidate {
	matches := listHeaderPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return candidates
	}

	seen := make(map[string]bool)

	for _, match := range matches {
		text := strings.TrimSpace(match[1])
		if text == "" || len(text) > maxBoldTermLen {
			continue
		}

		// Skip if the header text looks like a sentence (contains common verbs).
		if looksLikeSentence(text) {
			continue
		}

		lower := strings.ToLower(text)
		if seen[lower] {
			continue
		}
		seen[lower] = true

		candidates = append(candidates, candidate{
			canonical: text,
			source:    SourceStructuredPosition,
			chunkID:   chunkID,
		})
	}

	return candidates
}

// tableHeaderPattern splits a pipe-delimited table row.
var tableHeaderPattern = regexp.MustCompile(`\|`)

// tableSepPattern matches separator rows in tables (e.g., |---|---|).
var tableSepPattern = regexp.MustCompile(`^[\s|:-]+$`)

// appendTableHeaders extracts terms from table column headers (first row).
func appendTableHeaders(candidates []candidate, content string, chunkID string) []candidate {
	lines := strings.SplitN(content, "\n", 3)
	if len(lines) == 0 {
		return candidates
	}

	// First line is the header row.
	headerLine := strings.TrimSpace(lines[0])
	if headerLine == "" || tableSepPattern.MatchString(headerLine) {
		return candidates
	}

	cells := tableHeaderPattern.Split(headerLine, -1)

	seen := make(map[string]bool)

	for _, cell := range cells {
		text := strings.TrimSpace(cell)
		if text == "" || len(text) > maxBoldTermLen {
			continue
		}

		// Skip generic column headers that aren't useful as terms.
		if isGenericTableHeader(text) {
			continue
		}

		lower := strings.ToLower(text)
		if seen[lower] {
			continue
		}
		seen[lower] = true

		candidates = append(candidates, candidate{
			canonical: text,
			source:    SourceStructuredPosition,
			chunkID:   chunkID,
		})
	}

	return candidates
}

// genericHeaders are common table column headers that aren't useful as
// taxonomy terms.
var genericHeaders = map[string]bool{
	"name": true, "value": true, "type": true, "description": true,
	"default": true, "required": true, "example": true, "notes": true,
	"status": true, "key": true, "field": true, "column": true,
	"parameter": true, "option": true, "property": true, "attribute": true,
	"#": true, "no": true, "id": true, "index": true,
}

// isGenericTableHeader returns true if the text is a common generic
// table column header not useful as a taxonomy term.
func isGenericTableHeader(text string) bool {
	return genericHeaders[strings.ToLower(text)]
}

// sentenceVerbs are common verbs that indicate a phrase is a sentence
// rather than a term name.
var sentenceVerbs = regexp.MustCompile(`(?i)\b(?:is|are|was|were|will|should|must|can|could|would|has|have|had|does|do|did)\b`)

// looksLikeSentence does a rough check to see if text looks more like
// a natural language sentence than a term or concept name.
func looksLikeSentence(text string) bool {
	// If it has common sentence verbs and is relatively long, it's probably a sentence.
	if len(text) > 30 && sentenceVerbs.MatchString(text) {
		return true
	}
	return false
}
