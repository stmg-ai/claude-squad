package ui

import (
	"claude-squad/keys"
	"strings"

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

var separator = "  •  "

var menuStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205"))

type Menu struct {
	options       []keys.KeyName
	height, width int
}

var StartMenuOptions = []keys.KeyName{keys.KeyNew, keys.KeyKill, keys.KeyEnter, keys.KeyTab, keys.KeyPush, keys.KeyQuit}

func NewMenu() *Menu {
	return &Menu{
		options: StartMenuOptions,
	}
}

func (m *Menu) SetOptions(options []keys.KeyName) {
	m.options = options
}

// SetSize sets the width of the window. The menu will be centered horizontally within this width.
func (m *Menu) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Menu) String() string {
	var s strings.Builder
	for i, k := range m.options {
		binding := keys.GlobalkeyBindings[k]
		s.WriteString(keyStyle.Render(binding.Help().Key))
		s.WriteString(" ")
		s.WriteString(descStyle.Render(binding.Help().Desc))
		if i != len(m.options)-1 {
			s.WriteString(sepStyle.Render(separator))
		}
	}

	centeredMenuText := menuStyle.Render(s.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, centeredMenuText)
}
