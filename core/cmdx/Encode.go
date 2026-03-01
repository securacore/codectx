package cmdx

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Encode converts Markdown to CMDX format.
func Encode(markdown []byte, opts ...EncoderOptions) ([]byte, error) {
	opt := DefaultEncoderOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Pass 1: Parse markdown -> goldmark AST -> internal AST
	body, err := markdownToAST(markdown)
	if err != nil {
		return nil, fmt.Errorf("encode pass 1: %w", err)
	}

	doc := &Document{
		Version: "1",
		Body:    body,
	}

	// Pass 2: Domain pattern detection (Phase 3)
	if opt.EnableDomainBlocks {
		detectDomainPatterns(doc)
	}

	// D6: Escape @→@@ and $→$$ in source text FIRST.
	escapeAllText(doc.Body)

	// Pass 3: Dictionary compression on already-escaped text.
	if opt.MaxDictEntries > 0 {
		segments := collectTextSegments(doc.Body)
		dict := buildDictionary(segments, opt)
		if dict != nil && len(dict.Entries) > 0 {
			doc.Dict = dict
			applyDictionary(doc.Body, dict)
		}
	}

	// Pass 4: Serialize to CMDX (text already escaped + referenced).
	return serializeCMDX(doc), nil
}

// newGoldmark returns a consistently-configured goldmark instance (D8).
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
	// goldmark may include trailing spaces in text segments before a hard break,
	// but these spaces are syntactic (part of the hard break marker), not semantic.
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
// A list item can contain paragraphs, nested lists, etc.
func convertListItem(li ast.Node, source []byte) Node {
	var children []Node
	for child := li.FirstChild(); child != nil; child = child.NextSibling() {
		switch child := child.(type) {
		case *ast.Paragraph, *ast.TextBlock:
			// Paragraph (loose lists) or TextBlock (tight lists).
			inline := convertInlineChildren(child, source)
			children = append(children, inline...)
		case *ast.Heading:
			// Headings inside list items: emit heading markers as literal text
			// so that when decoded, goldmark re-parses them as headings.
			marker := strings.Repeat("#", child.Level)
			inline := convertInlineChildren(child, source)
			if len(inline) > 0 {
				// Escape trailing # in the last raw node to prevent ATX closing sequence stripping.
				escapeTrailingHashesInNodes(inline)
				// Prepend "## " before the first inline child.
				children = append(children, Node{Tag: TagRaw, Content: marker + " "})
				children = append(children, inline...)
			} else {
				// Empty heading, e.g., "* #"
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
func convertTable(table *east.Table, source []byte) Node {
	var headers []string
	var rows [][]string

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.(type) {
		case *east.TableHeader:
			// TableHeader directly contains TableCell children.
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					headers = append(headers, extractTextContent(cell, source))
				}
			}
		case *east.TableRow:
			// Body rows are direct children of Table (no TableBody wrapper).
			var cells []string
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if _, ok := cell.(*east.TableCell); ok {
					cells = append(cells, extractTextContent(cell, source))
				}
			}
			rows = append(rows, cells)
		}
	}

	return Node{
		Tag: TagTable,
		Attrs: NodeAttrs{
			Headers: headers,
			Cells:   rows,
		},
	}
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

// flattenText extracts plain text from a tree of inline nodes.
// isAllRawText returns true if all nodes in the list are TagRaw (plain text).
func isAllRawText(nodes []Node) bool {
	for _, n := range nodes {
		if n.Tag != TagRaw {
			return false
		}
	}
	return true
}

// escapeTrailingHashesInNodes escapes trailing # in the last TagRaw node of an
// inline node list, preventing ATX heading closing sequence stripping.
func escapeTrailingHashesInNodes(nodes []Node) {
	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].Tag == TagRaw {
			nodes[i].Content = escapeTrailingHashesEncoder(nodes[i].Content)
			return
		}
		// If non-raw node has children, recurse.
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
		// Don't double-escape: if the char before # is already \, skip.
		if i >= 0 && s[i] == '\\' {
			return s
		}
		s = s[:j] + "\\" + s[j:]
	}
	return s
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

// --- Pass 2: Domain pattern detection ---

// endpointPattern matches H3 headings with HTTP method + exactly one space + path.
// Requires exactly one space for lossless round-tripping (decoder outputs single space).
var endpointPattern = regexp.MustCompile(`^(GET|POST|PUT|DELETE|PATCH) /`)

// detectDomainPatterns transforms matching AST nodes into domain-specific blocks.
// Runs before escaping and dictionary building (Pass 2).
func detectDomainPatterns(doc *Document) {
	doc.Body = detectDomainInNodes(doc.Body)
}

func detectDomainInNodes(nodes []Node) []Node {
	result := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		// Recurse into children first (e.g., blockquote children).
		if node.Tag == TagBQ {
			node.Children = detectDomainInNodes(node.Children)
		}

		converted := detectSingleNode(node)
		result = append(result, converted)
	}
	return result
}

func detectSingleNode(node Node) Node {
	switch node.Tag {
	case TagTable:
		return detectTablePattern(node)
	case TagH3:
		return detectEndpointPattern(node)
	case TagBQ:
		return detectAdmonitionPattern(node)
	default:
		return node
	}
}

// --- Table pattern detection ---

// nameColumns are the allowed "name-like" header values (case-insensitive).
var nameColumns = map[string]bool{
	"field":     true,
	"key":       true,
	"name":      true,
	"parameter": true,
}

// kvDecodedHeaders are the headers that the decoder produces for KV blocks.
var kvDecodedHeaders = map[string]string{
	"field": "Field", "key": "Field", "name": "Field", "parameter": "Field",
}

// paramsDecodedHeaders maps name-like headers to the decoder's output for PARAMS.
var paramsDecodedHeaders = map[string]string{
	"field": "Name", "key": "Name", "name": "Name", "parameter": "Name",
}

func detectTablePattern(node Node) Node {
	headers := node.Attrs.Headers
	if len(headers) == 0 {
		return node
	}

	// Build lowercase header set and find column indices.
	lowerHeaders := make([]string, len(headers))
	for i, h := range headers {
		lowerHeaders[i] = strings.ToLower(strings.TrimSpace(h))
	}

	nameIdx := -1
	typeIdx := -1
	descIdx := -1
	reqIdx := -1

	for i, h := range lowerHeaders {
		if nameColumns[h] {
			nameIdx = i
		} else if h == "type" {
			typeIdx = i
		} else if h == "description" {
			descIdx = i
		} else if h == "required" {
			reqIdx = i
		}
	}

	// Verify that the round-trip will be lossless: check that original headers
	// match the canonical forms the decoder will produce. KV decodes to
	// "Field/Type/Description", PARAMS to "Name/Type/Required/Description".
	if nameIdx >= 0 && typeIdx >= 0 && reqIdx >= 0 && descIdx >= 0 {
		// PARAMS candidate — decoder will output "Name/Type/Required/Description".
		expectedName := paramsDecodedHeaders[lowerHeaders[nameIdx]]
		if strings.TrimSpace(headers[nameIdx]) != expectedName ||
			strings.TrimSpace(headers[typeIdx]) != "Type" ||
			strings.TrimSpace(headers[reqIdx]) != "Required" ||
			strings.TrimSpace(headers[descIdx]) != "Description" {
			return node // Headers don't match canonical output — keep as TABLE
		}
	} else if nameIdx >= 0 && typeIdx >= 0 && descIdx >= 0 && reqIdx < 0 {
		// KV candidate — decoder will output "Field/Type/Description".
		expectedName := kvDecodedHeaders[lowerHeaders[nameIdx]]
		if strings.TrimSpace(headers[nameIdx]) != expectedName ||
			strings.TrimSpace(headers[typeIdx]) != "Type" ||
			strings.TrimSpace(headers[descIdx]) != "Description" {
			return node // Headers don't match canonical output — keep as TABLE
		}
	}

	// PARAMS detection: name + type + required + description, exactly 4 columns.
	if nameIdx >= 0 && typeIdx >= 0 && reqIdx >= 0 && descIdx >= 0 && len(headers) == 4 {
		return tableToParams(node, nameIdx, typeIdx, reqIdx, descIdx)
	}

	// KV detection: name + type + description (NOT required), exactly 3 columns.
	if nameIdx >= 0 && typeIdx >= 0 && descIdx >= 0 && reqIdx < 0 && len(headers) == 3 {
		return tableToKV(node, nameIdx, typeIdx, descIdx)
	}

	// No match — keep as @TABLE{}.
	return node
}

func tableToKV(node Node, nameIdx, typeIdx, descIdx int) Node {
	var items []KVItem
	for _, row := range node.Attrs.Cells {
		items = append(items, KVItem{
			Key:         cellAt(row, nameIdx),
			Type:        cellAt(row, typeIdx),
			Description: cellAt(row, descIdx),
		})
	}
	return Node{Tag: TagKV, Attrs: NodeAttrs{Items: items}}
}

func tableToParams(node Node, nameIdx, typeIdx, reqIdx, descIdx int) Node {
	// Verify all Required column values are valid before converting.
	// Invalid or empty values mean this isn't a real PARAMS table.
	for _, row := range node.Attrs.Cells {
		val := strings.ToLower(strings.TrimSpace(cellAt(row, reqIdx)))
		if !isValidRequired(val) {
			return node // Keep as @TABLE{}.
		}
	}
	var params []ParamItem
	for _, row := range node.Attrs.Cells {
		params = append(params, ParamItem{
			Name:        cellAt(row, nameIdx),
			Type:        cellAt(row, typeIdx),
			Required:    isRequired(cellAt(row, reqIdx)),
			Description: cellAt(row, descIdx),
		})
	}
	return Node{Tag: TagParams, Attrs: NodeAttrs{Params: params}}
}

// cellAt safely gets a cell value by index.
func cellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

// isValidRequired checks if a Required column value can be losslessly
// round-tripped through PARAMS encoding. Only "yes" and "no" (case-insensitive)
// are accepted because the decoder always outputs "Yes"/"No".
func isValidRequired(val string) bool {
	switch val {
	case "yes", "no":
		return true
	default:
		return false
	}
}

// isRequired maps required column values to boolean.
func isRequired(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "yes", "true", "y", "1":
		return true
	default:
		return false
	}
}

// --- Endpoint detection ---

func detectEndpointPattern(node Node) Node {
	// Only H3 triggers detection, and only if the heading is purely text
	// (no inline formatting). Inline formatting would be lost in the conversion.
	if !isAllRawText(node.Children) {
		return node
	}
	text := flattenText(node.Children)
	if endpointPattern.MatchString(text) {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) == 2 {
			return Node{
				Tag: TagEndpoint,
				Attrs: NodeAttrs{
					Endpoint: &EndpointDef{
						Method: parts[0],
						Path:   strings.TrimSpace(parts[1]),
					},
				},
			}
		}
	}
	return node
}

// --- Admonition detection ---

// admonitionBoldPrefixes maps bold-prefix text to tag types.
// Keys use the exact case the decoder outputs for lossless round-tripping.
var admonitionBoldPrefixes = map[string]TagType{
	"Note:":      TagNote,
	"Warning:":   TagWarn,
	"Tip:":       TagTip,
	"Important:": TagImportant,
}

// admonitionGHPrefixes maps GitHub-style callout markers to tag types.
// Note: GH-style detection inherently changes format (decoded as bold-prefix).
// We still accept these case-insensitively since the output format differs anyway.
var admonitionGHPrefixes = map[string]TagType{
	"[!note]":      TagNote,
	"[!warning]":   TagWarn,
	"[!tip]":       TagTip,
	"[!important]": TagImportant,
}

func detectAdmonitionPattern(node Node) Node {
	if len(node.Children) == 0 {
		return node
	}

	firstChild := node.Children[0]
	if firstChild.Tag != TagP || len(firstChild.Children) == 0 {
		return node
	}

	// Check for bold-prefix style: first child is Bold, containing "Note:" etc.
	// The text after the bold must start with a space (or be empty on this line)
	// to be a proper admonition. E.g., "**Note:** text" but NOT "**Note:**text".
	if len(firstChild.Children) >= 1 && firstChild.Children[0].Tag == TagBold {
		boldNode := firstChild.Children[0]
		boldText := strings.TrimSpace(flattenText(boldNode.Children))

		// Check that the content after bold has proper spacing:
		// must start with exactly one space (the decoder outputs "> **Note:** text").
		// Multiple spaces or no space means it won't round-trip losslessly.
		hasProperSpacing := true
		if len(firstChild.Children) > 1 {
			next := firstChild.Children[1]
			if next.Tag == TagRaw {
				if len(next.Content) == 0 || next.Content[0] != ' ' {
					hasProperSpacing = false
				} else if len(next.Content) > 1 && next.Content[1] == ' ' {
					// Multiple leading spaces — would be collapsed to one.
					hasProperSpacing = false
				}
			}
		}

		if hasProperSpacing && isAdmonitionContentPlainText(firstChild.Children[1:], node.Children[1:]) {
			for prefix, tag := range admonitionBoldPrefixes {
				if boldText == prefix {
					// Extract the remaining text after the bold prefix.
					remaining := extractAdmonitionContent(firstChild.Children[1:], node.Children[1:])
					return Node{Tag: tag, Attrs: NodeAttrs{Callout: remaining}}
				}
			}
		}
	}

	// Check for GitHub-style: first raw text starts with [!NOTE] etc.
	if firstChild.Children[0].Tag == TagRaw {
		rawText := firstChild.Children[0].Content
		lowerRaw := strings.ToLower(strings.TrimSpace(rawText))

		for prefix, tag := range admonitionGHPrefixes {
			if strings.HasPrefix(lowerRaw, prefix) {
				// Extract text after the marker.
				afterMarker := strings.TrimSpace(rawText[len(prefix):])
				remaining := extractAdmonitionContent(nil, nil)
				if afterMarker != "" {
					remaining = afterMarker
				}
				// Also include remaining inline children and subsequent paragraphs.
				if len(firstChild.Children) > 1 {
					extraInline := flattenText(firstChild.Children[1:])
					remaining += extraInline
				}
				return Node{Tag: tag, Attrs: NodeAttrs{Callout: strings.TrimSpace(remaining)}}
			}
		}
	}

	return node
}

// extractAdmonitionContent collects the text content after an admonition prefix.
// isAdmonitionContentPlainText checks that all remaining content after the bold
// prefix is plain text (no inline formatting that would be lost in conversion).
func isAdmonitionContentPlainText(remainingInline []Node, remainingBlocks []Node) bool {
	if !isAllRawText(remainingInline) {
		return false
	}
	for _, block := range remainingBlocks {
		if block.Tag == TagP {
			if !isAllRawText(block.Children) {
				return false
			}
		}
	}
	return true
}

func extractAdmonitionContent(remainingInline []Node, remainingBlocks []Node) string {
	var parts []string

	// Inline content after the bold prefix (e.g., " This action is irreversible...")
	if len(remainingInline) > 0 {
		text := flattenText(remainingInline)
		text = strings.TrimSpace(text)
		if text != "" {
			parts = append(parts, text)
		}
	}

	// Additional paragraph blocks in the blockquote.
	for _, block := range remainingBlocks {
		if block.Tag == TagP {
			text := flattenText(block.Children)
			text = strings.TrimSpace(text)
			if text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.Join(parts, " ")
}

// --- Pass 4: Serialize internal AST to CMDX text ---

func serializeCMDX(doc *Document) []byte {
	var buf bytes.Buffer

	buf.WriteString("@CMDX v1\n")

	// Dictionary (Phase 2)
	if doc.Dict != nil && len(doc.Dict.Entries) > 0 {
		buf.WriteString("@DICT{\n")
		for _, e := range doc.Dict.Entries {
			fmt.Fprintf(&buf, "  $%d=%s\n", e.Index, e.Value)
		}
		buf.WriteString("}\n")
	}

	// Metadata (Phase 3)
	if len(doc.Meta) > 0 {
		buf.WriteString("@META{")
		first := true
		for k, v := range doc.Meta {
			if !first {
				buf.WriteByte(';')
			}
			buf.WriteString(k)
			buf.WriteByte(':')
			buf.WriteString(EscapeMeta(v))
			first = false
		}
		buf.WriteString("}\n")
	}

	// Body
	for i, node := range doc.Body {
		if i > 0 {
			buf.WriteByte('\n')
		}
		serializeNode(&buf, node, 0)
	}

	return buf.Bytes()
}

// serializeNode writes a single node in CMDX format.
func serializeNode(buf *bytes.Buffer, node Node, indent int) {
	prefix := strings.Repeat("  ", indent)

	switch node.Tag {
	case TagH1, TagH2, TagH3, TagH4, TagH5, TagH6:
		level := int(node.Tag-TagH1) + 1
		fmt.Fprintf(buf, "%s@H%d %s\n", prefix, level, serializeInlineCMDX(node.Children))

	case TagHR:
		fmt.Fprintf(buf, "%s@HR\n", prefix)

	case TagBR:
		fmt.Fprintf(buf, "%s@BR\n", prefix)

	case TagP:
		inline := serializeInlineCMDX(node.Children)
		fmt.Fprintf(buf, "%s@P %s\n", prefix, inline)

	case TagBQ:
		if len(node.Children) == 1 && node.Children[0].Tag == TagP {
			inline := serializeInlineCMDX(node.Children[0].Children)
			fmt.Fprintf(buf, "%s@BQ %s\n", prefix, inline)
		} else {
			fmt.Fprintf(buf, "%s@BQ{\n", prefix)
			for _, child := range node.Children {
				serializeNode(buf, child, indent)
			}
			fmt.Fprintf(buf, "%s}\n", prefix)
		}

	case TagCodeBlock:
		lang := node.Attrs.Language
		if lang != "" {
			// Escape \, {, } in language to prevent interference with CMDX blocks.
			escapedLang := strings.ReplaceAll(lang, "\\", "\\\\")
			escapedLang = strings.ReplaceAll(escapedLang, "{", "\\{")
			escapedLang = strings.ReplaceAll(escapedLang, "}", "\\}")
			fmt.Fprintf(buf, "%s@CODE:%s\n", prefix, escapedLang)
		} else {
			fmt.Fprintf(buf, "%s@CODE\n", prefix)
		}
		lines := strings.Split(node.Content, "\n")
		for _, line := range lines {
			buf.WriteString(prefix)
			buf.WriteString(EscapeCodeLine(line))
			buf.WriteByte('\n')
		}
		fmt.Fprintf(buf, "%s@/CODE\n", prefix)

	case TagUL:
		fmt.Fprintf(buf, "%s@UL{\n", prefix)
		for _, item := range node.Children {
			inline := serializeInlineCMDX(filterInlineChildren(item))
			fmt.Fprintf(buf, "%s- %s\n", prefix, inline)
			// Serialize nested lists.
			for _, child := range item.Children {
				if child.Tag == TagUL || child.Tag == TagOL {
					serializeNode(buf, child, indent+1)
				}
			}
		}
		fmt.Fprintf(buf, "%s}\n", prefix)

	case TagOL:
		fmt.Fprintf(buf, "%s@OL{\n", prefix)
		for i, item := range node.Children {
			inline := serializeInlineCMDX(filterInlineChildren(item))
			fmt.Fprintf(buf, "%s%d. %s\n", prefix, i+1, inline)
			for _, child := range item.Children {
				if child.Tag == TagUL || child.Tag == TagOL {
					serializeNode(buf, child, indent+1)
				}
			}
		}
		fmt.Fprintf(buf, "%s}\n", prefix)

	case TagTable:
		fmt.Fprintf(buf, "%s@TABLE{\n", prefix)
		if len(node.Attrs.Headers) > 0 {
			fmt.Fprintf(buf, "%s@THEAD{%s}\n", prefix, strings.Join(escapeAllCells(node.Attrs.Headers), "|"))
		}
		for _, row := range node.Attrs.Cells {
			fmt.Fprintf(buf, "%s%s\n", prefix, strings.Join(escapeAllCells(row), "|"))
		}
		fmt.Fprintf(buf, "%s}\n", prefix)

	case TagKV:
		fmt.Fprintf(buf, "%s@KV{\n", prefix)
		for _, item := range node.Attrs.Items {
			fmt.Fprintf(buf, "%s  %s:%s~%s\n", prefix, EscapeKVField(item.Key), EscapeKVField(item.Type), EscapeDesc(item.Description))
		}
		fmt.Fprintf(buf, "%s}\n", prefix)

	case TagParams:
		fmt.Fprintf(buf, "%s@PARAMS{\n", prefix)
		for _, p := range node.Attrs.Params {
			req := "O"
			if p.Required {
				req = "R"
			}
			fmt.Fprintf(buf, "%s  %s:%s:%s~%s\n", prefix, EscapeKVField(p.Name), EscapeKVField(p.Type), req, EscapeDesc(p.Description))
		}
		fmt.Fprintf(buf, "%s}\n", prefix)

	case TagEndpoint:
		if node.Attrs.Endpoint != nil {
			fmt.Fprintf(buf, "%s@ENDPOINT{%s %s}\n", prefix, node.Attrs.Endpoint.Method, node.Attrs.Endpoint.Path)
		}

	case TagReturns:
		if len(node.Attrs.Returns) > 0 {
			var parts []string
			for _, r := range node.Attrs.Returns {
				parts = append(parts, r.Status+":"+EscapeCell(r.Description))
			}
			fmt.Fprintf(buf, "%s@RETURNS{%s}\n", prefix, strings.Join(parts, "|"))
		}

	case TagDef:
		fmt.Fprintf(buf, "%s@DEF{\n", prefix)
		for _, item := range node.Attrs.Items {
			fmt.Fprintf(buf, "%s  %s~%s\n", prefix, EscapeKVField(item.Key), EscapeDesc(item.Description))
		}
		fmt.Fprintf(buf, "%s}\n", prefix)

	case TagNote:
		fmt.Fprintf(buf, "%s@NOTE{%s}\n", prefix, node.Attrs.Callout)

	case TagWarn:
		fmt.Fprintf(buf, "%s@WARN{%s}\n", prefix, node.Attrs.Callout)

	case TagTip:
		fmt.Fprintf(buf, "%s@TIP{%s}\n", prefix, node.Attrs.Callout)

	case TagImportant:
		fmt.Fprintf(buf, "%s@IMPORTANT{%s}\n", prefix, node.Attrs.Callout)

	case TagRaw:
		if node.Content != "" {
			buf.WriteString(prefix)
			buf.WriteString(node.Content) // Already escaped.
			buf.WriteByte('\n')
		}
	}
}

// serializeInlineCMDX renders inline nodes as a CMDX string.
// Text is already escaped (D6 escaping applied before serialization).
func serializeInlineCMDX(nodes []Node) string {
	var buf strings.Builder
	for _, n := range nodes {
		switch n.Tag {
		case TagRaw:
			// D6: @ and $ are already escaped. Also escape { and } to prevent
			// interference with braced block delimiters.
			s := strings.ReplaceAll(n.Content, "\\", "\\\\")
			s = strings.ReplaceAll(s, "{", "\\{")
			s = strings.ReplaceAll(s, "}", "\\}")
			buf.WriteString(s)
		case TagBold:
			buf.WriteString("@B{")
			buf.WriteString(serializeInlineCMDX(n.Children))
			buf.WriteByte('}')
		case TagItalic:
			buf.WriteString("@I{")
			buf.WriteString(serializeInlineCMDX(n.Children))
			buf.WriteByte('}')
		case TagBoldItalic:
			buf.WriteString("@BI{")
			buf.WriteString(serializeInlineCMDX(n.Children))
			buf.WriteByte('}')
		case TagCode:
			buf.WriteString("@C{")
			// Replace newlines with spaces — CMDX inline tags are single-line,
			// and goldmark normalizes newlines in code spans to spaces anyway.
			// Escape } and { in content to prevent brace matching issues.
			codeContent := strings.ReplaceAll(n.Content, "\n", " ")
			codeContent = strings.ReplaceAll(codeContent, "\\", "\\\\")
			codeContent = strings.ReplaceAll(codeContent, "{", "\\{")
			codeContent = strings.ReplaceAll(codeContent, "}", "\\}")
			buf.WriteString(codeContent)
			buf.WriteByte('}')
		case TagStrikethrough:
			buf.WriteString("@S{")
			buf.WriteString(serializeInlineCMDX(n.Children))
			buf.WriteByte('}')
		case TagLink:
			buf.WriteString("@LINK{")
			display := serializeInlineCMDX(n.Children)
			buf.WriteString(EscapeDisplay(display))
			buf.WriteByte('>')
			buf.WriteString(EscapeURL(n.Attrs.URL))
			buf.WriteByte('}')
		case TagImage:
			buf.WriteString("@IMG{")
			display := serializeInlineCMDX(n.Children)
			buf.WriteString(EscapeDisplay(display))
			buf.WriteByte('>')
			buf.WriteString(EscapeURL(n.Attrs.URL))
			buf.WriteByte('}')
		case TagBR:
			buf.WriteString("@BR{}")
		}
	}
	return buf.String()
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

// escapeAllCells escapes pipe characters in all cells.
func escapeAllCells(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = EscapeCell(c)
	}
	return out
}
