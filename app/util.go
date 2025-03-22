package app

import (
	"os/exec"
)

// isWorktreeDirty checks if the current git worktree has uncommitted changes
func isWorktreeDirty() bool {
	// Check for changes in tracked files only
	// git diff --quiet only checks tracked files with changes
	cmd := exec.Command("git", "diff", "--quiet")
	err := cmd.Run()
	return err != nil
}