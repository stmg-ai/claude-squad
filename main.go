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

	"sync"
	"time"
)

func main() {
	tmuxMain()
}

type stdinLisenter struct {
	mu struct {
		*sync.Mutex
		buf bytes.Buffer
	}
}

// copies stdin continuously into itself. then, it exposes a Read method for processes to read. this
// lets us intercept the stdin to detect keystrokes.
func newStdinListener() *stdinLisenter {
	listener := &stdinLisenter{}
	listener.mu.Mutex = &sync.Mutex{}
	go func() {
		if _, err := io.Copy(listener, os.Stdin); err != nil {
			log.Fatalf("Error copying stdin to buffer: %v", err)
		}
	}()
	return listener
}

func (sl *stdinLisenter) Write(p []byte) (n int, err error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.mu.buf.Write(p)
}

func (sl *stdinLisenter) Read(p []byte) (n int, err error) {
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
