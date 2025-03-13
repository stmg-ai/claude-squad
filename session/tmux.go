package session

import (
	"fmt"
	"github.com/creack/pty"
	"golang.org/x/term"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// TmuxSession represents a managed tmux session
type TmuxSession struct {
	// The name of the tmux session
	Name      string
	// The PTY for the session
	ptmx      *os.File
	// The channel for window resize events
	winchChan chan os.Signal
}

func NewTmuxSession(name string) *TmuxSession {
	return &TmuxSession{
		Name:      name,
		winchChan: make(chan os.Signal, 1),
	}
}

// Creates and starts a new detached tmux session
func (t *TmuxSession) Start() error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", t.Name)

	var err error
	t.ptmx, err = pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("error starting tmux session: %v", err)
	}

	if err := t.updateWindowSize(); err != nil {
		return fmt.Errorf("error updating window size: %v", err)
	}

	t.handleWindowResize()
	return nil
}

// Attaches to an existing tmux session
func (t *TmuxSession) Attach() error {
	attachCmd := exec.Command("tmux", "attach-session", "-t", t.Name)

	var err error
	t.ptmx, err = pty.Start(attachCmd)
	if err != nil {
		return fmt.Errorf("error attaching to tmux session: %v", err)
	}

	// Set terminal to raw mode so that all keyboard input is passed directly to tmux without
	// being processed by the terminal driver. This allows tmux to handle all key combinations
	// and control sequences properly. We restore the original terminal state when done.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("error setting raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var wg sync.WaitGroup
	wg.Add(2)

	// Handle output from tmux
	go func() {
		defer wg.Done()
		io.Copy(os.Stdout, t.ptmx)
	}()

	// Handle input to tmux
	go func() {
		defer wg.Done()
		io.Copy(t.ptmx, os.Stdin)
	}()

	wg.Wait()
	return nil
}

// Detaches from the current tmux session
func (t *TmuxSession) Detach() error {
	detachCmd := exec.Command("tmux", "detach-client", "-s", t.Name)
	if err := detachCmd.Run(); err != nil {
		return fmt.Errorf("error detaching from tmux session: %v", err)
	}
	return nil
}

// Closes the tmux session
func (t *TmuxSession) Close() {
	if t.ptmx != nil {
		t.ptmx.Close()
	}
	if t.winchChan != nil {
		signal.Stop(t.winchChan)
		close(t.winchChan)
	}
}

// Updates the window size of the PTY based on the current terminal dimensions
func (t *TmuxSession) updateWindowSize() error {
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	return pty.Setsize(t.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
		X:    0,
		Y:    0,
	})
}

func (t *TmuxSession) handleWindowResize() {
	signal.Notify(t.winchChan, syscall.SIGWINCH)

	go func() {
		// Send initial SIGWINCH to trigger the first resize
		syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)

		// Create a debounced channel for resize events
		debouncedWinch := make(chan os.Signal)
		go func() {
			var resizeTimer *time.Timer
			for range t.winchChan {
				if resizeTimer != nil {
					resizeTimer.Stop()
				}
				resizeTimer = time.AfterFunc(50*time.Millisecond, func() {
					debouncedWinch <- syscall.SIGWINCH
				})
			}
		}()

		for range debouncedWinch {
			if err := t.updateWindowSize(); err != nil {
				log.Printf("failed to update window size: %v", err)
			}
		}
	}()
}
