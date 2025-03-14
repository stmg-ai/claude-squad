package ui

import (
	"claude-squad/session"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var previewPaneStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), true, true, true, true).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
	MarginTop(1)

type PreviewPane struct {
	width     int
	maxHeight int

	// text is the raw text being rendered.
	text string
}

func NewPreviewPane(width, maxHeight int) *PreviewPane {
	// Use 70% of the provided width
	adjustedWidth := int(float64(width) * 0.7)
	return &PreviewPane{width: adjustedWidth, maxHeight: maxHeight}
}

func (p *PreviewPane) SetSize(width, maxHeight int) {
	// Use 70% of the provided width
	p.width = int(float64(width) * 0.7)
	p.maxHeight = maxHeight
}

// TODO: should we put a limit here to limit the amount we buffer? Maybe 5k chars?
func (p *PreviewPane) SetText(text string) {
	p.text = text
}

// Updates the preview pane content with the tmux pane content
func (p *PreviewPane) UpdateContent(instance *session.Instance) error {
	if instance == nil {
		p.text = ""
		return nil
	}

	content, err := instance.Preview()
	if err != nil {
		return err
	}

	p.text = content
	return nil
}

// Returns the preview pane content as a string.
func (p *PreviewPane) String() string {
	if p.width == 0 || p.maxHeight == 0 {
		return strings.Repeat("\n", p.maxHeight)
	}
	if len(p.text) == 0 {
		return previewPaneStyle.Render("No content to display")
	}
	return previewPaneStyle.Render(p.text)
}
