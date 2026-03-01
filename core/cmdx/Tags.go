package cmdx

import (
	"fmt"
	"strconv"
	"strings"
)

// TagParser parses CMDX body content into AST nodes.
type TagParser struct {
	lines []string
	pos   int
}

// newTagParser creates a parser from CMDX body text (after header/dict/meta).
func newTagParser(input string) *TagParser {
	return &TagParser{
		lines: strings.Split(input, "\n"),
		pos:   0,
	}
}

func (p *TagParser) hasMore() bool { return p.pos < len(p.lines) }

func (p *TagParser) currentLine() string {
	if p.pos >= len(p.lines) {
		return ""
	}
	return p.lines[p.pos]
}

func (p *TagParser) advance() { p.pos++ }

// ParseBody parses all body lines into AST nodes.
func (p *TagParser) ParseBody() ([]Node, error) {
	var nodes []Node
	for p.hasMore() {
		line := strings.TrimRight(p.currentLine(), "\r")
		if line == "" {
			p.advance()
			continue
		}
		if !strings.HasPrefix(line, "@") {
			// Non-tag line — skip.
			p.advance()
			continue
		}
		node, err := p.parseTag()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (p *TagParser) parseTag() (Node, error) {
	line := strings.TrimRight(p.currentLine(), "\r")

	// Headings @H1..@H6
	for level := 1; level <= 6; level++ {
		prefix := fmt.Sprintf("@H%d ", level)
		if strings.HasPrefix(line, prefix) {
			content := line[len(prefix):]
			p.advance()
			children := parseInline(content)
			return Node{Tag: TagH1 + TagType(level-1), Children: children}, nil
		}
	}

	// Horizontal rule
	if line == "@HR" {
		p.advance()
		return Node{Tag: TagHR}, nil
	}

	// Line break
	if line == "@BR" {
		p.advance()
		return Node{Tag: TagBR}, nil
	}

	// Paragraph
	if strings.HasPrefix(line, "@P ") {
		content := line[3:]
		p.advance()
		children := parseInline(content)
		return Node{Tag: TagP, Children: children}, nil
	}

	// Blockquote — block form
	if strings.HasPrefix(line, "@BQ{") {
		return p.parseBlockquoteBlock()
	}
	// Blockquote — single-line form
	if strings.HasPrefix(line, "@BQ ") {
		content := line[4:]
		p.advance()
		children := parseInline(content)
		return Node{
			Tag:      TagBQ,
			Children: []Node{{Tag: TagP, Children: children}},
		}, nil
	}

	// Code block
	if strings.HasPrefix(line, "@CODE") {
		return p.parseCodeBlock()
	}

	// Lists
	if strings.HasPrefix(line, "@UL{") {
		return p.parseListBlock(TagUL)
	}
	if strings.HasPrefix(line, "@OL{") {
		return p.parseListBlock(TagOL)
	}

	// Table
	if strings.HasPrefix(line, "@TABLE{") {
		return p.parseTableBlock()
	}

	// Domain blocks (Phase 3)
	if strings.HasPrefix(line, "@KV{") {
		return p.parseKVBlock()
	}
	if strings.HasPrefix(line, "@PARAMS{") {
		return p.parseParamsBlock()
	}
	if strings.HasPrefix(line, "@ENDPOINT{") {
		return p.parseEndpointTag()
	}
	if strings.HasPrefix(line, "@RETURNS{") {
		return p.parseReturnsTag()
	}
	if strings.HasPrefix(line, "@DEF{") {
		return p.parseDEFBlock()
	}
	if strings.HasPrefix(line, "@NOTE{") {
		return p.parseCalloutTag(TagNote)
	}
	if strings.HasPrefix(line, "@WARN{") {
		return p.parseCalloutTag(TagWarn)
	}
	if strings.HasPrefix(line, "@TIP{") {
		return p.parseCalloutTag(TagTip)
	}
	if strings.HasPrefix(line, "@IMPORTANT{") {
		return p.parseCalloutTag(TagImportant)
	}

	return Node{}, fmt.Errorf("unknown tag at line %d: %s", p.pos+1, line)
}

// parseCodeBlock parses @CODE:lang ... @/CODE.
func (p *TagParser) parseCodeBlock() (Node, error) {
	line := strings.TrimRight(p.currentLine(), "\r")
	lang := ""
	if strings.HasPrefix(line, "@CODE:") {
		lang = line[6:]
		// Unescape \{, \}, \\ in language string.
		lang = strings.ReplaceAll(lang, "\\}", "}")
		lang = strings.ReplaceAll(lang, "\\{", "{")
		lang = strings.ReplaceAll(lang, "\\\\", "\\")
	}
	p.advance()

	var contentLines []string
	for p.hasMore() {
		codeLine := strings.TrimRight(p.currentLine(), "\r")
		if codeLine == "@/CODE" {
			p.advance()
			break
		}
		contentLines = append(contentLines, UnescapeCodeLine(codeLine))
		p.advance()
	}

	return Node{
		Tag:     TagCodeBlock,
		Content: strings.Join(contentLines, "\n"),
		Attrs:   NodeAttrs{Language: lang},
	}, nil
}

// parseBlockquoteBlock parses @BQ{...} multi-line form.
func (p *TagParser) parseBlockquoteBlock() (Node, error) {
	lines, err := p.readBracedBlock()
	if err != nil {
		return Node{}, err
	}
	subParser := newTagParser(strings.Join(lines, "\n"))
	children, err := subParser.ParseBody()
	if err != nil {
		return Node{}, fmt.Errorf("blockquote body: %w", err)
	}
	return Node{Tag: TagBQ, Children: children}, nil
}

// parseListBlock parses @UL{...} or @OL{...}.
func (p *TagParser) parseListBlock(tag TagType) (Node, error) {
	lines, err := p.readBracedBlock()
	if err != nil {
		return Node{}, err
	}
	items := parseListItems(lines, tag)
	return Node{Tag: tag, Children: items}, nil
}

// parseListItems processes lines within a list block into item nodes.
func parseListItems(lines []string, tag TagType) []Node {
	var items []Node
	i := 0
	for i < len(lines) {
		line := strings.TrimRight(lines[i], "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			i++
			continue
		}

		var itemText string
		isItem := false
		if tag == TagUL && (strings.HasPrefix(trimmed, "- ") || trimmed == "-") {
			if len(trimmed) > 2 {
				itemText = trimmed[2:]
			}
			isItem = true
		} else if tag == TagOL {
			if idx := strings.Index(trimmed, ". "); idx > 0 {
				if _, err := strconv.Atoi(trimmed[:idx]); err == nil {
					itemText = trimmed[idx+2:]
					isItem = true
				}
			} else if strings.HasSuffix(trimmed, ".") {
				// Handle empty OL item: "1." (number + period, no content).
				numStr := trimmed[:len(trimmed)-1]
				if _, err := strconv.Atoi(numStr); err == nil {
					isItem = true
				}
			}
		}

		if isItem {
			children := parseInline(itemText)
			item := Node{Tag: TagP, Children: children}
			i++
			// Check for nested lists.
			for i < len(lines) {
				nextTrimmed := strings.TrimSpace(lines[i])
				if strings.HasPrefix(nextTrimmed, "@UL{") || strings.HasPrefix(nextTrimmed, "@OL{") {
					nestedTag := TagUL
					if strings.HasPrefix(nextTrimmed, "@OL{") {
						nestedTag = TagOL
					}
					nestedLines, end := collectNestedBraced(lines, i)
					nestedItems := parseListItems(nestedLines, nestedTag)
					item.Children = append(item.Children, Node{Tag: nestedTag, Children: nestedItems})
					i = end
				} else {
					break
				}
			}
			items = append(items, item)
		} else {
			i++
		}
	}
	return items
}

// collectNestedBraced finds the braced content starting at lines[start] and returns
// the inner lines and the index AFTER the closing brace.
func collectNestedBraced(lines []string, start int) ([]string, int) {
	openLine := strings.TrimSpace(lines[start])
	braceIdx := strings.Index(openLine, "{")
	if braceIdx < 0 {
		return nil, start + 1
	}

	depth := 1
	var inner []string
	// Content after { on the opening line.
	after := strings.TrimSpace(openLine[braceIdx+1:])
	if after != "" && after != "}" {
		inner = append(inner, after)
	}
	if after == "}" {
		return inner, start + 1
	}

	i := start + 1
	for i < len(lines) {
		line := strings.TrimRight(lines[i], "\r")
		for j := 0; j < len(line); j++ {
			if line[j] == '\\' && j+1 < len(line) {
				j++
				continue
			}
			if line[j] == '{' {
				depth++
			}
			if line[j] == '}' {
				depth--
				if depth == 0 {
					before := strings.TrimSpace(line[:j])
					if before != "" {
						inner = append(inner, before)
					}
					return inner, i + 1
				}
			}
		}
		inner = append(inner, line)
		i++
	}
	return inner, i
}

// parseTableBlock parses @TABLE{...}.
func (p *TagParser) parseTableBlock() (Node, error) {
	lines, err := p.readBracedBlock()
	if err != nil {
		return Node{}, err
	}

	var headers []string
	var rows [][]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "@THEAD{") && strings.HasSuffix(line, "}") {
			headerStr := line[7 : len(line)-1]
			headers = splitUnescapedPipe(headerStr)
			for i, h := range headers {
				headers[i] = UnescapeCell(strings.TrimSpace(h))
			}
		} else {
			cells := splitUnescapedPipe(line)
			for i, c := range cells {
				cells[i] = UnescapeCell(strings.TrimSpace(c))
			}
			rows = append(rows, cells)
		}
	}

	return Node{
		Tag:   TagTable,
		Attrs: NodeAttrs{Headers: headers, Cells: rows},
	}, nil
}

// Domain block parsers (Phase 3 — stubs available for the parser to handle)

func (p *TagParser) parseKVBlock() (Node, error) {
	lines, err := p.readBracedBlock()
	if err != nil {
		return Node{}, err
	}
	var items []KVItem
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: key:type~description (with escaped delimiters)
		colonIdx := findUnescaped(line, ':')
		if colonIdx < 0 {
			continue
		}
		key := UnescapeKVField(line[:colonIdx])
		rest := line[colonIdx+1:]
		tildeIdx := findUnescaped(rest, '~')
		if tildeIdx < 0 {
			items = append(items, KVItem{Key: key, Type: UnescapeKVField(rest)})
			continue
		}
		typ := UnescapeKVField(rest[:tildeIdx])
		desc := UnescapeDesc(rest[tildeIdx+1:])
		items = append(items, KVItem{Key: key, Type: typ, Description: desc})
	}
	return Node{Tag: TagKV, Attrs: NodeAttrs{Items: items}}, nil
}

func (p *TagParser) parseParamsBlock() (Node, error) {
	lines, err := p.readBracedBlock()
	if err != nil {
		return Node{}, err
	}
	var params []ParamItem
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: name:type:R|O~description (with escaped delimiters)
		tildeIdx := findUnescaped(line, '~')
		desc := ""
		var prefix string
		if tildeIdx >= 0 {
			desc = UnescapeDesc(line[tildeIdx+1:])
			prefix = line[:tildeIdx]
		} else {
			prefix = line
		}
		colons := splitNUnescaped(prefix, ':', 3)
		if len(colons) < 3 {
			continue
		}
		req := strings.TrimSpace(colons[2]) == "R"
		params = append(params, ParamItem{
			Name:        UnescapeKVField(strings.TrimSpace(colons[0])),
			Type:        UnescapeKVField(strings.TrimSpace(colons[1])),
			Required:    req,
			Description: desc,
		})
	}
	return Node{Tag: TagParams, Attrs: NodeAttrs{Params: params}}, nil
}

func (p *TagParser) parseEndpointTag() (Node, error) {
	line := strings.TrimRight(p.currentLine(), "\r")
	p.advance()
	// @ENDPOINT{METHOD /path}
	content := extractBracedContent(line, "@ENDPOINT{")
	parts := strings.SplitN(content, " ", 2)
	if len(parts) != 2 {
		return Node{Tag: TagEndpoint, Attrs: NodeAttrs{Endpoint: &EndpointDef{Method: content}}}, nil
	}
	return Node{
		Tag:   TagEndpoint,
		Attrs: NodeAttrs{Endpoint: &EndpointDef{Method: parts[0], Path: parts[1]}},
	}, nil
}

func (p *TagParser) parseReturnsTag() (Node, error) {
	line := strings.TrimRight(p.currentLine(), "\r")
	p.advance()
	content := extractBracedContent(line, "@RETURNS{")
	parts := splitUnescapedPipe(content)
	var returns []ReturnDef
	for _, part := range parts {
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			returns = append(returns, ReturnDef{Status: strings.TrimSpace(part)})
			continue
		}
		returns = append(returns, ReturnDef{
			Status:      strings.TrimSpace(part[:colonIdx]),
			Description: UnescapeCell(strings.TrimSpace(part[colonIdx+1:])),
		})
	}
	return Node{Tag: TagReturns, Attrs: NodeAttrs{Returns: returns}}, nil
}

func (p *TagParser) parseDEFBlock() (Node, error) {
	lines, err := p.readBracedBlock()
	if err != nil {
		return Node{}, err
	}
	var items []KVItem
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: term~definition (with escaped delimiters)
		tildeIdx := findUnescaped(line, '~')
		if tildeIdx < 0 {
			items = append(items, KVItem{Key: UnescapeKVField(line)})
			continue
		}
		items = append(items, KVItem{
			Key:         UnescapeKVField(line[:tildeIdx]),
			Description: UnescapeDesc(line[tildeIdx+1:]),
		})
	}
	return Node{Tag: TagDef, Attrs: NodeAttrs{Items: items}}, nil
}

func (p *TagParser) parseCalloutTag(tag TagType) (Node, error) {
	line := strings.TrimRight(p.currentLine(), "\r")
	p.advance()
	prefix := "@" + tag.String() + "{"
	content := extractBracedContent(line, prefix)
	return Node{Tag: tag, Attrs: NodeAttrs{Callout: content}}, nil
}

// readBracedBlock reads content between { and matching }, handling nesting.
// The parser position should be on the line containing the opening {.
func (p *TagParser) readBracedBlock() ([]string, error) {
	line := strings.TrimRight(p.currentLine(), "\r")
	braceIdx := strings.Index(line, "{")
	if braceIdx < 0 {
		return nil, fmt.Errorf("expected { on line %d: %s", p.pos+1, line)
	}

	depth := 1
	var inner []string
	// Content after { on the opening line.
	after := line[braceIdx+1:]
	if after != "" {
		// Check if closing brace is on the same line.
		for j := 0; j < len(after); j++ {
			if after[j] == '\\' && j+1 < len(after) {
				j++
				continue
			}
			if after[j] == '{' {
				depth++
			}
			if after[j] == '}' {
				depth--
				if depth == 0 {
					content := strings.TrimSpace(after[:j])
					if content != "" {
						inner = append(inner, content)
					}
					p.advance()
					return inner, nil
				}
			}
		}
		trimmed := strings.TrimSpace(after)
		if trimmed != "" {
			inner = append(inner, after)
		}
	}

	p.advance()
	for p.hasMore() {
		bLine := strings.TrimRight(p.currentLine(), "\r")
		for j := 0; j < len(bLine); j++ {
			if bLine[j] == '\\' && j+1 < len(bLine) {
				j++
				continue
			}
			if bLine[j] == '{' {
				depth++
			}
			if bLine[j] == '}' {
				depth--
				if depth == 0 {
					before := bLine[:j]
					trimmed := strings.TrimSpace(before)
					if trimmed != "" {
						inner = append(inner, before)
					}
					p.advance()
					return inner, nil
				}
			}
		}
		inner = append(inner, bLine)
		p.advance()
	}

	return inner, fmt.Errorf("unclosed brace starting at line %d", p.pos)
}

// extractBracedContent extracts content from a single-line tag like @TAG{content}.
func extractBracedContent(line, prefix string) string {
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	rest := line[len(prefix):]
	if strings.HasSuffix(rest, "}") {
		return rest[:len(rest)-1]
	}
	return rest
}

// --- Inline parser ---

// parseInline parses inline CMDX content into AST nodes.
func parseInline(s string) []Node {
	p := &inlineParser{input: s, pos: 0}
	return p.parse()
}

type inlineParser struct {
	input string
	pos   int
}

func (p *inlineParser) parse() []Node {
	var nodes []Node
	var text strings.Builder

	flush := func() {
		if text.Len() > 0 {
			nodes = append(nodes, Node{Tag: TagRaw, Content: text.String()})
			text.Reset()
		}
	}

	for p.pos < len(p.input) {
		ch := p.input[p.pos]

		if ch == '@' {
			// Check for escaped @@.
			if p.pos+1 < len(p.input) && p.input[p.pos+1] == '@' {
				text.WriteString("@@")
				p.pos += 2
				continue
			}
			// Try to parse an inline tag.
			if node, ok := p.tryInlineTag(); ok {
				flush()
				nodes = append(nodes, node)
				continue
			}
			// Bare @ — emit as-is.
			text.WriteByte('@')
			p.pos++
			continue
		}

		if ch == '$' {
			// Check for escaped $$.
			if p.pos+1 < len(p.input) && p.input[p.pos+1] == '$' {
				text.WriteString("$$")
				p.pos += 2
				continue
			}
			// $N dictionary reference — keep as-is for later expansion.
			text.WriteByte('$')
			p.pos++
			continue
		}

		if ch == '\\' && p.pos+1 < len(p.input) {
			next := p.input[p.pos+1]
			if next == '{' || next == '}' || next == '\\' {
				// Escaped brace or backslash — unescape.
				text.WriteByte(next)
				p.pos += 2
				continue
			}
		}

		text.WriteByte(ch)
		p.pos++
	}

	flush()
	return mergeAdjacentRaw(nodes)
}

// tryInlineTag attempts to parse an inline tag at the current position.
// Returns the node and true if successful, or zero-value and false.
func (p *inlineParser) tryInlineTag() (Node, bool) {
	if p.pos >= len(p.input) || p.input[p.pos] != '@' {
		return Node{}, false
	}

	// Read tag name.
	start := p.pos + 1
	i := start
	for i < len(p.input) && isTagChar(p.input[i]) {
		i++
	}

	tagName := p.input[start:i]
	tag, ok := inlineTagMap[tagName]
	if !ok {
		return Node{}, false
	}

	if i >= len(p.input) || p.input[i] != '{' {
		return Node{}, false
	}

	// Find matching closing brace.
	braceStart := i
	content, end, ok := matchBraces(p.input, braceStart)
	if !ok {
		return Node{}, false
	}

	p.pos = end

	switch tag {
	case TagBold, TagItalic, TagBoldItalic, TagStrikethrough:
		children := parseInline(content)
		return Node{Tag: tag, Children: children}, true
	case TagCode:
		// Unescape \{ \} \\ in code span content (escaped by encoder).
		content = unescapeCodeSpan(content)
		return Node{Tag: tag, Content: content}, true
	case TagLink:
		display, url := splitLinkContent(content)
		url = UnescapeURL(url)
		displayNodes := parseInline(display)
		return Node{
			Tag:      tag,
			Attrs:    NodeAttrs{URL: url, Display: flattenText(displayNodes)},
			Children: displayNodes,
		}, true
	case TagImage:
		alt, url := splitLinkContent(content)
		url = UnescapeURL(url)
		altNodes := parseInline(alt)
		return Node{
			Tag:      tag,
			Attrs:    NodeAttrs{URL: url, Display: flattenText(altNodes)},
			Children: altNodes,
		}, true
	case TagBR:
		return Node{Tag: TagBR}, true
	default:
		return Node{}, false
	}
}

var inlineTagMap = map[string]TagType{
	"B":    TagBold,
	"I":    TagItalic,
	"BI":   TagBoldItalic,
	"C":    TagCode,
	"S":    TagStrikethrough,
	"LINK": TagLink,
	"IMG":  TagImage,
	"BR":   TagBR,
}

func isTagChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// matchBraces finds the content between { and matching }, starting at pos.
// Returns content, end position (after }), and success.
func matchBraces(s string, pos int) (string, int, bool) {
	if pos >= len(s) || s[pos] != '{' {
		return "", 0, false
	}
	depth := 1
	i := pos + 1
	for i < len(s) {
		ch := s[i]
		if ch == '\\' && i+1 < len(s) {
			i += 2
			continue
		}
		if ch == '{' {
			depth++
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return s[pos+1 : i], i + 1, true
			}
		}
		i++
	}
	return "", 0, false
}

// splitLinkContent splits LINK/IMG content on the first unescaped > at brace depth 0.
// Per D4: first unescaped > separates display from URL.
func splitLinkContent(s string) (display, url string) {
	depth := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\\' && i+1 < len(s) {
			i++
			continue
		}
		if ch == '{' {
			depth++
		}
		if ch == '}' {
			depth--
		}
		if ch == '>' && depth == 0 {
			return UnescapeDisplay(s[:i]), s[i+1:]
		}
	}
	return s, ""
}

// splitUnescapedPipe splits a string on unescaped | characters.
func splitUnescapedPipe(s string) []string {
	var parts []string
	var current strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			// Pass through any escape sequence as-is.
			current.WriteByte(s[i])
			current.WriteByte(s[i+1])
			i++
			continue
		}
		if s[i] == '|' {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(s[i])
	}
	parts = append(parts, current.String())
	return parts
}

// unescapeCodeSpan reverses the encoder's brace escaping in code span content.
func unescapeCodeSpan(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			if next == '{' || next == '}' || next == '\\' {
				buf.WriteByte(next)
				i++
				continue
			}
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}

// findUnescaped finds the first occurrence of char that is not preceded by \.
func findUnescaped(s string, char byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++ // skip escaped char
			continue
		}
		if s[i] == char {
			return i
		}
	}
	return -1
}

// splitNUnescaped splits s on unescaped occurrences of sep, returning at most n parts.
func splitNUnescaped(s string, sep byte, n int) []string {
	var parts []string
	for n < 0 || len(parts) < n-1 {
		idx := findUnescaped(s, sep)
		if idx < 0 {
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	parts = append(parts, s)
	return parts
}
