package ui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var previewPaneStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), true, true, true, true).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

type PreviewPane struct {
	width     int
	maxHeight int

	// text is the raw text being rendered.
	text string
}

func NewPreviewPane(width, maxHeight int) *PreviewPane {
	return &PreviewPane{width: width, maxHeight: maxHeight}
}

func (p *PreviewPane) SetSize(width, maxHeight int) {
	p.width, p.maxHeight = width, maxHeight
}

// TODO: should we put a limit here to limit the amount we buffer? Maybe 5k chars?
func (p *PreviewPane) SetText(text string) {
	p.text = text
}

// TODO: render the text preview.
func (p *PreviewPane) String() string {
	//if len(p.text) == 0 {
	//	return strings.Repeat("\n", p.maxHeight)
	//}
	if p.width == 0 || p.maxHeight == 0 {
		return strings.Repeat("\n", p.maxHeight)
	}
	return previewPaneStyle.Render(strings.Repeat(strings.Repeat("a", p.width-1)+"\n", p.maxHeight))
}
