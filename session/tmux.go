package session

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// TmuxSession represents a managed tmux session
type TmuxSession struct {
	// The name of the tmux session
	Name string
	// The sanitized name used for tmux commands
	sanitizedName string
	// The PTY for the session
	ptmx *os.File
	// The channel for window resize events
	winchChan chan os.Signal
	// The channel to signal detach completion
	attachCh chan struct{}
	// Terminal state before attach
	oldState *term.State
	// The width of the terminal
	width int
}

func removeWhitespace(str string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(str, "")
}

var SendSessionToBackgroundKeys = map[string]struct{}{
	"esc": {},
}

func NewTmuxSession(name string) *TmuxSession {
	return &TmuxSession{
		Name:          name,
		sanitizedName: removeWhitespace(name),
		winchChan:     make(chan os.Signal, 1),
	}
}

// Start creates and starts a new detached tmux session
func (t *TmuxSession) Start() error {
	log.Printf("Starting new tmux session: %s", t.sanitizedName)
	// Check if the session already exists
	if DoesSessionExist(t.sanitizedName) {
		return fmt.Errorf("tmux session already exists: %s", t.sanitizedName)
	}

	// Create a new detached tmux session and start claude in it
	cmd := exec.Command("tmux", "new-session", "-d", "-s", t.sanitizedName)

	var err error
	if t.ptmx != nil {
		log.Printf("Warning: PTY already exists for session %s, closing it", t.sanitizedName)
		t.ptmx.Close()
	}
	t.ptmx, err = pty.Start(cmd)
	if err != nil {
		log.Printf("Error starting tmux session: %v", err)
		return fmt.Errorf("error starting tmux session: %v", err)
	}
	log.Printf("Successfully created PTY for session: %s", t.sanitizedName)

	if err := t.updateWindowSize(); err != nil {
		return fmt.Errorf("error updating window size: %v", err)
	}

	t.MonitorWindowSize(context.Background())
	return nil
}

// Restore attaches to an existing session and restores the window size
func (t *TmuxSession) Restore() error {
	ptmx, err := pty.Start(exec.Command("tmux", "attach-session", "-t", t.sanitizedName))
	if err != nil {
		return fmt.Errorf("error opening PTY: %v", err)
	}
	t.ptmx = ptmx

	if err := t.updateWindowSize(); err != nil {
		return fmt.Errorf("error updating window size: %v", err)
	}

	t.MonitorWindowSize(context.Background())
	return nil
}

func (t *TmuxSession) Attach() (exited chan struct{}) {
	log.Printf("Attaching to tmux session: %s", t.sanitizedName)
	t.attachCh = make(chan struct{})
	defer func() {
		exited = t.attachCh
	}()

	attachCmd := exec.Command("tmux", "attach-session", "-t", t.sanitizedName)

	if t.ptmx != nil {
		log.Printf("Warning: PTY already exists for session %s during attach, closing it", t.sanitizedName)
		t.ptmx.Close()
	}

	var err error
	t.ptmx, err = pty.Start(attachCmd)
	if err != nil {
		log.Printf("Error attaching to session: %v", err)
		return
	}
	log.Printf("Successfully created PTY for attach: %s", t.sanitizedName)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Printf("error making terminal raw %s", err)
		return
	}
	t.oldState = oldState

	// TODO: goroutines may outlive Session.
	go func() {
		// Copy tmux output to the local stdout.
		_, _ = io.Copy(os.Stdout, t.ptmx)
	}()
	go func() {
		// Copy tmux output to the local stdout.
		_, _ = io.Copy(t.ptmx, os.Stdin)
	}()

	return
}

// Detach disconnects from the current tmux session
func (t *TmuxSession) Detach() error {
	defer func() {
		if t.attachCh != nil {
			close(t.attachCh)
			t.attachCh = nil
		}
	}()

	detachCmd := exec.Command("tmux", "detach-client", "-s", t.sanitizedName)
	if err := detachCmd.Run(); err != nil {
		return fmt.Errorf("error detaching from tmux session: %v", err)
	}

	if t.ptmx != nil {
		if err := t.ptmx.Close(); err != nil {
			log.Printf("error closing attach pty session: %v", err)
		}
		t.ptmx = nil
	}

	if t.oldState != nil {
		if err := term.Restore(int(os.Stdin.Fd()), t.oldState); err != nil {
			log.Printf("error restoring terminal state: %v", err)
		}
		t.oldState = nil
	}

	return nil
}

// Close terminates the tmux session and cleans up resources
func (t *TmuxSession) Close() error {
	log.Printf("Closing tmux session: %s", t.sanitizedName)
	if t.ptmx != nil {
		log.Printf("Closing PTY for session: %s", t.sanitizedName)
		if err := t.ptmx.Close(); err != nil {
			log.Printf("Error closing PTY: %v", err)
			return fmt.Errorf("error closing PTY: %v", err)
		}
		t.ptmx = nil
	}

	cmd := exec.Command("tmux", "kill-session", "-t", t.sanitizedName)
	if err := cmd.Run(); err != nil {
		log.Printf("Error killing tmux session: %v", err)
		return fmt.Errorf("error killing tmux session: %v", err)
	}
	log.Printf("Successfully closed session: %s", t.sanitizedName)
	return nil
}

// MonitorWindowSize monitors and handles window resize events
func (t *TmuxSession) MonitorWindowSize(ctx context.Context) {
	signal.Notify(t.winchChan, syscall.SIGWINCH)

	// Send initial SIGWINCH to trigger the first resize
	syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)

	go func() {
		debouncedWinch := make(chan os.Signal, 1)
		defer signal.Stop(t.winchChan)

		// Debounce resize events
		go func() {
			var resizeTimer *time.Timer
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.winchChan:
					if resizeTimer != nil {
						resizeTimer.Stop()
					}
					resizeTimer = time.AfterFunc(50*time.Millisecond, func() {
						select {
						case debouncedWinch <- syscall.SIGWINCH:
						case <-ctx.Done():
						}
					})
				}
			}
		}()

		// Handle resize events
		for {
			select {
			case <-ctx.Done():
				return
			case <-debouncedWinch:
				if err := t.updateWindowSize(); err != nil {
					log.Printf("failed to update window size: %v", err)
				}
			}
		}
	}()
}

// updateWindowSize updates the window size of the PTY based on the current terminal dimensions
func (t *TmuxSession) updateWindowSize() error {
	if t.ptmx == nil {
		return nil
	}

	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	// Calculate the preview width as 70% of total width
	previewWidth := int(float64(cols) * 0.7)
	t.width = previewWidth

	// Set the PTY size
	if err := pty.Setsize(t.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(previewWidth), // Use preview width for the tmux pane
		X:    0,
		Y:    0,
	}); err != nil {
		return err
	}

	return nil
}

// DoesSessionExist checks if a tmux session exists
func DoesSessionExist(name string) bool {
	existsCmd := exec.Command("tmux", "has-session", "-t", name)
	return existsCmd.Run() == nil
}

// CapturePaneContent captures the content of the tmux pane
func (t *TmuxSession) CapturePaneContent() (string, error) {
	// Add -e flag to preserve escape sequences (ANSI color codes)
	cmd := exec.Command("tmux", "capture-pane", "-p", "-e", "-J", "-t", t.sanitizedName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error capturing pane content: %v", err)
	}
	return string(output), nil
}

// CapturePaneContentWithOptions captures the pane content with additional options
// start and end specify the starting and ending line numbers (use "-" for the start/end of history)
func (t *TmuxSession) CapturePaneContentWithOptions(start, end string) (string, error) {
	// Add -e flag to preserve escape sequences (ANSI color codes)
	cmd := exec.Command("tmux", "capture-pane", "-p", "-e", "-J", "-S", start, "-E", end, "-t", t.sanitizedName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture tmux pane content with options: %v", err)
	}
	return string(output), nil
}
