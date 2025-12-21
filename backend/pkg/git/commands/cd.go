package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("cd", func() git.Command { return &CdCommand{} })
}

type CdCommand struct{}

func (c *CdCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if len(args) < 2 {
		return s.CurrentDir, nil
	}

	target := args[1]
	
	// Handle absolute path
	var newPath string
	if len(target) > 0 && target[0] == '/' {
		newPath = target
	} else {
		newPath = filepath.Join(s.CurrentDir, target)
	}

	// Clean path (handle .., ., etc)
	newPath = filepath.Clean(newPath)
	
	// Ensure valid path (simple check: root exists, subdirs need check)
	if newPath == "/" || newPath == "." {
		s.CurrentDir = "/"
		return "/", nil
	}

	// Check if directory exists in FS
    // Note: memfs might behave weirdly with "Stat" on directories sometimes, but usually works.
	// Stripping leading slash for billy if needed?
    // billy usually takes paths. simple memfs:
    
    // Check if it's one of our known repos
    displayPath := newPath
    if strings.HasPrefix(newPath, "/") {
        newPath = newPath[1:]
    }

	fi, err := s.Filesystem.Stat(newPath)
	if err != nil {
		return "", fmt.Errorf("directory not found: %s", displayPath)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("not a directory: %s", displayPath)
	}

	s.CurrentDir = displayPath // Keep the absolute path convention for state
	return s.CurrentDir, nil
}

func (c *CdCommand) Help() string {
	return "usage: cd <directory>\n\nChange current directory."
}
