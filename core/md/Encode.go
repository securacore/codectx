package md

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/securacore/codectx/core/tokenizer"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Encode converts Markdown to compact, normalized Markdown.
// The output is BPE-optimized: it uses native markdown syntax (which
// tokenizes efficiently) rather than custom tags.
func Encode(markdown []byte) ([]byte, error) {
	// Pass 1: Parse markdown -> goldmark AST -> internal AST.
	body, err := markdownToAST(markdown)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	// Pass 1.5: Compute reference-style link map for URLs where it saves tokens.
	refs := computeRefLinks(body)

	// Pass 2: Serialize AST -> compact markdown.
	return serializeMarkdownWithRefs(body, refs), nil
}

// newGoldmark returns a consistently-configured goldmark instance.
func newGoldmark() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.GFM))
}

// markdownToAST parses markdown with goldmark and converts to internal AST nodes.
func markdownToAST(source []byte) ([]Node, error) {
	md := newGoldmark()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var nodes []Node
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		n, ok := convertBlock(child, source)
		if ok {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

// convertBlock converts a goldmark block-level node to an internal Node.
func convertBlock(n ast.Node, source []byte) (Node, bool) {
	switch v := n.(type) {
	case *ast.Heading:
		tag := TagH1 + TagType(v.Level-1)
		children := convertInlineChildren(v, source)
		return Node{Tag: tag, Children: children}, true

	case *ast.Paragraph:
		children := convertInlineChildren(v, source)
		return Node{Tag: TagP, Children: children}, true

	case *ast.ThematicBreak:
		return Node{Tag: TagHR}, true

	case *ast.Blockquote:
		return convertBlockquote(v, source), true

	case *ast.FencedCodeBlock:
		lang := ""
		if v.Language(source) != nil {
			lang = strings.TrimRight(string(v.Language(source)), " \t")
		}
		content := readBlockLines(v, source)
		return Node{
			Tag:     TagCodeBlock,
			Content: content,
			Attrs:   NodeAttrs{Language: lang},
		}, true

	case *ast.CodeBlock:
		content := readBlockLines(v, source)
		return Node{
			Tag:     TagCodeBlock,
			Content: content,
		}, true

	case *ast.List:
		return convertList(v, source), true

	case *east.Table:
		return convertTable(v, source), true

	case *ast.TextBlock:
		children := convertInlineChildren(v, source)
		return Node{Tag: TagP, Children: children}, true

	case *ast.HTMLBlock:
		content := readBlockLines(v, source)
		return Node{Tag: TagRaw, Content: content}, true

	default:
		// Unknown block type — skip silently.
		return Node{}, false
	}
}

// convertInlineChildren converts all inline children of a goldmark node.
func convertInlineChildren(parent ast.Node, source []byte) []Node {
	var nodes []Node
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		inline := convertInline(child, source)
		nodes = append(nodes, inline...)
	}
	nodes = mergeAdjacentRaw(nodes)

	// Strip trailing whitespace from raw text nodes that precede a hard break.
	for i := 0; i+1 < len(nodes); i++ {
		if nodes[i].Tag == TagRaw && nodes[i+1].Tag == TagBR {
			nodes[i].Content = strings.TrimRight(nodes[i].Content, " ")
		}
	}
	return nodes
}

// convertInline converts a single goldmark inline node to internal Nodes.
func convertInline(n ast.Node, source []byte) []Node {
	switch v := n.(type) {
	case *ast.Text:
		content := string(v.Segment.Value(source))
		var nodes []Node
		nodes = append(nodes, Node{Tag: TagRaw, Content: content})
		if v.SoftLineBreak() {
			nodes = append(nodes, Node{Tag: TagRaw, Content: " "})
		}
		if v.HardLineBreak() {
			nodes = append(nodes, Node{Tag: TagBR})
		}
		return nodes

	case *ast.Emphasis:
		children := convertInlineChildren(v, source)
		tag := TagItalic
		if v.Level == 2 {
			tag = TagBold
		}
		return []Node{{Tag: tag, Children: children}}

	case *ast.CodeSpan:
		content := extractTextContent(v, source)
		return []Node{{Tag: TagCode, Content: content}}

	case *ast.Link:
		children := convertInlineChildren(v, source)
		return []Node{{
			Tag:      TagLink,
			Attrs:    NodeAttrs{URL: string(v.Destination), Display: flattenText(children)},
			Children: children,
		}}

	case *ast.Image:
		children := convertInlineChildren(v, source)
		return []Node{{
			Tag:      TagImage,
			Attrs:    NodeAttrs{URL: string(v.Destination), Display: flattenText(children)},
			Children: children,
		}}

	case *east.Strikethrough:
		children := convertInlineChildren(v, source)
		return []Node{{Tag: TagStrikethrough, Children: children}}

	case *ast.RawHTML:
		return []Node{{Tag: TagRaw, Content: string(v.Segments.Value(source))}}

	case *ast.AutoLink:
		url := string(v.URL(source))
		label := string(v.Label(source))
		return []Node{{
			Tag:      TagLink,
			Attrs:    NodeAttrs{URL: url, Display: label},
			Children: []Node{{Tag: TagRaw, Content: label}},
		}}

	default:
		// Unknown inline — try to extract text from children.
		children := convertInlineChildren(n, source)
		if len(children) > 0 {
			return children
		}
		return nil
	}
}

// convertBlockquote converts a goldmark Blockquote to an internal Node.
func convertBlockquote(bq *ast.Blockquote, source []byte) Node {
	var children []Node
	for child := bq.FirstChild(); child != nil; child = child.NextSibling() {
		n, ok := convertBlock(child, source)
		if ok {
			children = append(children, n)
		}
	}
	return Node{Tag: TagBQ, Children: children}
}

// convertList converts a goldmark List to an internal UL or OL Node.
func convertList(list *ast.List, source []byte) Node {
	tag := TagUL
	if list.IsOrdered() {
		tag = TagOL
	}

	var items []Node
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.ListItem); ok {
			item := convertListItem(child, source)
			items = append(items, item)
		}
	}
	return Node{Tag: tag, Children: items}
}

// convertListItem converts a goldmark ListItem to an internal Node.
func convertListItem(li ast.Node, source []byte) Node {
	var children []Node
	for child := li.FirstChild(); child != nil; child = child.NextSibling() {
		switch child := child.(type) {
		case *ast.Paragraph, *ast.TextBlock:
			inline := convertInlineChildren(child, source)
			children = append(children, inline...)
		case *ast.Heading:
			marker := strings.Repeat("#", child.Level)
			inline := convertInlineChildren(child, source)
			if len(inline) > 0 {
				escapeTrailingHashesInNodes(inline)
				children = append(children, Node{Tag: TagRaw, Content: marker + " "})
				children = append(children, inline...)
			} else {
				children = append(children, Node{Tag: TagRaw, Content: marker})
			}
		case *ast.List:
			nested := convertList(child, source)
			children = append(children, nested)
		default:
			n, ok := convertBlock(child, source)
			if ok {
				children = append(children, n)
			}
		}
	}
	return Node{Tag: TagP, Children: children}
}

// convertTable converts a goldmark GFM Table to an internal TABLE node.
// If the table is a TOC table (2-column, col1 is entirely a link), it is
// converted to a UL list instead, which is more token-efficient.
func convertTable(table *east.Table, source []byte) Node {
	var headers []string
	var rows [][]string

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.(type) {
		case *east.TableHeader:
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					headers = append(headers, extractTextContent(cell, source))
				}
			}
		case *east.TableRow:
			var cells []string
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					cells = append(cells, extractTextContent(cell, source))
				}
			}
			rows = append(rows, cells)
		}
	}

	// Check if this is a TOC table and convert to list if so.
	if tocList, ok := tryConvertTOCTable(table, source); ok {
		return tocList
	}

	return Node{
		Tag: TagTable,
		Attrs: NodeAttrs{
			Headers: headers,
			Cells:   rows,
		},
	}
}

// tryConvertTOCTable checks if a table is a TOC-style table (2 columns,
// col1 is entirely a markdown link in every data row) and converts it
// to a UL list if so. Each row becomes: `- [text](url) -- description`
//
// This saves tokens by eliminating the header row, separator row, and
// pipe delimiters that add overhead without carrying information in
// index/navigation tables.
func tryConvertTOCTable(table *east.Table, source []byte) (Node, bool) {
	// Collect header count and row data from the goldmark AST.
	headerCount := 0
	type tocRow struct {
		linkText string
		linkURL  string
		desc     string
	}
	var tocRows []tocRow

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.(type) {
		case *east.TableHeader:
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					headerCount++
				}
			}
		case *east.TableRow:
			cellCount := 0
			var linkText, linkURL, desc string
			isLinkCell := false
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				tc, ok := cell.(*east.TableCell)
				if !ok {
					continue
				}
				switch cellCount {
				case 0:
					// Check if first cell is exactly one Link node
					// (possibly with surrounding whitespace text nodes).
					linkText, linkURL, isLinkCell = extractSingleLink(tc, source)
				case 1:
					desc = extractTextContent(cell, source)
				}
				cellCount++
			}
			if cellCount != 2 || !isLinkCell {
				return Node{}, false
			}
			tocRows = append(tocRows, tocRow{linkText, linkURL, desc})
		}
	}

	// Must be exactly 2 columns with at least 1 data row.
	if headerCount != 2 || len(tocRows) == 0 {
		return Node{}, false
	}

	// Build UL node with list items.
	var items []Node
	for _, r := range tocRows {
		var children []Node
		children = append(children, Node{
			Tag:      TagLink,
			Attrs:    NodeAttrs{URL: r.linkURL, Display: r.linkText},
			Children: []Node{{Tag: TagRaw, Content: r.linkText}},
		})
		if r.desc != "" {
			children = append(children, Node{Tag: TagRaw, Content: " -- " + r.desc})
		}
		items = append(items, Node{Tag: TagP, Children: children})
	}
	return Node{Tag: TagUL, Children: items}, true
}

// extractSingleLink checks if a table cell contains exactly one Link node
// (with no other meaningful content besides optional whitespace). Returns
// the link text, URL, and whether the cell is a single link.
func extractSingleLink(cell *east.TableCell, source []byte) (string, string, bool) {
	var link *ast.Link
	for child := cell.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *ast.Link:
			if link != nil {
				return "", "", false // multiple links
			}
			link = v
		case *ast.Text:
			// Allow whitespace-only text nodes (padding around the link).
			content := string(v.Segment.Value(source))
			if strings.TrimSpace(content) != "" {
				return "", "", false // non-whitespace text alongside link
			}
		default:
			return "", "", false // other inline elements
		}
	}
	if link == nil {
		return "", "", false
	}
	linkText := extractTextContent(link, source)
	return linkText, string(link.Destination), true
}

// extractTextContent recursively extracts all text from a node's children.
func extractTextContent(n ast.Node, source []byte) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
			if t.SoftLineBreak() {
				buf.WriteByte(' ')
			}
		} else {
			buf.WriteString(extractTextContent(child, source))
		}
	}
	return buf.String()
}

// readBlockLines reads the raw lines of a block node (code blocks).
func readBlockLines(n ast.Node, source []byte) string {
	lines := n.Lines()
	var buf bytes.Buffer
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(source))
	}
	// Trim trailing newline since we add it during serialization.
	return strings.TrimRight(buf.String(), "\n")
}

// escapeTrailingHashesInNodes escapes trailing # in the last TagRaw node of an
// inline node list, preventing ATX heading closing sequence stripping.
func escapeTrailingHashesInNodes(nodes []Node) {
	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].Tag == TagRaw {
			nodes[i].Content = escapeTrailingHashesEncoder(nodes[i].Content)
			return
		}
		if len(nodes[i].Children) > 0 {
			escapeTrailingHashesInNodes(nodes[i].Children)
			return
		}
	}
}

// escapeTrailingHashesEncoder escapes trailing # characters to prevent stripping.
func escapeTrailingHashesEncoder(s string) string {
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

// computeRefLinks walks the AST, counts URL occurrences, and determines which
// URLs should use reference-style links based on BPE token cost comparison.
// Returns a map from URL -> reference label for URLs where reference style is cheaper.
func computeRefLinks(body []Node) map[string]string {
	// Count how many times each URL appears across all link nodes.
	urlCounts := make(map[string]int)
	walkURLs(body, func(url string) {
		urlCounts[url]++
	})

	// For each URL appearing 2+ times, compare token cost of inline vs reference.
	refs := make(map[string]string)
	labelIdx := 1
	for url, count := range urlCounts {
		if count < 2 {
			continue
		}
		if !refLinkSavesTokens(url, count) {
			continue
		}
		refs[url] = fmt.Sprintf("%d", labelIdx)
		labelIdx++
	}
	return refs
}

// walkURLs calls fn for every URL in link and image nodes within the AST.
func walkURLs(nodes []Node, fn func(string)) {
	for _, n := range nodes {
		if (n.Tag == TagLink || n.Tag == TagImage) && n.Attrs.URL != "" {
			fn(n.Attrs.URL)
		}
		walkURLs(n.Children, fn)
	}
}

// refLinkSavesTokens computes whether reference-style links save tokens
// compared to inline links for a given URL appearing `count` times.
//
// Inline cost per occurrence: `](URL)` = tokens("]("+URL+")")
// Reference cost per occurrence: `][ref]` = tokens("]["+ref+"]")
// Reference definition (once): `[ref]: URL\n` = tokens("["+ref+"]: "+URL+"\n")
//
// Total inline:    count * inlineCost
// Total reference: count * refCost + defCost
func refLinkSavesTokens(url string, count int) bool {
	label := "1" // label length doesn't vary much; use representative value
	inlineSuffix := "](" + url + ")"
	refSuffix := "][" + label + "]"
	refDef := "[" + label + "]: " + url + "\n"

	inlineTotal := count * tokenizer.CountTokens(inlineSuffix)
	refTotal := count*tokenizer.CountTokens(refSuffix) + tokenizer.CountTokens(refDef)

	return refTotal < inlineTotal
}

func flattenText(nodes []Node) string {
	var buf strings.Builder
	for _, n := range nodes {
		if n.Tag == TagRaw {
			buf.WriteString(n.Content)
		} else {
			buf.WriteString(flattenText(n.Children))
		}
	}
	return buf.String()
}

// mergeAdjacentRaw merges consecutive TagRaw nodes into one.
func mergeAdjacentRaw(nodes []Node) []Node {
	if len(nodes) <= 1 {
		return nodes
	}
	var merged []Node
	for _, n := range nodes {
		if n.Tag == TagRaw && len(merged) > 0 && merged[len(merged)-1].Tag == TagRaw {
			merged[len(merged)-1].Content += n.Content
		} else {
			merged = append(merged, n)
		}
	}
	return merged
}

// filterInlineChildren returns only inline children (not nested lists).
func filterInlineChildren(item Node) []Node {
	var inline []Node
	for _, child := range item.Children {
		if child.Tag != TagUL && child.Tag != TagOL {
			inline = append(inline, child)
		}
	}
	return inline
}
