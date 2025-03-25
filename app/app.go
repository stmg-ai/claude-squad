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
)

type home struct {
	ctx context.Context

	program string

	// ui components
	list         *ui.List
	menu         *ui.Menu
	tabbedWindow *ui.TabbedWindow
	errBox       *ui.ErrBox
	// global spinner instance. we plumb this down to where it's needed
	spinner spinner.Model

	// storage
	storage *session.Storage

	// state
	state state
	// newInstanceFinalizer is called when the state is stateNew and then you press enter.
	// It registers the new instance in the list after the instance has been started.
	newInstanceFinalizer func()
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
		ctx:          ctx,
		spinner:      spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		menu:         ui.NewMenu(),
		tabbedWindow: ui.NewTabbedWindow(ui.NewPreviewPane(), ui.NewDiffPane()),
		errBox:       ui.NewErrBox(),
		storage:      storage,
		program:      program,
		state:        stateDefault,
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
		// Call the finalizer immediately.
		h.list.AddInstance(instance)()
	}

	return h
}

// updateHandleWindowSizeEvent sets the sizes of the components.
// The components will try to render inside their bounds.
func (m *home) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
	// List takes 30% of width, preview takes 70%
	listWidth := int(float32(msg.Width) * 0.3)
	tabsWidth := msg.Width - listWidth

	// Menu takes 10% of height, list and window take 90%
	contentHeight := int(float32(msg.Height) * 0.9)
	menuHeight := msg.Height - contentHeight

	m.tabbedWindow.SetSize(tabsWidth, contentHeight)
	m.list.SetSize(listWidth, contentHeight)

	previewWidth, previewHeight := m.tabbedWindow.GetPreviewSize()
	if err := m.list.SetSessionPreviewSize(previewWidth, previewHeight); err != nil {
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
		tickUpdateMetadataCmd,
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
	case tickUpdateMetadataMessage:
		for _, instance := range m.list.GetInstances() {
			if !instance.Started() || instance.Paused() {
				continue
			}
			updated, prompt := instance.HasUpdated()
			if updated {
				instance.SetStatus(session.Running)
			} else {
				if prompt {
					instance.TapEnter()
				} else {
					instance.SetStatus(session.Ready)
				}
			}
			if err := instance.UpdateDiffStats(); err != nil {
				log.Printf("could not update diff stats: %v", err)
			}
		}
		return m, tickUpdateMetadataCmd
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
	return m, tea.Quit
}

func (m *home) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state == stateNew {
		// Handle quit commands first. Don't handle q because the user might want to type that.
		if msg.String() == "ctrl+c" {
			return m.handleQuit()
		}

		instance := m.list.GetInstances()[m.list.NumInstances()-1]
		switch msg.Type {
		// Start the instance (enable previews etc) and go back to the main menu state.
		case tea.KeyEnter:
			if len(instance.Title) == 0 {
				return m.showErrorMessageForShortTime(fmt.Errorf("title cannot be empty"))
			}

			defer func() {
				m.state = stateDefault
				m.menu.SetState(ui.StateDefault)
			}()
			if err := instance.Start(true); err != nil {
				m.list.Kill()
				return m.showErrorMessageForShortTime(err)
			}
			// Save after adding new instance
			if err := m.storage.SaveInstances(m.list.GetInstances()); err != nil {
				return m.showErrorMessageForShortTime(err)
			}
			// Instance added successfully, call the finalizer.
			m.newInstanceFinalizer()
			return m, tea.WindowSize()
		case tea.KeyRunes:
			if len(instance.Title) >= 20 {
				return m.showErrorMessageForShortTime(fmt.Errorf("title cannot be longer than 32 characters"))
			}
			if err := instance.SetTitle(instance.Title + string(msg.Runes)); err != nil {
				return m.showErrorMessageForShortTime(err)
			}
		case tea.KeyBackspace:
			if len(instance.Title) == 0 {
				return m, nil
			}
			if err := instance.SetTitle(instance.Title[:len(instance.Title)-1]); err != nil {
				return m.showErrorMessageForShortTime(err)
			}
		case tea.KeySpace:
			if err := instance.SetTitle(instance.Title + " "); err != nil {
				return m.showErrorMessageForShortTime(err)
			}
		case tea.KeyEsc:
			m.list.Kill()
			m.state = stateDefault
			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					return nil
				},
			)
		default:
		}
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
	case keys.KeyShiftUp:
		if m.tabbedWindow.IsInDiffTab() {
			m.tabbedWindow.ScrollUp()
		}
		return m.updatePreview()
	case keys.KeyShiftDown:
		if m.tabbedWindow.IsInDiffTab() {
			m.tabbedWindow.ScrollDown()
		}
		return m.updatePreview()
	case keys.KeyTab:
		m.tabbedWindow.Toggle()
		m.menu.SetInDiffTab(m.tabbedWindow.IsInDiffTab())
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
	case keys.KeyAutoYes:
		selected := m.list.GetSelectedInstance()
		selected.ToggleAutoYes()
		return m, nil
	case keys.KeyNew:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m.showErrorMessageForShortTime(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}
		instance, err := session.NewInstance(session.InstanceOptions{
			Title:   "",
			Path:    ".",
			Program: m.program,
		})
		if err != nil {
			return m.showErrorMessageForShortTime(err)
		}

		m.newInstanceFinalizer = m.list.AddInstance(instance)
		m.list.SetSelectedInstance(m.list.NumInstances() - 1)
		m.state = stateNew
		m.menu.SetState(ui.StateNewInstance)

		return m, nil
	case keys.KeySubmit:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		// Default commit message with timestamp
		commitMsg := fmt.Sprintf("Update from session %s at %s", selected.Title, time.Now().Format(time.RFC3339))
		worktree, err := selected.GetGitWorktree()
		if err != nil {
			return m.showErrorMessageForShortTime(err)
		}
		if err = worktree.PushChanges(commitMsg); err != nil {
			return m.showErrorMessageForShortTime(err)
		}

		return m, nil
	case keys.KeyPause:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}
		if err := selected.Pause(); err != nil {
			return m.showErrorMessageForShortTime(err)
		}
		return m.updatePreview()
	case keys.KeyResume:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}
		if err := selected.Resume(); err != nil {
			return m.showErrorMessageForShortTime(err)
		}
		return m, tea.WindowSize()
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
	selected := m.list.GetSelectedInstance()

	if err := m.tabbedWindow.UpdatePreview(selected); err != nil {
		return m.showErrorMessageForShortTime(err)
	}

	if err := m.tabbedWindow.UpdateDiff(selected); err != nil {
		return m.showErrorMessageForShortTime(err)
	}

	// Update menu with current instance
	m.menu.SetInstance(selected)
	return m, nil
}

// hideErrMsg implements tea.Msg and clears the error text from the screen.
type hideErrMsg struct{}

// previewTickMsg implements tea.Msg and triggers a preview update
type previewTickMsg struct{}

type tickUpdateMetadataMessage struct{}

// tickUpdateMetadataCmd is the callback to update the metadata of the instances every 500ms. Note that we iterate
// overall the instances and capture their output. It's a pretty expensive operation. Let's do it 2x a second only.
var tickUpdateMetadataCmd = func() tea.Msg {
	time.Sleep(500 * time.Millisecond)
	return tickUpdateMetadataMessage{}
}

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
	listWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.list.String())
	previewWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.tabbedWindow.String())
	listAndPreview := lipgloss.JoinHorizontal(lipgloss.Top, listWithPadding, previewWithPadding)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		listAndPreview,
		m.menu.String(),
		m.errBox.String(),
	)
}
