package commands

import (
	"context"
	"fmt"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("init", func() git.Command { return &InitCommand{} })
}

type InitCommand struct{}

// Ensure InitCommand implements git.Command
var _ git.Command = (*InitCommand)(nil)

func (c *InitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// Initialize repo in current directory ("." relative to session root or current dir)
	path := ""
	if len(args) > 1 {
		path = args[1]
		// Note: args[0] is "init". args[1] might be directory name.
		// If "git init", args=["init"].
		// If "git init foo", args=["init", "foo"].
	}

	_, err := s.InitRepo(path)
	if err != nil {
		return "", fmt.Errorf("failed to init repo: %w", err)
	}
	return fmt.Sprintf("Initialized empty Git repository in %s", path), nil
}

func (c *InitCommand) Help() string {
	return "usage: git init [directory]\n\nCreate an empty Git repository or reinitialize an existing one."
}
