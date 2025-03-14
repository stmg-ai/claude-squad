package main

import (
	"bytes"
	"claude-squad/app"
	"claude-squad/logger"
	"claude-squad/session"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/spf13/cobra"
)

var (
	resetFlag bool
	rootCmd   = &cobra.Command{
		Use:   "claude-squad",
		Short: "Claude Squad - A terminal-based session manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			logger.Initialize()
			defer logger.Close()

			if resetFlag {
				storage, err := session.NewStorage()
				if err != nil {
					return fmt.Errorf("failed to initialize storage: %w", err)
				}
				if err := storage.DeleteAllInstances(); err != nil {
					return fmt.Errorf("failed to reset storage: %w", err)
				}
				fmt.Println("Storage has been reset successfully")
				return nil
			}

			app.Run(ctx)
			return nil
		},
	}
)

func init() {
	rootCmd.Flags().BoolVar(&resetFlag, "reset", false, "Reset all stored instances")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
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
