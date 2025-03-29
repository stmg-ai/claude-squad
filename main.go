// main package is the entry point for the claude-squad application,
// providing CLI commands for session management and configuration
package main

import (
	"claude-squad/app"
	"claude-squad/config"
	"claude-squad/daemon"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/session/git"
	"claude-squad/session/tmux"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	resetFlag   bool
	programFlag string
	autoYesFlag bool
	daemonFlag  bool
	rootCmd     = &cobra.Command{
		Use:   "claude-squad",
		Short: "Claude Squad - A terminal-based session manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			log.Initialize(daemonFlag)
			defer log.Close()

			if daemonFlag {
				err := daemon.RunDaemon()
				return err
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if resetFlag {
				storage, err := session.NewStorage()
				if err != nil {
					return fmt.Errorf("failed to initialize storage: %w", err)
				}
				if err := storage.DeleteAllInstances(); err != nil {
					return fmt.Errorf("failed to reset storage: %w", err)
				}
				fmt.Println("Storage has been reset successfully")

				if err := tmux.CleanupSessions(); err != nil {
					return fmt.Errorf("failed to cleanup tmux sessions: %w", err)
				}
				fmt.Println("Tmux sessions have been cleaned up")

				if err := git.CleanupWorktrees(); err != nil {
					return fmt.Errorf("failed to cleanup worktrees: %w", err)
				}
				fmt.Println("Worktrees have been cleaned up")

				// Kill any daemon that's running.
				if err := daemon.StopDaemon(); err != nil {
					log.ErrorLog.Printf("failed to stop daemon: %v", err)
				}
				fmt.Println("Daemon has been stopped")

				return nil
			}

			// Program flag overrides config
			program := cfg.DefaultProgram
			if programFlag != "" {
				program = programFlag
			}
			// AutoYes flag overrides config
			autoYes := cfg.AutoYes
			if autoYesFlag {
				autoYes = true
			}
			if autoYes {
				defer func() {
					if err := daemon.LaunchDaemon(); err != nil {
						log.ErrorLog.Printf("failed to launch daemon: %v", err)
					}
				}()
			}
			// Kill any daemon that's running.
			if err := daemon.StopDaemon(); err != nil {
				log.ErrorLog.Printf("failed to stop daemon: %v", err)
			}

			return app.Run(ctx, program, autoYes)
		},
	}

	debugCmd = &cobra.Command{
		Use:   "debug",
		Short: "Print debug information like config paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			configDir, err := config.GetConfigDir()
			if err != nil {
				return fmt.Errorf("failed to get config directory: %w", err)
			}
			configJson, _ := json.MarshalIndent(cfg, "", "  ")

			fmt.Printf("Config: %s\n%s\n", filepath.Join(configDir, "config.json"), configJson)
			return nil
		},
	}
)

func init() {
	rootCmd.Flags().BoolVar(&resetFlag, "reset", false, "Reset all stored instances")
	rootCmd.Flags().StringVarP(&programFlag, "program", "p", "",
		"Program to run in new instances (e.g. 'aider --model ollama_chat/gemma3:1b')")
	rootCmd.Flags().BoolVarP(&autoYesFlag, "autoyes", "y", false,
		"[experimental] If enabled, all instances will automatically accept prompts")
	rootCmd.Flags().BoolVar(&daemonFlag, "daemon", false, "Run a program that loads all sessions"+
		" and runs autoyes mode on them.")
	// Hide the daemonFlag as it's only for internal use
	err := rootCmd.Flags().MarkHidden("daemon")
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(debugCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
