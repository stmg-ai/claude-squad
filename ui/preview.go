package ui

import (
	"claude-squad/session"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var fallBackText = lipgloss.JoinVertical(lipgloss.Center, `
░█████╗░██╗░░░░░░█████╗░██╗░░░██╗██████╗░███████╗
██╔══██╗██║░░░░░██╔══██╗██║░░░██║██╔══██╗██╔════╝
██║░░╚═╝██║░░░░░███████║██║░░░██║██║░░██║█████╗░░
██║░░██╗██║░░░░░██╔══██║██║░░░██║██║░░██║██╔══╝░░
╚█████╔╝███████╗██║░░██║╚██████╔╝██████╔╝███████╗
░╚════╝░╚══════╝╚═╝░░╚═╝░╚═════╝░╚═════╝░╚══════╝
`, `
░██████╗░██████╗░██╗░░░██╗░█████╗░██████╗░
██╔════╝██╔═══██╗██║░░░██║██╔══██╗██╔══██╗
╚█████╗░██║██╗██║██║░░░██║███████║██║░░██║
░╚═══██╗╚██████╔╝██║░░░██║██╔══██║██║░░██║
██████╔╝░╚═██╔═╝░╚██████╔╝██║░░██║██████╔╝
`)

var previewPaneStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

type PreviewPane struct {
	width  int
	height int

	previewState previewState
}

type previewState struct {
	// fallback is true if the preview pane is displaying fallback text
	fallback bool
	// text is the text displayed in the preview pane
	text string
}

func NewPreviewPane() *PreviewPane {
	return &PreviewPane{}
}

func (p *PreviewPane) SetSize(width, maxHeight int) {
	p.width = width
	p.height = maxHeight
}

// setFallbackState sets the preview state with fallback text and a message
func (p *PreviewPane) setFallbackState(message string) {
	p.previewState = previewState{
		fallback: true,
		text:     lipgloss.JoinVertical(lipgloss.Center, fallBackText, "", message),
	}
}

// Updates the preview pane content with the tmux pane content
func (p *PreviewPane) UpdateContent(instance *session.Instance) error {
	switch {
	case instance == nil:
		p.setFallbackState("No agents running yet. Spin up a new instance with 'n' to get started!")
		return nil
	case instance.Status == session.Paused:
		p.setFallbackState("Session is paused. Press 'p' to resume.")
		return nil
	}

	content, err := instance.Preview()
	if err != nil {
		return err
	}

	if len(content) == 0 {
		p.setFallbackState("No agents running yet. Spin up a new instance with 'n' to get started!")
		return nil
	}

	p.previewState = previewState{
		fallback: false,
		text:     content,
	}
	return nil
}

// Returns the preview pane content as a string.
func (p *PreviewPane) String() string {
	if p.width == 0 || p.height == 0 {
		return strings.Repeat("\n", p.height)
	}

	if p.previewState.fallback {
		// Calculate available height for fallback text
		availableHeight := p.height - 3 - 4 // 2 for borders, 1 for margin, 1 for padding
		
		// Count the number of lines in the fallback text
		fallbackLines := len(strings.Split(p.previewState.text, "\n"))
		
		// Calculate padding needed above and below to center the content
		totalPadding := availableHeight - fallbackLines
		topPadding := totalPadding / 2
		bottomPadding := totalPadding - topPadding // accounts for odd numbers
		
		// Build the centered content
		var lines []string
		lines = append(lines, strings.Repeat("\n", topPadding))
		lines = append(lines, p.previewState.text)
		if bottomPadding > 0 {
			lines = append(lines, strings.Repeat("\n", bottomPadding))
		}
		
		// Center both vertically and horizontally
		return previewPaneStyle.
			Width(p.width).
			Align(lipgloss.Center).
			Render(strings.Join(lines, ""))
	}

	// Calculate available height accounting for border and margin
	availableHeight := p.height - 3 - 4 // 2 for borders, 1 for margin, 1 for ellipsis

	lines := strings.Split(p.previewState.text, "\n")

	// Truncate if we have more lines than available height
	if availableHeight > 0 {
		if len(lines) > availableHeight {
			lines = lines[:availableHeight]
			lines = append(lines, "...")
		} else {
			// Pad with empty lines to fill available height
			padding := availableHeight - len(lines)
			lines = append(lines, make([]string, padding)...)
		}
	}

	content := strings.Join(lines, "\n")
	return previewPaneStyle.Width(p.width).Render(content)
}
