package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GitWorktree manages git worktree operations for a session
type GitWorktree struct {
	// Path to the repository
	repoPath string
	// Path to the worktree
	worktreePath string
	// Name of the session
	sessionName string
	// Branch name for the worktree
	branchName string
}

// NewGitWorktree creates a new GitWorktree instance
func NewGitWorktree(repoPath string, sessionName string) (tree *GitWorktree, branchname string) {
	branchName := fmt.Sprintf("session-%s", sessionName)
	return &GitWorktree{
		repoPath:     repoPath,
		sessionName:  sessionName,
		branchName:   branchName,
		worktreePath: filepath.Join(filepath.Dir(repoPath), fmt.Sprintf("worktree-%s", sessionName)),
	}, branchName
}

// runGitCommand executes a git command and returns any error
func (g *GitWorktree) runGitCommand(args ...string) error {
	baseArgs := []string{"-C", g.repoPath}
	cmd := exec.Command("git", append(baseArgs, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git command failed: %s (%w)", output, err)
	}
	return nil
}

// Setup creates a new worktree for the session
func (g *GitWorktree) Setup() error {
	// Prevent accidental overwrites
	if _, err := os.Stat(g.worktreePath); !os.IsNotExist(err) {
		return fmt.Errorf("worktree directory already exists at %s", g.worktreePath)
	}

	// Open the repository
	repo, err := git.PlainOpen(g.repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get current HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Clean up any existing branch or reference
	if err := g.cleanupExistingBranch(repo); err != nil {
		return fmt.Errorf("failed to cleanup existing branch: %w", err)
	}

	// Create a new worktree using git command
	if err := g.runGitCommand("worktree", "add", "-b", g.branchName, g.worktreePath, ref.Hash().String()); err != nil {
		return err
	}

	return nil
}

// copyDir recursively copies a directory tree
func (g *GitWorktree) copyDir(src string, dst string) error {
	si, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error getting source directory info for %s: %w", src, err)
	}

	if err := os.MkdirAll(dst, si.Mode()); err != nil {
		return fmt.Errorf("error creating destination directory %s: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error reading source directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := g.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := g.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file using io.Copy for efficient streaming
func (g *GitWorktree) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file %s: %w", src, err)
	}
	defer in.Close()

	si, err := in.Stat()
	if err != nil {
		return fmt.Errorf("error getting source file info for %s: %w", src, err)
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, si.Mode())
	if err != nil {
		return fmt.Errorf("error creating destination file %s: %w", dst, err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			err = fmt.Errorf("error closing destination file %s: %w", dst, cerr)
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("error copying data from %s to %s: %w", src, dst, err)
	}

	return nil
}

// cleanupExistingBranch performs a thorough cleanup of any existing branch or reference
func (g *GitWorktree) cleanupExistingBranch(repo *git.Repository) error {
	branchRef := plumbing.NewBranchReferenceName(g.branchName)

	// Try to remove the branch reference
	if err := repo.Storer.RemoveReference(branchRef); err != nil && err != plumbing.ErrReferenceNotFound {
		return fmt.Errorf("failed to remove branch reference %s: %w", g.branchName, err)
	}

	// Remove any worktree-specific references
	worktreeRef := plumbing.NewReferenceFromStrings(
		fmt.Sprintf("worktrees/%s/HEAD", g.branchName),
		"",
	)
	if err := repo.Storer.RemoveReference(worktreeRef.Name()); err != nil && err != plumbing.ErrReferenceNotFound {
		return fmt.Errorf("failed to remove worktree reference for %s: %w", g.branchName, err)
	}

	// Clean up configuration entries
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repository config: %w", err)
	}

	delete(cfg.Branches, g.branchName)
	worktreeSection := fmt.Sprintf("worktree.%s", g.branchName)
	cfg.Raw.RemoveSection(worktreeSection)

	if err := repo.Storer.SetConfig(cfg); err != nil {
		return fmt.Errorf("failed to update repository config after removing branch %s: %w", g.branchName, err)
	}

	return nil
}

// Cleanup removes the worktree and associated branch
func (g *GitWorktree) Cleanup() error {
	var errs []error

	// Remove the worktree using git command
	if err := g.runGitCommand("worktree", "remove", "-f", g.worktreePath); err != nil {
		errs = append(errs, err)
	}

	// Open the repository for branch cleanup
	repo, err := git.PlainOpen(g.repoPath)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to open repository for cleanup: %w", err))
		return g.combineErrors(errs)
	}

	branchRef := plumbing.NewBranchReferenceName(g.branchName)

	// Check if branch exists before attempting removal
	if _, err := repo.Reference(branchRef, false); err == nil {
		if err := repo.Storer.RemoveReference(branchRef); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove branch %s: %w", g.branchName, err))
		}
	} else if err != plumbing.ErrReferenceNotFound {
		errs = append(errs, fmt.Errorf("error checking branch %s existence: %w", g.branchName, err))
	}

	// Prune the worktree to clean up any remaining references
	if err := g.runGitCommand("worktree", "prune"); err != nil {
		errs = append(errs, err)
	}

	return g.combineErrors(errs)
}

// combineErrors combines multiple errors into a single error
func (g *GitWorktree) combineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	errMsg := "multiple errors occurred:"
	for _, err := range errs {
		errMsg += "\n  - " + err.Error()
	}
	return fmt.Errorf(errMsg)
}

// GetWorktreePath returns the path to the worktree
func (g *GitWorktree) GetWorktreePath() string {
	return g.worktreePath
}

// GetBranchName returns the name of the branch associated with this worktree
func (g *GitWorktree) GetBranchName() string {
	return g.branchName
}

// PushChanges commits and pushes changes in the worktree to the remote branch
func (g *GitWorktree) PushChanges(commitMessage string) error {
	if err := CheckGHCLI(); err != nil {
		return err
	}

	// Stage all changes
	if err := g.runGitCommand("-C", g.worktreePath, "add", "."); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Create commit
	if err := g.runGitCommand("-C", g.worktreePath, "commit", "-m", commitMessage); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push changes using gh cli to handle authentication
	cmd := exec.Command("gh", "repo", "sync", "-b", g.branchName)
	cmd.Dir = g.worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push changes: %s (%w)", output, err)
	}

	return nil
}

// CheckGHCLI checks if GitHub CLI is installed and configured
func CheckGHCLI() error {
	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is not installed. Please install it first")
	}

	// Check if gh is authenticated
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GitHub CLI is not configured. Please run 'gh auth login' first")
	}

	return nil
}
