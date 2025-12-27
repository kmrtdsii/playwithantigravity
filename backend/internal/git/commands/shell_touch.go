package commands

// shell_touch.go - Shell Command: Create/Update File
//
// This is a SHELL COMMAND (not a git command).
// Creates a new file or updates modification time of existing file.

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("touch", func() git.Command { return &TouchCommand{} })
}

type TouchCommand struct{}

type TouchOptions struct {
	Files []string
}

func (c *TouchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	return c.executeTouch(s, opts)
}

func (c *TouchCommand) parseArgs(args []string) (*TouchOptions, error) {
	cmdArgs := args[1:]
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("usage: touch <file>...")
	}
	return &TouchOptions{Files: cmdArgs}, nil
}

func (c *TouchCommand) executeTouch(s *git.Session, opts *TouchOptions) (string, error) {
	var created []string
	var updated []string

	for _, filename := range opts.Files {
		// Resolve path relative to CurrentDir
		fullPath := filename
		if !strings.HasPrefix(filename, "/") {
			fullPath = path.Join(s.CurrentDir, filename)
		}
		// ensure no leading double slash if CurrentDir is /
		if strings.HasPrefix(fullPath, "//") {
			fullPath = fullPath[1:]
		}

		// Check if file exists
		_, err := s.Filesystem.Stat(fullPath)
		if err != nil {
			// File doesn't exist, create it (empty)
			f, createErr := s.Filesystem.Create(fullPath)
			if createErr != nil {
				return "", createErr
			}
			f.Close()
			created = append(created, filename)
		} else {
			// File exists, try to update modification time
			// Try type assertion for Chtimes
			if changeFs, ok := s.Filesystem.(interface {
				Chtimes(name string, atime time.Time, mtime time.Time) error
			}); ok {
				now := time.Now()
				if err := changeFs.Chtimes(fullPath, now, now); err != nil {
					return "", err
				}
				updated = append(updated, filename)
			} else {
				// No Chtimes support.
				// We do NOT modify content (avoid corruption).
				// We just report updated (simulation bias: success if it exists).
				updated = append(updated, filename)
			}
		}
	}

	if len(created) > 0 {
		return fmt.Sprintf("Created '%s'", strings.Join(created, ", ")), nil
	}
	if len(updated) > 0 {
		return fmt.Sprintf("Updated '%s'", strings.Join(updated, ", ")), nil
	}
	return "", nil
}

func (c *TouchCommand) Help() string {
	return `📘 TOUCH (1)                                            Shell Manual

 💡 DESCRIPTION
    ・空のファイルを新規作成する
    ・ファイルの最終更新日時を更新する

 📋 SYNOPSIS
    touch <file>...

 🛠  EXAMPLES
    1. 新しいファイルを作成
       $ touch newfile.txt
`
}
