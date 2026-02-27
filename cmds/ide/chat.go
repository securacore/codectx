package ide

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/securacore/codectx/ui"
)

// renderMarkdown renders markdown content for terminal display using glamour.
// Falls back to plain text if rendering fails.
func renderMarkdown(content string, width int) string {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimRight(out, "\n")
}

// chatMessage represents a single message in the chat viewport.
type chatMessage struct {
	role    string // "user", "assistant", "tool", "error"
	content string
}

// renderChat formats the chat message history into a single string for
// the viewport. Each message is styled based on its role.
func renderChat(messages []chatMessage, streamingContent string, width int) string {
	var b strings.Builder

	for _, msg := range messages {
		renderMessage(&b, msg, width)
		b.WriteString("\n")
	}

	// If there is content currently streaming, show it as an incomplete
	// assistant message with a cursor indicator.
	if streamingContent != "" {
		b.WriteString("  ")
		b.WriteString(ui.DimStyle.Render(ui.SymbolActive))
		b.WriteString(" ")
		b.WriteString(streamingContent)
		b.WriteString(ui.DimStyle.Render(" ..."))
		b.WriteString("\n")
	}

	return b.String()
}

// renderMessage formats a single chat message.
func renderMessage(b *strings.Builder, msg chatMessage, width int) {
	switch msg.role {
	case "user":
		b.WriteString("  ")
		b.WriteString(ui.AccentStyle.Render(string(ui.SymbolBullet)))
		b.WriteString(" ")
		b.WriteString(ui.AccentStyle.Render(msg.content))

	case "assistant":
		b.WriteString("  ")
		b.WriteString(ui.GreenStyle.Render(ui.SymbolActive))
		b.WriteString(" ")
		rendered := renderMarkdown(msg.content, width-4)
		// Indent rendered markdown to align with the symbol prefix.
		for i, line := range strings.Split(rendered, "\n") {
			if i > 0 {
				b.WriteString("\n    ")
			}
			b.WriteString(line)
		}

	case "tool":
		b.WriteString("    ")
		b.WriteString(ui.DimStyle.Render(fmt.Sprintf("[%s]", msg.content)))

	case "error":
		b.WriteString("  ")
		b.WriteString(ui.RedStyle.Render(ui.SymbolFail))
		b.WriteString(" ")
		b.WriteString(ui.RedStyle.Render(msg.content))
	}
}
