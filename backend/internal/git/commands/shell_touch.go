package commands

// shell_touch.go - Shell Command: Create/Update File
//
// This is a SHELL COMMAND (not a git command).
// Creates a new file or updates modification time of existing file.

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("touch", func() git.Command { return &TouchCommand{} })
}

type TouchCommand struct{}

func (c *TouchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: touch <filename>")
	}

	filename := args[1]

	s.Lock()
	defer s.Unlock()

	// Resolve path relative to CurrentDir
	fullPath := filename
	if !strings.HasPrefix(filename, "/") {
		// s.CurrentDir usually starts with /
		// path.Join handles clean paths
		fullPath = path.Join(s.CurrentDir, filename)
	}
	// ensure no leading double slash if CurrentDir is /
	if strings.HasPrefix(fullPath, "//") {
		fullPath = fullPath[1:]
	}

	// Check if file exists
	_, err := s.Filesystem.Stat(fullPath)
	if err != nil {
		// File likely doesn't exist, create it (empty)
		f, createErr := s.Filesystem.Create(fullPath)
		if createErr != nil {
			return "", createErr
		}
		f.Close()
		return fmt.Sprintf("Created '%s'", filename), nil
	}

	// File exists, append to it
	f, err := s.Filesystem.OpenFile(fullPath, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.Write([]byte("\n// Update")); err != nil {
		return "", err
	}

	return fmt.Sprintf("Updated '%s'", filename), nil
}

func (c *TouchCommand) Help() string {
	return "usage: touch <filename>\n\nUpdate modifications timestamp of a file or create it."
}
