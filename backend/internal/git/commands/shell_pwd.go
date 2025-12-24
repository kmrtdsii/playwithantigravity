package commands

import (
	"context"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("pwd", func() git.Command { return &PwdCommand{} })
}

type PwdCommand struct{}

func (c *PwdCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.RLock()
	defer s.RUnlock()

	dir := s.CurrentDir
	if dir == "" {
		return "/", nil
	}
	return dir, nil
}

func (c *PwdCommand) Help() string {
	return "usage: pwd\n\nPrint name of current/working directory."
}
