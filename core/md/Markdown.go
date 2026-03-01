package md

import (
	"fmt"
	"strings"
)

// serializeMarkdownWithRefs converts an AST node list to compact markdown,
// using reference-style links for URLs in the refs map.
func serializeMarkdownWithRefs(body []Node, refs map[string]string) []byte {
	var buf strings.Builder
	for i, node := range body {
		if i > 0 {
			buf.WriteByte('\n')
		}
		serializeNodeMDWithRefs(&buf, node, 0, refs)
	}
	// Append reference link definitions at the end of the document.
	if len(refs) > 0 {
		defs := buildRefDefs(refs)
		buf.WriteByte('\n')
		for _, def := range defs {
			buf.WriteString(def)
			buf.WriteByte('\n')
		}
	}
	return []byte(strings.TrimRight(buf.String(), "\n") + "\n")
}

// buildRefDefs builds sorted reference link definitions from a refs map.
// Returns lines like "[1]: https://example.com/path".
func buildRefDefs(refs map[string]string) []string {
	// Sort by label (numeric order) for deterministic output.
	type entry struct {
		label string
		url   string
	}
	entries := make([]entry, 0, len(refs))
	for url, label := range refs {
		entries = append(entries, entry{label, url})
	}
	// Sort numerically by label.
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].label > entries[j].label {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	defs := make([]string, len(entries))
	for i, e := range entries {
		defs[i] = fmt.Sprintf("[%s]: %s", e.label, e.url)
	}
	return defs
}

func serializeNodeMDWithRefs(buf *strings.Builder, node Node, indent int, refs map[string]string) {
	prefix := strings.Repeat("  ", indent)

	switch node.Tag {
	case TagH1, TagH2, TagH3, TagH4, TagH5, TagH6:
		level := int(node.Tag-TagH1) + 1
		inline := serializeInlineMDWithRefs(node.Children, refs)
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
		inline := serializeInlineMDWithRefs(node.Children, refs)
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
				serializeNodeMDWithRefs(&childBuf, child, 0, refs)
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
			inline := serializeInlineMDWithRefs(filterInlineChildren(item), refs)
			marker := "- "
			// If "- " + content forms a thematic break, use "* " instead.
			if looksLikeThematicBreak("- " + inline) {
				marker = "* "
			}
			fmt.Fprintf(buf, "%s%s%s\n", prefix, marker, inline)
			for _, child := range item.Children {
				if child.Tag == TagUL || child.Tag == TagOL {
					serializeNodeMDWithRefs(buf, child, indent+1, refs)
				}
			}
		}

	case TagOL:
		for i, item := range node.Children {
			inline := serializeInlineMDWithRefs(filterInlineChildren(item), refs)
			fmt.Fprintf(buf, "%s%d. %s\n", prefix, i+1, inline)
			for _, child := range item.Children {
				if child.Tag == TagUL || child.Tag == TagOL {
					serializeNodeMDWithRefs(buf, child, indent+1, refs)
				}
			}
		}

	case TagTable:
		serializeTableMD(buf, node.Attrs.Headers, node.Attrs.Cells, prefix)

	case TagRaw:
		if node.Content != "" {
			buf.WriteString(prefix)
			buf.WriteString(node.Content)
			buf.WriteByte('\n')
		}
	}
}

// serializeInlineMDWithRefs renders inline nodes as Markdown with reference links.
func serializeInlineMDWithRefs(nodes []Node, refs map[string]string) string {
	return serializeInlineMDFull(nodes, byte(0), refs)
}

func serializeInlineMDFull(nodes []Node, parentEmphChar byte, refs map[string]string) string {
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
			buf.WriteString(serializeInlineMDFull(n.Children, open[0], refs))
			buf.WriteString(close)
		case TagItalic:
			open, close := emphDelimiters("*", parentEmphChar, &buf)
			buf.WriteString(open)
			buf.WriteString(serializeInlineMDFull(n.Children, open[0], refs))
			buf.WriteString(close)
		case TagBoldItalic:
			open, close := emphDelimiters("***", parentEmphChar, &buf)
			buf.WriteString(open)
			buf.WriteString(serializeInlineMDFull(n.Children, open[0], refs))
			buf.WriteString(close)
		case TagCode:
			// Use enough backticks to avoid conflicts with content.
			ticks := "`"
			for strings.Contains(n.Content, ticks) {
				ticks += "`"
			}
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
			buf.WriteString(serializeInlineMDFull(n.Children, byte(0), refs))
			buf.WriteString("~~")
		case TagLink:
			if label, ok := refs[n.Attrs.URL]; ok {
				// Reference-style link: [text][label]
				buf.WriteString("[")
				buf.WriteString(serializeInlineMDFull(n.Children, byte(0), refs))
				buf.WriteString("][")
				buf.WriteString(label)
				buf.WriteString("]")
			} else {
				buf.WriteString("[")
				buf.WriteString(serializeInlineMDFull(n.Children, byte(0), refs))
				buf.WriteString("](")
				buf.WriteString(escapeMDURL(n.Attrs.URL))
				buf.WriteString(")")
			}
		case TagImage:
			if label, ok := refs[n.Attrs.URL]; ok {
				// Reference-style image: ![alt][label]
				buf.WriteString("![")
				buf.WriteString(serializeInlineMDFull(n.Children, byte(0), refs))
				buf.WriteString("][")
				buf.WriteString(label)
				buf.WriteString("]")
			} else {
				buf.WriteString("![")
				buf.WriteString(serializeInlineMDFull(n.Children, byte(0), refs))
				buf.WriteString("](")
				buf.WriteString(escapeMDURL(n.Attrs.URL))
				buf.WriteString(")")
			}
		case TagBR:
			if buf.Len() > 0 && buf.String()[buf.Len()-1] == '\\' {
				buf.WriteString("  \n")
			} else {
				buf.WriteString("\\\n")
			}
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

// serializeTableMD writes a compact markdown table.
// Uses minimal syntax: no spaces around pipes, short separator.
// This is BPE-optimized — compact pipe syntax tokenizes ~6% more
// efficiently than spaced syntax in o200k_base.
func serializeTableMD(buf *strings.Builder, headers []string, rows [][]string, prefix string) {
	if len(headers) == 0 {
		return
	}
	// Header row (compact: no spaces around pipes).
	fmt.Fprintf(buf, "%s|%s|\n", prefix, joinEscapedCells(headers))
	// Separator row (minimal: single dash per column).
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "-"
	}
	fmt.Fprintf(buf, "%s|%s|\n", prefix, strings.Join(seps, "|"))
	// Data rows (compact).
	for _, row := range rows {
		// Pad row to header length.
		for len(row) < len(headers) {
			row = append(row, "")
		}
		fmt.Fprintf(buf, "%s|%s|\n", prefix, joinEscapedCells(row))
	}
}

// joinEscapedCells joins table cell values with "|", escaping pipe characters
// and trailing backslashes within cell content to prevent misparse.
func joinEscapedCells(cells []string) string {
	escaped := make([]string, len(cells))
	for i, cell := range cells {
		escaped[i] = escapeTableCell(cell)
	}
	return strings.Join(escaped, "|")
}

// escapeTableCell escapes characters in a table cell that would break
// compact pipe-delimited table parsing.
func escapeTableCell(s string) string {
	// In compact format (no spaces around pipes), we must escape:
	// 1. Literal pipe characters in cell content (would be read as column delimiter)
	// 2. Trailing backslash (would escape the closing pipe delimiter)
	needsEscape := strings.ContainsRune(s, '|') ||
		(len(s) > 0 && s[len(s)-1] == '\\')
	if !needsEscape {
		return s
	}
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			buf.WriteString("\\|")
		} else {
			buf.WriteByte(s[i])
		}
	}
	// If the cell ends with a backslash, add a space to prevent it from
	// escaping the closing pipe delimiter.
	result := buf.String()
	if len(result) > 0 && result[len(result)-1] == '\\' {
		result += " "
	}
	return result
}

// --- Markdown escaping helpers ---

// escapeMDInline escapes markdown-significant characters in literal text
// so they don't get interpreted as formatting when re-parsed.
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
func emphDelimiters(delim string, parentEmphChar byte, buf *strings.Builder) (string, string) {
	ch := delim[0]
	altCh := byte('_')
	if ch == '_' {
		altCh = '*'
	}
	alt := strings.Repeat(string(altCh), len(delim))

	if parentEmphChar == ch {
		return alt, alt
	}

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

// escapeMDURL escapes characters in a URL that would break markdown link syntax.
func escapeMDURL(url string) string {
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
// goldmark would interpret as block-level constructs.
func escapeLeadingBlockMarker(s string) string {
	if len(s) == 0 {
		return s
	}
	switch s[0] {
	case '#':
		if len(s) == 1 || s[1] == ' ' || s[1] == '#' {
			return "\\" + s
		}
	case '+', '-':
		if len(s) == 1 || s[1] == ' ' || s[1] == '\t' {
			return "\\" + s
		}
	case '*':
		if len(s) > 1 && (s[1] == ' ' || s[1] == '\t') {
			return "\\" + s
		}
	case '>':
		return "\\" + s
	}
	// Check for ordered list markers.
	i := 0
	for i < len(s) && i < 9 && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) && (s[i] == '.' || s[i] == ')') {
		if i+1 >= len(s) || s[i+1] == ' ' || s[i+1] == '\t' {
			return "\\" + s
		}
	}
	return s
}

// escapeTrailingHashes escapes trailing # characters in heading text.
func escapeTrailingHashes(s string) string {
	i := len(s) - 1
	for i >= 0 && s[i] == '#' {
		i--
	}
	if i < len(s)-1 {
		j := i + 1
		if i >= 0 && s[i] == '\\' {
			return s
		}
		s = s[:j] + "\\" + s[j:]
	}
	return s
}

// looksLikeThematicBreak checks if a line would be parsed as a thematic break.
func looksLikeThematicBreak(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < 3 {
		return false
	}
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
