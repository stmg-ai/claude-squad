package main

import (
	"bytes"
	"claude-squad/app"
	"claude-squad/logger"
	"claude-squad/session"
	"context"
	"log"
	"sync"
)

func main() {
	ctx := context.Background()
	logger.Initialize()
	defer logger.Close()

	app.Run(ctx)
	// tmuxMain()
}

type stdinListener struct {
	mu struct {
		*sync.Mutex
		buf bytes.Buffer
	}
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
	tmux := session.NewTmuxSession("my_session")
	defer tmux.Close()

	if err := tmux.Start(); err != nil {
		log.Fatalf("Error starting tmux session: %v", err)
	}

	if err := tmux.Attach(); err != nil {
		log.Fatalf("Error attaching to tmux session: %v", err)
	}

	if err := tmux.Detach(); err != nil {
		log.Fatalf("Error detaching from tmux session: %v", err)
	}
}
