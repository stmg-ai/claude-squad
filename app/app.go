package app

import (
	"claude-squad/keys"
	"claude-squad/ui"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"strings"
)

// Run is the main entrypoint into the application.
func Run() {
	p := tea.NewProgram(newHome(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type state int

const (
	stateDefault state = iota
	// stateNew is the state when the user is creating a new instance.
	stateNew
)

type home struct {
	spinner  spinner.Model
	quitting bool
	err      error

	// ui components
	menu *ui.Menu
	list *ui.List

	// state
	windowWidth  int
	windowHeight int
	state        state
}

func newHome() *home {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &home{
		spinner: s,
		menu:    ui.NewMenu(),
		list:    ui.NewList(),
	}
}

func (m *home) handleWindowSizeEvent(msg tea.WindowSizeMsg) {
	m.windowWidth, m.windowHeight = msg.Width, msg.Height
}

func (m *home) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case ui.ErrMsg:
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:
		m.handleWindowSizeEvent(msg)
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *home) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return m, nil
	}
	switch name {
	case keys.KeyQuit:
		m.quitting = true
		return m, tea.Quit

		// TODO: add more key bindings
	default:
		return m, nil
	}
}

func (m *home) View() string {
	//if m.err != nil {
	//	return m.err.Error()
	//}
	//str := fmt.Sprintf("\n\n   %s Loading forever...press q to quit\n\n", m.spinner.View())
	//if m.quitting {
	//	return str + "\n"
	//}

	// 0.1 means 10% from the bottom

	var s strings.Builder
	//s.WriteString(lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, 0.1, m.list.String()))
	//s.WriteString(lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, 0.1, m.menu.String()))
	//lipgloss.JoinHorizontal()
	lipgloss.JoinVertical()
	s.WriteString(lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, 0.1, fmt.Sprintf("%d %d", m.windowHeight, m.windowWidth)))

	return s.String()
}
