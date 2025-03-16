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
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// TmuxSession represents a managed tmux session
type TmuxSession struct {
	// Initialized by NewTmuxSession
	//
	// The name of the tmux session and the sanitized name used for tmux commands.
	Name          string
	sanitizedName string

	// Initialized by Start or Restore
	//
	// ptmx is a PTY is running the tmux attach command. This can be resized to change the
	// stdout dimensions of the tmux pane. On detach, we close it and set a new one.
	// This should never be nil.
	ptmx *os.File

	// Initialized by Attach
	// Deinitilaized by Detach
	//
	// Channel to be closed at the very end of detaching. Used to signal callers.
	attachCh chan struct{}
	// Terminal state before attach. Gets restored when detaching.
	oldState *term.State
	// While attached, we use some goroutines to manage the window size and stdin/stdout. This stuff
	// is used to terminate them on Detach. We don't want them to outlive the attached window.
	ctx    context.Context
	cancel func()
	wg     *sync.WaitGroup
}

func removeWhitespace(str string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(str, "")
}

func NewTmuxSession(name string) *TmuxSession {
	return &TmuxSession{
		Name:          name,
		sanitizedName: removeWhitespace(name),
	}
}

// Start creates and starts a new tmux session, then attaches to it. Program is the command to run in
// the session (ex. claude). workdir is the git worktree directory.
func (t *TmuxSession) Start(program string, workDir string) error {
	// Check if the session already exists
	if DoesSessionExist(t.sanitizedName) {
		return fmt.Errorf("tmux session already exists: %s", t.sanitizedName)
	}

	// Create a new detached tmux session and start claude in it
	cmd := exec.Command("tmux", "new-session", "-d", "-s", t.sanitizedName, "-c", workDir, program)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		// Cleanup any partially created session if any exists.
		if DoesSessionExist(t.sanitizedName) {
			cleanupCmd := exec.Command("tmux", "kill-session", "-t", t.sanitizedName)
			if cleanupErr := cleanupCmd.Run(); cleanupErr != nil {
				err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
			}
		}
		return fmt.Errorf("error starting tmux session: %w", err)
	}

	// We need to close the ptmx, but we shouldn't close it before the command above finishes.
	// So, we poll for completion before closing.
	timeout := time.After(2 * time.Second)
	for !DoesSessionExist(t.sanitizedName) {
		select {
		case <-timeout:
			// Cleanup on window size update failure
			if cleanupErr := t.Close(); cleanupErr != nil {
				err = fmt.Errorf("%v (cleanup error: %v)", err, cleanupErr)
			}
			return fmt.Errorf("timed out waiting for tmux session: %v", err)
		default:
			time.Sleep(time.Millisecond * 10)
		}
	}
	ptmx.Close()

	return t.Restore()
}

// Restore attaches to an existing session and restores the window size
func (t *TmuxSession) Restore() error {
	ptmx, err := pty.Start(exec.Command("tmux", "attach-session", "-t", t.sanitizedName))
	if err != nil {
		return fmt.Errorf("error opening PTY: %w", err)
	}
	t.ptmx = ptmx
	return nil
}

func (t *TmuxSession) Attach() (chan struct{}, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("error making terminal raw: %v", err)
	}
	t.oldState = oldState
	t.attachCh = make(chan struct{})

	t.wg = &sync.WaitGroup{}
	t.wg.Add(1)
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// The first goroutine should terminate when the ptmx is closed. We use the
	// waitgroup to wait for it to finish.
	// The 2nd one returns when you press escape to Detach. It doesn't need to be
	// in the waitgroup because is the goroutine doing the Detaching; it waits for
	// all the other ones.
	go func() {
		defer t.wg.Done()
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

	t.monitorWindowSize()
	return t.attachCh, nil
}

// Detach disconnects from the current tmux session
func (t *TmuxSession) Detach() (err error) {
	// TODO: control flow is a bit messy here. If there's an error,
	// I'm not sure if we get into a bad state. Needs testing.
	defer func() {
		close(t.attachCh)
		t.attachCh = nil
		t.oldState = nil
		t.cancel = nil
		t.ctx = nil
		t.wg = nil
	}()

	// Close the attached pty session.
	if err := t.ptmx.Close(); err != nil {
		return fmt.Errorf("error closing attach pty session: %w", err)
	}
	// Attach goroutines should die on EOF due to the ptmx closing. Call
	// t.Restore to set a new t.ptmx.
	if err := t.Restore(); err != nil {
		return err
	}
	// Yeild the stdin/stdout back to the UI.
	if err := term.Restore(int(os.Stdin.Fd()), t.oldState); err != nil {
		return fmt.Errorf("error restoring terminal state: %w", err)
	}

	// Cancel goroutines created by Attach.
	t.cancel()
	t.wg.Wait()

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

// monitorWindowSize monitors and handles window resize events while attached.
func (t *TmuxSession) monitorWindowSize() {
	winchChan := make(chan os.Signal)
	signal.Notify(winchChan, syscall.SIGWINCH)
	// Send initial SIGWINCH to trigger the first resize
	syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)

	doUpdate := func() {
		// Use the current terminal height and width.
		cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			log.Printf("failed to update window size: %v", err)
		} else {
			if err := t.updateWindowSize(cols, rows); err != nil {
				log.Printf("failed to update window size: %v", err)
			}
		}
	}
	// Do one at the end of the function to set the initial size.
	defer doUpdate()

	// Debounce resize events
	t.wg.Add(2)
	debouncedWinch := make(chan os.Signal, 1)
	go func() {
		defer t.wg.Done()
		var resizeTimer *time.Timer
		for {
			select {
			case <-t.ctx.Done():
				return
			case <-winchChan:
				if resizeTimer != nil {
					resizeTimer.Stop()
				}
				resizeTimer = time.AfterFunc(50*time.Millisecond, func() {
					select {
					case debouncedWinch <- syscall.SIGWINCH:
					case <-t.ctx.Done():
					}
				})
			}
		}
	}()
	go func() {
		defer t.wg.Done()
		defer signal.Stop(winchChan)
		// Handle resize events
		for {
			select {
			case <-t.ctx.Done():
				return
			case <-debouncedWinch:
				doUpdate()
			}
		}
	}()
}

// SetDetachedSize set the width and height of the session while detached. This makes the
// tmux output conform to the specified shape.
func (t *TmuxSession) SetDetachedSize(width, height int) error {
	return t.updateWindowSize(width, height)
}

// updateWindowSize updates the window size of the PTY.
func (t *TmuxSession) updateWindowSize(cols, rows int) error {
	return pty.Setsize(t.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
		X:    0,
		Y:    0,
	})
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
