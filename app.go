package main

import (
	"claude-squad/ui"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
)

// Run is the main entrypoint into the application.
func Run() {
	p := tea.NewProgram(newHome(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type home struct {
	spinner  spinner.Model
	quitting bool
	err      error
	menu     *ui.Menu

	windowWidth  int
	windowHeight int
}

func newHome() *home {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &home{
		spinner: s,
		menu:    ui.NewMenu(),
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
	name, ok := ui.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return m, nil
	}
	switch name {
	case ui.Quit:
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
	//var s strings.Builder
	//

	// 0.1 means 10% from the bottom
	block := lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, 0.1, m.menu.String())

	return block
}
