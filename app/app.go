package app

import (
	"claude-squad/keys"
	"claude-squad/ui"
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"time"
)

const GlobalInstanceLimit = 10

// Run is the main entrypoint into the application.
func Run(ctx context.Context) {
	p := tea.NewProgram(newHome(ctx), tea.WithAltScreen())
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
	ctx context.Context

	quitting bool

	// ui components
	list    *ui.List
	preview *ui.PreviewPane
	menu    *ui.Menu
	errBox  *ui.ErrBox
	// global spinner instance. we plumb this down to where it's needed
	spinner spinner.Model

	// input
	inputDisabled bool

	// state
	windowWidth  int
	windowHeight int
	state        state
}

func newHome(ctx context.Context) *home {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	h := &home{
		ctx:     ctx,
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		menu:    ui.NewMenu(),
		errBox:  ui.NewErrBox(),
		preview: ui.NewPreviewPane(0, 0),
	}
	h.list = ui.NewList(&h.spinner)
	return h
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
	// Upon starting, we want to start the spinner. Whenever we get a spinner.TickMsg, we
	// update the spinner, which sends a new spinner.TickMsg. I think this lasts forever lol.
	return m.spinner.Tick
}

func (m *home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hideErrMsg:
		m.errBox.Clear()
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.updateHandleWindowSizeEvent(msg)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
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
	case keys.KeyKill:
		m.list.Kill()
		return m, nil
	case keys.KeyUp:
		m.list.Up()
		return m, nil
	case keys.KeyNew:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m.showErrorMessageForShortTime(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}
		return m, nil
		// TODO: add more key bindings
	default:
		return m, nil
	}
	return m, nil
}

// hideErrMsg implements tea.Msg and clears the error text from the screen.
type hideErrMsg struct{}

// showErrorMessageForShortTime sets the error message. We return a callback. I assume bubbletea calls the
// callback in a goroutine because it says that tea.Msg / tea.Cmd should be used for IO operations. These
// tend to block... Eventually, the callback returns a message which is sent back to the Update function.
// Then, we clear the error.
func (m *home) showErrorMessageForShortTime(err error) (tea.Model, tea.Cmd) {
	m.errBox.SetError(err)
	return m, func() tea.Msg {
		select {
		case <-m.ctx.Done():
		case <-time.After(3 * time.Second):
		}

		return hideErrMsg{}
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

	return lipgloss.JoinVertical(
		lipgloss.Center,
		listAndPreview,
		m.menu.String(),
		m.errBox.String(),
	)

	//return m.header.String() + listAndPreview + menu
	//s.WriteString(lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, 0.1, fmt.Sprintf("%d %d", m.windowHeight, m.windowWidth)))

	//return s.String()
}
