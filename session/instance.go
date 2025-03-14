package session

import (
	"fmt"
	"time"
)

type Status int

const (
	// Running is the status when the instance is running and claude is working.
	Running Status = iota
	// Ready is if the claude instance is ready to be interacted with (waiting for user input).
	Ready
	// Loading is if the instance is loading (if we are starting it up or something).
	Loading
)

// A running instance of claude code.
type Instance struct {
	Title       string
	Path        string // workspace path?
	Status      Status
	Height      int
	Width       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	tmuxSession *TmuxSession
}

// ToInstanceData converts an Instance to its serializable form
func (i *Instance) ToInstanceData() InstanceData {
	return InstanceData{
		Title:     i.Title,
		Path:      i.Path,
		Status:    i.Status,
		Height:    i.Height,
		Width:     i.Width,
		CreatedAt: i.CreatedAt,
		UpdatedAt: time.Now(), // Update timestamp when saving
	}
}

// FromInstanceData creates a new Instance from serialized data
func FromInstanceData(data InstanceData) (*Instance, error) {
	instance, err := NewInstance(InstanceOptions{
		Title:              data.Title,
		Path:               data.Path,
		RestoreFromStorage: true,
	})
	if err != nil {
		return nil, err
	}
	instance.Status = data.Status
	instance.Height = data.Height
	instance.Width = data.Width
	instance.CreatedAt = data.CreatedAt
	instance.UpdatedAt = data.UpdatedAt
	return instance, nil
}

// Options for creating a new instance
type InstanceOptions struct {
	Title string
	Path  string
	// ReuseExisting determines whether to reuse an existing session if it exists
	ReuseExisting bool
	// ForceNew forces creation of a new session, killing existing if necessary
	ForceNew bool
	// RestoreFromStorage indicates this is being restored from storage, skip tmux operations
	RestoreFromStorage bool
}

func NewInstance(opts InstanceOptions) (*Instance, error) {
	if opts.Title == "" {
		return nil, fmt.Errorf("instance title cannot be empty")
	}

	tmuxSession := NewTmuxSession(opts.Title)

	// If restoring from storage, create instance without tmux operations
	if opts.RestoreFromStorage {
		now := time.Now()
		return &Instance{
			Title:       opts.Title,
			Path:        opts.Path,
			Status:      Loading,
			tmuxSession: tmuxSession,
			CreatedAt:   now,
			UpdatedAt:   now,
		}, nil
	}

	sessionExists := DoesSessionExist(opts.Title)

	if sessionExists {
		if opts.ForceNew {
			// Kill existing session if force new is requested
			if err := tmuxSession.Close(); err != nil {
				return nil, fmt.Errorf("failed to kill existing session: %v", err)
			}
			sessionExists = false
		} else if !opts.ReuseExisting {
			return nil, fmt.Errorf("session already exists: %s (use ReuseExisting or ForceNew to handle existing session)", opts.Title)
		}
	}

	var err error
	if sessionExists {
		// Reuse existing session
		if err = tmuxSession.Restore(); err != nil {
			return nil, fmt.Errorf("failed to restore existing session: %v", err)
		}
	} else {
		// Create new session
		if err = tmuxSession.Start(); err != nil {
			return nil, fmt.Errorf("failed to start new session: %v", err)
		}
	}

	now := time.Now()
	return &Instance{
		Title:       opts.Title,
		Path:        opts.Path,
		Status:      Loading,
		tmuxSession: tmuxSession,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (i *Instance) Kill() error {
	return i.tmuxSession.Close()
}

func (i *Instance) Preview() (string, error) {
	return i.tmuxSession.CapturePaneContent()
}

func (i *Instance) Attach() chan struct{} {
	return i.tmuxSession.Attach()
}

func (i *Instance) Close() error {
	return i.tmuxSession.Close()
}
