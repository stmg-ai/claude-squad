package app

import (
	"claude-squad/keys"
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
	list    *ui.List
	preview *ui.PreviewPane
	menu    *ui.Menu

	// input
	inputDisabled bool

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
		preview: ui.NewPreviewPane(0, 0),
	}
}

// updateHandleWindowSizeEvent is really important since it sets the sizes of the components.
// The components will try to render inside their bounds.
func (m *home) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
	m.windowWidth, m.windowHeight = msg.Width, msg.Height

	m.preview.SetSize(int(float32(msg.Width)*0.7), int(float32(msg.Height)*0.8))
	m.list.SetSize(int(float32(msg.Width)*0.3), int(float32(msg.Height)*0.8))
	m.menu.SetSize(msg.Width, int(float32(msg.Height)*0.1))
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
		m.updateHandleWindowSizeEvent(msg)
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *home) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.inputDisabled {
		return m, nil
	}
	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return m, nil
	}
	switch name {
	case keys.KeyQuit:
		m.quitting = true
		return m, tea.Quit
	case keys.KeyDown:
		m.list.Down()
		return m, nil
	case keys.KeyUp:
		m.list.Up()
		return m, nil
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
	//

	// 0.1 means 10% from the bottom

	//var s strings.Builder
	//s.WriteString(lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, 0.1, m.list.String()))
	//s.WriteString(lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, 0.1, m.menu.String()))
	//lipgloss.JoinHorizontal()
	listAndPreview := lipgloss.JoinHorizontal(lipgloss.Top, m.list.String(), m.preview.String())
	menu := m.menu.String()

	return lipgloss.JoinVertical(lipgloss.Center, listAndPreview, menu)

	//return m.header.String() + listAndPreview + menu
	//s.WriteString(lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, 0.1, fmt.Sprintf("%d %d", m.windowHeight, m.windowWidth)))

	//return s.String()
}
