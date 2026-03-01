package cmdx

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse parses CMDX into a Document AST without decoding to Markdown.
func Parse(cmdx []byte) (*Document, error) {
	input := string(cmdx)
	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty CMDX input")
	}

	doc := &Document{Version: "1"}
	pos := 0

	// Header: @CMDX v1
	header := strings.TrimRight(lines[pos], "\r")
	if header != "@CMDX v1" {
		return nil, fmt.Errorf("invalid CMDX header: expected '@CMDX v1', got %q", header)
	}
	pos++

	// Skip blank lines after header.
	for pos < len(lines) && strings.TrimSpace(lines[pos]) == "" {
		pos++
	}

	// Optional @DICT{...}
	if pos < len(lines) && strings.HasPrefix(strings.TrimRight(lines[pos], "\r"), "@DICT{") {
		dict, newPos, err := parseDictBlock(lines, pos)
		if err != nil {
			return nil, fmt.Errorf("parse dict: %w", err)
		}
		doc.Dict = dict
		pos = newPos
	}

	// Skip blank lines.
	for pos < len(lines) && strings.TrimSpace(lines[pos]) == "" {
		pos++
	}

	// Optional @META{...}
	if pos < len(lines) && strings.HasPrefix(strings.TrimRight(lines[pos], "\r"), "@META{") {
		meta, newPos := parseMetaBlock(lines, pos)
		doc.Meta = meta
		pos = newPos
	}

	// Skip blank lines.
	for pos < len(lines) && strings.TrimSpace(lines[pos]) == "" {
		pos++
	}

	// Body
	bodyText := strings.Join(lines[pos:], "\n")
	parser := newTagParser(bodyText)
	body, err := parser.ParseBody()
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}
	doc.Body = body

	return doc, nil
}

// Decode converts CMDX back to Markdown.
func Decode(cmdx []byte) ([]byte, error) {
	doc, err := Parse(cmdx)
	if err != nil {
		return nil, err
	}

	// Pass 2: Expand dictionary references.
	if doc.Dict != nil && len(doc.Dict.Entries) > 0 {
		expandDictionary(doc)
	}

	// Unescape @@ and $$ in all text fields.
	unescapeDocument(doc)

	// Pass 3: Convert domain blocks to standard structures (handled during MD serialization).

	// Pass 4: Serialize to Markdown.
	return serializeMarkdown(doc), nil
}

// parseDictBlock parses @DICT{...} from lines starting at pos.
func parseDictBlock(lines []string, pos int) (*Dictionary, int, error) {
	dict := &Dictionary{}
	pos++ // skip @DICT{ line
	for pos < len(lines) {
		line := strings.TrimRight(lines[pos], "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "}" {
			pos++
			return dict, pos, nil
		}
		// Format: $N=value
		if strings.HasPrefix(trimmed, "$") {
			eqIdx := strings.Index(trimmed, "=")
			if eqIdx > 1 {
				idxStr := trimmed[1:eqIdx]
				idx, err := strconv.Atoi(idxStr)
				if err == nil {
					dict.Entries = append(dict.Entries, DictEntry{
						Index: idx,
						Value: trimmed[eqIdx+1:],
					})
				}
			}
		}
		pos++
	}
	return dict, pos, nil
}

// parseMetaBlock parses @META{key:val;key:val} from lines starting at pos.
func parseMetaBlock(lines []string, pos int) (map[string]string, int) {
	meta := make(map[string]string)
	line := strings.TrimRight(lines[pos], "\r")
	// @META{key:val;key:val}
	start := strings.Index(line, "{")
	end := strings.LastIndex(line, "}")
	if start >= 0 && end > start {
		content := line[start+1 : end]
		pairs := splitUnescapedSemicolon(content)
		for _, pair := range pairs {
			colonIdx := strings.Index(pair, ":")
			if colonIdx > 0 {
				key := strings.TrimSpace(pair[:colonIdx])
				val := UnescapeMeta(strings.TrimSpace(pair[colonIdx+1:]))
				meta[key] = val
			}
		}
	}
	pos++
	return meta, pos
}

// splitUnescapedSemicolon splits on ; that are not preceded by \.
func splitUnescapedSemicolon(s string) []string {
	var parts []string
	var current strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == ';' {
			current.WriteString("\\;")
			i++
			continue
		}
		if s[i] == ';' {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(s[i])
	}
	parts = append(parts, current.String())
	return parts
}

// expandDictionary replaces $N references in all string fields (D7).
func expandDictionary(doc *Document) {
	for i := range doc.Body {
		expandNodeDict(&doc.Body[i], doc.Dict)
	}
}

func expandNodeDict(node *Node, dict *Dictionary) {
	node.Content = expandRefs(node.Content, dict)
	node.Attrs.URL = expandRefs(node.Attrs.URL, dict)
	node.Attrs.Display = expandRefs(node.Attrs.Display, dict)
	node.Attrs.Callout = expandRefs(node.Attrs.Callout, dict)
	node.Attrs.Language = expandRefs(node.Attrs.Language, dict)

	for i := range node.Attrs.Headers {
		node.Attrs.Headers[i] = expandRefs(node.Attrs.Headers[i], dict)
	}
	for i := range node.Attrs.Cells {
		for j := range node.Attrs.Cells[i] {
			node.Attrs.Cells[i][j] = expandRefs(node.Attrs.Cells[i][j], dict)
		}
	}
	for i := range node.Attrs.Items {
		node.Attrs.Items[i].Key = expandRefs(node.Attrs.Items[i].Key, dict)
		node.Attrs.Items[i].Type = expandRefs(node.Attrs.Items[i].Type, dict)
		node.Attrs.Items[i].Description = expandRefs(node.Attrs.Items[i].Description, dict)
	}
	for i := range node.Attrs.Params {
		node.Attrs.Params[i].Name = expandRefs(node.Attrs.Params[i].Name, dict)
		node.Attrs.Params[i].Type = expandRefs(node.Attrs.Params[i].Type, dict)
		node.Attrs.Params[i].Description = expandRefs(node.Attrs.Params[i].Description, dict)
	}
	for i := range node.Attrs.Returns {
		node.Attrs.Returns[i].Status = expandRefs(node.Attrs.Returns[i].Status, dict)
		node.Attrs.Returns[i].Description = expandRefs(node.Attrs.Returns[i].Description, dict)
	}
	if node.Attrs.Endpoint != nil {
		node.Attrs.Endpoint.Method = expandRefs(node.Attrs.Endpoint.Method, dict)
		node.Attrs.Endpoint.Path = expandRefs(node.Attrs.Endpoint.Path, dict)
	}

	for i := range node.Children {
		expandNodeDict(&node.Children[i], dict)
	}
}

// expandRefs replaces $N references in a string with their dictionary values.
// Skips escaped $$ (literal dollar signs) to avoid misinterpreting them.
func expandRefs(s string, dict *Dictionary) string {
	if s == "" || dict == nil {
		return s
	}
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '$' {
			// Check for $$ escape (literal $).
			if i+1 < len(s) && s[i+1] == '$' {
				result.WriteString("$$")
				i += 2
				continue
			}
			// Check for $N reference.
			j := i + 1
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			if j > i+1 {
				numStr := s[i+1 : j]
				idx, err := strconv.Atoi(numStr)
				if err == nil {
					if val, ok := dict.Lookup(idx); ok {
						result.WriteString(val)
						i = j
						continue
					}
				}
			}
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

// unescapeDocument walks all string fields and applies UnescapeBody.
func unescapeDocument(doc *Document) {
	for i := range doc.Body {
		unescapeNode(&doc.Body[i])
	}
}

func unescapeNode(node *Node) {
	// Don't unescape code block content — no @@ or $$ escaping there.
	if node.Tag == TagCodeBlock {
		return
	}

	node.Content = UnescapeBody(node.Content)
	node.Attrs.URL = UnescapeBody(node.Attrs.URL)
	node.Attrs.Display = UnescapeBody(node.Attrs.Display)
	node.Attrs.Callout = UnescapeBody(node.Attrs.Callout)

	for i := range node.Attrs.Headers {
		node.Attrs.Headers[i] = UnescapeBody(node.Attrs.Headers[i])
	}
	for i := range node.Attrs.Cells {
		for j := range node.Attrs.Cells[i] {
			node.Attrs.Cells[i][j] = UnescapeBody(node.Attrs.Cells[i][j])
		}
	}
	for i := range node.Attrs.Items {
		node.Attrs.Items[i].Key = UnescapeBody(node.Attrs.Items[i].Key)
		node.Attrs.Items[i].Type = UnescapeBody(node.Attrs.Items[i].Type)
		node.Attrs.Items[i].Description = UnescapeBody(node.Attrs.Items[i].Description)
	}
	for i := range node.Attrs.Params {
		node.Attrs.Params[i].Name = UnescapeBody(node.Attrs.Params[i].Name)
		node.Attrs.Params[i].Type = UnescapeBody(node.Attrs.Params[i].Type)
		node.Attrs.Params[i].Description = UnescapeBody(node.Attrs.Params[i].Description)
	}
	for i := range node.Attrs.Returns {
		node.Attrs.Returns[i].Status = UnescapeBody(node.Attrs.Returns[i].Status)
		node.Attrs.Returns[i].Description = UnescapeBody(node.Attrs.Returns[i].Description)
	}

	for i := range node.Children {
		unescapeNode(&node.Children[i])
	}
}

// --- Pass 4: Serialize internal AST to Markdown ---

func serializeMarkdown(doc *Document) []byte {
	var buf strings.Builder
	for i, node := range doc.Body {
		if i > 0 {
			buf.WriteByte('\n')
		}
		serializeNodeMD(&buf, node, 0)
	}
	return []byte(strings.TrimRight(buf.String(), "\n") + "\n")
}

func serializeNodeMD(buf *strings.Builder, node Node, indent int) {
	prefix := strings.Repeat("  ", indent)

	switch node.Tag {
	case TagH1, TagH2, TagH3, TagH4, TagH5, TagH6:
		level := int(node.Tag-TagH1) + 1
		inline := serializeInlineMD(node.Children)
		// Check if inline content contains hard breaks (newlines).
		// ATX headings can't span multiple lines, so use setext for H1/H2.
		if strings.Contains(inline, "\n") && (level == 1 || level == 2) {
			underline := "="
			if level == 2 {
				underline = "-"
			}
			fmt.Fprintf(buf, "%s%s\n%s%s\n", prefix, inline, prefix, underline)
		} else {
			hashes := strings.Repeat("#", level)
			// Escape trailing # to prevent ATX heading closing sequence stripping.
			inline = escapeTrailingHashes(inline)
			fmt.Fprintf(buf, "%s%s %s\n", prefix, hashes, inline)
		}

	case TagHR:
		fmt.Fprintf(buf, "%s---\n", prefix)

	case TagBR:
		// Hard line break in block context.
		buf.WriteString(prefix)
		buf.WriteString("  \n")

	case TagP:
		inline := serializeInlineMD(node.Children)
		// Escape leading # to prevent paragraph text from being parsed as a heading.
		inline = escapeLeadingBlockMarker(inline)
		fmt.Fprintf(buf, "%s%s\n", prefix, inline)

	case TagBQ:
		if len(node.Children) == 0 {
			// Empty blockquote.
			fmt.Fprintf(buf, "%s>\n", prefix)
		} else {
			for i, child := range node.Children {
				// Separate block-level children with a blank blockquote line.
				if i > 0 {
					fmt.Fprintf(buf, "%s>\n", prefix)
				}
				var childBuf strings.Builder
				serializeNodeMD(&childBuf, child, 0)
				for _, line := range strings.Split(strings.TrimRight(childBuf.String(), "\n"), "\n") {
					fmt.Fprintf(buf, "%s> %s\n", prefix, line)
				}
			}
		}

	case TagCodeBlock:
		lang := node.Attrs.Language
		// Choose fence character: use tildes if language contains backtick
		// (CommonMark disallows backticks in backtick-fence info strings).
		fenceChar := "`"
		if strings.ContainsRune(lang, '`') {
			fenceChar = "~"
		}
		// Choose fence length that doesn't conflict with content.
		fence := strings.Repeat(fenceChar, 3)
		for strings.Contains(node.Content, fence) {
			fence += fenceChar
		}
		fmt.Fprintf(buf, "%s%s%s\n", prefix, fence, lang)
		lines := strings.Split(node.Content, "\n")
		for _, line := range lines {
			buf.WriteString(prefix)
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
		fmt.Fprintf(buf, "%s%s\n", prefix, fence)

	case TagUL:
		for _, item := range node.Children {
			inline := serializeInlineMD(filterInlineChildren(item))
			marker := "- "
			// If "- " + content forms a thematic break, use "* " instead.
			if looksLikeThematicBreak("- " + inline) {
				marker = "* "
			}
			fmt.Fprintf(buf, "%s%s%s\n", prefix, marker, inline)
			for _, child := range item.Children {
				if child.Tag == TagUL || child.Tag == TagOL {
					serializeNodeMD(buf, child, indent+1)
				}
			}
		}

	case TagOL:
		for i, item := range node.Children {
			inline := serializeInlineMD(filterInlineChildren(item))
			fmt.Fprintf(buf, "%s%d. %s\n", prefix, i+1, inline)
			for _, child := range item.Children {
				if child.Tag == TagUL || child.Tag == TagOL {
					serializeNodeMD(buf, child, indent+1)
				}
			}
		}

	case TagTable:
		serializeTableMD(buf, node.Attrs.Headers, node.Attrs.Cells, prefix)

	// Domain blocks decode to markdown structures.
	case TagKV:
		headers := []string{"Field", "Type", "Description"}
		var cells [][]string
		for _, item := range node.Attrs.Items {
			cells = append(cells, []string{item.Key, item.Type, item.Description})
		}
		serializeTableMD(buf, headers, cells, prefix)

	case TagParams:
		headers := []string{"Name", "Type", "Required", "Description"}
		var cells [][]string
		for _, p := range node.Attrs.Params {
			req := "No"
			if p.Required {
				req = "Yes"
			}
			cells = append(cells, []string{p.Name, p.Type, req, p.Description})
		}
		serializeTableMD(buf, headers, cells, prefix)

	case TagEndpoint:
		if node.Attrs.Endpoint != nil {
			fmt.Fprintf(buf, "%s### %s %s\n", prefix, node.Attrs.Endpoint.Method, node.Attrs.Endpoint.Path)
		}

	case TagReturns:
		// Serialize as inline text: "Status: Description | Status: Description"
		var parts []string
		for _, r := range node.Attrs.Returns {
			if r.Description != "" {
				parts = append(parts, r.Status+": "+r.Description)
			} else {
				parts = append(parts, r.Status)
			}
		}
		if len(parts) > 0 {
			fmt.Fprintf(buf, "%s%s\n", prefix, strings.Join(parts, " | "))
		}

	case TagDef:
		headers := []string{"Term", "Definition"}
		var cells [][]string
		for _, item := range node.Attrs.Items {
			cells = append(cells, []string{item.Key, item.Description})
		}
		serializeTableMD(buf, headers, cells, prefix)

	case TagNote:
		fmt.Fprintf(buf, "%s> **Note:** %s\n", prefix, node.Attrs.Callout)

	case TagWarn:
		fmt.Fprintf(buf, "%s> **Warning:** %s\n", prefix, node.Attrs.Callout)

	case TagTip:
		fmt.Fprintf(buf, "%s> **Tip:** %s\n", prefix, node.Attrs.Callout)

	case TagImportant:
		fmt.Fprintf(buf, "%s> **Important:** %s\n", prefix, node.Attrs.Callout)

	case TagRaw:
		if node.Content != "" {
			buf.WriteString(prefix)
			buf.WriteString(node.Content)
			buf.WriteByte('\n')
		}
	}
}

// serializeInlineMD renders inline nodes as Markdown.
// escapeMDInline escapes markdown-significant characters in literal text
// so they don't get interpreted as formatting when re-parsed.
// Characters that are already backslash-escaped in the source are left as-is.
func escapeMDInline(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			// Already an escape sequence — pass through as-is.
			buf.WriteByte(s[i])
			buf.WriteByte(s[i+1])
			i++
			continue
		}
		switch s[i] {
		case '*', '_':
			buf.WriteByte('\\')
			buf.WriteByte(s[i])
		default:
			buf.WriteByte(s[i])
		}
	}
	return buf.String()
}

// emphDelimiters chooses emphasis delimiters that won't merge with adjacent text.
// If the buffer ends with * or _ (same char as delim), switch to the alternate.
// Similarly, if the next node's text starts with * or _, switch closing delimiter.
func emphDelimiters(delim string, parentEmphChar byte, buf *strings.Builder) (string, string) {
	ch := delim[0] // '*' or '_'
	altCh := byte('_')
	if ch == '_' {
		altCh = '*'
	}
	alt := strings.Repeat(string(altCh), len(delim))

	// If parent emphasis uses the same char, switch to alternate to prevent
	// delimiter merging (e.g., @I{@I{0}} → *_0_* instead of **0**).
	if parentEmphChar == ch {
		return alt, alt
	}

	// If the buffer ends with an unescaped emphasis char (e.g., consecutive
	// emphasis nodes *a**b*), switch to alternate to prevent merging.
	// Don't switch if the char is backslash-escaped (e.g., \*) since
	// escaped chars don't act as emphasis delimiters.
	if buf.Len() > 0 {
		s := buf.String()
		if s[len(s)-1] == ch {
			escaped := buf.Len() >= 2 && s[len(s)-2] == '\\'
			if !escaped {
				return alt, alt
			}
		}
	}

	return delim, delim
}

func serializeInlineMD(nodes []Node) string {
	return serializeInlineMDCtx(nodes, byte(0))
}

func serializeInlineMDCtx(nodes []Node, parentEmphChar byte) string {
	var buf strings.Builder
	for i, n := range nodes {
		_ = i // used for lookahead below
		switch n.Tag {
		case TagRaw:
			// If this raw text follows a strikethrough and starts with ~,
			// add a space to prevent ~~text~~~ ambiguity in goldmark.
			if i > 0 && nodes[i-1].Tag == TagStrikethrough && strings.HasPrefix(n.Content, "~") {
				buf.WriteByte(' ')
			}
			buf.WriteString(escapeMDInline(n.Content))
		case TagBold:
			open, close := emphDelimiters("**", parentEmphChar, &buf)
			buf.WriteString(open)
			buf.WriteString(serializeInlineMDCtx(n.Children, open[0]))
			buf.WriteString(close)
		case TagItalic:
			open, close := emphDelimiters("*", parentEmphChar, &buf)
			buf.WriteString(open)
			buf.WriteString(serializeInlineMDCtx(n.Children, open[0]))
			buf.WriteString(close)
		case TagBoldItalic:
			open, close := emphDelimiters("***", parentEmphChar, &buf)
			buf.WriteString(open)
			buf.WriteString(serializeInlineMDCtx(n.Children, open[0]))
			buf.WriteString(close)
		case TagCode:
			// Use enough backticks to avoid conflicts with content.
			ticks := "`"
			for strings.Contains(n.Content, ticks) {
				ticks += "`"
			}
			// Add space padding when needed:
			// 1. If content starts/ends with backtick (to prevent delimiter merging).
			// 2. If content starts AND ends with space AND is not all spaces
			//    (to prevent goldmark's space-stripping from removing spaces).
			//    Content that is all spaces is NOT stripped by goldmark, so
			//    no padding needed.
			content := n.Content
			needsPadding := false
			if len(content) > 0 && (content[0] == '`' || content[len(content)-1] == '`') {
				needsPadding = true
			}
			if len(content) > 0 && content[0] == ' ' && content[len(content)-1] == ' ' &&
				strings.TrimSpace(content) != "" {
				needsPadding = true
			}
			if needsPadding {
				content = " " + content + " "
			}
			buf.WriteString(ticks)
			buf.WriteString(content)
			buf.WriteString(ticks)
		case TagStrikethrough:
			// If previous output ends with ~, insert space to prevent ~~~
			// being parsed as a code fence.
			if buf.Len() > 0 {
				s := buf.String()
				if s[len(s)-1] == '~' {
					buf.WriteByte(' ')
				}
			}
			buf.WriteString("~~")
			buf.WriteString(serializeInlineMD(n.Children))
			buf.WriteString("~~")
		case TagLink:
			buf.WriteString("[")
			buf.WriteString(serializeInlineMD(n.Children))
			buf.WriteString("](")
			buf.WriteString(escapeMDURL(n.Attrs.URL))
			buf.WriteString(")")
		case TagImage:
			buf.WriteString("![")
			buf.WriteString(serializeInlineMD(n.Children))
			buf.WriteString("](")
			buf.WriteString(escapeMDURL(n.Attrs.URL))
			buf.WriteString(")")
		case TagBR:
			// Use trailing-spaces hard break if buffer ends with backslash
			// (to avoid backslash merging with \-newline form).
			// Use backslash hard break otherwise (works even at start of line).
			if buf.Len() > 0 && buf.String()[buf.Len()-1] == '\\' {
				buf.WriteString("  \n")
			} else {
				buf.WriteString("\\\n")
			}
			// After a hard break, text starts at a new line. If the next
			// text starts with a block-level marker (unordered list, heading,
			// blockquote), escape it to prevent markdown reinterpretation.
			// Note: ordered list markers (digits + ./)) can't be escaped with
			// backslash since \digit is not a valid markdown escape.
			if i+1 < len(nodes) && nodes[i+1].Tag == TagRaw {
				c := nodes[i+1].Content
				if len(c) > 0 && (c[0] == '+' || c[0] == '-' || c[0] == '#' || c[0] == '>') {
					nodes[i+1].Content = "\\" + c
				}
			}
		}
	}
	return buf.String()
}

// serializeTableMD writes a standard markdown table.
func serializeTableMD(buf *strings.Builder, headers []string, rows [][]string, prefix string) {
	if len(headers) == 0 {
		return
	}
	// Header row.
	fmt.Fprintf(buf, "%s| %s |\n", prefix, strings.Join(headers, " | "))
	// Separator row.
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Fprintf(buf, "%s| %s |\n", prefix, strings.Join(seps, " | "))
	// Data rows.
	for _, row := range rows {
		// Pad row to header length.
		for len(row) < len(headers) {
			row = append(row, "")
		}
		fmt.Fprintf(buf, "%s| %s |\n", prefix, strings.Join(row, " | "))
	}
}

// escapeMDURL escapes characters in a URL that would break markdown link syntax.
// URL-encodes problematic characters so goldmark stores the encoded form,
// which normalizer must also handle.
func escapeMDURL(url string) string {
	// URL-encode characters that break inline link destinations.
	needsEscape := strings.ContainsAny(url, "() \t\\") ||
		strings.Contains(url, "](") || strings.Contains(url, "[](")
	if !needsEscape {
		return url
	}
	var buf strings.Builder
	for i := 0; i < len(url); i++ {
		switch url[i] {
		case '(':
			buf.WriteString("%28")
		case ')':
			buf.WriteString("%29")
		case ' ':
			buf.WriteString("%20")
		case '\t':
			buf.WriteString("%09")
		case '\\':
			buf.WriteString("%5C")
		default:
			buf.WriteByte(url[i])
		}
	}
	return buf.String()
}

// escapeLeadingBlockMarker escapes leading characters in paragraph text that
// goldmark would interpret as block-level constructs:
//   - ATX headings: # followed by space or end of line
//   - List markers: +, -, * followed by space/tab/end-of-line
//   - Blockquote markers: > (any position)
//   - Ordered list markers: digit(s) followed by . or ) then space/tab/end-of-line
func escapeLeadingBlockMarker(s string) string {
	if len(s) == 0 {
		return s
	}
	switch s[0] {
	case '#':
		// # is a heading if followed by space or is the entire string (empty heading)
		if len(s) == 1 || s[1] == ' ' || s[1] == '#' {
			return "\\" + s
		}
	case '+', '-':
		// List marker requires space/tab after, or be the entire content
		if len(s) == 1 || s[1] == ' ' || s[1] == '\t' {
			return "\\" + s
		}
	case '*':
		// * is a list marker only with space/tab after. Bare * is emphasis.
		if len(s) > 1 && (s[1] == ' ' || s[1] == '\t') {
			return "\\" + s
		}
	case '>':
		return "\\" + s
	}
	// Check for ordered list markers: 1-9 digit(s) followed by . or ) then space/tab/end.
	// CommonMark spec limits ordered list markers to at most 9 digits.
	i := 0
	for i < len(s) && i < 9 && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) && (s[i] == '.' || s[i] == ')') {
		// Must be followed by space/tab or end of string
		if i+1 >= len(s) || s[i+1] == ' ' || s[i+1] == '\t' {
			return "\\" + s
		}
	}
	return s
}

// escapeTrailingHashes escapes trailing # characters in heading text to prevent
// goldmark from stripping them as ATX heading closing sequences.
func escapeTrailingHashes(s string) string {
	// Find the rightmost non-# non-space character.
	i := len(s) - 1
	for i >= 0 && s[i] == '#' {
		i--
	}
	// Check if there are trailing # after spaces.
	if i < len(s)-1 {
		j := i + 1 // position of first trailing #
		// Don't double-escape: if the char before # is already \, skip.
		if i >= 0 && s[i] == '\\' {
			return s
		}
		s = s[:j] + "\\" + s[j:]
	}
	return s
}

// looksLikeThematicBreak checks if a line would be parsed as a thematic break.
// A thematic break is 3+ of the same character (-, *, _) with optional spaces.
func looksLikeThematicBreak(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < 3 {
		return false
	}
	// Determine which character
	ch := line[0]
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	count := 0
	for _, r := range line {
		if r == rune(ch) {
			count++
		} else if r != ' ' && r != '\t' {
			return false
		}
	}
	return count >= 3
}
