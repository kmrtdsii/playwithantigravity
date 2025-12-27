package commands

import (
	"context"
	"fmt"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("worktree", func() git.Command { return &WorktreeCommand{} })
}

type WorktreeCommand struct{}

// Ensure WorktreeCommand implements git.Command
var _ git.Command = (*WorktreeCommand)(nil)

func (c *WorktreeCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	return "", fmt.Errorf("git worktree is not supported in GitGym's current UI.")
}

func (c *WorktreeCommand) Help() string {
	return "usage: git worktree\n\n(Not supported in GitGym)"
}
