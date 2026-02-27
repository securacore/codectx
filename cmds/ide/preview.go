package ide

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/securacore/codectx/core/ide"
	"github.com/securacore/codectx/ui"
)

// previewModel manages the document preview overlay.
type previewModel struct {
	blocks   []ide.DocumentBlock
	current  int // Index of the currently viewed block
	viewport viewport.Model
	width    int
	height   int
}

// newPreview creates a preview model for the given document blocks.
func newPreview(blocks []ide.DocumentBlock, width, height int) previewModel {
	vp := viewport.New(width, height-4) // Subtract header + help bar
	if len(blocks) > 0 {
		vp.SetContent(renderMarkdown(blocks[0].Content, width-4))
	}
	return previewModel{
		blocks:   blocks,
		current:  0,
		viewport: vp,
		width:    width,
		height:   height,
	}
}

// renderPreview produces the full preview screen.
func (p *previewModel) renderPreview() string {
	var lines []string

	// Header line.
	if len(p.blocks) > 0 {
		path := p.blocks[p.current].Path
		counter := fmt.Sprintf("%d/%d", p.current+1, len(p.blocks))
		header := fmt.Sprintf("  %s  %s    %s",
			ui.BoldStyle.Render("PREVIEW"),
			ui.AccentStyle.Render(path),
			ui.DimStyle.Render(counter),
		)
		lines = append(lines, header)
	} else {
		lines = append(lines, "  "+ui.BoldStyle.Render("PREVIEW")+"  "+ui.DimStyle.Render("(no documents)"))
	}

	// Separator.
	lines = append(lines, ui.DimStyle.Render(strings.Repeat("─", p.width)))

	// Viewport content.
	lines = append(lines, p.viewport.View())

	// Help bar.
	lines = append(lines, ui.DimStyle.Render(strings.Repeat("─", p.width)))
	help := "  " + ui.DimStyle.Render("enter approve . tab next file . r request changes . esc back")
	lines = append(lines, help)

	return strings.Join(lines, "\n")
}

// nextFile cycles to the next document block.
func (p *previewModel) nextFile() {
	if len(p.blocks) <= 1 {
		return
	}
	p.current = (p.current + 1) % len(p.blocks)
	p.viewport.SetContent(renderMarkdown(p.blocks[p.current].Content, p.width-4))
	p.viewport.GotoTop()
}

// resize updates the preview dimensions.
func (p *previewModel) resize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = width
	p.viewport.Height = height - 4
}
