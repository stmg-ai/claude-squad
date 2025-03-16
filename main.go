package main

import (
	"claude-squad/app"
	"claude-squad/logger"
	"claude-squad/session"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"log"
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
