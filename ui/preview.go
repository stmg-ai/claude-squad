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

	// text is the raw text being rendered, including ANSI color codes
	text string
}

// AdjustPreviewWidth adjusts the width of the preview pane to be 90% of the provided width.
func AdjustPreviewWidth(width int) int {
	return int(float64(width) * 0.9)
}

func NewPreviewPane(width, maxHeight int) *PreviewPane {
	// Use 70% of the provided width
	adjustedWidth := AdjustPreviewWidth(width)
	return &PreviewPane{width: adjustedWidth, maxHeight: maxHeight}
}

func (p *PreviewPane) SetSize(width, maxHeight int) {
	p.width = AdjustPreviewWidth(width)
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
		return previewPaneStyle.Width(p.width).Render("No content to display")
	}

	// Calculate available height accounting for border and margin
	availableHeight := p.maxHeight - 3 - 4 // 2 for borders, 1 for margin, 1 for ellipsis

	// Split the raw text into lines first, preserving ANSI codes
	lines := strings.Split(p.text, "\n")
	if len(lines) > availableHeight && availableHeight > 0 {
		lines = lines[:availableHeight]
		lines = append(lines, "...")
	}

	// Join lines and wrap in preview pane style while preserving ANSI codes
	content := strings.Join(lines, "\n")
	return previewPaneStyle.Width(p.width).Render(content)
}
