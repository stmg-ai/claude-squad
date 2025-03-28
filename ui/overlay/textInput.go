package overlay

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TextInputOverlay represents a text input overlay with state management
type TextInputOverlay struct {
	Value      string
	Title      string
	Multiline  bool
	FocusIndex int // 0 for text input, 1 for enter button
	Submitted  bool
	Canceled   bool
	OnSubmit   func()
}

// NewTextInputOverlay creates a new text input overlay with the given title and initial value
func NewTextInputOverlay(title string, initialValue string) *TextInputOverlay {
	return &TextInputOverlay{
		Value:      initialValue,
		Title:      title,
		Multiline:  true,
		FocusIndex: 0,
		Submitted:  false,
		Canceled:   false,
	}
}

// HandleKeyPress processes a key press and updates the state accordingly
// Returns true if the overlay should be closed
func (t *TextInputOverlay) HandleKeyPress(key tea.KeyMsg) bool {
	switch key.Type {
	case tea.KeyTab:
		// Toggle focus between input and enter button
		t.FocusIndex = (t.FocusIndex + 1) % 2
		return false
	case tea.KeyShiftTab:
		// Toggle focus in reverse
		t.FocusIndex = (t.FocusIndex + 1) % 2
		return false
	case tea.KeyEnter:
		if t.FocusIndex == 1 {
			// Enter button is focused, submit the form
			t.Submitted = true
			if t.OnSubmit != nil {
				t.OnSubmit()
			}
			return true
		}
		// Input is focused, add a new line
		t.Value += "\n"
		return false
	case tea.KeyEsc:
		t.Canceled = true
		return true
	case tea.KeyRunes:
		// Handle other keys only when input is focused
		if t.FocusIndex == 0 && len(key.Runes) > 0 {
			// Add character to input
			t.Value += string(key.Runes)
		}
		return false
	case tea.KeyBackspace:
		// Handle backspace only when input is focused
		if t.FocusIndex == 0 && len(t.Value) > 0 {
			// Remove last character
			t.Value = t.Value[:len(t.Value)-1]
		}
		return false
	case tea.KeySpace:
		// Handle space only when input is focused
		if t.FocusIndex == 0 {
			t.Value += " "
		}
		return false
	default:
		return false
	}
}

// IsAltEnterPressed checks if Alt+Enter was pressed
func (t *TextInputOverlay) IsAltEnterPressed(key tea.KeyMsg) bool {
	return key.Type == tea.KeyEnter && key.Alt
}

// GetValue returns the current value of the text input
func (t *TextInputOverlay) GetValue() string {
	return t.Value
}

// IsSubmitted returns whether the form was submitted
func (t *TextInputOverlay) IsSubmitted() bool {
	return t.Submitted
}

// IsCanceled returns whether the form was canceled
func (t *TextInputOverlay) IsCanceled() bool {
	return t.Canceled
}

// SetOnSubmit sets a callback function for form submission
func (t *TextInputOverlay) SetOnSubmit(onSubmit func()) {
	t.OnSubmit = onSubmit
}

// Render renders the text input overlay
func (t *TextInputOverlay) Render(height, width int, opts ...WhitespaceOption) string {
	// Style the title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	// Style the input value
	inputStyle := lipgloss.NewStyle()

	// Style for the input box
	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2) // Add padding inside the text input box

	// Style for the enter button
	enterButtonStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2)

	// Highlight the focused element
	if t.FocusIndex == 0 {
		inputBoxStyle = inputBoxStyle.BorderForeground(lipgloss.Color("#7D56F4"))
		enterButtonStyle = enterButtonStyle.BorderForeground(lipgloss.Color("#AAAAAA"))
	} else {
		inputBoxStyle = inputBoxStyle.BorderForeground(lipgloss.Color("#AAAAAA"))
		enterButtonStyle = enterButtonStyle.BorderForeground(lipgloss.Color("#7D56F4"))
	}

	// Create a terminal-style cursor (block cursor)
	cursor := ""
	if t.FocusIndex == 0 {
		cursor = lipgloss.NewStyle().
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Render(" ")
	}

	// Format multiline text with cursor at the end
	formattedText := ""
	if t.Multiline {
		lines := strings.Split(t.Value, "\n")
		for i, line := range lines {
			formattedText += inputStyle.Render(line)
			if i < len(lines)-1 {
				formattedText += "\n"
			}
		}
		if t.FocusIndex == 0 {
			formattedText += cursor
		}
	} else {
		formattedText = inputStyle.Render(t.Value)
		if t.FocusIndex == 0 {
			formattedText += cursor
		}
	}

	// Create input box with border
	inputBox := inputBoxStyle.
		Width(width - 6).
		Height(height - 8). // Leave room for the enter button
		Render(
			titleStyle.Render(t.Title) + "\n" +
				formattedText,
		)

	// Create enter button
	enterButton := enterButtonStyle.
		Render("Enter")

	// Get the actual rendered width of the input box
	inputBoxWidth := lipgloss.Width(inputBox)

	// Position the enter button at the bottom right
	enterButtonWidth := lipgloss.Width(enterButton)
	// Calculate padding to align with the right edge of the input box
	enterButtonPadding := inputBoxWidth - enterButtonWidth

	// Combine input box and enter button
	content := inputBox + "\n" +
		lipgloss.NewStyle().
			PaddingLeft(enterButtonPadding).
			Render(enterButton)

	return PlaceOverlay(0, 0, content, strings.Repeat("\n", height), true, true, opts...)
}
