package chunk

import (
	"fmt"
	"strings"
)

// RenderMeta produces the <!-- codectx:meta ... --> HTML comment header
// for a chunk. This header contains all chunk metadata needed for the
// compilation pipeline. Returns an empty string for nil chunks.
func RenderMeta(c *Chunk) string {
	if c == nil {
		return ""
	}
	var b strings.Builder

	b.WriteString("<!-- codectx:meta\n")
	fmt.Fprintf(&b, "id: %s\n", c.ID)
	fmt.Fprintf(&b, "type: %s\n", c.Type)
	fmt.Fprintf(&b, "source: %s\n", c.Source)
	fmt.Fprintf(&b, "heading: %s\n", c.Heading)
	fmt.Fprintf(&b, "chunk: %d of %d\n", c.Sequence, c.TotalInFile)
	fmt.Fprintf(&b, "tokens: %d\n", c.Tokens)

	if c.Oversized {
		b.WriteString("oversized: true\n")
	}

	b.WriteString("-->")

	return b.String()
}

// Render produces the complete chunk file content: meta header followed
// by a blank line and the chunk content. Returns an empty string for nil chunks.
func Render(c *Chunk) string {
	if c == nil {
		return ""
	}
	meta := RenderMeta(c)
	if c.Content == "" {
		return meta + "\n"
	}
	return meta + "\n\n" + c.Content + "\n"
}
