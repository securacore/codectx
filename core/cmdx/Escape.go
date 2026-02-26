package cmdx

import "strings"

// EscapeBody escapes @ and $ for CMDX body text.
// @ -> @@, $ -> $$
func EscapeBody(s string) string {
	s = strings.ReplaceAll(s, "@", "@@")
	s = strings.ReplaceAll(s, "$", "$$")
	return s
}

// UnescapeBody reverses EscapeBody using a single-pass scanner.
// @@ -> @, $$ -> $
func UnescapeBody(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	i := 0
	for i < len(s) {
		if i+1 < len(s) {
			if s[i] == '@' && s[i+1] == '@' {
				buf.WriteByte('@')
				i += 2
				continue
			}
			if s[i] == '$' && s[i+1] == '$' {
				buf.WriteByte('$')
				i += 2
				continue
			}
		}
		buf.WriteByte(s[i])
		i++
	}
	return buf.String()
}

// EscapeDisplay escapes > in LINK/IMG display text.
func EscapeDisplay(s string) string {
	return strings.ReplaceAll(s, ">", "\\>")
}

// UnescapeDisplay reverses EscapeDisplay.
func UnescapeDisplay(s string) string {
	return strings.ReplaceAll(s, "\\>", ">")
}

// EscapeURL escapes \, {, } in LINK/IMG URLs so they don't interfere with
// CMDX brace-block parsing.
func EscapeURL(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	return s
}

// UnescapeURL reverses EscapeURL.
func UnescapeURL(s string) string {
	s = strings.ReplaceAll(s, "\\}", "}")
	s = strings.ReplaceAll(s, "\\{", "{")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

// EscapeMeta escapes ; in META values.
func EscapeMeta(s string) string {
	return strings.ReplaceAll(s, ";", "\\;")
}

// UnescapeMeta reverses EscapeMeta.
func UnescapeMeta(s string) string {
	return strings.ReplaceAll(s, "\\;", ";")
}

// EscapeCell escapes |, {, and } in TABLE cells and RETURNS blocks.
// Braces must be escaped to prevent readBracedBlock from misinterpreting
// cell content as block structure.
func EscapeCell(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	return s
}

// UnescapeCell reverses EscapeCell.
func UnescapeCell(s string) string {
	s = strings.ReplaceAll(s, "\\}", "}")
	s = strings.ReplaceAll(s, "\\{", "{")
	s = strings.ReplaceAll(s, "\\|", "|")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

// EscapeDesc escapes ~, {, and } in KV/PARAMS/DEF descriptions.
// Braces must be escaped to prevent readBracedBlock from misinterpreting
// description content as block structure.
func EscapeDesc(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "~", "\\~")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	return s
}

// UnescapeDesc reverses EscapeDesc.
func UnescapeDesc(s string) string {
	s = strings.ReplaceAll(s, "\\}", "}")
	s = strings.ReplaceAll(s, "\\{", "{")
	s = strings.ReplaceAll(s, "\\~", "~")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

// EscapeKVField escapes \, {, }, :, and ~ in KV/PARAMS key and type fields.
func EscapeKVField(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	s = strings.ReplaceAll(s, ":", "\\:")
	s = strings.ReplaceAll(s, "~", "\\~")
	return s
}

// UnescapeKVField reverses EscapeKVField.
func UnescapeKVField(s string) string {
	s = strings.ReplaceAll(s, "\\~", "~")
	s = strings.ReplaceAll(s, "\\:", ":")
	s = strings.ReplaceAll(s, "\\}", "}")
	s = strings.ReplaceAll(s, "\\{", "{")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

// EscapeCodeLine escapes @ at the start of a line inside a code block.
// Per D3: prefix with \ so @/CODE in source code is not mistaken for a block terminator.
func EscapeCodeLine(line string) string {
	if strings.HasPrefix(line, "@") || strings.HasPrefix(line, "\\") {
		return "\\" + line
	}
	return line
}

// UnescapeCodeLine reverses EscapeCodeLine.
func UnescapeCodeLine(line string) string {
	if strings.HasPrefix(line, "\\@") || strings.HasPrefix(line, "\\\\") {
		return line[1:]
	}
	return line
}
