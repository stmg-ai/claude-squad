package ui

import (
	"claude-squad/keys"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var keyStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#909090",
	Dark:  "#626262",
})

var descStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#B2B2B2",
	Dark:  "#4A4A4A",
})

var sepStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#DDDADA",
	Dark:  "#3C3C3C",
})

var separator = " • "

var menuStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205"))

type Menu struct {
	style   lipgloss.Style
	options []keys.KeyName
}

func NewMenu() *Menu {
	return &Menu{
		style:   menuStyle,
		options: []keys.KeyName{keys.KeyNew, keys.KeyKill, keys.KeyEnter, keys.KeyReview, keys.KeyQuit},
	}
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
	return m.style.Render(s.String())
}
