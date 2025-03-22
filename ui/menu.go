package ui

import (
	"claude-squad/keys"
	"strings"

	"claude-squad/session"

	"github.com/charmbracelet/lipgloss"
)

var keyStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#655F5F",
	Dark:  "#7F7A7A",
})

var descStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#7A7474",
	Dark:  "#9C9494",
})

var sepStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#DDDADA",
	Dark:  "#3C3C3C",
})

var actionGroupStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

var separator = "  •  "
var verticalSeparator = "  │  "

var menuStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205"))

// MenuState represents different states the menu can be in
type MenuState int

const (
	StateDefault MenuState = iota
	StateNewInstance
)

type Menu struct {
	options       []keys.KeyName
	height, width int
	state         MenuState
	instance      *session.Instance
}

var defaultMenuOptions = []keys.KeyName{keys.KeyNew, keys.KeyQuit}
var newInstanceMenuOptions = []keys.KeyName{keys.KeySubmitName}

func NewMenu() *Menu {
	return &Menu{
		options: defaultMenuOptions,
		state:   StateDefault,
	}
}

// SetState updates the menu state and options accordingly
func (m *Menu) SetState(state MenuState) {
	m.state = state
	m.updateOptions()
}

// SetInstance updates the current instance and refreshes menu options
func (m *Menu) SetInstance(instance *session.Instance) {
	m.instance = instance
	if m.state == StateDefault {
		m.updateOptions()
	}
}

// updateOptions updates the menu options based on current state and instance
func (m *Menu) updateOptions() {
	switch m.state {
	case StateNewInstance:
		m.options = newInstanceMenuOptions
	case StateDefault:
		if m.instance == nil {
			m.options = defaultMenuOptions
			return
		}

		// Instance management group
		options := []keys.KeyName{keys.KeyNew, keys.KeyKill}

		// Action group
		actionGroup := []keys.KeyName{keys.KeyEnter, keys.KeySubmit}
		if m.instance.Status == session.Paused {
			actionGroup = append(actionGroup, keys.KeyResume)
		} else {
			actionGroup = append(actionGroup, keys.KeyPause)
		}

		// System group
		systemGroup := []keys.KeyName{keys.KeyTab, keys.KeyQuit}

		// Combine all groups
		options = append(options, actionGroup...)
		options = append(options, systemGroup...)

		m.options = options
	}
}

// SetSize sets the width of the window. The menu will be centered horizontally within this width.
func (m *Menu) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Menu) String() string {
	var s strings.Builder

	// Define group boundaries
	groups := []struct {
		start int
		end   int
	}{
		{0, 2}, // Instance management group (n, d)
		{2, 5}, // Action group (enter, submit, pause/resume)
		{5, 7}, // System group (tab, q)
	}

	for i, k := range m.options {
		binding := keys.GlobalkeyBindings[k]

		// Check if we're in the action group (middle group)
		inActionGroup := i >= groups[1].start && i < groups[1].end

		if inActionGroup {
			s.WriteString(actionGroupStyle.Render(binding.Help().Key))
			s.WriteString(" ")
			s.WriteString(actionGroupStyle.Render(binding.Help().Desc))
		} else {
			s.WriteString(keyStyle.Render(binding.Help().Key))
			s.WriteString(" ")
			s.WriteString(descStyle.Render(binding.Help().Desc))
		}

		// Add appropriate separator
		if i != len(m.options)-1 {
			isGroupEnd := false
			for _, group := range groups {
				if i == group.end-1 {
					s.WriteString(sepStyle.Render(verticalSeparator))
					isGroupEnd = true
					break
				}
			}
			if !isGroupEnd {
				s.WriteString(sepStyle.Render(separator))
			}
		}
	}

	centeredMenuText := menuStyle.Render(s.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, centeredMenuText)
}
