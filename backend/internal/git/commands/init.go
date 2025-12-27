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
	return "", fmt.Errorf("git init is not supported in GitGym. Please use 'git clone' to start a session.")
}

func (c *InitCommand) Help() string {
	return "usage: git init\n\n(Not supported in GitGym)"
}
