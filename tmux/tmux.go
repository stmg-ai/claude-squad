package tmux

import (
	"context"
	"github.com/creack/pty"
	"golang.org/x/term"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

var SendSessionToBackgroundKeys = map[string]struct{}{
	"esc": {},
}

// Session contains a shell running tmux, which runs claude. It can be put in the foreground
// or background.
type Session struct {
	// tmux session name
	sanitizedKey string

	// ptmx running the attach command which gets piped to the terminal
	attachPtmx *os.File
	oldState   *term.State
	attachCh   chan struct{} // signals that detaching is done to the caller
}

func removeWhitespace(str string) string {
	// Regular expression that matches all whitespace characters (space, tab, newline, etc.)
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(str, "")
}

// NewTmuxSession starts a new tmux session. The name denotes the underlying tmux session name.
// If you make two sessions with the same name, it'll open the same tmux.
func NewTmuxSession(key string) *Session {
	sanitizedKey := removeWhitespace(key)

	// Start tmux session using pty in detached mode. Have claude running in there.
	cmd := exec.Command(
		"tmux",
		"new-session",
		"-d",
		"-s",
		sanitizedKey,
		"'claude'")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Fatalf("Error starting tmux with pty: %v", err)
	} else {
		defer func() {
			if err := ptmx.Close(); err != nil {
				log.Printf("error closing pty session: %v", err)
			}
		}()
	}

	cmd.Wait()

	s := &Session{
		sanitizedKey: sanitizedKey,
	}

	return s
}

func (s *Session) Close() {
	// Kill the session.
	detachCmd := exec.Command("tmux", "kill-session", "-t", s.sanitizedKey)
	err := detachCmd.Run()
	if err != nil {
		// TODO: This is very bad. Should we panic and nuke everything?
		log.Printf("error detaching from tmux session: %v", err)
	}
}

// MonitorWindowSize monitors the window size of the terminal in goroutines and updates it.
func (s *Session) MonitorWindowSize(ctx context.Context) {
	winch := make(chan os.Signal, 1)
	debouncedWinch := make(chan os.Signal, 1)

	// Subscribe to SIGWINCH signals.
	signal.Notify(winch, syscall.SIGWINCH)
	// Send initial SIGWINCH to trigger the first resize
	err := syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)
	if err != nil {
		log.Printf("error resizing window size: %v", err)
	}

	go func() {
		defer signal.Stop(winch)
		var resizeTimer *time.Timer
		for {
			select {
			case <-winch:
			}

			if resizeTimer != nil {
				resizeTimer.Stop()
			}

			// TODO: goroutine may outlive Session.
			resizeTimer = time.AfterFunc(50*time.Millisecond, func() {
				debouncedWinch <- syscall.SIGWINCH
			})
		}
	}()

	go func() {
		// Handle the debounced resize events
		for {
			select {
			case <-debouncedWinch:
			}
			// TODO: handle resizing. We need to make sure we don't resize after the attachPtmx is closed ideally.
			//s.updateWindowSize()
		}
	}()
}

// updateWindowSize updates the window size of the PTY based on the current terminal dimensions
func (s *Session) updateWindowSize() {
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("error getting window size: %v", err)
		return
	}

	if err := pty.Setsize(s.attachPtmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
		X:    0,
		Y:    0,
	}); err != nil {
		log.Fatalf("error updating window size: %v", err)
		return
	}
}

func (s *Session) Attach() (exited chan struct{}) {
	s.attachCh = make(chan struct{})
	defer func() {
		exited = s.attachCh
	}()

	attachCmd := exec.Command("tmux", "attach-session", "-t", s.sanitizedKey)

	ptmx, err := pty.Start(attachCmd)
	if err != nil {
		log.Printf("error starting tmux with pty: %v", err)
		return
	}
	s.attachPtmx = ptmx

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Printf("error making terminal raw %s", err)
		return
	}
	s.oldState = oldState

	// TODO: goroutines may outlive Session.
	go func() {
		// Copy tmux output to the local stdout.
		_, _ = io.Copy(os.Stdout, ptmx)
	}()
	go func() {
		// Copy tmux output to the local stdout.
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// For now, just detach after 5 seconds.
	time.AfterFunc(5*time.Second, func() {
		s.Detach()
	})

	return
}

func (s *Session) Detach() {
	defer close(s.attachCh)

	detachCmd := exec.Command("tmux", "detach-client", "-s", s.sanitizedKey)
	err := detachCmd.Run()
	if err != nil {
		// TODO: This is very bad. Should we panic and nuke everything?
		log.Printf("error detaching from tmux session: %v", err)
	}

	if s.attachPtmx == nil {
		if err := s.attachPtmx.Close(); err != nil {
			log.Printf("error closing attach pty session: %v", err)
		}
	}
	if s.oldState != nil {
		if err := term.Restore(int(os.Stdin.Fd()), s.oldState); err != nil {
			log.Printf("error restoring terminal state %v", err)
		}
	}
}
