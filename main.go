package main

import (
	"claude-squad/app"
	"claude-squad/config"
	"claude-squad/logger"
	"claude-squad/session"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	resetFlag   bool
	programFlag string
	rootCmd     = &cobra.Command{
		Use:   "claude-squad",
		Short: "Claude Squad - A terminal-based session manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			logger.Initialize()
			defer logger.Close()

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
				return nil
			}

			// Program flag overrides config
			program := cfg.DefaultProgram
			if programFlag != "" {
				program = programFlag
			}

			app.Run(ctx, program)
			return nil
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
	rootCmd.Flags().StringVarP(&programFlag, "program", "p", "", "Program to run in new instances (e.g. 'aider --model ollama_chat/gemma3:1b')")
	rootCmd.AddCommand(debugCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
