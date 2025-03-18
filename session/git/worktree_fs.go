package git

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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