package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("init", func() git.Command { return &InitCommand{} })
}

type InitCommand struct{}

// Ensure InitCommand implements git.Command
var _ git.Command = (*InitCommand)(nil)

func (c *InitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	var path string
	if len(args) > 1 {
		path = args[1]
	}

	// If no path provided, init in current directory
	if path == "" {
		// Get the directory name from currentDir
		if s.CurrentDir == "/" {
			return "", fmt.Errorf("cannot init repository at root. Run 'mkdir <name>' first, then 'cd <name>' and 'git init'")
		}
		// Current dir is like "/kurobon", strip leading slash
		path = strings.TrimPrefix(s.CurrentDir, "/")
	}

	_, err := s.InitRepo(path)
	if err != nil {
		return "", fmt.Errorf("failed to init repo: %w", err)
	}

	return fmt.Sprintf("Initialized empty Git repository in /%s/.git/", path), nil
}

func (c *InitCommand) Help() string {
	return "usage: git init [directory]\n\nCreate an empty Git repository or reinitialize an existing one."
}
