package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kmrtdsii/playwithantigravity/backend/internal/git"
)

func init() {
	git.RegisterCommand("rm", func() git.Command { return &RmCommand{} })
}

type RmCommand struct{}

func (c *RmCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if len(args) < 2 {
		return "", fmt.Errorf("usage: rm [-rf] <path>")
	}

	// Simple flag parsing
	path := args[len(args)-1]

	// Safety check: Don't allow deleting root or critical paths if possible, though FS is isolated.
	if path == "/" || path == "." || path == ".." {
		return "", fmt.Errorf("refusing to remove root or current directory")
	}

	// Normalize path
	targetPath := path
	if !strings.HasPrefix(targetPath, "/") {
		// Relative path, but for projects at root we usually expect them at root
		// If s.CurrentDir is /, then payload is just the name.
		if s.CurrentDir == "/" {
			targetPath = "/" + targetPath
		} else {
			// Handle relative paths properly if needed, but for project deletion
			// we assume we are likely at root or refer to it by name.
			// For now, let's just assume we are deleting a folder in current dir.
			targetPath = s.CurrentDir
			if targetPath == "/" {
				targetPath = ""
			}
			targetPath = targetPath + "/" + path
		}
	}

	// Check if it exists
	fi, err := s.Filesystem.Stat(targetPath)
	if err != nil {
		return "", fmt.Errorf("path not found: %s", path)
	}

	// Check if it is a directory representing a repo
	if fi.IsDir() {
		// Remove from Repos map if it exists there
		// Name in Repos map is usually the relative name from root (e.g. "json-server")
		// targetPath might be "/json-server"
		repoName := strings.TrimPrefix(targetPath, "/")
		delete(s.Repos, repoName)

		// Remove from Filesystem
		// standard Remove might not be recursive for billy?
		// simple-git-fs usually supports Remove (which might be recursive depending on impl)
		// or we need to implement walk delete.
		// memfs usually requires empty dir for Remove.
		// But let's see if there is RemoveAll.
		// Billy interface usually has Remove.
		// Note: memfs might not verify non-empty?
		// Actually billy's basic Remove often mimics 'rm' which fails on non-empty directories.
		// But we need 'rm -rf'.
		// Let's try attempting a recursive delete helper if needed, or assume library support.
		// checking billy docs (memory): it usually errors on non-empty Remove.
		// We might need to walk it.

		err = s.RemoveAll(targetPath)
		if err != nil {
			return "", fmt.Errorf("failed to remove: %v", err)
		}
	} else {
		// File
		err = s.Filesystem.Remove(targetPath)
		if err != nil {
			return "", fmt.Errorf("failed to remove file: %v", err)
		}
	}

	return fmt.Sprintf("Removed '%s'", path), nil
}

func (c *RmCommand) Help() string {
	return "usage: rm [-rf] <path>\n\nRemove files or directories."
}
