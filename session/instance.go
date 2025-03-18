package session

import (
	"claude-squad/session/git"

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

// Instance is a running instance of claude code.
type Instance struct {
	// Title is the title of the instance.
	Title string
	// Path is the path to the workspace.
	Path string
	// Status is the status of the instance.
	Status Status
	// Height is the height of the instance.
	Height int
	// Width is the width of the instance.
	Width int
	// CreatedAt is the time the instance was created.
	CreatedAt time.Time
	// UpdatedAt is the time the instance was last updated.
	UpdatedAt time.Time
	// Program is the program to run in the instance.
	Program string
	// tmuxSession is the tmux session for the instance.
	tmuxSession *TmuxSession
	// gitWorktree is the git worktree for the instance.
	gitWorktree *git.GitWorktree
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
		UpdatedAt: time.Now(),
		Program:   i.Program,
	}
}

// FromInstanceData creates a new Instance from serialized data
func FromInstanceData(data InstanceData) (*Instance, error) {
	instance, err := NewInstance(InstanceOptions{
		Title:   data.Title,
		Path:    data.Path,
		Program: data.Program,
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
	// Title is the title of the instance.
	Title string
	// Path is the path to the workspace.
	Path string
	// Program is the program to run in the instance (e.g. "claude", "aider --model ollama_chat/gemma3:1b")
	Program string
}

func NewInstance(opts InstanceOptions) (*Instance, error) {
	if opts.Title == "" {
		return nil, fmt.Errorf("instance title cannot be empty")
	}

	tmuxSession := NewTmuxSession(opts.Title)
	gitWorktree := git.NewGitWorktree(opts.Path, opts.Title)

	// Create instance first so we can use its cleanup methods
	now := time.Now()
	instance := &Instance{
		Title:       opts.Title,
		Path:        opts.Path,
		Status:      Loading,
		tmuxSession: tmuxSession,
		gitWorktree: gitWorktree,
		CreatedAt:   now,
		UpdatedAt:   now,
		Program:     opts.Program,
	}

	// Setup error handler to cleanup resources on any error
	var setupErr error
	defer func() {
		if setupErr != nil {
			if cleanupErr := instance.Kill(); cleanupErr != nil {
				setupErr = fmt.Errorf("%v (cleanup error: %v)", setupErr, cleanupErr)
			}
		}
	}()

	sessionExists := DoesSessionExist(opts.Title)

	if sessionExists {
		// Reuse existing session
		if err := tmuxSession.Restore(); err != nil {
			setupErr = fmt.Errorf("failed to restore existing session: %w", err)
			return nil, setupErr
		}
	} else {
		// Setup git worktree first
		if err := gitWorktree.Setup(); err != nil {
			setupErr = fmt.Errorf("failed to setup git worktree: %w", err)
			return nil, setupErr
		}

		// Create new session
		if err := tmuxSession.Start(opts.Program, gitWorktree.GetWorktreePath()); err != nil {
			// Cleanup git worktree if tmux session creation fails
			if cleanupErr := gitWorktree.Cleanup(); cleanupErr != nil {
				err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
			}
			setupErr = fmt.Errorf("failed to start new session: %w", err)
			return nil, setupErr
		}
	}

	return instance, nil
}

// Kill terminates the instance and cleans up all resources
func (i *Instance) Kill() error {
	var errs []error

	// Always try to cleanup both resources, even if one fails
	// Clean up tmux session first since it's using the git worktree
	if err := i.tmuxSession.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close tmux session: %w", err))
	}

	// Then clean up git worktree
	if err := i.gitWorktree.Cleanup(); err != nil {
		errs = append(errs, fmt.Errorf("failed to cleanup git worktree: %w", err))
	}

	return i.combineErrors(errs)
}

// combineErrors combines multiple errors into a single error
func (i *Instance) combineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	errMsg := "multiple cleanup errors occurred:"
	for _, err := range errs {
		errMsg += "\n  - " + err.Error()
	}
	return fmt.Errorf(errMsg)
}

// Close is an alias for Kill to maintain backward compatibility
func (i *Instance) Close() error {
	return i.Kill()
}

func (i *Instance) Preview() (string, error) {
	return i.tmuxSession.CapturePaneContent()
}

func (i *Instance) Attach() (chan struct{}, error) {
	return i.tmuxSession.Attach()
}

func (i *Instance) SetPreviewSize(width, height int) error {
	return i.tmuxSession.SetDetachedSize(width, height)
}

// GetGitWorktree returns the git worktree for the instance
func (i *Instance) GetGitWorktree() *git.GitWorktree {
	return i.gitWorktree
}
