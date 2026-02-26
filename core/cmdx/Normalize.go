package cmdx

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// CompareASTs parses two markdown documents and returns whether they are
// semantically equivalent (ignoring presentational metadata per D1).
func CompareASTs(a, b []byte) (bool, string, error) {
	dumpA, err := DumpAST(a)
	if err != nil {
		return false, "", fmt.Errorf("dump A: %w", err)
	}
	dumpB, err := DumpAST(b)
	if err != nil {
		return false, "", fmt.Errorf("dump B: %w", err)
	}

	if dumpA == dumpB {
		return true, "", nil
	}

	// Build a diff message showing the first divergence.
	linesA := strings.Split(dumpA, "\n")
	linesB := strings.Split(dumpB, "\n")
	maxLen := len(linesA)
	if len(linesB) > maxLen {
		maxLen = len(linesB)
	}
	for i := 0; i < maxLen; i++ {
		la := ""
		lb := ""
		if i < len(linesA) {
			la = linesA[i]
		}
		if i < len(linesB) {
			lb = linesB[i]
		}
		if la != lb {
			diff := fmt.Sprintf("first difference at line %d:\n  A: %s\n  B: %s", i+1, la, lb)
			return false, diff, nil
		}
	}
	return false, "dumps differ but no line difference found", nil
}

// DumpAST parses markdown with goldmark and produces a canonical tree dump.
// Per D1: strips source positions, table alignment, link/image titles.
// Per D2: merges adjacent text nodes to normalize soft line breaks.
func DumpAST(markdown []byte) (string, error) {
	md := newGoldmark()
	reader := text.NewReader(markdown)
	doc := md.Parser().Parse(reader)

	var buf strings.Builder
	dumpNode(&buf, doc, markdown, 0)
	return buf.String(), nil
}

func dumpNode(buf *strings.Builder, n ast.Node, source []byte, depth int) {
	indent := strings.Repeat("  ", depth)

	switch v := n.(type) {
	case *ast.Document:
		fmt.Fprintf(buf, "%sDocument\n", indent)
		dumpChildren(buf, n, source, depth+1)

	case *ast.Heading:
		fmt.Fprintf(buf, "%sHeading[level=%d]\n", indent, v.Level)
		dumpInlineChildren(buf, n, source, depth+1)

	case *ast.Paragraph:
		fmt.Fprintf(buf, "%sParagraph\n", indent)
		dumpInlineChildren(buf, n, source, depth+1)

	case *ast.TextBlock:
		// Normalize: TextBlock (tight lists) and Paragraph (loose lists)
		// are functionally identical inside list items. Use same label.
		if _, inListItem := n.Parent().(*ast.ListItem); inListItem {
			fmt.Fprintf(buf, "%sParagraph\n", indent)
		} else {
			fmt.Fprintf(buf, "%sTextBlock\n", indent)
		}
		dumpInlineChildren(buf, n, source, depth+1)

	case *ast.ThematicBreak:
		fmt.Fprintf(buf, "%sThematicBreak\n", indent)

	case *ast.Blockquote:
		fmt.Fprintf(buf, "%sBlockquote\n", indent)
		dumpChildren(buf, n, source, depth+1)

	case *ast.FencedCodeBlock:
		lang := ""
		if v.Language(source) != nil {
			lang = strings.TrimRight(string(v.Language(source)), " \t")
		}
		content := readBlockLines(v, source)
		fmt.Fprintf(buf, "%sCodeBlock[lang=%q]\n", indent, lang)
		fmt.Fprintf(buf, "%s  %q\n", indent, content)

	case *ast.CodeBlock:
		content := readBlockLines(v, source)
		fmt.Fprintf(buf, "%sCodeBlock[lang=%q]\n", indent, "")
		fmt.Fprintf(buf, "%s  %q\n", indent, content)

	case *ast.List:
		kind := "unordered"
		if v.IsOrdered() {
			kind = "ordered"
		}
		fmt.Fprintf(buf, "%sList[%s]\n", indent, kind)
		dumpChildren(buf, n, source, depth+1)

	case *ast.ListItem:
		fmt.Fprintf(buf, "%sListItem\n", indent)
		dumpChildren(buf, n, source, depth+1)

	case *ast.Emphasis:
		fmt.Fprintf(buf, "%sEmphasis[level=%d]\n", indent, v.Level)
		dumpInlineChildren(buf, n, source, depth+1)

	case *ast.CodeSpan:
		text := extractTextContent(v, source)
		// Normalize: goldmark preserves raw \n in code span text but renders
		// them as spaces. Normalize for consistent comparison.
		text = strings.ReplaceAll(text, "\n", " ")
		fmt.Fprintf(buf, "%sCodeSpan %q\n", indent, text)

	case *ast.Link:
		// D1: strip title. Normalize percent-encoded URL chars.
		fmt.Fprintf(buf, "%sLink[dest=%q]\n", indent, normalizeLinkURL(string(v.Destination)))
		dumpInlineChildren(buf, n, source, depth+1)

	case *ast.Image:
		// D1: strip title. Normalize percent-encoded URL chars.
		fmt.Fprintf(buf, "%sImage[dest=%q]\n", indent, normalizeLinkURL(string(v.Destination)))
		dumpInlineChildren(buf, n, source, depth+1)

	case *ast.AutoLink:
		url := string(v.URL(source))
		fmt.Fprintf(buf, "%sLink[dest=%q]\n", indent, url)
		label := string(v.Label(source))
		fmt.Fprintf(buf, "%s  Text %q\n", strings.Repeat("  ", depth), label)

	case *east.Table:
		// D1: strip alignment.
		fmt.Fprintf(buf, "%sTable\n", indent)
		dumpChildren(buf, n, source, depth+1)

	case *east.TableHeader:
		fmt.Fprintf(buf, "%sTableHeader\n", indent)
		dumpChildren(buf, n, source, depth+1)

	case *east.TableRow:
		fmt.Fprintf(buf, "%sTableRow\n", indent)
		dumpChildren(buf, n, source, depth+1)

	case *east.TableCell:
		// D1: strip alignment.
		fmt.Fprintf(buf, "%sTableCell\n", indent)
		dumpInlineChildren(buf, n, source, depth+1)

	case *east.Strikethrough:
		fmt.Fprintf(buf, "%sStrikethrough\n", indent)
		dumpInlineChildren(buf, n, source, depth+1)

	case *ast.HTMLBlock:
		content := readBlockLines(v, source)
		fmt.Fprintf(buf, "%sHTMLBlock %q\n", indent, content)

	case *ast.RawHTML:
		content := string(v.Segments.Value(source))
		fmt.Fprintf(buf, "%sRawHTML %q\n", indent, content)

	case *ast.Text:
		// Text nodes are handled by dumpInlineChildren to merge adjacent ones.
		content := string(v.Segment.Value(source))
		fmt.Fprintf(buf, "%sText %q\n", indent, content)

	default:
		fmt.Fprintf(buf, "%s%s\n", indent, n.Kind().String())
		dumpChildren(buf, n, source, depth+1)
	}
}

// dumpChildren dumps all children of a block node.
// It merges consecutive lists of the same type (both unordered or both ordered)
// since different list markers (+, -, *) are presentational per D1.
func dumpChildren(buf *strings.Builder, n ast.Node, source []byte, depth int) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		list, isList := child.(*ast.List)
		if isList {
			// Dump list header once, then merge items from consecutive same-type lists.
			indent := strings.Repeat("  ", depth)
			kind := "unordered"
			if list.IsOrdered() {
				kind = "ordered"
			}
			fmt.Fprintf(buf, "%sList[%s]\n", indent, kind)
			// Dump items of this list.
			for item := list.FirstChild(); item != nil; item = item.NextSibling() {
				dumpNode(buf, item, source, depth+1)
			}
			// Merge any consecutive same-type lists.
			for next := child.NextSibling(); next != nil; next = child.NextSibling() {
				nextList, ok := next.(*ast.List)
				if !ok || nextList.IsOrdered() != list.IsOrdered() {
					break
				}
				// Same type — dump its items under the same list header.
				for item := nextList.FirstChild(); item != nil; item = item.NextSibling() {
					dumpNode(buf, item, source, depth+1)
				}
				child = next // Advance past this merged list.
			}
		} else {
			dumpNode(buf, child, source, depth)
		}
	}
}

// dumpInlineChildren dumps inline children, merging adjacent Text nodes.
// This normalizes soft line break differences (D9) and whitespace at
// inline formatting boundaries (e.g., space after strikethrough closing ~~).
func dumpInlineChildren(buf *strings.Builder, n ast.Node, source []byte, depth int) {
	indent := strings.Repeat("  ", depth)
	var textBuf strings.Builder
	hasHardBreak := false
	prevWasStrikethrough := false

	flushText := func() {
		if textBuf.Len() > 0 {
			text := textBuf.String()
			// Normalize: if this text follows a strikethrough and has a
			// leading space before ~, that space was inserted by the
			// serializer to prevent ~~text~~~ ambiguity. Strip it.
			if prevWasStrikethrough && len(text) >= 2 && text[0] == ' ' && text[1] == '~' {
				text = text[1:]
			}
			// Normalize: strip trailing spaces before hard breaks.
			// goldmark may include trailing spaces in text segments as
			// part of hard break syntax, not as semantic content.
			if hasHardBreak {
				text = strings.TrimRight(text, " ")
			}
			text = unescapeMDBackslash(text)
			fmt.Fprintf(buf, "%sText %q\n", indent, text)
			textBuf.Reset()
		}
		if hasHardBreak {
			fmt.Fprintf(buf, "%sHardBreak\n", indent)
			hasHardBreak = false
		}
	}

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			content := string(t.Segment.Value(source))
			textBuf.WriteString(content)
			if t.SoftLineBreak() {
				textBuf.WriteString(" ")
			}
			if t.HardLineBreak() {
				// Flush accumulated text immediately before the hard break,
				// then emit the hard break. This prevents trailing spaces
				// (syntactic hard break markers) from merging with subsequent text.
				hasHardBreak = true
				flushText()
			}
		} else {
			_, isStrikethrough := child.(*east.Strikethrough)
			// Normalize: strip trailing space before strikethrough when text
			// ends with "~ " — the space was inserted by the decoder to prevent
			// ~~~ being parsed as a code fence.
			if isStrikethrough && textBuf.Len() > 0 {
				t := textBuf.String()
				if len(t) >= 2 && t[len(t)-2] == '~' && t[len(t)-1] == ' ' {
					textBuf.Reset()
					textBuf.WriteString(t[:len(t)-1])
				}
			}
			flushText()
			prevWasStrikethrough = isStrikethrough
			dumpNode(buf, child, source, depth)
		}
	}
	flushText()
}

// unescapeMDBackslash removes markdown backslash escapes from text.
// e.g., `\*` → `*`, `\\` → `\`, `\_` → `_`.
func unescapeMDBackslash(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var buf strings.Builder
	buf.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			// Markdown spec: any ASCII punctuation can be backslash-escaped.
			if (next >= '!' && next <= '/') || (next >= ':' && next <= '@') ||
				(next >= '[' && next <= '`') || (next >= '{' && next <= '~') {
				buf.WriteByte(next)
				i++ // skip the escaped char
				continue
			}
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}

// normalizeLinkURL decodes common percent-encoded characters in URLs to allow
// comparison between original URLs and URLs that were percent-encoded for
// markdown safety.
func normalizeLinkURL(url string) string {
	url = strings.ReplaceAll(url, "%28", "(")
	url = strings.ReplaceAll(url, "%29", ")")
	url = strings.ReplaceAll(url, "%20", " ")
	url = strings.ReplaceAll(url, "%09", "\t")
	url = strings.ReplaceAll(url, "%5C", "\\")
	url = strings.ReplaceAll(url, "%3E", ">")
	url = strings.ReplaceAll(url, "%3C", "<")
	url = strings.ReplaceAll(url, "%0A", "\n")
	return url
}
