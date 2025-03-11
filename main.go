package main

import (
	"bytes"
	"fmt"
	"github.com/creack/pty"
	"golang.org/x/term"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"sync"
	"time"
)

func main() {
	//app.Run()
	tmuxMain()
}

type stdinListener struct {
	mu struct {
		*sync.Mutex
		buf bytes.Buffer
	}
}

// copies stdin continuously into itself. then, it exposes a Read method for processes to read. this
// lets us intercept the stdin to detect keystrokes.
func newStdinListener() *stdinListener {
	listener := &stdinListener{}
	listener.mu.Mutex = &sync.Mutex{}
	go func() {
		if _, err := io.Copy(listener, os.Stdin); err != nil {
			log.Fatalf("Error copying stdin to buffer: %v", err)
		}
	}()
	return listener
}

// updateWindowSize updates the window size of the PTY based on the current terminal dimensions
func updateWindowSize(ptmx *os.File) error {
	cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	return pty.Setsize(ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
		X:    0,
		Y:    0,
	})
}

func (sl *stdinListener) Write(p []byte) (n int, err error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.mu.buf.Write(p)
}

func (sl *stdinListener) Read(p []byte) (n int, err error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.mu.buf.Read(p)
}

func tmuxMain() {
	// Step 1: Start tmux session using pty (pseudo-terminal)
	cmd := exec.Command("tmux", "new-session", "-d", "-s", "my_session", "'claude'") // Start tmux in detached mode

	// Create a pty (pseudo-terminal) for tmux
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Fatalf("Error starting tmux with pty: %v", err)
	}
	defer ptmx.Close()

	// Update the window size of the PTY
	if err := updateWindowSize(ptmx); err != nil {
		log.Fatalf("Error updating window size: %v", err)
	}

	// Handle window resize events
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	go func() {
		// Send initial SIGWINCH to trigger the first resize
		syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)

		// Create a debounced channel for resize events
		debouncedWinch := make(chan os.Signal)
		go func() {
			var resizeTimer *time.Timer
			for range winch {
				if resizeTimer != nil {
					resizeTimer.Stop()
				}
				resizeTimer = time.AfterFunc(50*time.Millisecond, func() {
					debouncedWinch <- syscall.SIGWINCH
				})
			}
		}()

		// Handle the debounced resize events
		for range debouncedWinch {
			if err := updateWindowSize(ptmx); err != nil {
				log.Printf("failed to update window size: %v", err)
			}
		}
	}()
	defer signal.Stop(winch)

	time.Sleep(1 * time.Second)

	// Step 2: Ensure tmux process is running
	fmt.Println("tmux session 'my_session' started in the background")

	// Step 3: Attach to the tmux session and bring it to the foreground
	// This uses a separate command to attach and run tmux in the foreground
	attachCmd := exec.Command("tmux", "attach-session", "-t", "my_session")
	//attachCmd.Stdout = os.Stdout
	//attachCmd.Stderr = os.Stderr
	//attachCmd.Stdin = os.Stdin

	ptmx, err = pty.Start(attachCmd)
	if err != nil {
		log.Fatalf("Error starting tmux with pty: %v", err)
	}
	defer ptmx.Close()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	go func() {
		// Copy tmux output to the local stdout.
		_, _ = io.Copy(os.Stdout, ptmx)
	}()
	go func() {
		// Copy tmux output to the local stdout.
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	time.Sleep(5 * time.Second)

	detachCmd := exec.Command("tmux", "detach-client", "-s", "my_session")
	err = detachCmd.Run()
	if err != nil {
		log.Fatalf("Error detaching from tmux session: %v", err)
	}

	// Output to indicate tmux is in the foreground
	fmt.Println("tmux session 'my_session' is now in the foreground.")
	time.Sleep(1 * time.Second)
}
