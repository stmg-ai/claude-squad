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
	// The working directory for the session
	workDir string
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

// SetWorkDir sets the working directory for the session
func (t *TmuxSession) SetWorkDir(dir string) {
	t.workDir = dir
}

// Start creates and starts a new detached tmux session
func (t *TmuxSession) Start(command string) error {
	// Check if the session already exists
	if DoesSessionExist(t.sanitizedName) {
		return fmt.Errorf("tmux session already exists: %s", t.sanitizedName)
	}

	// Create a new detached tmux session and start claude in it
	var cmd *exec.Cmd
	if t.workDir != "" {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", t.sanitizedName, "-c", t.workDir, command)
	} else {
		cmd = exec.Command("tmux", "new-session", "-d", "-s", t.sanitizedName, command)
	}

	// Close existing PTY if it exists
	if t.ptmx != nil {
		t.ptmx.Close()
		t.ptmx = nil
	}

	var err error
	t.ptmx, err = pty.Start(cmd)
	if err != nil {
		// Cleanup any partially created session
		if DoesSessionExist(t.sanitizedName) {
			cleanupCmd := exec.Command("tmux", "kill-session", "-t", t.sanitizedName)
			if cleanupErr := cleanupCmd.Run(); cleanupErr != nil {
				err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
			}
		}
		return fmt.Errorf("error starting tmux session: %w", err)
	}

	if err := t.updateWindowSize(); err != nil {
		// Cleanup on window size update failure
		if cleanupErr := t.Close(); cleanupErr != nil {
			err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
		}
		return fmt.Errorf("error updating window size: %w", err)
	}

	t.MonitorWindowSize(context.Background())
	return nil
}

// Restore attaches to an existing session and restores the window size
func (t *TmuxSession) Restore() error {
	ptmx, err := pty.Start(exec.Command("tmux", "attach-session", "-t", t.sanitizedName))
	if err != nil {
		return fmt.Errorf("error opening PTY: %w", err)
	}

	// Store old PTY to cleanup if something goes wrong
	oldPtmx := t.ptmx
	t.ptmx = ptmx

	if err := t.updateWindowSize(); err != nil {
		// Restore old PTY and cleanup new one on failure
		if oldPtmx != nil {
			t.ptmx = oldPtmx
		}
		if cleanupErr := ptmx.Close(); cleanupErr != nil {
			err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
		}
		return fmt.Errorf("error updating window size: %w", err)
	}

	// Cleanup old PTY now that new one is successfully set up
	if oldPtmx != nil {
		oldPtmx.Close()
	}

	t.MonitorWindowSize(context.Background())
	return nil
}

func (t *TmuxSession) Attach() (exited chan struct{}) {
	t.attachCh = make(chan struct{})
	defer func() {
		exited = t.attachCh
	}()

	attachCmd := exec.Command("tmux", "attach-session", "-t", t.sanitizedName)

	// Store old state to restore on failure
	oldPtmx := t.ptmx
	oldState := t.oldState

	if oldPtmx != nil {
		oldPtmx.Close()
	}

	var err error
	t.ptmx, err = pty.Start(attachCmd)
	if err != nil {
		// Restore old state on failure
		t.ptmx = oldPtmx
		t.oldState = oldState
		return
	}

	newState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Cleanup new PTY and restore old state on failure
		if cleanupErr := t.ptmx.Close(); cleanupErr != nil {
			err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
		}
		t.ptmx = oldPtmx
		t.oldState = oldState
		return
	}
	t.oldState = newState

	// TODO: goroutines may outlive Session.
	go func() {
		// Copy tmux output to the local stdout.
		_, _ = io.Copy(os.Stdout, t.ptmx)
	}()
	go func() {
		// Read input from stdin and check for escape key
		buf := make([]byte, 32)
		for {
			nr, err := os.Stdin.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				continue
			}

			// Check for escape key (ASCII 27)
			if nr == 1 && buf[0] == 27 {
				// Detach from the session
				if err := t.Detach(); err != nil {
					log.Printf("Error detaching from tmux session: %v", err)
				}
				return
			}

			// Forward other input to tmux
			_, _ = t.ptmx.Write(buf[:nr])
		}
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
		return fmt.Errorf("error detaching from tmux session: %w", err)
	}

	if t.ptmx != nil {
		if err := t.ptmx.Close(); err != nil {
			return fmt.Errorf("error closing attach pty session: %w", err)
		}
		t.ptmx = nil
	}

	if t.oldState != nil {
		if err := term.Restore(int(os.Stdin.Fd()), t.oldState); err != nil {
			return fmt.Errorf("error restoring terminal state: %w", err)
		}
		t.oldState = nil
	}

	return nil
}

// Close terminates the tmux session and cleans up resources
func (t *TmuxSession) Close() error {
	var errs []error

	if t.ptmx != nil {
		if err := t.ptmx.Close(); err != nil {
			errs = append(errs, fmt.Errorf("error closing PTY: %w", err))
		}
		t.ptmx = nil
	}

	cmd := exec.Command("tmux", "kill-session", "-t", t.sanitizedName)
	if err := cmd.Run(); err != nil {
		errs = append(errs, fmt.Errorf("error killing tmux session: %w", err))
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	errMsg := "multiple errors occurred during cleanup:"
	for _, err := range errs {
		errMsg += "\n  - " + err.Error()
	}
	return fmt.Errorf(errMsg)
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
