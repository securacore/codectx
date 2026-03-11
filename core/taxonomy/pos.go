package taxonomy

import (
	"regexp"
	"strings"

	prose "github.com/zuvaai/prose/v3"

	"github.com/securacore/codectx/core/chunk"
)

// Noun POS tag prefixes used for noun phrase extraction.
// Penn Treebank tags: NN (singular/mass), NNS (plural),
// NNP (proper singular), NNPS (proper plural).
const (
	tagNN   = "NN"
	tagNNS  = "NNS"
	tagNNP  = "NNP"
	tagNNPS = "NNPS"
)

// Adjective POS tag prefixes: JJ, JJR (comparative), JJS (superlative).
const (
	tagJJ  = "JJ"
	tagJJR = "JJR"
	tagJJS = "JJS"
)

// posStopWords are common English words that should not become taxonomy
// terms even when tagged as nouns by the POS tagger.
var posStopWords = map[string]bool{
	"example": true, "thing": true, "way": true, "case": true,
	"time": true, "place": true, "point": true, "part": true,
	"lot": true, "number": true, "kind": true, "set": true,
	"list": true, "use": true, "end": true, "section": true,
	"note": true, "step": true, "result": true, "order": true,
	"line": true, "form": true, "level": true, "area": true,
	"side": true, "fact": true, "change": true, "group": true,
	"work": true, "home": true, "need": true, "world": true,
	"state": true, "issue": true, "problem": true, "piece": true,
}

// extractPOS performs Pass 3 of the taxonomy extraction pipeline.
// It uses POS (part-of-speech) tagging to identify noun phrases and
// named entities from chunk body text that structural extraction missed.
//
// Extraction rules:
//   - Noun phrases: sequences of JJ* + NN+ (adjectives + nouns)
//   - Single proper nouns: NNP/NNPS tokens (library names, product names)
//   - Named entities from the prose NER model
//
// Each extracted term becomes a candidate with SourcePOS.
func extractPOS(chunks []chunk.Chunk) []candidate {
	var candidates []candidate

	for i := range chunks {
		c := &chunks[i]

		// Strip markdown formatting for cleaner POS tagging.
		text := stripMarkdown(c.Content)
		if text == "" {
			continue
		}

		// Create a prose document with POS tagging and NER enabled,
		// but segmentation disabled (we don't need sentence boundaries).
		doc, err := prose.NewDocument(text, prose.WithSegmentation(false))
		if err != nil {
			continue
		}

		// Extract noun phrases from POS tags.
		candidates = appendNounPhrases(candidates, doc.Tokens(), c.ID)

		// Extract named entities.
		candidates = appendNamedEntities(candidates, doc.Entities(), c.ID)
	}

	return candidates
}

// isNoun returns true if the POS tag is a noun tag (NN, NNS, NNP, NNPS).
func isNoun(tag string) bool {
	return tag == tagNN || tag == tagNNS || tag == tagNNP || tag == tagNNPS
}

// isProperNoun returns true if the POS tag is a proper noun (NNP, NNPS).
func isProperNoun(tag string) bool {
	return tag == tagNNP || tag == tagNNPS
}

// isAdjective returns true if the POS tag is an adjective (JJ, JJR, JJS).
func isAdjective(tag string) bool {
	return tag == tagJJ || tag == tagJJR || tag == tagJJS
}

// appendNounPhrases walks POS-tagged tokens and extracts noun phrases
// matching the pattern JJ* + NN+ (zero or more adjectives followed by
// one or more nouns). Single proper nouns (NNP/NNPS) are also extracted.
func appendNounPhrases(candidates []candidate, tokens []prose.Token, chunkID string) []candidate {
	seen := make(map[string]bool)

	i := 0
	for i < len(tokens) {
		tok := tokens[i]

		// Look for the start of a noun phrase: adjective or noun.
		if !isAdjective(tok.Tag) && !isNoun(tok.Tag) {
			i++
			continue
		}

		// Collect the full noun phrase: JJ* + NN+.
		start := i
		hasNoun := false

		for i < len(tokens) && (isAdjective(tokens[i].Tag) || isNoun(tokens[i].Tag)) {
			if isNoun(tokens[i].Tag) {
				hasNoun = true
			}
			i++
		}

		if !hasNoun {
			continue
		}

		// Build the phrase text from the collected tokens.
		phrase := buildPhrase(tokens[start:i])

		// Apply filters.
		if !isValidPOSTerm(phrase, tokens[start:i]) {
			continue
		}

		lower := strings.ToLower(phrase)
		if seen[lower] {
			continue
		}
		seen[lower] = true

		candidates = append(candidates, candidate{
			canonical: phrase,
			source:    SourcePOS,
			chunkID:   chunkID,
		})
	}

	return candidates
}

// buildPhrase joins token texts into a single phrase string.
func buildPhrase(tokens []prose.Token) string {
	parts := make([]string, len(tokens))
	for i, t := range tokens {
		parts[i] = t.Text
	}
	return strings.Join(parts, " ")
}

// isValidPOSTerm filters out invalid POS-extracted terms.
func isValidPOSTerm(phrase string, tokens []prose.Token) bool {
	// Too short.
	if len(phrase) < minIdentifierLen {
		return false
	}

	// Too long (likely a sentence fragment).
	if len(phrase) > maxBoldTermLen {
		return false
	}

	// Single-word common nouns are too generic.
	if len(tokens) == 1 && !isProperNoun(tokens[0].Tag) {
		return false
	}

	// Single-word proper nouns must pass stop word check.
	if len(tokens) == 1 {
		lower := strings.ToLower(tokens[0].Text)
		return !posStopWords[lower]
	}

	// Multi-word phrases: check that the phrase isn't entirely stop words.
	for _, t := range tokens {
		lower := strings.ToLower(t.Text)
		if !posStopWords[lower] && isNoun(t.Tag) {
			return true
		}
	}

	return false
}

// appendNamedEntities extracts named entities identified by the prose
// NER model. Entity types include PERSON and GPE (geopolitical entity).
func appendNamedEntities(candidates []candidate, entities []prose.Entity, chunkID string) []candidate {
	seen := make(map[string]bool)

	for _, ent := range entities {
		text := strings.TrimSpace(ent.Text)
		if text == "" || len(text) < minIdentifierLen || len(text) > maxBoldTermLen {
			continue
		}

		lower := strings.ToLower(text)
		if seen[lower] || posStopWords[lower] {
			continue
		}
		seen[lower] = true

		candidates = append(candidates, candidate{
			canonical: text,
			source:    SourcePOS,
			chunkID:   chunkID,
		})
	}

	return candidates
}

// Markdown stripping patterns for pre-POS-tagging cleanup.
var (
	// stripCodeFence removes fenced code blocks (```...``` or ~~~...~~~).
	stripCodeFence = regexp.MustCompile("(?s)```[^`]*```|~~~[^~]*~~~")

	// stripInlineCode removes inline code (`...`).
	stripInlineCode = regexp.MustCompile("`[^`]+`")

	// stripLink replaces [text](url) with just text.
	stripLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

	// stripBold removes bold markers (**text** or __text__).
	stripBold = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)

	// stripHTML removes HTML tags.
	stripHTML = regexp.MustCompile(`<[^>]+>`)

	// stripHeadingMarker removes leading # markers from headings.
	stripHeadingMarker = regexp.MustCompile(`(?m)^#{1,6}\s+`)

	// stripListMarker removes leading list markers (-, *, +, 1.).
	stripListMarkerRe = regexp.MustCompile(`(?m)^[\t ]*[-*+][\t ]+|^[\t ]*\d+[.)]\s+`)

	// collapseWhitespace normalizes runs of whitespace to single spaces.
	collapseWhitespace = regexp.MustCompile(`\s+`)
)

// stripMarkdown removes markdown formatting from text to produce
// cleaner input for POS tagging. This is a lightweight regex-based
// approach, not a full markdown parser.
func stripMarkdown(text string) string {
	if text == "" {
		return ""
	}

	// Remove code blocks first (they contain non-prose content).
	text = stripCodeFence.ReplaceAllString(text, " ")

	// Remove inline code.
	text = stripInlineCode.ReplaceAllString(text, " ")

	// Replace links with their display text.
	text = stripLink.ReplaceAllString(text, "$1")

	// Remove bold markers, keeping the text.
	text = stripBold.ReplaceAllStringFunc(text, func(s string) string {
		// Extract the text between ** or __.
		if strings.HasPrefix(s, "**") {
			return strings.TrimSuffix(strings.TrimPrefix(s, "**"), "**")
		}
		return strings.TrimSuffix(strings.TrimPrefix(s, "__"), "__")
	})

	// Remove HTML tags.
	text = stripHTML.ReplaceAllString(text, " ")

	// Remove heading markers.
	text = stripHeadingMarker.ReplaceAllString(text, "")

	// Remove list markers.
	text = stripListMarkerRe.ReplaceAllString(text, "")

	// Collapse whitespace.
	text = collapseWhitespace.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
