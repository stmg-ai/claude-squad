package app

import (
	"claude-squad/keys"
	"claude-squad/session"
	"claude-squad/ui"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const GlobalInstanceLimit = 10

// Run is the main entrypoint into the application.
func Run(ctx context.Context, program string) {
	p := tea.NewProgram(newHome(ctx, program), tea.WithAltScreen())
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
	// statePush is the state when the user is pushing changes
	statePush
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

	// storage
	storage *session.Storage

	// input
	inputDisabled bool

	// state
	windowWidth  int
	windowHeight int
	state        state
	program      string
}

func newHome(ctx context.Context, program string) *home {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize storage
	storage, err := session.NewStorage()
	if err != nil {
		fmt.Printf("Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	h := &home{
		ctx:     ctx,
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		menu:    ui.NewMenu(),
		errBox:  ui.NewErrBox(),
		preview: ui.NewPreviewPane(0, 0),
		storage: storage,
		program: program,
	}
	h.list = ui.NewList(&h.spinner)

	// Load saved instances
	instances, err := storage.LoadInstances()
	if err != nil {
		fmt.Printf("Failed to load instances: %v\n", err)
		os.Exit(1)
	}

	// Add loaded instances to the list
	for _, instance := range instances {
		h.list.AddInstance(instance)
	}

	return h
}

// updateHandleWindowSizeEvent sets the sizes of the components.
// The components will try to render inside their bounds.
func (m *home) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
	m.windowWidth, m.windowHeight = msg.Width, msg.Height

	// List takes 30% of width, preview takes 70%
	listWidth := int(float32(msg.Width) * 0.3)
	previewWidth := msg.Width - listWidth

	// Menu takes 10% of height, list and preview take 90%
	contentHeight := int(float32(msg.Height) * 0.9)
	menuHeight := msg.Height - contentHeight

	m.preview.SetSize(previewWidth, contentHeight)
	m.list.SetSize(listWidth, contentHeight)
	if err := m.list.SetSessionPreviewSize(ui.AdjustPreviewWidth(previewWidth), 40); err != nil {
		log.Println(err)
	}
	m.menu.SetSize(msg.Width, menuHeight)
}

func (m *home) Init() tea.Cmd {
	// Upon starting, we want to start the spinner. Whenever we get a spinner.TickMsg, we
	// update the spinner, which sends a new spinner.TickMsg. I think this lasts forever lol.
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			time.Sleep(100 * time.Millisecond)
			return previewTickMsg{}
		},
	)
}

func (m *home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hideErrMsg:
		m.errBox.Clear()
	case previewTickMsg:
		var cmd tea.Cmd
		model, cmd := m.updatePreview()
		m = model.(*home)
		return m, tea.Batch(
			cmd,
			func() tea.Msg {
				time.Sleep(100 * time.Millisecond)
				return previewTickMsg{}
			},
		)
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

func (m *home) handleQuit() (tea.Model, tea.Cmd) {
	if err := m.storage.SaveInstances(m.list.GetInstances()); err != nil {
		return m.showErrorMessageForShortTime(err)
	}
	m.quitting = true
	return m, tea.Quit
}

func (m *home) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.inputDisabled {
		return m, nil
	}

	// Handle quit commands first
	if msg.String() == "ctrl+c" || msg.String() == "q" {
		return m.handleQuit()
	}

	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return m, nil
	}

	switch name {
	case keys.KeyUp:
		m.list.Up()
		return m.updatePreview()
	case keys.KeyDown:
		m.list.Down()
		return m.updatePreview()
	case keys.KeyKill:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		// Delete from storage first
		if err := m.storage.DeleteInstance(selected.Title); err != nil {
			return m.showErrorMessageForShortTime(err)
		}

		// Then kill the instance
		m.list.Kill()
		return m, tea.WindowSize()
	case keys.KeyNew:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m.showErrorMessageForShortTime(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}

		instance := session.NewInstance(session.InstanceOptions{
			Title:   fmt.Sprintf("instance-%d", m.list.NumInstances()+1),
			Path:    ".",
			Program: m.program,
		})
		if err := instance.Start(); err != nil {
			return m.showErrorMessageForShortTime(err)
		}
		m.list.AddInstance(instance)

		// Save after adding new instance
		if err := m.storage.SaveInstances(m.list.GetInstances()); err != nil {
			return m.showErrorMessageForShortTime(err)
		}

		return m, nil
	case keys.KeyPush:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		// Default commit message with timestamp
		commitMsg := fmt.Sprintf("Update from session %s at %s", selected.Title, time.Now().Format(time.RFC3339))

		if err := selected.GetGitWorktree().PushChanges(commitMsg); err != nil {
			return m.showErrorMessageForShortTime(err)
		}

		return m, nil
	// TODO: add more key bindings
	case keys.KeyEnter:
		if m.list.NumInstances() == 0 {
			return m, nil
		}
		ch, err := m.list.Attach()
		if err != nil {
			return m.showErrorMessageForShortTime(err)
		}
		<-ch
		// WindowSize clears the screen.
		return m, tea.WindowSize()
	default:
		return m, nil
	}
}

// updatePreview updates the preview pane with the currently selected instance
func (m *home) updatePreview() (tea.Model, tea.Cmd) {
	if err := m.preview.UpdateContent(m.list.GetSelectedInstance()); err != nil {
		return m.showErrorMessageForShortTime(err)
	}
	return m, nil
}

// hideErrMsg implements tea.Msg and clears the error text from the screen.
type hideErrMsg struct{}

// previewTickMsg implements tea.Msg and triggers a preview update
type previewTickMsg struct{}

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
	listAndPreview := lipgloss.JoinHorizontal(lipgloss.Top, m.list.String(), m.preview.String())

	return lipgloss.JoinVertical(
		lipgloss.Center,
		listAndPreview,
		m.menu.String(),
		m.errBox.String(),
	)
}
