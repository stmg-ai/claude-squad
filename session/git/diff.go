package git

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// DiffStats holds statistics about the changes in a diff
type DiffStats struct {
	// Content is the full diff content
	Content string
	// Added is the number of added lines
	Added int
	// Removed is the number of removed lines
	Removed int
	// Error holds any error that occurred during diff computation
	// This allows propagating setup errors (like missing base commit) without breaking the flow
	Error error
}

func (d *DiffStats) IsEmpty() bool {
	return d.Added == 0 && d.Removed == 0 && d.Content == ""
}

// Diff returns the git diff between the worktree and the base branch along with statistics
func (g *GitWorktree) Diff() *DiffStats {
	worktree, baseTree, stats := g.prepareGitObjectsForDiff()
	if stats.Error != nil {
		return stats
	}

	status, err := worktree.Status()
	if err != nil {
		stats.Error = fmt.Errorf("failed to get worktree status: %w", err)
		return stats
	}

	// Create a new buffer to store the diff output
	var diffOutput bytes.Buffer

	statuses, paths := sortStatuses(status)

	// Process each changed file
	for i, fileStatus := range statuses {
		if fileStatus.Worktree == git.Unmodified {
			continue
		}

		filePath := paths[i]

		// Get the file content from the worktree
		var currentContent []byte
		if fileStatus.Worktree != git.Deleted {
			currentContent, err = os.ReadFile(filepath.Join(g.worktreePath, filePath))
			if err != nil {
				stats.Error = fmt.Errorf("failed to read file %s: %w", filePath, err)
				return stats
			}
		}

		// Get the base content
		var baseContent []byte
		if baseFile, err := baseTree.File(filePath); err == nil {
			content, err := baseFile.Contents()
			if err != nil {
				stats.Error = fmt.Errorf("failed to get base content for %s: %w", filePath, err)
				return stats
			}
			baseContent = []byte(content)
		}

		// Write file header
		diffOutput.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
		if fileStatus.Worktree == git.Added {
			diffOutput.WriteString("new file mode 100644\n")
		} else if fileStatus.Worktree == git.Deleted {
			diffOutput.WriteString("deleted file mode 100644\n")
		}
		diffOutput.WriteString("--- a/" + filePath + "\n")
		diffOutput.WriteString("+++ b/" + filePath + "\n")

		// Generate unified diff
		baseLines := strings.Split(string(baseContent), "\n")
		currentLines := strings.Split(string(currentContent), "\n")

		// Simple line-by-line comparison
		var i, j int
		for i < len(baseLines) || j < len(currentLines) {
			if i < len(baseLines) && j < len(currentLines) && baseLines[i] == currentLines[j] {
				diffOutput.WriteString(" " + baseLines[i] + "\n")
				i++
				j++
			} else if i < len(baseLines) {
				diffOutput.WriteString("-" + baseLines[i] + "\n")
				stats.Removed++
				i++
			} else if j < len(currentLines) {
				diffOutput.WriteString("+" + currentLines[j] + "\n")
				stats.Added++
				j++
			}
		}
	}

	stats.Content = diffOutput.String()

	return stats
}

func sortStatuses(status git.Status) ([]*git.FileStatus, []string) {
	paths := make([]string, 0, len(status))
	for path := range status {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	statuses := make([]*git.FileStatus, len(paths))
	for i, path := range paths {
		statuses[i] = status[path]
	}
	return statuses, paths
}

// getGitObjects initializes the diff operation by setting up the repository, worktree and base tree
func (g *GitWorktree) prepareGitObjectsForDiff() (*git.Worktree, *object.Tree, *DiffStats) {
	stats := &DiffStats{}

	if g.baseCommitSHA == "" {
		stats.Error = fmt.Errorf("base commit SHA not set")
		return nil, nil, stats
	}

	// Open the repository using the main repo path, not the worktree path
	repo, err := git.PlainOpen(g.repoPath)
	if err != nil {
		stats.Error = fmt.Errorf("failed to open repository: %w", err)
		return nil, nil, stats
	}

	// Get the base commit
	baseCommit, err := repo.CommitObject(plumbing.NewHash(g.baseCommitSHA))
	if err != nil {
		stats.Error = fmt.Errorf("failed to get base commit: %w", err)
		return nil, nil, stats
	}

	repo, err = git.PlainOpen(g.worktreePath)
	if err != nil {
		stats.Error = fmt.Errorf("failed to open worktree: %w", err)
		return nil, nil, stats
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		stats.Error = fmt.Errorf("failed to get worktree: %w", err)
		return nil, nil, stats
	}

	// Get the base tree
	baseTree, err := baseCommit.Tree()
	if err != nil {
		stats.Error = fmt.Errorf("failed to get base tree: %w", err)
		return nil, nil, stats
	}

	return worktree, baseTree, stats
}
