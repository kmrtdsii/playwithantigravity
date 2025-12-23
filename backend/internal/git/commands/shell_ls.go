package commands

// shell_ls.go - Shell Command: List Directory
//
// This is a SHELL COMMAND (not a git command).
// Lists the contents of the current or specified directory.

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("ls", func() git.Command { return &LsCommand{} })
}

type LsCommand struct{}

func (c *LsCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	path := s.CurrentDir
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	if path == "" {
		path = "."
	}

	infos, err := s.Filesystem.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("ls failed: %w", err)
	}

	var output []string
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() {
			name = name + "/"
		}
		output = append(output, name)
	}

	return strings.Join(output, "\n"), nil
}

func (c *LsCommand) Help() string {
	return "usage: ls\n\nList directory contents."
}
